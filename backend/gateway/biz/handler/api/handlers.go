package api

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"hk_stock_assistant/backend/gateway/biz/rpc"
	"hk_stock_assistant/backend/stock_service/kitex_gen/stock"
)

// GetRealtime GET /api/stocks/:code/realtime
func GetRealtime(ctx context.Context, c *app.RequestContext) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		c.String(consts.StatusBadRequest, "missing code")
		return
	}
	code = normalizeHKCode(code)

	rpcReq := &stock.GetRealtimeRequest{Code: code}
	rpcResp, err := rpc.StockClient.GetRealtime(ctx, rpcReq)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}
	if rpcResp.Stock == nil {
		c.String(consts.StatusNotFound, "stock not found")
		return
	}
	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":           rpcResp.Stock.Code,
		"name":           rpcResp.Stock.Name,
		"current_price":  rpcResp.Stock.CurrentPrice,
		"change_percent": rpcResp.Stock.ChangePercent,
		"volume":         rpcResp.Stock.Volume,
		"timestamp":      rpcResp.Stock.Timestamp,
	})
}

// GetMarketSummary GET /api/market/summary
func GetMarketSummary(ctx context.Context, c *app.RequestContext) {
	rpcResp, err := rpc.StockClient.GetMarketSummary(ctx, &stock.GetMarketSummaryRequest{})
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}
	indices := make([]map[string]interface{}, 0, len(rpcResp.Indices))
	for _, idx := range rpcResp.Indices {
		indices = append(indices, map[string]interface{}{
			"name":            idx.Name,
			"value":          idx.Value,
			"change":         idx.Change,
			"change_percent": idx.ChangePercent,
		})
	}
	c.JSON(consts.StatusOK, map[string]interface{}{"indices": indices})
}

func normalizeHKCode(code string) string {
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
