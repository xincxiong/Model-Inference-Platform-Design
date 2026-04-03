'use client'

import { useEffect, useState } from 'react'
import { fetchModels } from '@/lib/api'

interface Model {
  id: string
  name: string
  type: string
  provider: string
  input_price_per_million: number
  output_price_per_million: number
  max_context: number
}

const typeColors: Record<string, string> = {
  'text-to-text': 'tag-blue',
  'embedding': 'tag-green',
  'rerank': 'tag-amber',
  'text-to-image': 'tag-cyan',
}

const typeIcons: Record<string, string> = {
  'text-to-text': '💬',
  'embedding': '📐',
  'rerank': '🔀',
  'text-to-image': '🎨',
}

export default function ModelsPage() {
  const [models, setModels] = useState<Model[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    fetchModels()
      .then((data) => setModels(data.models || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return (
    <div className="flex items-center gap-2 text-[var(--text-muted)]">
      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
      Loading models...
    </div>
  )
  if (error) return <div className="text-[var(--danger)] bg-[var(--danger-bg)] px-4 py-3 rounded-lg">Error: {error}</div>

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">模型列表</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">浏览可用的模型，查看定价和规格信息</p>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {models.map((m) => (
          <div key={m.id} className="card group cursor-default">
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-2.5">
                <span className="text-lg">{typeIcons[m.type] || '🤖'}</span>
                <h3 className="font-semibold text-sm text-[var(--text)]">{m.name}</h3>
              </div>
              <span className={`tag ${typeColors[m.type] || 'tag-blue'}`}>{m.type}</span>
            </div>
            <p className="text-xs text-[var(--text-muted)] mb-4 font-mono bg-[var(--bg-secondary)] px-2.5 py-1.5 rounded-md inline-block">{m.id}</p>
            <div className="pt-3 border-t border-[var(--border-light)]">
              <div className="flex items-center justify-between text-xs">
                <span className="text-[var(--text-secondary)] font-medium">{m.provider}</span>
                <span className="text-[var(--text-muted)]">
                  ${m.input_price_per_million} / ${m.output_price_per_million} per 1M tokens
                </span>
              </div>
              {m.max_context > 0 && (
                <div className="text-xs text-[var(--text-muted)] mt-1.5 flex items-center gap-1">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="2" y="2" width="20" height="20" rx="5"/><line x1="8" y1="12" x2="16" y2="12"/></svg>
                  Context: {(m.max_context / 1024).toFixed(0)}K tokens
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
