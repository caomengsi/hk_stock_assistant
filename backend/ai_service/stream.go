package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"hk_stock_assistant/backend/ai_service/biz/predictor"
)

const streamAddr = ":8890"

// RunStreamServer 启动 HTTP 流式预测服务，供网关代理 SSE。
func RunStreamServer(p *predictor.Predictor) {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) { handleStream(p, w, r) })
	log.Printf("[stream] listening on %s", streamAddr)
	if err := http.ListenAndServe(streamAddr, mux); err != nil {
		log.Printf("[stream] server error: %v", err)
	}
}

func handleStream(p *predictor.Predictor, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	days := int32(3)
	modelOverride := ""
	if r.Method == http.MethodPost && r.Body != nil {
		var body struct {
			Code  string `json:"code"`
			Days  int32  `json:"days"`
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		r.Body.Close()
		code = strings.TrimSpace(body.Code)
		if body.Days > 0 {
			days = body.Days
		}
		modelOverride = strings.TrimSpace(body.Model)
	} else {
		// GET: code 来自 query，days/model 可扩展
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
	}
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	if p == nil {
		writeSSE(w, flusher, "error", "predictor not initialized")
		return
	}
	err := p.StreamPredict(r.Context(), code, days, modelOverride, func(eventType string, chunk string) error {
		return writeSSE(w, flusher, eventType, chunk)
	})
	if err != nil {
		writeSSE(w, flusher, "error", err.Error())
		return
	}
	writeSSE(w, flusher, "done", "")
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event, data string) error {
	if event != "" {
		if _, err := w.Write([]byte("event: " + event + "\n")); err != nil {
			return err
		}
	}
	// data 行：对内容做 JSON 编码避免换行等问题
	payload, _ := json.Marshal(data)
	if _, err := w.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
