package predictor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	stock "hk_stock_assistant/backend/stock_service/kitex_gen/stock"
	"hk_stock_assistant/backend/stock_service/kitex_gen/stock/stockservice"
)

// 智谱推理模型（如 GLM-5）响应较慢，默认 120s；可通过环境变量 LLM_TIMEOUT_SEC 覆盖（单位：秒）
var (
	httpClientOnce    sync.Once
	defaultHTTPClient *http.Client
)

func getLLMTimeoutSec() int {
	sec := 120
	if s := os.Getenv("LLM_TIMEOUT_SEC"); s != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && n > 0 {
			sec = n
		}
	}
	return sec
}

func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		sec := getLLMTimeoutSec()
		defaultHTTPClient = &http.Client{Timeout: time.Duration(sec) * time.Second}
	})
	return defaultHTTPClient
}

func loadEnv() {
	dir, _ := os.Getwd()
	if dir == "" {
		dir = "."
	}
	f, err := os.Open(filepath.Join(dir, ".env"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if key != "" && os.Getenv(key) == "" && val != "" {
			os.Setenv(key, val)
		}
	}
}

func init() {
	loadEnv()
}

// Predictor 使用股票服务与可选 LLM 产出港股分析（参考 A 股助手：预拉取数据 + 结构化 prompt）。
type Predictor struct {
	stockClient stockservice.Client
	apiKey      string
	baseURL     string
	model       string
}

const (
	zhipuBaseURL = "https://open.bigmodel.cn/api/paas/v4"
	zhipuModel   = "glm-4-flash"
)

// New 创建 Predictor。优先 ZHIPU_API_KEY（智谱），否则 LLM_API_KEY + LLM_BASE_URL + LLM_MODEL（OpenAI 兼容）。
func New(stockClient stockservice.Client) *Predictor {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	baseURL := zhipuBaseURL
	model := os.Getenv("ZHIPU_MODEL")
	if model == "" {
		model = zhipuModel
	}
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
		baseURL = os.Getenv("LLM_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		model = os.Getenv("LLM_MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
	}
	return &Predictor{stockClient: stockClient, apiKey: apiKey, baseURL: baseURL, model: model}
}

// IsHKTradingTime 判断当前是否港股交易时段（香港时间 9:30-12:00, 13:00-16:00，周一至周五）。
func IsHKTradingTime() bool {
	loc, err := time.LoadLocation("Asia/Hong_Kong")
	if err != nil {
		loc = time.FixedZone("HKT", 8*3600)
	}
	now := time.Now().In(loc)
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	min := now.Hour()*60 + now.Minute()
	// 9:30-12:00
	if min >= 9*60+30 && min < 12*60+0 {
		return true
	}
	// 13:00-16:00
	if min >= 13*60+0 && min < 16*60+0 {
		return true
	}
	return false
}

// fetchStockData 预拉取个股实时行情。
func (p *Predictor) fetchStockData(ctx context.Context, code string) string {
	rpcResp, err := p.stockClient.GetRealtime(ctx, &stock.GetRealtimeRequest{Code: code})
	if err != nil {
		return fmt.Sprintf("获取行情失败: %v", err)
	}
	if rpcResp == nil || rpcResp.Stock == nil {
		return "无行情数据"
	}
	s := rpcResp.Stock
	return fmt.Sprintf("名称=%s, 代码=%s, 现价=%.2f, 涨跌幅=%.2f%%, 成交量=%d",
		s.Name, s.Code, s.CurrentPrice, s.ChangePercent, s.Volume)
}

// fetchMarketData 预拉取大盘指数（恒生等）。
func (p *Predictor) fetchMarketData(ctx context.Context) string {
	rpcResp, err := p.stockClient.GetMarketSummary(ctx, &stock.GetMarketSummaryRequest{})
	if err != nil {
		return fmt.Sprintf("获取大盘失败: %v", err)
	}
	if rpcResp == nil || len(rpcResp.Indices) == 0 {
		return "无大盘数据"
	}
	var lines []string
	for _, idx := range rpcResp.Indices {
		if idx == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %.2f, 涨跌%.2f%%, 变动%.2f",
			idx.Name, idx.Value, idx.ChangePercent, idx.Change))
	}
	return strings.Join(lines, "\n")
}

// Predict 返回 analysis, confidence, newsSummary。
func (p *Predictor) Predict(ctx context.Context, code string, days int32, modelOverride string) (string, float64, string, error) {
	log.Printf("[Predict] start code=%s days=%d", code, days)
	// 1. 预拉取数据（参考 A 股：先拿齐再拼 prompt）
	stockStr := p.fetchStockData(ctx, code)
	marketStr := p.fetchMarketData(ctx)
	log.Printf("[Predict] data fetched, stock=%s", truncate(stockStr, 80))

	// 2. 无 API Key 时返回占位
	if p.apiKey == "" {
		return fmt.Sprintf("【港股 %s】\n当前数据：%s\n\n大盘：\n%s\n\n请设置环境变量 ZHIPU_API_KEY 或 LLM_API_KEY 后使用 AI 预测。", code, stockStr, marketStr),
			0.5, "参见分析内容。", nil
	}

	// 3. 港股交易时段与预测焦点
	isTrading := IsHKTradingTime()
	tradingStatusStr := "港股休市"
	predictionFocus := "未来 1 个交易日及未来 " + fmt.Sprintf("%d", days) + " 天走势"
	timeInstruction := `
- 当前状态：港股休市（盘后/周末）
- 重点：结合全日表现与大盘环境，给出下一交易日及未来数日的展望。
`

	if isTrading {
		tradingStatusStr = "港股盘中（9:30-12:00, 13:00-16:00 香港时间）"
		predictionFocus = "今日收盘走势及未来 " + fmt.Sprintf("%d", days) + " 天"
		timeInstruction = `
- 当前状态：港股盘中交易中
- 重点：结合实时价格、涨跌幅、成交量与大盘联动，判断尾盘及短期方向。
`
	}

	model := modelOverride
	if model == "" {
		model = p.model
	}

	// 4. 结构化 prompt（参考 A 股：角色 + 时间 + 数据块 + 分析框架 + 输出要求）
	prompt := fmt.Sprintf(`你是一位港股分析专家（专业基金经理水平）。请根据以下数据对港股 %s 做简明分析与预测。

当前时间与状态：%s（%s）

[个股实时数据]
%s

[大盘指数]
%s

请按以下逻辑组织回答（不必逐条标题，但需覆盖要点）：
1. 时间与大盘环境：结合当前是否盘中、大盘涨跌，说明对个股的影响。
2. 个股逻辑：价格、涨跌幅、成交量反映的资金与情绪。
3. 风险提示：若波动剧烈或大盘偏弱，需提示风险。
4. 预测：对「%s」给出方向判断（看多/看空/震荡）及简要理由。
5. 置信度：0～1 之间的数值。

输出要求：
- 语言：简体中文。
- 风格：专业、客观、简洁（2～4 段即可）。
- 不要编造未提供的数据。
`, code, time.Now().Format("2006-01-02 15:04:05"), tradingStatusStr, stockStr, marketStr, predictionFocus)
	prompt += "\n" + strings.TrimSpace(timeInstruction) + "\n\n请直接输出你的分析结论。"

	// 5. 调用 OpenAI 兼容 API
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 4000, // 推理模型（如 GLM-5）需更多 token，避免 finish_reason=length 时仅 reasoning_content 有内容
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", 0, "", fmt.Errorf("构建请求体: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	// 使用独立 context，避免调用方（网关）RPC 超时后取消导致 LLM 请求被取消
	timeoutSec := getLLMTimeoutSec()
	llmCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	req = req.WithContext(llmCtx)
	log.Printf("[Predict] calling LLM model=%s (timeout=%ds)", model, timeoutSec)
	resp, err := getHTTPClient().Do(req)
	if err != nil {
		log.Printf("[Predict] LLM request error: %v", err)
		return "", 0, "", fmt.Errorf("调用 LLM 失败: %w", err)
	}
	defer resp.Body.Close()
	log.Printf("[Predict] LLM response status=%d", resp.StatusCode)
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, "", fmt.Errorf("读取 LLM 响应: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, "", fmt.Errorf("LLM 返回 %d: %s", resp.StatusCode, string(respBytes))
	}
	var out struct {
		Error *struct {
			Message string `json:"message"`
			Code   string `json:"code"`
		} `json:"error"`
		Choices []struct {
			Message struct {
				Content          interface{} `json:"content"`           // 最终回答
				ReasoningContent string      `json:"reasoning_content"` // 智谱推理模型（如 GLM-5）的思考过程，finish_reason=length 时可能只有此项
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return "", 0, "", fmt.Errorf("解析 LLM 响应: %w", err)
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", 0, "", fmt.Errorf("LLM 错误: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		log.Printf("[Predict] LLM 响应无 choices，原始响应(前500字): %s", truncate(string(respBytes), 500))
		return "", 0, "", fmt.Errorf("LLM 未返回内容")
	}
	content := out.Choices[0].Message.Content
	analysis := contentToString(content)
	analysis = strings.TrimSpace(analysis)
	// 智谱推理模型在 finish_reason=length 时可能只填了 reasoning_content，content 为空，则用推理内容作为分析
	if analysis == "" {
		analysis = strings.TrimSpace(out.Choices[0].Message.ReasoningContent)
	}
	if analysis == "" {
		log.Printf("[Predict] LLM 返回 content 为空, finish_reason=%s, 原始响应(前500字): %s",
			out.Choices[0].FinishReason, truncate(string(respBytes), 500))
		return "", 0, "", fmt.Errorf("LLM 返回内容为空（可能触发内容策略或模型限制，请稍后重试或换用其他模型）")
	}
	return analysis, 0.85, "参见分析内容。", nil
}

// buildPromptForLLM 返回 (prompt, model, error)。无 API Key 时返回 error。
func (p *Predictor) buildPromptForLLM(ctx context.Context, code string, days int32, modelOverride string) (prompt, model string, err error) {
	stockStr := p.fetchStockData(ctx, code)
	marketStr := p.fetchMarketData(ctx)
	if p.apiKey == "" {
		return "", "", fmt.Errorf("未配置 ZHIPU_API_KEY 或 LLM_API_KEY")
	}
	isTrading := IsHKTradingTime()
	tradingStatusStr := "港股休市"
	predictionFocus := "未来 1 个交易日及未来 " + fmt.Sprintf("%d", days) + " 天走势"
	timeInstruction := "- 当前状态：港股休市（盘后/周末）\n- 重点：结合全日表现与大盘环境，给出下一交易日及未来数日的展望。"
	if isTrading {
		tradingStatusStr = "港股盘中（9:30-12:00, 13:00-16:00 香港时间）"
		predictionFocus = "今日收盘走势及未来 " + fmt.Sprintf("%d", days) + " 天"
		timeInstruction = "- 当前状态：港股盘中交易中\n- 重点：结合实时价格、涨跌幅、成交量与大盘联动，判断尾盘及短期方向。"
	}
	model = modelOverride
	if model == "" {
		model = p.model
	}
	prompt = fmt.Sprintf(`你是一位港股分析专家（专业基金经理水平）。请根据以下数据对港股 %s 做简明分析与预测。

当前时间与状态：%s（%s）

[个股实时数据]
%s

[大盘指数]
%s

请按以下逻辑组织回答（不必逐条标题，但需覆盖要点）：
1. 时间与大盘环境：结合当前是否盘中、大盘涨跌，说明对个股的影响。
2. 个股逻辑：价格、涨跌幅、成交量反映的资金与情绪。
3. 风险提示：若波动剧烈或大盘偏弱，需提示风险。
4. 预测：对「%s」给出方向判断（看多/看空/震荡）及简要理由。
5. 置信度：0～1 之间的数值。

输出要求：
- 语言：简体中文。
- 风格：专业、客观、简洁（2～4 段即可）。
- 不要编造未提供的数据。
%s

请直接输出你的分析结论。`, code, time.Now().Format("2006-01-02 15:04:05"), tradingStatusStr, stockStr, marketStr, predictionFocus, timeInstruction)
	return prompt, model, nil
}

// StreamPredict 流式调用 LLM，每收到一段内容就调用 onChunk(delta)；智谱/OpenAI 兼容 stream 格式。
func (p *Predictor) StreamPredict(ctx context.Context, code string, days int32, modelOverride string, onChunk func(string) error) error {
	prompt, model, err := p.buildPromptForLLM(ctx, code, days, modelOverride)
	if err != nil {
		return err
	}
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 4000,
		"stream":     true,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("构建请求体: %w", err)
	}
	timeoutSec := getLLMTimeoutSec()
	llmCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(llmCtx, "POST", p.baseURL+"/chat/completions", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	log.Printf("[StreamPredict] calling LLM model=%s stream=true", model)
	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("调用 LLM 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bs, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM 返回 %d: %s", resp.StatusCode, string(bs))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"` // 智谱 GLM-5 流式推理内容
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		text := delta.Content
		if text == "" && delta.ReasoningContent != "" {
			text = delta.ReasoningContent
		}
		if text != "" && onChunk != nil {
			if err := onChunk(text); err != nil {
				return err
			}
		}
	}
	return sc.Err()
}

func contentToString(c interface{}) string {
	if c == nil {
		return ""
	}
	switch v := c.(type) {
	case string:
		return v
	case []interface{}:
		var b strings.Builder
		for _, part := range v {
			if m, ok := part.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					b.WriteString(t)
				}
			}
		}
		return b.String()
	default:
		return fmt.Sprint(c)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
