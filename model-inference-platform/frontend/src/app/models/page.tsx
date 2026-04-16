'use client'

import { useEffect, useState } from 'react'
import { fetchModels } from '@/lib/api'

interface Model {
  id: string
  name: string
  type: string
  provider: string
  description: string
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

const typeIcons: Record<string, string> = {
  'text-to-text': '💬',
  'vision': '👁️',
  'embedding': '📐',
  'rerank': '🔀',
  'text-to-image': '🎨',
  'text-to-video': '🎬',
  'speech': '🎙️',
}

// 能力特性定义
const featureConfig: { key: string; label: string; icon: string }[] = [
  { key: 'function_calling', label: 'Function Calling', icon: '⚙️' },
  { key: 'json_mode', label: '结构化输出', icon: '📋' },
  { key: 'streaming', label: '流式输出', icon: '⚡' },
  { key: 'vision', label: '图像理解', icon: '🖼️' },
  { key: 'multilingual', label: '多语言支持', icon: '🌐' },
  { key: 'tts', label: '文字转语音', icon: '🔊' },
  { key: 'asr', label: '语音识别', icon: '🎤' },
  { key: 'code', label: '代码能力', icon: '💻' },
  { key: 'math', label: '数学推理', icon: '🧮' },
  { key: 'long_context', label: '长上下文', icon: '📄' },
]

function ProviderIcon({ provider }: { provider: string }) {
  const p = provider.toLowerCase()
  if (p.includes('qwen') || p.includes('alibaba') || p.includes('阿里')) {
    return (
      <span className="inline-flex items-center justify-center w-5 h-5 rounded text-[10px] font-bold bg-[#FF6A00]/10 text-[#FF6A00]">Q</span>
    )
  }
  if (p.includes('deepseek')) {
    return (
      <span className="inline-flex items-center justify-center w-5 h-5 rounded text-[10px] font-bold bg-[#5B6AF0]/10 text-[#5B6AF0]">D</span>
    )
  }
  if (p.includes('openai')) {
    return (
      <span className="inline-flex items-center justify-center w-5 h-5 rounded text-[10px] font-bold bg-[#10A37F]/10 text-[#10A37F]">O</span>
    )
  }
  if (p.includes('google') || p.includes('gemini')) {
    return (
      <span className="inline-flex items-center justify-center w-5 h-5 rounded text-[10px] font-bold bg-[#4285F4]/10 text-[#4285F4]">G</span>
    )
  }
  if (p.includes('anthropic') || p.includes('claude')) {
    return (
      <span className="inline-flex items-center justify-center w-5 h-5 rounded text-[10px] font-bold bg-[#D97706]/10 text-[#D97706]">A</span>
    )
  }
  return (
    <span className="inline-flex items-center justify-center w-5 h-5 rounded text-[10px] font-bold bg-[var(--bg-hover)] text-[var(--text-muted)]">M</span>
  )
}

function CapabilityRow({ label, supported }: { label: string; supported: boolean }) {
  return (
    <div className="flex items-center justify-between py-2.5 border-b border-[var(--border-light)] last:border-0">
      <span className="text-sm text-[var(--text-secondary)]">{label}</span>
      {supported ? (
        <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-[var(--primary)]/10">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" className="text-[var(--primary)]">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        </span>
      ) : (
        <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-[var(--bg-hover)]">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="text-[var(--text-muted)]">
            <circle cx="12" cy="12" r="9"/>
            <line x1="9" y1="9" x2="15" y2="15"/><line x1="15" y1="9" x2="9" y2="15"/>
          </svg>
        </span>
      )}
    </div>
  )
}

function ModelDetailView({ model }: { model: Model }) {
  const ctxLabel = model.max_context >= 1000000
    ? `${(model.max_context / 1000000).toFixed(0)}M tokens`
    : model.max_context >= 1024
      ? `${(model.max_context / 1024).toFixed(0)}K tokens`
      : model.max_context > 0 ? `${model.max_context} tokens` : null

  const hasPrice = model.input_price_per_million > 0 || model.output_price_per_million > 0

  // 将模型能力分为左右两列
  const leftFeatures = featureConfig.slice(0, Math.ceil(featureConfig.length / 2))
  const rightFeatures = featureConfig.slice(Math.ceil(featureConfig.length / 2))

  return (
    <div className="h-full overflow-y-auto">
      {/* 顶部标题栏 */}
      <div className="px-6 pt-6 pb-4 border-b border-[var(--border)]">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-[var(--primary)]/20 to-[var(--accent)]/20 flex items-center justify-center text-xl">
              {typeIcons[model.type] || '🤖'}
            </div>
            <div>
              <h2 className="text-lg font-semibold text-[var(--text)]">{model.name}</h2>
              <div className="flex items-center gap-2 mt-1">
                <ProviderIcon provider={model.provider} />
                <span className="text-xs text-[var(--text-muted)]">{model.provider}</span>
                <span className={`tag ${typeColors[model.type] || 'tag-blue'} text-[10px]`}>
                  {typeLabels[model.type] || model.type}
                </span>
              </div>
            </div>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              className="btn-secondary text-xs px-3 py-1.5"
              onClick={() => navigator.clipboard?.writeText(model.id)}
            >
              复制 ID
            </button>
          </div>
        </div>

        {/* 模型 Code 行 */}
        <div className="mt-4 flex items-center gap-2">
          <span className="text-xs text-[var(--text-muted)] shrink-0">模型 Code</span>
          <div className="flex-1 flex items-center gap-2 bg-[var(--bg-secondary)] rounded-lg px-3 py-2 border border-[var(--border-light)]">
            <code className="flex-1 text-xs font-mono text-[var(--text)]">{model.id}</code>
            <button
              type="button"
              onClick={() => navigator.clipboard?.writeText(model.id)}
              className="text-[var(--text-muted)] hover:text-[var(--text)] transition-colors shrink-0"
            >
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
              </svg>
            </button>
          </div>
        </div>
      </div>

      <div className="px-6 py-5 space-y-6">
        {/* 模型介绍 */}
        {model.description && (
          <section>
            <h3 className="text-sm font-semibold text-[var(--text)] mb-2">模型介绍</h3>
            <p className="text-sm text-[var(--text-secondary)] leading-relaxed">{model.description}</p>
          </section>
        )}

        {/* 规格速览 */}
        <section className="grid grid-cols-3 gap-3">
          {ctxLabel && (
            <div className="bg-[var(--bg-secondary)] rounded-xl p-3 text-center border border-[var(--border-light)]">
              <p className="text-[11px] text-[var(--text-muted)] mb-1">上下文长度</p>
              <p className="text-sm font-semibold text-[var(--text)]">{ctxLabel}</p>
            </div>
          )}
          {model.speed > 0 && (
            <div className="bg-[var(--bg-secondary)] rounded-xl p-3 text-center border border-[var(--border-light)]">
              <p className="text-[11px] text-[var(--text-muted)] mb-1">推理速度</p>
              <p className="text-sm font-semibold text-[var(--text)]">{model.speed} <span className="text-[11px] font-normal text-[var(--text-muted)]">tok/s</span></p>
            </div>
          )}
          {model.quality_score > 0 && (
            <div className="bg-[var(--bg-secondary)] rounded-xl p-3 text-center border border-[var(--border-light)]">
              <p className="text-[11px] text-[var(--text-muted)] mb-1">质量评分</p>
              <p className="text-sm font-semibold text-[var(--text)]">{model.quality_score}</p>
            </div>
          )}
        </section>

        {/* 模型能力 */}
        <section>
          <h3 className="text-sm font-semibold text-[var(--text)] mb-3">模型能力</h3>
          <div className="grid grid-cols-2 gap-x-6">
            <div>
              {leftFeatures.map(f => (
                <CapabilityRow
                  key={f.key}
                  label={f.label}
                  supported={model.features?.includes(f.key) ?? false}
                />
              ))}
            </div>
            <div>
              {rightFeatures.map(f => (
                <CapabilityRow
                  key={f.key}
                  label={f.label}
                  supported={model.features?.includes(f.key) ?? false}
                />
              ))}
            </div>
          </div>
        </section>

        {/* 模型价格 */}
        {hasPrice && (
          <section>
            <h3 className="text-sm font-semibold text-[var(--text)] mb-3">模型价格</h3>
            <div className="bg-[var(--bg-secondary)] rounded-xl border border-[var(--border-light)] overflow-hidden">
              {model.input_price_per_million > 0 && (
                <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
                  <span className="text-sm text-[var(--text-secondary)]">输入</span>
                  <div className="text-right">
                    <span className="text-base font-semibold text-[var(--text)]">${model.input_price_per_million}</span>
                    <span className="text-xs text-[var(--text-muted)] ml-1">元/每百万 tokens</span>
                  </div>
                </div>
              )}
              {model.output_price_per_million > 0 && (
                <div className="flex items-center justify-between px-4 py-3">
                  <span className="text-sm text-[var(--text-secondary)]">输出</span>
                  <div className="text-right">
                    <span className="text-base font-semibold text-[var(--text)]">${model.output_price_per_million}</span>
                    <span className="text-xs text-[var(--text-muted)] ml-1">元/每百万 tokens</span>
                  </div>
                </div>
              )}
            </div>
          </section>
        )}
      </div>
    </div>
  )
}

export default function ModelsPage() {
  const [models, setModels] = useState<Model[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  useEffect(() => {
    fetchModels()
      .then((data) => {
        const list: Model[] = data.models || []
        setModels(list)
        if (list.length > 0) setSelectedId(list[0].id)
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  const modelTypes = ['all', ...Array.from(new Set(models.map(m => m.type)))]

  const filtered = models.filter(m => {
    const matchType = typeFilter === 'all' || m.type === typeFilter
    const matchSearch = !search || m.name.toLowerCase().includes(search.toLowerCase()) || m.id.toLowerCase().includes(search.toLowerCase())
    return matchType && matchSearch
  })

  // 按 provider 分组
  const grouped = filtered.reduce<Record<string, Model[]>>((acc, m) => {
    const key = m.provider || '其他'
    if (!acc[key]) acc[key] = []
    acc[key].push(m)
    return acc
  }, {})

  const selectedModel = models.find(m => m.id === selectedId) || null

  if (loading) return (
    <div className="flex items-center gap-2 text-[var(--text-muted)] p-8">
      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
      Loading models...
    </div>
  )
  if (error) return <div className="text-[var(--danger)] bg-[var(--danger-bg)] px-4 py-3 rounded-lg m-4">Error: {error}</div>

  return (
    <div className="flex h-[calc(100vh-120px)] -mx-6 -my-4">
      {/* ── 左侧列表面板 ── */}
      <div className="w-72 shrink-0 border-r border-[var(--border)] flex flex-col bg-[var(--bg)]">
        {/* 顶部标题 + 搜索 */}
        <div className="px-4 pt-4 pb-3 border-b border-[var(--border)]">
          <h2 className="text-sm font-semibold text-[var(--text)] mb-3">模型列表</h2>
          <div className="relative">
            <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input
              type="text"
              placeholder="搜索模型..."
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-light)] text-[var(--text)] placeholder:text-[var(--text-muted)] outline-none focus:border-[var(--primary)]/50 transition-colors"
            />
          </div>
        </div>

        {/* 类型过滤 tabs */}
        <div className="px-3 py-2 border-b border-[var(--border)] flex gap-1 flex-wrap">
          {modelTypes.slice(0, 5).map(t => (
            <button
              key={t}
              onClick={() => setTypeFilter(t)}
              className={`px-2 py-1 rounded-md text-[10px] font-medium transition-colors ${
                typeFilter === t
                  ? 'bg-[var(--primary)] text-white'
                  : 'text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text)]'
              }`}
            >
              {typeLabels[t] || t}
            </button>
          ))}
          {modelTypes.length > 5 && modelTypes.slice(5).map(t => (
            <button
              key={t}
              onClick={() => setTypeFilter(t)}
              className={`px-2 py-1 rounded-md text-[10px] font-medium transition-colors ${
                typeFilter === t
                  ? 'bg-[var(--primary)] text-white'
                  : 'text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text)]'
              }`}
            >
              {typeLabels[t] || t}
            </button>
          ))}
        </div>

        {/* 分组模型列表 */}
        <div className="flex-1 overflow-y-auto py-2">
          {Object.entries(grouped).map(([provider, providerModels]) => (
            <div key={provider} className="mb-1">
              {/* Provider 分组标题 */}
              <div className="flex items-center gap-2 px-4 py-1.5">
                <ProviderIcon provider={provider} />
                <span className="text-[11px] font-medium text-[var(--text-muted)] uppercase tracking-wide truncate">{provider}</span>
                <span className="ml-auto text-[10px] text-[var(--text-muted)] bg-[var(--bg-secondary)] px-1.5 py-0.5 rounded-full">{providerModels.length}</span>
              </div>
              {/* 该 provider 下的模型 */}
              {providerModels.map(m => (
                <button
                  key={m.id}
                  type="button"
                  onClick={() => setSelectedId(m.id)}
                  className={`w-full text-left px-4 py-2 transition-colors flex items-center gap-2.5 ${
                    selectedId === m.id
                      ? 'bg-[var(--primary)]/8 border-r-2 border-[var(--primary)]'
                      : 'hover:bg-[var(--bg-hover)]'
                  }`}
                >
                  <span className="shrink-0 text-sm">{typeIcons[m.type] || '🤖'}</span>
                  <div className="flex-1 min-w-0">
                    <p className={`text-xs font-medium truncate leading-tight ${selectedId === m.id ? 'text-[var(--primary)]' : 'text-[var(--text)]'}`}>
                      {m.name}
                    </p>
                    <p className="text-[10px] text-[var(--text-muted)] truncate mt-0.5 font-mono">{m.id}</p>
                  </div>
                  <span className={`tag ${typeColors[m.type] || 'tag-blue'} text-[9px] shrink-0 !py-0`}>
                    {typeLabels[m.type]?.slice(0, 2) || m.type.slice(0, 2)}
                  </span>
                </button>
              ))}
            </div>
          ))}
          {filtered.length === 0 && (
            <div className="px-4 py-8 text-center text-xs text-[var(--text-muted)]">暂无匹配模型</div>
          )}
        </div>

        {/* 底部统计 */}
        <div className="px-4 py-3 border-t border-[var(--border)] text-[11px] text-[var(--text-muted)]">
          共 {models.length} 个模型
          {typeFilter !== 'all' && <span className="ml-1 text-[var(--primary)]">· 已过滤 {filtered.length} 个</span>}
        </div>
      </div>

      {/* ── 右侧详情面板 ── */}
      <div className="flex-1 overflow-hidden bg-[var(--bg)]">
        {selectedModel ? (
          <ModelDetailView model={selectedModel} />
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-[var(--text-muted)]">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" className="mb-3 opacity-30">
              <rect x="3" y="3" width="18" height="18" rx="3"/>
              <path d="M3 9h18M9 21V9"/>
            </svg>
            <p className="text-sm">请从左侧选择一个模型</p>
          </div>
        )}
      </div>
    </div>
  )
}
