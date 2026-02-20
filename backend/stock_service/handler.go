package main

import (
	"context"
	"fmt"

	"hk_stock_assistant/backend/stock_service/biz/provider/eastmoney_hk"
	"hk_stock_assistant/backend/stock_service/biz/provider/sina_hk"
	stock "hk_stock_assistant/backend/stock_service/kitex_gen/stock"
)

// StockServiceImpl implements stock.StockService
// 个股实时行情使用东方财富 push2（与券商/华盛通一致、更实时），大盘指数仍用新浪
type StockServiceImpl struct {
	stockClient *eastmoney_hk.Client
	indexClient *sina_hk.Client
}

// NewStockServiceImpl creates a new StockServiceImpl
func NewStockServiceImpl() *StockServiceImpl {
	return &StockServiceImpl{
		stockClient: eastmoney_hk.NewClient(),
		indexClient: sina_hk.NewClient(),
	}
}

// GetRealtime implements stock.StockService（东方财富数据源，更接近华盛通等券商）
func (s *StockServiceImpl) GetRealtime(ctx context.Context, req *stock.GetRealtimeRequest) (*stock.GetRealtimeResponse, error) {
	if req == nil || req.Code == "" {
		return &stock.GetRealtimeResponse{}, nil
	}
	info, err := s.stockClient.GetStockInfo(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	return &stock.GetRealtimeResponse{Stock: info}, nil
}

// GetMarketSummary implements stock.StockService（恒生指数 + 恒生科技指数，优先东方财富）
func (s *StockServiceImpl) GetMarketSummary(ctx context.Context, req *stock.GetMarketSummaryRequest) (*stock.GetMarketSummaryResponse, error) {
	indices := make([]*stock.MarketIndex, 0)
	// 恒生指数：优先东方财富 100.HSI，失败则新浪 int_hangseng
	name, value, change, changePct, err := s.stockClient.GetIndexInfo(ctx, "100.HSI")
	if err != nil {
		name, value, change, changePct, err = s.indexClient.GetIndexInfo(ctx, "int_hangseng")
	}
	if err == nil {
		indices = append(indices, &stock.MarketIndex{
			Name:          name,
			Value:         value,
			Change:        change,
			ChangePercent: changePct,
		})
	}
	// 恒生科技指数：东方财富 124.HSTECH
	if name2, value2, change2, changePct2, err2 := s.stockClient.GetIndexInfo(ctx, "124.HSTECH"); err2 == nil {
		indices = append(indices, &stock.MarketIndex{
			Name:          name2,
			Value:         value2,
			Change:        change2,
			ChangePercent: changePct2,
		})
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("failed to fetch any HK market indices")
	}
	return &stock.GetMarketSummaryResponse{Indices: indices}, nil
}
