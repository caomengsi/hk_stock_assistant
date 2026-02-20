import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import { getPredictionStream } from '../api/stock'
import type { PredictionRequest } from '../types'

/** 流式结束后仅展示：从「时间与大盘环境」开始到结尾的总结内容 */
function extractSummary(fullText: string): string {
  const trimmed = fullText.trim()
  if (!trimmed) return trimmed
  const markers = ['时间与大盘环境分析', '时间与大盘环境', '1. 时间与大盘环境']
  for (const marker of markers) {
    const idx = trimmed.indexOf(marker)
    if (idx !== -1) {
      return trimmed.slice(idx).trim()
    }
  }
  return trimmed
}

const MODEL_OPTIONS = [
  { value: 'glm-5', label: 'GLM-5' },
  { value: 'GLM-4.7-Flash', label: 'GLM-4.7-Flash' },
] as const

export default function Prediction() {
  const [searchParams] = useSearchParams()
  const codeFromQuery = searchParams.get('code') ?? ''
  const [code, setCode] = useState(codeFromQuery || 'hk02513')
  const [days, setDays] = useState(3)
  const [model, setModel] = useState<string>(MODEL_OPTIONS[0].value)
  const [streamingText, setStreamingText] = useState('')
  const [summary, setSummary] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const fullTextRef = useRef('')
  const streamContainerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (codeFromQuery) setCode(codeFromQuery)
  }, [codeFromQuery])

  // 实时输出区域超出时自动滚到底部
  useEffect(() => {
    const el = streamContainerRef.current
    if (el && streamingText) el.scrollTop = el.scrollHeight
  }, [streamingText])

  const runStream = useCallback(() => {
    setError('')
    setSummary('')
    setStreamingText('')
    fullTextRef.current = ''
    setLoading(true)
    const req: PredictionRequest = {
      code,
      days,
      include_news: true,
      model,
    }
    const cancel = getPredictionStream(req, {
      onChunk(t) {
        fullTextRef.current += t
        setStreamingText(fullTextRef.current)
      },
      onDone() {
        setLoading(false)
        const s = extractSummary(fullTextRef.current)
        setSummary(s)
        setStreamingText('')
      },
      onError(msg) {
        setLoading(false)
        setError(msg)
      },
    })
    return cancel
  }, [code, days, model])

  const cancelRef = useRef<(() => void) | null>(null)
  useEffect(() => {
    return () => {
      cancelRef.current?.()
    }
  }, [])

  const handleStart = () => {
    cancelRef.current?.()
    cancelRef.current = runStream()
  }

  return (
    <div className="page">
      <header className="header">
        <h1>个股预测</h1>
      </header>
      <div className="card">
        <div style={{ marginBottom: 12 }}>
          <label>
            股票代码
            <input
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="hk02513"
              style={{ marginLeft: 8, width: 120 }}
            />
          </label>
          <label style={{ marginLeft: 16 }}>
            预测天数
            <select value={days} onChange={(e) => setDays(Number(e.target.value))}>
              {[1, 3, 5, 7].map((d) => (
                <option key={d} value={d}>
                  {d} 天
                </option>
              ))}
            </select>
          </label>
          <label style={{ marginLeft: 16 }}>
            模型
            <select value={model} onChange={(e) => setModel(e.target.value)}>
              {MODEL_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </label>
        </div>
        <button type="button" onClick={handleStart} disabled={loading} className="btn primary">
          {loading ? '生成中…' : '开始预测'}
        </button>
      </div>
      {error && (
        <div className="card" style={{ color: '#c62828' }}>
          {error}
        </div>
      )}
      {streamingText && (
        <div className="card">
          <h3>实时输出</h3>
          <div
            ref={streamContainerRef}
            className="markdown-stream"
            style={{ maxHeight: 300, overflow: 'auto' }}
          >
            <ReactMarkdown>{streamingText}</ReactMarkdown>
          </div>
        </div>
      )}
      {summary && (
        <div className="card">
          <h3>时间与大盘环境分析 · 总结</h3>
          <div className="markdown-content">
            <ReactMarkdown>{summary}</ReactMarkdown>
          </div>
        </div>
      )}
    </div>
  )
}
