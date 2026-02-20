module hk_stock_assistant/backend/gateway

go 1.21

require (
	github.com/cloudwego/hertz v0.10.0
	github.com/cloudwego/kitex v0.15.4
	hk_stock_assistant/backend/ai_service v0.0.0
	hk_stock_assistant/backend/stock_service v0.0.0
)

replace (
	hk_stock_assistant/backend/ai_service => ../ai_service
	hk_stock_assistant/backend/stock_service => ../stock_service
)
