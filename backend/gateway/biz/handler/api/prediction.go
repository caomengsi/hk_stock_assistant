package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/protocol/http1/resp"
	"hk_stock_assistant/backend/ai_service/kitex_gen/ai"
	"hk_stock_assistant/backend/gateway/biz/rpc"
)

const streamBackendURL = "http://127.0.0.1:8890/stream"

// PredictionBody request body for POST /api/prediction/:code
type PredictionBody struct {
	Days        int32  `json:"days"`
	IncludeNews bool   `json:"include_news"`
	Model       string `json:"model"`
}

// GetPrediction POST /api/prediction/:code
func GetPrediction(ctx context.Context, c *app.RequestContext) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.String(consts.StatusBadRequest, "missing code")
		return
	}
	code = normalizeHKCode(code)

	var body PredictionBody
	_ = c.BindJSON(&body)
	if body.Days <= 0 {
		body.Days = 3
	}

	rpcReq := &ai.GetPredictionRequest{
		Code:        code,
		Days:        body.Days,
		IncludeNews: body.IncludeNews,
		Model:       body.Model,
	}
	rpcResp, err := rpc.AIClient.GetPrediction(ctx, rpcReq)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}
	if rpcResp.Result_ == nil {
		c.String(consts.StatusInternalServerError, "AI service returned empty result")
		return
	}
	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":         rpcResp.Result_.Code,
		"confidence":   rpcResp.Result_.Confidence,
		"analysis":     rpcResp.Result_.Analysis,
		"news_summary": rpcResp.Result_.NewsSummary_,
	})
}

// GetPredictionStream POST /api/prediction/:code/stream，流式返回 SSE。
func GetPredictionStream(ctx context.Context, c *app.RequestContext) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.String(consts.StatusBadRequest, "missing code")
		return
	}
	code = normalizeHKCode(code)
	var body PredictionBody
	_ = c.BindJSON(&body)
	if body.Days <= 0 {
		body.Days = 3
	}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"code":  code,
		"days":  body.Days,
		"model": body.Model,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, streamBackendURL, bytes.NewReader(reqBody))
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	backendResp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}
	defer backendResp.Body.Close()
	if backendResp.StatusCode != http.StatusOK {
		bs, _ := io.ReadAll(backendResp.Body)
		c.String(backendResp.StatusCode, string(bs))
		return
	}
	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	c.Response.HijackWriter(resp.NewChunkedBodyWriter(&c.Response, c.GetWriter()))
	buf := make([]byte, 4096)
	for {
		n, err := backendResp.Body.Read(buf)
		if n > 0 {
			_, _ = c.Write(buf[:n])
			_ = c.Flush()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
}
