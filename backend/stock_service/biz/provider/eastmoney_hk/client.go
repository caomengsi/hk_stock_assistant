package eastmoney_hk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"hk_stock_assistant/backend/stock_service/kitex_gen/stock"
)

// 东方财富 push2 港股实时行情，与券商/华盛通数据源一致，较新浪更实时
// 文档参考: push2.eastmoney.com/api/qt/stock/get

const push2URL = "http://push2.eastmoney.com/api/qt/stock/get"

var httpClient = &http.Client{Timeout: 8 * time.Second}

// NormalizeHKCode 统一为 hk + 5 位数字
func NormalizeHKCode(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))
	if strings.HasPrefix(code, "hk") {
		if len(code) == 7 {
			return code
		}
		if len(code) == 6 {
			return "hk0" + code[2:]
		}
		return code
	}
	if len(code) <= 5 {
		return "hk" + strings.Repeat("0", 5-len(code)) + code
	}
	return "hk" + code
}

// hkCodeToSecID 转为东方财富 secid：港股 116.02513
func hkCodeToSecID(hkCode string) string {
	// hk02513 -> 116.02513
	if strings.HasPrefix(hkCode, "hk") && len(hkCode) >= 7 {
		return "116." + hkCode[2:]
	}
	return "116." + hkCode
}

// push2Data 单只股票数据，API 无数据时可能为 null
type push2Data struct {
	F43 int64   `json:"f43"` // 最新价 * 1000
	F44 int64   `json:"f44"` // 最高 * 1000
	F45 int64   `json:"f45"` // 最低 * 1000
	F46 int64   `json:"f46"` // 今开 * 1000
	F47 int64   `json:"f47"` // 成交量
	F48 float64 `json:"f48"` // 成交额
	F57 string  `json:"f57"` // 代码
	F58 string  `json:"f58"` // 名称
	F60 int64   `json:"f60"` // 昨收 * 1000
}

// push2Resp 与 push2.eastmoney.com/api/qt/stock/get 返回结构一致；data 可能为 null
type push2Resp struct {
	Data *push2Data `json:"data"`
}

// Client 东方财富港股行情
type Client struct{}

// NewClient 创建东方财富港股客户端
func NewClient() *Client {
	return &Client{}
}

// GetStockInfo 获取港股实时行情（与券商数据源一致，较新浪更实时）
func (c *Client) GetStockInfo(ctx context.Context, code string) (*stock.StockInfo, error) {
	code = NormalizeHKCode(code)
	secID := hkCodeToSecID(code)
	// 字段: 最新价,最高,最低,今开,成交量,成交额,代码,名称,昨收
	fields := "f43,f44,f45,f46,f47,f48,f57,f58,f60"
	url := fmt.Sprintf("%s?secid=%s&fields=%s&ut=fa5fd1943c7b386f172d6893dbfba10b", push2URL, secID, fields)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r push2Resp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if r.Data == nil || (r.Data.F57 == "" && r.Data.F58 == "") {
		return nil, fmt.Errorf("invalid code or no data: %s", code)
	}

	// 价格字段为整数，需 /1000 得到元
	currentPrice := float64(r.Data.F43) / 1000
	prevClose := float64(r.Data.F60) / 1000
	changePercent := 0.0
	if prevClose > 0 {
		changePercent = (currentPrice - prevClose) / prevClose * 100
	}

	name := r.Data.F58
	if name == "" {
		name = code
	}
	return &stock.StockInfo{
		Code:          code,
		Name:          name,
		CurrentPrice:  currentPrice,
		ChangePercent: changePercent,
		Volume:        r.Data.F47,
		Timestamp:     "",
	}, nil
}

// indexPush2Data 全球指数 push2 返回（secid=100.HSI 等），价格与涨跌为 *100
type indexPush2Data struct {
	F43  int64  `json:"f43"`  // 最新价 * 100
	F58  string `json:"f58"`  // 名称
	F60  int64  `json:"f60"`  // 昨收 * 100
	F169 int64  `json:"f169"` // 涨跌额 * 100
	F170 int64  `json:"f170"` // 涨跌幅 * 100（如 -0.82 表示 -0.82%）
}

// GetIndexInfo 获取全球指数（如恒生 100.HSI），与东方财富行情页一致
func (c *Client) GetIndexInfo(ctx context.Context, secID string) (name string, value, change, changePercent float64, err error) {
	fields := "f43,f58,f60,f169,f170"
	url := fmt.Sprintf("%s?secid=%s&fields=%s&ut=fa5fd1943c7b386f172d6893dbfba10b", push2URL, secID, fields)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", 0, 0, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, 0, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, 0, err
	}
	var r struct {
		Data *indexPush2Data `json:"data"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.Data == nil {
		return "", 0, 0, 0, fmt.Errorf("invalid index response")
	}
	d := r.Data
	name = d.F58
	if name == "" {
		name = "恒生指数"
	}
	value = float64(d.F43) / 100
	change = float64(d.F169) / 100
	changePercent = float64(d.F170) / 100
	return name, value, change, changePercent, nil
}
