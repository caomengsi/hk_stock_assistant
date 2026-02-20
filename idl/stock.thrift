namespace go stock

struct StockInfo {
    1: string code
    2: string name
    3: double current_price
    4: double change_percent
    5: i64 volume
    6: string timestamp
}

struct GetRealtimeRequest {
    1: string code
}

struct GetRealtimeResponse {
    1: StockInfo stock
}

struct MarketIndex {
    1: string name
    2: double value
    3: double change
    4: double change_percent
}

struct GetMarketSummaryRequest {
}

struct GetMarketSummaryResponse {
    1: list<MarketIndex> indices
}

service StockService {
    GetRealtimeResponse GetRealtime(1: GetRealtimeRequest req)
    GetMarketSummaryResponse GetMarketSummary(1: GetMarketSummaryRequest req)
}
