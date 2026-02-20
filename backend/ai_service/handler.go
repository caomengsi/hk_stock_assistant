package main

import (
	"context"
	"log"

	ai "hk_stock_assistant/backend/ai_service/kitex_gen/ai"
	"hk_stock_assistant/backend/ai_service/biz/predictor"
	"hk_stock_assistant/backend/stock_service/kitex_gen/stock/stockservice"
)

// AIServiceImpl implements ai.AIService
type AIServiceImpl struct {
	stockClient stockservice.Client
	predictor   *predictor.Predictor
}

func NewAIServiceImpl(stockClient stockservice.Client, p *predictor.Predictor) *AIServiceImpl {
	return &AIServiceImpl{stockClient: stockClient, predictor: p}
}

func (s *AIServiceImpl) GetPrediction(ctx context.Context, req *ai.GetPredictionRequest) (*ai.GetPredictionResponse, error) {
	log.Printf("GetPrediction: code=%s", req.Code)
	analysis, confidence, newsSummary, err := s.predictor.Predict(ctx, req.Code, req.Days, req.Model)
	if err != nil {
		return nil, err
	}
	return &ai.GetPredictionResponse{
		Result_: &ai.PredictionResult_{
			Code:         req.Code,
			Confidence:   confidence,
			Analysis:     analysis,
			NewsSummary_: newsSummary,
		},
	}, nil
}
