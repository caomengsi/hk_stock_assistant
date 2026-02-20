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

// GetMarketSummary implements stock.StockService（指数仍用新浪）
func (s *StockServiceImpl) GetMarketSummary(ctx context.Context, req *stock.GetMarketSummaryRequest) (*stock.GetMarketSummaryResponse, error) {
	indices := make([]*stock.MarketIndex, 0)
	// Hang Seng Index
	name, value, changePct, err := s.indexClient.GetIndexInfo(ctx, "int_hangseng")
	if err == nil {
		indices = append(indices, &stock.MarketIndex{
			Name:          name,
			Value:         value,
			Change:        value * changePct / 100,
			ChangePercent: changePct,
		})
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("failed to fetch any HK market indices")
	}
	return &stock.GetMarketSummaryResponse{Indices: indices}, nil
}
