package main

import (
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"hk_stock_assistant/backend/gateway/biz/rpc"
)

func main() {
	rpc.InitStock()
	rpc.InitAI()
	h := server.Default(
		server.WithHostPorts(":8080"),
		server.WithReadTimeout(60*time.Second),
		server.WithWriteTimeout(60*time.Second),
		server.WithIdleTimeout(60*time.Second),
	)
	register(h)
	h.Spin()
}
