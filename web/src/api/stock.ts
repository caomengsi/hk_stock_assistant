import client from './client'
import type {
  RealtimeResponse,
  MarketSummaryResponse,
  PredictionResponse,
  PredictionRequest,
  SectorsResponse,
} from '../types'

function normalizeCode(code: string): string {
  code = code.trim().toLowerCase()
  if (code.startsWith('hk')) {
    if (code.length === 7) return code
    if (code.length === 6) return 'hk0' + code.slice(2)
    return code
  }
  if (code.length <= 5) {
    return 'hk' + '0'.repeat(5 - code.length) + code
  }
  return 'hk' + code
}

export async function getRealtime(code: string): Promise<RealtimeResponse> {
  const c = normalizeCode(code)
  const { data } = await client.get<RealtimeResponse>(`/api/stocks/${encodeURIComponent(c)}/realtime`)
  return data
}

export async function getMarketSummary(): Promise<MarketSummaryResponse> {
  const { data } = await client.get<MarketSummaryResponse>('/api/market/summary')
  return data
}

export async function getSectors(): Promise<SectorsResponse> {
  const { data } = await client.get<SectorsResponse>('/api/market/sectors')
  return data
}

export async function getPrediction(req: PredictionRequest): Promise<PredictionResponse> {
  const code = normalizeCode(req.code)
  const { data } = await client.post<PredictionResponse>(
    `/api/prediction/${encodeURIComponent(code)}`,
    { days: req.days, include_news: req.include_news, model: req.model },
    { timeout: 180000 }
  )
  return data
}

/** 流式预测：通过 SSE 逐段接收分析内容。onChunk(event, 片段)，event 为 'reasoning'（思考过程）或 'content'（最终输出）。 */
export function getPredictionStream(
  req: PredictionRequest,
  callbacks: {
    onChunk: (event: 'reasoning' | 'content', text: string) => void
    onDone: () => void
    onError: (message: string) => void
  }
): () => void {
  const code = normalizeCode(req.code)
  const abort = new AbortController()
  const baseURL = client.defaults.baseURL || ''
  const url = `${baseURL}/api/prediction/${encodeURIComponent(code)}/stream`
  fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      days: req.days,
      include_news: req.include_news,
      model: req.model,
    }),
    signal: abort.signal,
  })
    .then(async (res) => {
      if (!res.ok) {
        const text = await res.text()
        callbacks.onError(text || `请求失败 ${res.status}`)
        return
      }
      const reader = res.body?.getReader()
      if (!reader) {
        callbacks.onError('不支持流式响应')
        return
      }
      const dec = new TextDecoder()
      let buffer = ''
      try {
        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          buffer += dec.decode(value, { stream: true })
          const parts = buffer.split('\n\n')
          buffer = parts.pop() ?? ''
          for (const part of parts) {
            let event = ''
            let data = ''
            for (const line of part.split('\n')) {
              if (line.startsWith('event: ')) event = line.slice(7).trim()
              else if (line.startsWith('data: ')) data = line.slice(6)
            }
            if (event === 'reasoning' || event === 'content') {
              try {
                const s = JSON.parse(data) as string
                callbacks.onChunk(event as 'reasoning' | 'content', s)
              } catch {
                callbacks.onChunk(event as 'reasoning' | 'content', data)
              }
            } else if (event === 'error') {
              try {
                callbacks.onError(JSON.parse(data) as string)
              } catch {
                callbacks.onError(data)
              }
              return
            } else if (event === 'done') {
              callbacks.onDone()
              return
            }
          }
        }
        if (buffer.trim()) {
          const lines = buffer.split('\n')
          let event = ''
          let data = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) event = line.slice(7).trim()
            else if (line.startsWith('data: ')) data = line.slice(6)
          }
          if (event === 'reasoning' || event === 'content') {
            try {
              callbacks.onChunk(event as 'reasoning' | 'content', JSON.parse(data) as string)
            } catch {
              callbacks.onChunk(event as 'reasoning' | 'content', data)
            }
          }
        }
        callbacks.onDone()
      } finally {
        reader.releaseLock()
      }
    })
    .catch((e: unknown) => {
      if (e instanceof Error && e.name === 'AbortError') return
      callbacks.onError(e instanceof Error ? e.message : String(e))
    })
  return () => abort.abort()
}
