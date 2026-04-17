'use client'

import { useState, useEffect, useMemo } from 'react'
import { fetchUsage, redeemPromo } from '@/lib/api'

interface DailyUsage {
  date: string
  input_tokens: number
  output_tokens: number
  cost: number
  request_count: number
}

interface ModelUsage {
  model: string
  input_tokens: number
  output_tokens: number
  cost: number
  request_count: number
}

interface UsageData {
  balance: number
  total_spent: number
  total_tokens: number
  total_requests: number
  daily_breakdown: DailyUsage[]
  by_model: ModelUsage[]
}

// ── Mini sparkline bar chart ──────────────────────────────────────────────
function SparkBar({ values, color = 'var(--accent)' }: { values: number[]; color?: string }) {
  const max = Math.max(...values, 0.0001)
  return (
    <div className="flex items-end gap-px h-12">
      {values.map((v, i) => (
        <div
          key={i}
          className="flex-1 rounded-sm transition-all"
          style={{ height: `${Math.max((v / max) * 100, 2)}%`, background: color, opacity: 0.85 }}
        />
      ))}
    </div>
  )
}

// ── Simple pie / donut chart using SVG ────────────────────────────────────
function DonutChart({ slices }: { slices: { label: string; value: number; color: string }[] }) {
  const total = slices.reduce((s, d) => s + d.value, 0)
  if (total === 0) return <div className="text-xs text-[var(--text-muted)] text-center py-4">暂无数据</div>

  const r = 40
  const cx = 56
  const cy = 56
  let startAngle = -Math.PI / 2

  const arcs = slices.map((s) => {
    const angle = (s.value / total) * 2 * Math.PI
    const x1 = cx + r * Math.cos(startAngle)
    const y1 = cy + r * Math.sin(startAngle)
    const x2 = cx + r * Math.cos(startAngle + angle)
    const y2 = cy + r * Math.sin(startAngle + angle)
    const large = angle > Math.PI ? 1 : 0
    const d = `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${large} 1 ${x2} ${y2} Z`
    startAngle += angle
    return { ...s, d }
  })

  return (
    <div className="flex items-center gap-4">
      <svg width={112} height={112} viewBox="0 0 112 112">
        {arcs.map((a, i) => (
          <path key={i} d={a.d} fill={a.color} opacity={0.9} />
        ))}
        <circle cx={cx} cy={cy} r={24} fill="var(--bg-card)" />
      </svg>
      <div className="space-y-1.5 flex-1 min-w-0">
        {arcs.map((a, i) => (
          <div key={i} className="flex items-center gap-2 text-xs">
            <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ background: a.color }} />
            <span className="truncate text-[var(--text-muted)] flex-1">{a.label}</span>
            <span className="font-medium text-[var(--text)]">{((a.value / total) * 100).toFixed(1)}%</span>
          </div>
        ))}
      </div>
    </div>
  )
}

// ── Palette for model colors ───────────────────────────────────────────────
const PALETTE = [
  '#6366f1', '#22d3ee', '#f59e0b', '#10b981', '#f43f5e',
  '#8b5cf6', '#0ea5e9', '#84cc16', '#fb923c', '#a78bfa',
]

export default function UsagePage() {
  const [usage, setUsage] = useState<UsageData | null>(null)
  const [promoCode, setPromoCode] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [activeTab, setActiveTab] = useState<'cost' | 'tokens' | 'requests'>('cost')

  const loadUsage = () => {
    fetchUsage()
      .then(setUsage)
      .catch((e) => setError(e.message))
  }

  useEffect(() => { loadUsage() }, [])

  const handleRedeem = async () => {
    if (!promoCode) return
    try {
      const res = await redeemPromo(promoCode)
      setMessage(`Redeemed $${res.amount}!`)
      setPromoCode('')
      loadUsage()
    } catch (e: any) {
      setError(e.message)
    }
  }

  const breakdown = useMemo(() => (usage?.daily_breakdown || []).slice().reverse(), [usage])
  const byModel   = usage?.by_model || []

  const chartValues = useMemo(() => {
    if (activeTab === 'cost')     return breakdown.map(d => d.cost)
    if (activeTab === 'tokens')   return breakdown.map(d => d.input_tokens + d.output_tokens)
    return breakdown.map(d => d.request_count)
  }, [breakdown, activeTab])

  const chartColor = activeTab === 'cost' ? 'var(--warning)' : activeTab === 'tokens' ? 'var(--accent)' : 'var(--success)'

  const modelSlices = useMemo(() => byModel.slice(0, 8).map((m, i) => ({
    label: m.model.split('/').pop() || m.model,
    value: m.cost,
    color: PALETTE[i % PALETTE.length],
  })), [byModel])

  const totalInputTokens  = breakdown.reduce((s, d) => s + d.input_tokens, 0)
  const totalOutputTokens = breakdown.reduce((s, d) => s + d.output_tokens, 0)

  if (!usage) return (
    <div className="flex items-center gap-2 text-[var(--text-muted)]">
      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
      </svg>
      Loading...
    </div>
  )

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">用量统计</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">查看账户余额、消费记录和每日用量</p>
      </div>

      {/* ── 4 stat cards ─────────────────────────────────────────────── */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-5">
        <div className="card text-center">
          <div className="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">余额</div>
          <div className="text-2xl font-semibold text-[var(--success)]">${usage.balance.toFixed(4)}</div>
        </div>
        <div className="card text-center">
          <div className="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">累计消费</div>
          <div className="text-2xl font-semibold text-[var(--warning)]">${usage.total_spent.toFixed(4)}</div>
        </div>
        <div className="card text-center">
          <div className="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">累计 Tokens</div>
          <div className="text-2xl font-semibold text-[var(--accent)]">{usage.total_tokens.toLocaleString()}</div>
        </div>
        <div className="card text-center">
          <div className="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">累计请求数</div>
          <div className="text-2xl font-semibold text-[var(--text)]">{(usage.total_requests ?? 0).toLocaleString()}</div>
        </div>
      </div>

      {/* ── Token type breakdown ──────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-4 mb-5">
        <div className="card">
          <div className="text-xs text-[var(--text-muted)] mb-1">Input Tokens (30d)</div>
          <div className="text-lg font-semibold text-[var(--text)]">{totalInputTokens.toLocaleString()}</div>
          <div className="mt-1 text-xs text-[var(--text-muted)]">
            {usage.total_tokens > 0 ? ((totalInputTokens / (totalInputTokens + totalOutputTokens)) * 100).toFixed(1) : '0'}% of total
          </div>
        </div>
        <div className="card">
          <div className="text-xs text-[var(--text-muted)] mb-1">Output Tokens (30d)</div>
          <div className="text-lg font-semibold text-[var(--text)]">{totalOutputTokens.toLocaleString()}</div>
          <div className="mt-1 text-xs text-[var(--text-muted)]">
            {usage.total_tokens > 0 ? ((totalOutputTokens / (totalInputTokens + totalOutputTokens)) * 100).toFixed(1) : '0'}% of total
          </div>
        </div>
      </div>

      {/* ── Trend chart ───────────────────────────────────────────────── */}
      <div className="card mb-5">
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-medium text-sm text-[var(--text)]">趋势图（近 30 天）</h3>
          <div className="flex gap-1">
            {(['cost', 'tokens', 'requests'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`text-xs px-2.5 py-1 rounded-md transition-colors ${
                  activeTab === tab
                    ? 'bg-[var(--accent)] text-white'
                    : 'text-[var(--text-muted)] hover:text-[var(--text)]'
                }`}
              >
                {tab === 'cost' ? '成本' : tab === 'tokens' ? 'Tokens' : '请求数'}
              </button>
            ))}
          </div>
        </div>

        {breakdown.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-4 text-center">暂无用量数据</p>
        ) : (
          <>
            <SparkBar values={chartValues} color={chartColor} />
            <div className="flex justify-between text-xs text-[var(--text-muted)] mt-1.5">
              <span>{breakdown[0]?.date}</span>
              <span>{breakdown[breakdown.length - 1]?.date}</span>
            </div>
          </>
        )}
      </div>

      {/* ── Model cost distribution + table ──────────────────────────── */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-5">
        <div className="card">
          <h3 className="font-medium text-sm text-[var(--text)] mb-3">成本分布（按模型）</h3>
          <DonutChart slices={modelSlices} />
        </div>

        <div className="card">
          <h3 className="font-medium text-sm text-[var(--text)] mb-3">模型用量明细</h3>
          {byModel.length === 0 ? (
            <p className="text-sm text-[var(--text-muted)] py-4 text-center">暂无数据</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                    <th className="text-left pb-2 font-medium">模型</th>
                    <th className="text-right pb-2 font-medium">请求</th>
                    <th className="text-right pb-2 font-medium">Tokens</th>
                    <th className="text-right pb-2 font-medium">成本</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--border)]">
                  {byModel.map((m, i) => (
                    <tr key={m.model} className="hover:bg-[var(--bg-secondary)]">
                      <td className="py-2 pr-2">
                        <div className="flex items-center gap-1.5">
                          <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: PALETTE[i % PALETTE.length] }} />
                          <span className="truncate max-w-[110px] text-[var(--text)]" title={m.model}>
                            {m.model.split('/').pop()}
                          </span>
                        </div>
                      </td>
                      <td className="py-2 text-right text-[var(--text-muted)]">{m.request_count.toLocaleString()}</td>
                      <td className="py-2 text-right text-[var(--text-muted)]">{(m.input_tokens + m.output_tokens).toLocaleString()}</td>
                      <td className="py-2 text-right font-medium text-[var(--warning)]">${m.cost.toFixed(4)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* ── Promo code ────────────────────────────────────────────────── */}
      <div className="card mb-5">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">Promo Code</h3>
        <div className="flex gap-3">
          <input value={promoCode} onChange={(e) => setPromoCode(e.target.value)} placeholder="输入促销码" />
          <button onClick={handleRedeem} className="btn-primary whitespace-nowrap">Redeem</button>
        </div>
        {message && <p className="text-[var(--success)] text-sm mt-2 font-medium">{message}</p>}
        {error && <p className="text-[var(--danger)] text-sm mt-2">{error}</p>}
      </div>

      {/* ── Daily breakdown table ─────────────────────────────────────── */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">每日明细（近 30 天）</h3>
        {breakdown.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-4 text-center">暂无用量数据</p>
        ) : (
          <div className="space-y-2">
            {[...breakdown].reverse().map((d) => {
              const maxCost = Math.max(...breakdown.map(x => x.cost), 0.0001)
              return (
                <div key={d.date} className="flex items-center gap-3 text-xs">
                  <span className="w-24 text-[var(--text-muted)] font-mono flex-shrink-0">{d.date}</span>
                  <div className="flex-1 bg-[var(--bg-secondary)] rounded-full h-4 overflow-hidden">
                    <div
                      className="h-full bg-[var(--warning)] rounded-full transition-all opacity-75"
                      style={{ width: `${(d.cost / maxCost) * 100}%`, minWidth: '2px' }}
                    />
                  </div>
                  <span className="w-20 text-right font-medium text-[var(--warning)]">${d.cost.toFixed(6)}</span>
                  <span className="w-20 text-right text-[var(--text-muted)]">
                    {(d.input_tokens + d.output_tokens).toLocaleString()} tok
                  </span>
                  <span className="w-16 text-right text-[var(--text-muted)]">
                    {d.request_count.toLocaleString()} req
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
