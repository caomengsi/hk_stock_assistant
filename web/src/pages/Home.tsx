import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { getRealtime } from '../api/stock'
import type { RealtimeResponse } from '../types'

const WATCHLIST_KEY = 'hk_watchlist'
const DEFAULT_LIST = ['hk00700', 'hk09988', 'hk09618']

function loadWatchlist(): string[] {
  try {
    const s = localStorage.getItem(WATCHLIST_KEY)
    if (s) return [...new Set(JSON.parse(s) as string[])]
  } catch (_) {}
  return DEFAULT_LIST
}

function saveWatchlist(list: string[]) {
  localStorage.setItem(WATCHLIST_KEY, JSON.stringify(list))
}

function getColor(change: number) {
  if (change > 0) return '#F44336'
  if (change < 0) return '#4CAF50'
  return '#333'
}

export default function Home() {
  const [watchlist, setWatchlist] = useState<string[]>(loadWatchlist)
  const [stocks, setStocks] = useState<RealtimeResponse[]>([])
  const [loading, setLoading] = useState(false)
  const [addCode, setAddCode] = useState('')
  const [showAdd, setShowAdd] = useState(false)

  useEffect(() => {
    saveWatchlist(watchlist)
  }, [watchlist])

  const watchlistRef = useRef(watchlist)
  watchlistRef.current = watchlist

  const fetchStocks = async (silent = false) => {
    const list = watchlistRef.current
    if (list.length === 0) return
    if (!silent) setLoading(true)
    try {
      const results = await Promise.allSettled(list.map((code) => getRealtime(code)))
      const out: RealtimeResponse[] = []
      const seen = new Set<string>()
      results.forEach((r) => {
        if (r.status === 'fulfilled' && !seen.has(r.value.code)) {
          seen.add(r.value.code)
          out.push(r.value)
        }
      })
      setStocks(out)
    } finally {
      if (!silent) setLoading(false)
    }
  }

  useEffect(() => {
    if (watchlist.length === 0) return
    fetchStocks()
    const timer = window.setInterval(() => {
      const list = watchlistRef.current
      if (list.length === 0) return
      Promise.allSettled(list.map((code) => getRealtime(code)))
        .then((results) => {
          const out: RealtimeResponse[] = []
          const seen = new Set<string>()
          results.forEach((r) => {
            if (r.status === 'fulfilled' && !seen.has(r.value.code)) {
              seen.add(r.value.code)
              out.push(r.value)
            }
          })
          setStocks(out)
        })
        .catch(() => {})
    }, 2000)
    return () => window.clearInterval(timer)
  }, [watchlist])

  const addStock = () => {
    let code = addCode.trim().toLowerCase()
    if (!code) return
    if (!code.startsWith('hk')) code = 'hk' + (code.length <= 5 ? '0'.repeat(5 - code.length) + code : code)
    if (!watchlist.includes(code)) {
      setWatchlist((prev) => [...prev, code])
    }
    setAddCode('')
    setShowAdd(false)
  }

  const removeStock = (code: string) => {
    setWatchlist((prev) => prev.filter((c) => c !== code))
  }

  return (
    <div className="page">
      <header className="header">
        <h1>港股助手</h1>
      </header>
      <div className="toolbar">
        <button type="button" onClick={() => setShowAdd(true)} className="btn primary">
          添加股票
        </button>
        <button type="button" onClick={() => fetchStocks()} disabled={loading} className="btn">
          刷新
        </button>
        {watchlist.length > 0 && (
          <span className="muted" style={{ marginLeft: 8, fontSize: 12 }}>
            每 2 秒自动刷新
          </span>
        )}
      </div>
      <div className="card-list">
        {loading && stocks.length === 0 && <p className="muted">加载中…</p>}
        {!loading && watchlist.length === 0 && <p className="muted">暂无自选，请添加港股代码（如 hk00700）</p>}
        {!loading && watchlist.length > 0 && stocks.length === 0 && (
          <p className="muted">行情获取失败，请检查网络或稍后重试</p>
        )}
        {stocks.map((s) => (
          <div key={s.code} className="card">
            <div className="card-row">
              <div>
                <Link to={`/prediction?code=${encodeURIComponent(s.code)}`} className="stock-name">
                  {s.name}
                </Link>
                <div className="stock-code">{s.code}</div>
              </div>
              <div className="price-block">
                <span className="price" style={{ color: getColor(s.change_percent) }}>
                  {s.current_price.toFixed(2)}
                </span>
                <span className="change" style={{ color: getColor(s.change_percent) }}>
                  {s.change_percent > 0 ? '+' : ''}{s.change_percent.toFixed(2)}%
                </span>
              </div>
            </div>
            <div className="card-meta">
              成交量 {s.volume.toLocaleString()}
              <button type="button" onClick={() => removeStock(s.code)} className="link-btn">
                移除
              </button>
            </div>
          </div>
        ))}
      </div>
      {showAdd && (
        <div className="modal">
          <div className="modal-content">
            <h3>添加股票</h3>
            <input
              type="text"
              placeholder="股票代码 如 hk00700 或 700"
              value={addCode}
              onChange={(e) => setAddCode(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && addStock()}
            />
            <div className="modal-actions">
              <button type="button" onClick={() => setShowAdd(false)} className="btn">
                取消
              </button>
              <button type="button" onClick={addStock} className="btn primary">
                添加
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
