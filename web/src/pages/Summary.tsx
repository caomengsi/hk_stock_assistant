import { useEffect, useState } from 'react'
import { getMarketSummary } from '../api/stock'
import type { MarketIndexItem } from '../types'

function getColor(change: number) {
  if (change > 0) return '#F44336'
  if (change < 0) return '#4CAF50'
  return '#333'
}

export default function Summary() {
  const [indices, setIndices] = useState<MarketIndexItem[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getMarketSummary()
      .then((r) => setIndices(r.indices || []))
      .catch(() => setIndices([]))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="page">
      <header className="header">
        <h1>大盘总结</h1>
      </header>
      {loading && <p className="muted">加载中…</p>}
      {!loading && indices.length === 0 && <p className="muted">暂无指数数据</p>}
      <div className="index-grid">
        {indices.map((idx) => (
          <div key={idx.name} className="card index-card">
            <div className="index-name">{idx.name}</div>
            <div className="index-value" style={{ color: getColor(idx.change_percent) }}>
              {idx.value.toFixed(2)}
            </div>
            <div className="index-change" style={{ color: getColor(idx.change_percent) }}>
              {idx.change_percent > 0 ? '+' : ''}{idx.change_percent.toFixed(2)}%
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
