namespace go api

struct RealtimeResponse {
    1: string code
    2: string name
    3: double current_price
    4: double change_percent
    5: i64 volume
    6: string timestamp
}

struct GetRealtimeRequest {
    1: string code (api.path="code")
}

struct MarketIndexItem {
    1: string name
    2: double value
    3: double change
    4: double change_percent
}

struct MarketSummaryResponse {
    1: list<MarketIndexItem> indices
}

struct GetMarketSummaryRequest {
}

struct PredictionRequest {
    1: string code (api.path="code")
    2: i32 days (api.body="days")
    3: bool include_news (api.body="include_news")
    4: string model (api.body="model")
}

struct PredictionResponse {
    1: string code
    2: double confidence
    3: string analysis
    4: string news_summary
}

service StockAPI {
    RealtimeResponse GetRealtime(1: GetRealtimeRequest req) (api.get="/api/stocks/:code/realtime")
    MarketSummaryResponse GetMarketSummary(1: GetMarketSummaryRequest req) (api.get="/api/market/summary")
    PredictionResponse GetPrediction(1: PredictionRequest req) (api.post="/api/prediction/:code")
}
