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
  speed: number
  quality_score: number
  features: string[]
}

const typeColors: Record<string, string> = {
  'text-to-text': 'tag-blue',
  'vision': 'tag-purple',
  'embedding': 'tag-green',
  'rerank': 'tag-amber',
  'text-to-image': 'tag-cyan',
  'text-to-video': 'tag-pink',
  'speech': 'tag-orange',
}

const typeIcons: Record<string, string> = {
  'text-to-text': '💬',
  'vision': '👁️',
  'embedding': '📐',
  'rerank': '🔀',
  'text-to-image': '🎨',
  'text-to-video': '🎬',
  'speech': '🎙️',
}

const typeLabels: Record<string, string> = {
  'all': '全部',
  'text-to-text': '对话补全',
  'vision': '多模态',
  'embedding': '向量嵌入',
  'rerank': '重排序',
  'text-to-image': '图像生成',
  'text-to-video': '视频生成',
  'speech': '语音',
}

export default function ModelsPage() {
  const [models, setModels] = useState<Model[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')

  useEffect(() => {
    fetchModels()
      .then((data) => setModels(data.models || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  const modelTypes = ['all', ...Array.from(new Set(models.map(m => m.type)))]
  const filtered = typeFilter === 'all' ? models : models.filter(m => m.type === typeFilter)

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
        <p className="text-sm text-[var(--text-muted)] mt-1">
          {models.length} 个模型，覆盖对话、多模态、嵌入、重排序、图像、视频、语音 7 大类型
        </p>
      </div>

      <div className="flex flex-wrap gap-2 mb-5">
        {modelTypes.map(t => (
          <button
            key={t}
            onClick={() => setTypeFilter(t)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${
              typeFilter === t
                ? 'bg-[var(--accent)] text-white'
                : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]'
            }`}
          >
            {t !== 'all' && <span className="mr-1">{typeIcons[t]}</span>}
            {typeLabels[t] || t}
            {t !== 'all' && <span className="ml-1 opacity-60">({models.filter(m => m.type === t).length})</span>}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {filtered.map((m) => (
          <div key={m.id} className="card group cursor-default">
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-2.5">
                <span className="text-lg">{typeIcons[m.type] || '🤖'}</span>
                <h3 className="font-semibold text-sm text-[var(--text)]">{m.name}</h3>
              </div>
              <span className={`tag ${typeColors[m.type] || 'tag-blue'}`}>{m.type}</span>
            </div>
            <p className="text-xs text-[var(--text-muted)] mb-3 font-mono bg-[var(--bg-secondary)] px-2.5 py-1.5 rounded-md inline-block">{m.id}</p>

            {m.features && m.features.length > 0 && (
              <div className="flex flex-wrap gap-1 mb-3">
                {m.features.map(f => (
                  <span key={f} className="px-1.5 py-0.5 bg-[var(--bg-secondary)] text-[var(--text-muted)] rounded text-[10px] font-medium">{f}</span>
                ))}
              </div>
            )}

            <div className="pt-3 border-t border-[var(--border-light)]">
              <div className="flex items-center justify-between text-xs">
                <span className="text-[var(--text-secondary)] font-medium">{m.provider}</span>
                <span className="text-[var(--text-muted)]">
                  ${m.input_price_per_million} / ${m.output_price_per_million} per 1M
                </span>
              </div>
              <div className="flex items-center gap-3 mt-2 text-xs text-[var(--text-muted)]">
                {m.max_context > 0 && (
                  <span className="flex items-center gap-1">
                    📏 {m.max_context >= 1000000 ? `${(m.max_context / 1000000).toFixed(0)}M` : `${(m.max_context / 1024).toFixed(0)}K`}
                  </span>
                )}
                {m.speed > 0 && (
                  <span className="flex items-center gap-1">⚡ {m.speed} tok/s</span>
                )}
                {m.quality_score > 0 && (
                  <span className="flex items-center gap-1">📊 {m.quality_score}</span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
