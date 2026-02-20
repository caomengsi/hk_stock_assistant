package main

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"hk_stock_assistant/backend/gateway/biz/handler/api"
)

func register(r *server.Hertz) {
	r.GET("/ping", func(c context.Context, ctx *app.RequestContext) {
		ctx.String(consts.StatusOK, "pong")
	})
	apiGroup := r.Group("/api")
	apiGroup.GET("/stocks/:code/realtime", api.GetRealtime)
	apiGroup.GET("/market/summary", api.GetMarketSummary)
	apiGroup.POST("/prediction/:code", api.GetPrediction)
	apiGroup.POST("/prediction/:code/stream", api.GetPredictionStream)
}
