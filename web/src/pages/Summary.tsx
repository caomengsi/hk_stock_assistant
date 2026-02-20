import { useEffect, useState } from 'react'
import { getMarketSummary, getSectors } from '../api/stock'
import type { MarketIndexItem, SectorItem, SectorsResponse } from '../types'

function getColor(change: number) {
  if (change > 0) return '#F44336'
  if (change < 0) return '#4CAF50'
  return '#333'
}

function formatFlow(v: number): string {
  const abs = Math.abs(v)
  const sign = v < 0 ? '-' : ''
  if (abs >= 1e8) return sign + (abs / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return sign + (abs / 1e4).toFixed(2) + '万'
  return String(Math.round(v))
}

type SectorTab = 'change' | 'capital'

export default function Summary() {
  const [indices, setIndices] = useState<MarketIndexItem[]>([])
  const [sectors, setSectors] = useState<SectorsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [sectorsLoading, setSectorsLoading] = useState(true)
  const [sectorTab, setSectorTab] = useState<SectorTab>('change')

  useEffect(() => {
    getMarketSummary()
      .then((r) => setIndices(r.indices || []))
      .catch(() => setIndices([]))
      .finally(() => setLoading(false))
    const timer = window.setInterval(() => {
      getMarketSummary()
        .then((r) => setIndices(r.indices || []))
        .catch(() => {})
    }, 2000)
    return () => window.clearInterval(timer)
  }, [])

  useEffect(() => {
    getSectors()
      .then(setSectors)
      .catch(() => setSectors(null))
      .finally(() => setSectorsLoading(false))
  }, [])

  return (
    <div className="page">
      <header className="header">
        <h1>大盘总结</h1>
        {indices.length > 0 && (
          <span className="muted" style={{ fontSize: 12, fontWeight: 'normal' }}>每 2 秒自动刷新</span>
        )}
      </header>
      {loading && <p className="muted">加载中…</p>}
      {!loading && indices.length === 0 && <p className="muted">暂无指数数据</p>}
      <div className="index-grid">
        {indices.map((idx) => (
          <div key={idx.name} className="card index-card">
            <div className="index-name">{idx.name}</div>
            <div className="index-value">
              {idx.value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
            </div>
            <div className="index-change" style={{ color: getColor(idx.change_percent) }}>
              {idx.change > 0 ? '+' : ''}{idx.change.toFixed(2)}（{idx.change_percent > 0 ? '+' : ''}{idx.change_percent.toFixed(2)}%）{idx.change_percent > 0 ? '↑' : idx.change_percent < 0 ? '↓' : ''}
            </div>
          </div>
        ))}
      </div>

      <section className="sector-section">
        <h2 className="section-title">港股涨跌与资金流向</h2>
        {sectorsLoading && <p className="muted">港股数据加载中…</p>}
        {!sectorsLoading && sectors && (
          <div className="card">
            <div className="sector-tabs">
              <button
                type="button"
                className={`sector-tab ${sectorTab === 'change' ? 'active' : ''}`}
                onClick={() => setSectorTab('change')}
              >
                涨跌幅排行
              </button>
              <button
                type="button"
                className={`sector-tab ${sectorTab === 'capital' ? 'active' : ''}`}
                onClick={() => setSectorTab('capital')}
              >
                资金流向
              </button>
            </div>
            <div className="sector-tab-panel">
            {sectorTab === 'change' && (
              <ul className="sector-list sector-list-fixed">
                {(sectors.by_change || []).slice(0, 20).map((s: SectorItem) => (
                  <li key={s.code} className="sector-row">
                    <span className="sector-name">
                      {s.name}
                      <span className="sector-code">{s.code}</span>
                    </span>
                    <span className="sector-value">{typeof s.value === 'number' ? s.value.toFixed(2) : '—'}</span>
                    <span style={{ color: getColor(s.change_percent) }}>
                      {s.change_percent > 0 ? '+' : ''}{s.change_percent.toFixed(2)}%
                    </span>
                  </li>
                ))}
              </ul>
            )}
            {sectorTab === 'capital' && (
              <ul className="sector-list sector-list-fixed">
                  {(sectors.by_capital || []).map((s: SectorItem) => {
                    const totalNet = (s.main_net_inflow ?? 0) + (s.retail_net_inflow ?? 0)
                    return (
                      <li key={s.code} className="sector-row-capital-block">
                        <div className="sector-row sector-row-capital-line1">
                          <span className="sector-name">
                            {s.name}
                            <span className="sector-code">{s.code}</span>
                          </span>
                          <span className="capital-flow-item">
                            <span className="capital-flow-label">净流入</span>
                            <span style={{ color: totalNet >= 0 ? '#F44336' : '#4CAF50' }}>
                              {totalNet >= 0 ? '+' : ''}{formatFlow(totalNet)}
                            </span>
                          </span>
                          <span className="capital-flow-item" style={{ minWidth: '3.5rem' }}>
                            <span style={{ color: getColor(s.change_percent) }}>
                              {s.change_percent > 0 ? '+' : ''}{s.change_percent.toFixed(2)}%
                            </span>
                          </span>
                        </div>
                        <div className="sector-row-capital-line2">
                          <span className="capital-flow-item">
                            <span className="capital-flow-label">主力</span>
                            <span style={{ color: (s.main_net_inflow ?? 0) >= 0 ? '#F44336' : '#4CAF50' }}>
                              {(s.main_net_inflow ?? 0) >= 0 ? '+' : ''}{formatFlow(s.main_net_inflow ?? 0)}
                            </span>
                          </span>
                          <span className="capital-flow-item">
                            <span className="capital-flow-label">散户</span>
                            <span style={{ color: (s.retail_net_inflow ?? 0) >= 0 ? '#F44336' : '#4CAF50' }}>
                              {(s.retail_net_inflow ?? 0) >= 0 ? '+' : ''}{formatFlow(s.retail_net_inflow ?? 0)}
                            </span>
                          </span>
                        </div>
                      </li>
                    )
                  })}
                </ul>
            )}
            </div>
          </div>
        )}
      </section>
    </div>
  )
}
