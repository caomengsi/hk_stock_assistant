package rpc

import (
	"sync"
	"time"

	"github.com/cloudwego/kitex/client"
	"hk_stock_assistant/backend/ai_service/kitex_gen/ai/aiservice"
)

var (
	AIClient aiservice.Client
	aiOnce   sync.Once
)

func InitAI() {
	aiOnce.Do(func() {
		var err error
		AIClient, err = aiservice.NewClient("ai_service",
			client.WithHostPorts("127.0.0.1:8889"),
			client.WithRPCTimeout(180*time.Second), // 智谱推理模型响应慢，需与 LLM_TIMEOUT_SEC 匹配
			client.WithConnectTimeout(3*time.Second),
		)
		if err != nil {
			panic(err)
		}
	})
}
