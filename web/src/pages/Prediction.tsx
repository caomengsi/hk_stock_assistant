import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import { getPredictionStream } from '../api/stock'
import type { PredictionRequest } from '../types'

const MODEL_OPTIONS = [
  { value: 'GLM-4.7-Flash', label: 'GLM-4.7-Flash' },
  { value: 'glm-5', label: 'GLM-5' },
] as const

export default function Prediction() {
  const [searchParams] = useSearchParams()
  const codeFromQuery = searchParams.get('code') ?? ''
  const [code, setCode] = useState(codeFromQuery || 'hk02513')
  const [days, setDays] = useState(3)
  const [model, setModel] = useState<string>(MODEL_OPTIONS[0].value)
  const [streamingText, setStreamingText] = useState('')
  const [finalOutput, setFinalOutput] = useState('')
  const [fullStreamedText, setFullStreamedText] = useState('')
  const [resultTab, setResultTab] = useState<'summary' | 'stream'>('summary')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const contentRef = useRef('')
  const fullTextRef = useRef('')
  const streamContainerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (codeFromQuery) setCode(codeFromQuery)
  }, [codeFromQuery])

  // 实时分析区域超出时自动滚到底部
  useEffect(() => {
    const el = streamContainerRef.current
    if (el && streamingText) el.scrollTop = el.scrollHeight
  }, [streamingText])

  const runStream = useCallback(() => {
    setError('')
    setFinalOutput('')
    setFullStreamedText('')
    setStreamingText('')
    contentRef.current = ''
    fullTextRef.current = ''
    setLoading(true)
    const req: PredictionRequest = {
      code,
      days,
      include_news: true,
      model,
    }
    const cancel = getPredictionStream(req, {
      onChunk(event, t) {
        fullTextRef.current += t
        setStreamingText(fullTextRef.current)
        if (event === 'content') {
          contentRef.current += t
        }
      },
      onDone() {
        setLoading(false)
        setFinalOutput(contentRef.current)
        setFullStreamedText(fullTextRef.current)
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
        <div className="prediction-form-row">
          <label className="prediction-field">
            <span className="prediction-field-label">股票代码</span>
            <input
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="hk02513"
              className="prediction-input"
            />
          </label>
          <label className="prediction-field">
            <span className="prediction-field-label">预测天数</span>
            <select value={days} onChange={(e) => setDays(Number(e.target.value))} className="prediction-select">
              {[1, 3, 5, 7].map((d) => (
                <option key={d} value={d}>
                  {d} 天
                </option>
              ))}
            </select>
          </label>
          <label className="prediction-field">
            <span className="prediction-field-label">模型</span>
            <select value={model} onChange={(e) => setModel(e.target.value)} className="prediction-select">
              {MODEL_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </label>
        </div>
        <button type="button" onClick={handleStart} disabled={loading} className="btn primary prediction-btn">
          {loading ? '生成中…' : '开始预测'}
        </button>
      </div>
      {error && (
        <div className="card" style={{ color: '#c62828' }}>
          {error}
        </div>
      )}
      {((loading && streamingText) || (!loading && (finalOutput || fullStreamedText))) && (
        <div className="card prediction-result-card">
          <div className="prediction-result-tabs">
            {!loading && (
              <button
                type="button"
                className={`prediction-tab ${resultTab === 'summary' ? 'active' : ''}`}
                onClick={() => setResultTab('summary')}
              >
                总结走势
              </button>
            )}
            <button
              type="button"
              className={`prediction-tab ${loading || resultTab === 'stream' ? 'active' : ''}`}
              onClick={() => !loading && setResultTab('stream')}
            >
              实时分析
            </button>
          </div>
          {loading && (
            <div
              ref={streamContainerRef}
              className="markdown-stream prediction-tab-panel"
            >
              <ReactMarkdown>{streamingText}</ReactMarkdown>
            </div>
          )}
          {!loading && resultTab === 'summary' && (
            <div className="markdown-content prediction-tab-panel">
              <ReactMarkdown>{finalOutput.trim() || '—'}</ReactMarkdown>
            </div>
          )}
          {!loading && resultTab === 'stream' && (
            <div className="markdown-stream prediction-tab-panel">
              <ReactMarkdown>{fullStreamedText.trim() || '—'}</ReactMarkdown>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
