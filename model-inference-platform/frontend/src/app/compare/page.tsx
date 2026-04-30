'use client'

import { useState, useMemo, useEffect } from 'react'

// ── Mock benchmark data ──────────────────────────────────────────────────────
interface ModelBenchmark {
  id: string
  name: string
  provider: string
  color: string
  mmlu: number
  humaneval: number
  gsm8k: number
  math: number
  ceval: number
  ttft: number
  tps: number
  costPerMInput: number
  costPerMOutput: number
  contextWindow: string
}

const MODELS: ModelBenchmark[] = [
  { id: 'ds', name: 'DeepSeek-V3', provider: '深度求索', color: '#1a73e8', mmlu: 86.4, humaneval: 87.2, gsm8k: 94.1, math: 58.3, ceval: 88.7, ttft: 142, tps: 87, costPerMInput: 0.27, costPerMOutput: 1.10, contextWindow: '128K' },
  { id: 'qw', name: 'Qwen3.5-72B', provider: '阿里云', color: '#e37400', mmlu: 85.1, humaneval: 82.4, gsm8k: 91.3, math: 55.7, ceval: 92.1, ttft: 218, tps: 64, costPerMInput: 0.33, costPerMOutput: 1.32, contextWindow: '256K' },
  { id: 'gl', name: 'GLM-5', provider: '智谱 AI', color: '#1e8e3e', mmlu: 82.3, humaneval: 79.8, gsm8k: 88.5, math: 51.2, ceval: 90.4, ttft: 167, tps: 72, costPerMInput: 0.20, costPerMOutput: 0.80, contextWindow: '128K' },
  { id: 'll', name: 'Llama-4-70B', provider: 'Meta', color: '#7627bb', mmlu: 84.7, humaneval: 81.3, gsm8k: 90.2, math: 54.1, ceval: 72.5, ttft: 195, tps: 58, costPerMInput: 0.15, costPerMOutput: 0.60, contextWindow: '128K' },
  { id: 'km', name: 'Kimi-K2.5', provider: '月之暗面', color: '#c5221f', mmlu: 83.9, humaneval: 84.1, gsm8k: 92.0, math: 60.2, ceval: 85.3, ttft: 178, tps: 71, costPerMInput: 0.40, costPerMOutput: 1.60, contextWindow: '256K' },
  { id: 'db', name: '豆包 2.0', provider: '字节跳动', color: '#007b83', mmlu: 80.2, humaneval: 76.5, gsm8k: 86.7, math: 47.8, ceval: 87.9, ttft: 131, tps: 95, costPerMInput: 0.12, costPerMOutput: 0.48, contextWindow: '64K' },
]

const BENCHMARKS = [
  { key: 'mmlu', label: 'MMLU', desc: '综合知识', max: 100 },
  { key: 'humaneval', label: 'HumanEval', desc: '代码生成', max: 100 },
  { key: 'gsm8k', label: 'GSM8K', desc: '数学推理', max: 100 },
  { key: 'math', label: 'MATH', desc: '高等数学', max: 100 },
  { key: 'ceval', label: 'C-Eval', desc: '中文评测', max: 100 },
]

const PERF_METRICS = [
  { key: 'ttft', label: 'TTFT', unit: 'ms', lower: true },
  { key: 'tps', label: 'TPS', unit: 'tok/s', lower: false },
]

// ── Empty state ──────────────────────────────────────────────────────────────
function EmptyState({ message }: { message: string }) {
  return (
    <div className="card text-center py-16">
      <div className="text-2xl mb-3">📊</div>
      <p className="text-sm text-[var(--text)] font-medium">{message}</p>
      <p className="text-xs text-[var(--text-muted)] mt-1">请至少选择 2 个模型进行对比</p>
    </div>
  )
}

// ── Loading skeleton ─────────────────────────────────────────────────────────
function LoadingSkeleton() {
  return (
    <div className="space-y-4 animate-pulse">
      <div className="card h-8 bg-[var(--bg-secondary)] rounded-lg" />
      <div className="card h-64 bg-[var(--bg-secondary)] rounded-lg" />
      <div className="card h-48 bg-[var(--bg-secondary)] rounded-lg" />
    </div>
  )
}

// ── Error state ──────────────────────────────────────────────────────────────
function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="card text-center py-16">
      <div className="text-2xl mb-3">️</div>
      <p className="text-sm text-[var(--danger)] font-medium">数据加载失败</p>
      <p className="text-xs text-[var(--text-muted)] mt-1 mb-4">无法获取模型对比数据，请检查网络连接</p>
      <button onClick={onRetry} className="btn-primary text-xs px-4 py-2">重新加载</button>
    </div>
  )
}

export default function ComparePage() {
  const [selected, setSelected] = useState<string[]>(['ds', 'qw', 'll'])
  const [activeTab, setActiveTab] = useState<'benchmarks' | 'performance' | 'cost'>('benchmarks')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Simulate data fetch
  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 800)
    return () => clearTimeout(timer)
  }, [])

  const selectedModels = useMemo(() => MODELS.filter(m => selected.includes(m.id)), [selected])

  const winner = (key: string, lower = false) => {
    const vals = selectedModels.map(m => ({ id: m.id, v: (m as any)[key] }))
    const best = lower ? Math.min(...vals.map(v => v.v)) : Math.max(...vals.map(v => v.v))
    return vals.filter(v => v.v === best).map(v => v.id)
  }

  if (error) return <ErrorState onRetry={() => { setError(null); setLoading(true); setTimeout(() => setLoading(false), 800) }} />
  if (loading) return <LoadingSkeleton />
  if (selectedModels.length < 2) return <EmptyState message="请选择至少 2 个模型" />

  return (
    <div>
      {/* ── Header ─────────────────────────────────────────────────── */}
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">模型对比矩阵</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">基准评测、性能指标、成本效益对比</p>
      </div>

      {/* ── Model selector ────────────────────────────────────────── */}
      <div className="card mb-5">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">选择对比模型（2-6 个）</h3>
        <div className="flex flex-wrap gap-2">
          {MODELS.map(m => {
            const isSelected = selected.includes(m.id)
            return (
              <button
                key={m.id}
                onClick={() => {
                  if (isSelected && selected.length > 2) setSelected(s => s.filter(id => id !== m.id))
                  else if (!isSelected && selected.length < 6) setSelected(s => [...s, m.id])
                }}
                className={`flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium border transition-all ${
                  isSelected
                    ? 'border-[var(--accent)] bg-[var(--accent-light)] text-[var(--accent)]'
                    : 'border-[var(--border)] text-[var(--text-secondary)] hover:border-[var(--accent)]'
                }`}
              >
                <span className="w-2.5 h-2.5 rounded-full" style={{ background: m.color }} />
                <span>{m.name}</span>
                <span className="text-[var(--text-muted)]">· {m.provider}</span>
              </button>
            )
          })}
        </div>
      </div>

      {/* ── Tabs ───────────────────────────────────────────────────── */}
      <div className="flex gap-1 mb-5 bg-white border border-[var(--border)] rounded-lg p-1 w-fit">
        {([
          ['benchmarks', '基准评测'],
          ['performance', '性能指标'],
          ['cost', '成本效益'],
        ] as const).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setActiveTab(key)}
            className={`text-xs px-4 py-1.5 rounded-md font-medium transition-colors ${
              activeTab === key ? 'bg-[var(--accent)] text-white' : 'text-[var(--text-muted)] hover:text-[var(--text)]'
            }`}
          >{label}</button>
        ))}
      </div>

      {/* ── Benchmarks ─────────────────────────────────────────────── */}
      {activeTab === 'benchmarks' && (
        <div className="card">
          <h3 className="font-medium text-sm text-[var(--text)] mb-4">基准评测得分</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                  <th className="text-left pb-2.5 font-medium w-32">评测集</th>
                  {selectedModels.map(m => (
                    <th key={m.id} className="text-right pb-2.5 font-medium">
                      <div className="flex items-center justify-end gap-1.5">
                        <span className="w-2.5 h-2.5 rounded-full" style={{ background: m.color }} />
                        {m.name}
                      </div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--border-light)]">
                {BENCHMARKS.map(b => {
                  const winners = winner(b.key)
                  return (
                    <tr key={b.key} className="hover:bg-[var(--bg-secondary)]">
                      <td className="py-3 pr-3">
                        <div>
                          <div className="font-medium text-[var(--text)]">{b.label}</div>
                          <div className="text-label-xs text-[var(--text-muted)]">{b.desc}</div>
                        </div>
                      </td>
                      {selectedModels.map(m => {
                        const isWinner = winners.includes(m.id)
                        return (
                          <td key={m.id} className="py-3 text-right">
                            <div className="flex items-center justify-end gap-2">
                              <div className="w-24 bg-[var(--bg-secondary)] rounded h-3 overflow-hidden">
                                <div className="h-full rounded transition-all" style={{ width: `${(m as any)[b.key]}%`, background: m.color }} />
                              </div>
                              <span className={`font-mono font-medium w-12 ${isWinner ? 'text-[var(--accent)]' : 'text-[var(--text)]'}`}>
                                {(m as any)[b.key]}
                              </span>
                              {isWinner && <span className="tag tag-green text-label-xs">最佳</span>}
                            </div>
                          </td>
                        )
                      })}
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          {/* Radar-style summary */}
          <div className="mt-5 pt-4 border-t border-[var(--border-light)]">
            <h4 className="text-xs font-medium text-[var(--text-muted)] mb-3">综合能力总览</h4>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              {selectedModels.map(m => {
                const avg = (m.mmlu + m.humaneval + m.gsm8k + m.math + m.ceval) / 5
                return (
                  <div key={m.id} className="flex items-center gap-3 p-3 rounded-lg border border-[var(--border-light)]">
                    <div className="relative w-14 h-14">
                      <svg viewBox="0 0 36 36" className="w-full h-full -rotate-90">
                        <circle cx="18" cy="18" r="15.9" fill="none" stroke="var(--bg-secondary)" strokeWidth="3" />
                        <circle cx="18" cy="18" r="15.9" fill="none" stroke={m.color} strokeWidth="3" strokeDasharray={`${avg} ${100 - avg}`} strokeLinecap="round" />
                      </svg>
                      <div className="absolute inset-0 flex items-center justify-center">
                        <span className="text-xs font-bold" style={{ color: m.color }}>{avg.toFixed(1)}</span>
                      </div>
                    </div>
                    <div>
                      <div className="text-xs font-medium text-[var(--text)]">{m.name}</div>
                      <div className="text-label-xs text-[var(--text-muted)]">五项平均</div>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      )}

      {/* ── Performance ────────────────────────────────────────────── */}
      {activeTab === 'performance' && (
        <div className="space-y-4">
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-4">性能指标</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                    <th className="text-left pb-2.5 font-medium w-32">指标</th>
                    {selectedModels.map(m => (
                      <th key={m.id} className="text-right pb-2.5 font-medium">
                        <div className="flex items-center justify-end gap-1.5">
                          <span className="w-2.5 h-2.5 rounded-full" style={{ background: m.color }} />
                          {m.name.split('-')[0]}
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--border-light)]">
                  {PERF_METRICS.map(metric => {
                    const winners = winner(metric.key, metric.lower)
                    return (
                      <tr key={metric.key} className="hover:bg-[var(--bg-secondary)]">
                        <td className="py-3 pr-3">
                          <div className="font-medium text-[var(--text)]">{metric.label}</div>
                          <div className="text-label-xs text-[var(--text-muted)]">{metric.unit} · {metric.lower ? '越低越好' : '越高越好'}</div>
                        </td>
                        {selectedModels.map(m => {
                          const isWinner = winners.includes(m.id)
                          return (
                            <td key={m.id} className={`py-3 text-right font-mono ${isWinner ? 'font-semibold text-[var(--accent)]' : 'text-[var(--text)]'}`}>
                              {(m as any)[metric.key]}{metric.unit === 'ms' ? 'ms' : ''}
                              {isWinner && ' ★'}
                            </td>
                          )
                        })}
                      </tr>
                    )
                  })}
                  <tr className="hover:bg-[var(--bg-secondary)]">
                    <td className="py-3 pr-3">
                      <div className="font-medium text-[var(--text)]">上下文窗口</div>
                    </td>
                    {selectedModels.map(m => (
                      <td key={m.id} className="py-3 text-right font-mono text-[var(--text)]">{m.contextWindow}</td>
                    ))}
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          {/* TTFT comparison bars */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-3">首 Token 延迟对比</h3>
            <div className="space-y-2.5">
              {selectedModels.sort((a, b) => a.ttft - b.ttft).map(m => {
                const best = Math.min(...selectedModels.map(x => x.ttft))
                const worst = Math.max(...selectedModels.map(x => x.ttft))
                const pct = ((m.ttft - best) / (worst - best || 1)) * 100
                return (
                  <div key={m.id} className="flex items-center gap-3">
                    <span className="w-24 text-xs font-medium text-[var(--text)]">{m.name.split('-')[0]}</span>
                    <div className="flex-1 bg-[var(--bg-secondary)] rounded h-4 overflow-hidden">
                      <div className="h-full rounded transition-all" style={{
                        width: `${100 - pct}%`,
                        background: m.ttft === best ? 'var(--success)' : m.ttft === worst ? 'var(--danger)' : 'var(--warning)',
                      }} />
                    </div>
                    <span className={`w-14 text-right font-mono text-xs ${m.ttft === best ? 'text-[var(--success)] font-semibold' : 'text-[var(--text-muted)]'}`}>
                      {m.ttft}ms
                    </span>
                  </div>
                )
              })}
            </div>
          </div>

          {/* TPS comparison bars */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-3">生成速度 (TPS)</h3>
            <div className="space-y-2.5">
              {selectedModels.sort((a, b) => b.tps - a.tps).map(m => {
                const best = Math.max(...selectedModels.map(x => x.tps))
                const worst = Math.min(...selectedModels.map(x => x.tps))
                const pct = ((m.tps - worst) / (best - worst || 1)) * 100
                return (
                  <div key={m.id} className="flex items-center gap-3">
                    <span className="w-24 text-xs font-medium text-[var(--text)]">{m.name.split('-')[0]}</span>
                    <div className="flex-1 bg-[var(--bg-secondary)] rounded h-4 overflow-hidden">
                      <div className="h-full rounded transition-all" style={{ width: `${pct}%`, background: m.color }} />
                    </div>
                    <span className="w-14 text-right font-mono text-xs font-medium text-[var(--text)]">{m.tps}</span>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      )}

      {/* ── Cost ───────────────────────────────────────────────────── */}
      {activeTab === 'cost' && (
        <div className="space-y-4">
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-4">定价对比</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                    <th className="text-left pb-2.5 font-medium w-32">模型</th>
                    <th className="text-right pb-2.5 font-medium">Input $/M</th>
                    <th className="text-right pb-2.5 font-medium">Output $/M</th>
                    <th className="text-right pb-2.5 font-medium">上下文</th>
                    <th className="text-right pb-2.5 font-medium">综合评分</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--border-light)]">
                  {selectedModels.map(m => {
                    const score = ((m.mmlu + m.humaneval + m.gsm8k + m.math + m.ceval) / 5) / ((m.costPerMInput + m.costPerMOutput) / 2) * 10
                    return (
                      <tr key={m.id} className="hover:bg-[var(--bg-secondary)]">
                        <td className="py-3 pr-3">
                          <div className="flex items-center gap-2">
                            <span className="w-2.5 h-2.5 rounded-full" style={{ background: m.color }} />
                            <span className="font-medium text-[var(--text)]">{m.name}</span>
                          </div>
                        </td>
                        <td className="py-3 pr-3 text-right font-mono text-[var(--text)]">${m.costPerMInput.toFixed(2)}</td>
                        <td className="py-3 pr-3 text-right font-mono text-[var(--text)]">${m.costPerMOutput.toFixed(2)}</td>
                        <td className="py-3 pr-3 text-right text-[var(--text-muted)]">{m.contextWindow}</td>
                        <td className="py-3 text-right">
                          <span className="tag" style={{
                            background: score > 3000 ? 'var(--success-bg)' : score > 2000 ? 'var(--warning-bg)' : 'var(--bg-secondary)',
                            color: score > 3000 ? 'var(--success)' : score > 2000 ? 'var(--warning)' : 'var(--text-muted)',
                          }}>{score.toFixed(0)}</span>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>

          {/* Cost-performance ratio bars */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-3">性价比（综合得分 / 平均成本）</h3>
            <div className="space-y-2.5">
              {selectedModels
                .map(m => ({ m, score: ((m.mmlu + m.humaneval + m.gsm8k + m.math + m.ceval) / 5) / ((m.costPerMInput + m.costPerMOutput) / 2) * 10 }))
                .sort((a, b) => b.score - a.score)
                .map(({ m, score }) => {
                  const maxScore = Math.max(...selectedModels.map(x => ((x.mmlu + x.humaneval + x.gsm8k + x.math + x.ceval) / 5) / ((x.costPerMInput + x.costPerMOutput) / 2) * 10))
                  return (
                    <div key={m.id} className="flex items-center gap-3">
                      <span className="w-24 text-xs font-medium text-[var(--text)]">{m.name.split('-')[0]}</span>
                      <div className="flex-1 bg-[var(--bg-secondary)] rounded h-4 overflow-hidden">
                        <div className="h-full rounded transition-all" style={{ width: `${(score / maxScore) * 100}%`, background: m.color }} />
                      </div>
                      <span className="w-14 text-right font-mono text-xs font-medium text-[var(--text)]">{score.toFixed(0)}</span>
                    </div>
                  )
                })}
            </div>
          </div>

          {/* Scenario cost comparison */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-3">场景成本估算（每 100 万 Token 对话）</h3>
            <div className="text-xs text-[var(--text-muted)] mb-3">假设 Input 700K + Output 300K</div>
            <div className="space-y-2.5">
              {selectedModels.sort((a, b) => (a.costPerMInput * 0.7 + a.costPerMOutput * 0.3) - (b.costPerMInput * 0.7 + b.costPerMOutput * 0.3)).map(m => {
                const cost = m.costPerMInput * 0.7 + m.costPerMOutput * 0.3
                const maxCost = Math.max(...selectedModels.map(x => x.costPerMInput * 0.7 + x.costPerMOutput * 0.3))
                return (
                  <div key={m.id} className="flex items-center gap-3">
                    <span className="w-24 text-xs font-medium text-[var(--text)]">{m.name.split('-')[0]}</span>
                    <div className="flex-1 bg-[var(--bg-secondary)] rounded h-4 overflow-hidden">
                      <div className="h-full rounded transition-all" style={{ width: `${(cost / maxCost) * 100}%`, background: m.color }} />
                    </div>
                    <span className="w-14 text-right font-mono text-xs font-medium text-[var(--warning)]">${cost.toFixed(2)}</span>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
