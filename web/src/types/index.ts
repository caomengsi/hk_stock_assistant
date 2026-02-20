export interface RealtimeResponse {
  code: string
  name: string
  current_price: number
  change_percent: number
  volume: number
  timestamp: string
}

export interface MarketIndexItem {
  name: string
  value: number
  change: number
  change_percent: number
}

export interface MarketSummaryResponse {
  indices: MarketIndexItem[]
}

export interface PredictionResponse {
  code: string
  confidence: number
  analysis: string
  news_summary: string
}

export interface PredictionRequest {
  code: string
  days: number
  include_news: boolean
  model: string
}

export interface SectorItem {
  code: string
  name: string
  value: number
  change_percent: number
  main_net_inflow: number
  main_net_ratio: number
  retail_net_inflow?: number
}

export interface SectorsResponse {
  hot: SectorItem[]
  by_change: SectorItem[]
  by_capital: SectorItem[]
}
