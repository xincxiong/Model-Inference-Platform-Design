'use client'

import { useState, useMemo } from 'react'
import { LineChart, BarChart } from '@/components/charts'

// ── Mock cost data ───────────────────────────────────────────────────────────
const MODELS = [
  { name: 'DeepSeek-V3', inputPrice: 0.27, outputPrice: 1.10, color: '#1a73e8' },
  { name: 'Qwen3.5-72B', inputPrice: 0.33, outputPrice: 1.32, color: '#e37400' },
  { name: 'GLM-5', inputPrice: 0.20, outputPrice: 0.80, color: '#1e8e3e' },
  { name: 'Llama-4-70B', inputPrice: 0.15, outputPrice: 0.60, color: '#7627bb' },
  { name: 'Kimi-K2.5', inputPrice: 0.40, outputPrice: 1.60, color: '#c5221f' },
  { name: '豆包2.0', inputPrice: 0.12, outputPrice: 0.48, color: '#007b83' },
]

function generateDailyCost(days: number) {
  return Array.from({ length: days }, (_, i) => {
    const d = new Date(Date.now() - (days - 1 - i) * 86400000)
    const base = 5 + Math.random() * 15
    return {
      date: d.toLocaleDateString('zh', { month: '2-digit', day: '2-digit' }),
      cost: parseFloat(base.toFixed(2)),
      inputTokens: Math.floor(500000 + Math.random() * 2000000),
      outputTokens: Math.floor(100000 + Math.random() * 800000),
      requests: Math.floor(2000 + Math.random() * 8000),
    }
  })
}

const DAILY_COST = generateDailyCost(30)

interface Budget {
  name: string
  limit: number
  spent: number
  period: 'month' | 'project'
  alertThresholds: number[]
}

const BUDGETS: Budget[] = [
  { name: '生产环境', limit: 500, spent: 342.87, period: 'month', alertThresholds: [50, 80, 100] },
  { name: '开发测试', limit: 100, spent: 67.23, period: 'month', alertThresholds: [50, 80, 100] },
  { name: 'RLHF 训练', limit: 2000, spent: 1247.50, period: 'project', alertThresholds: [50, 80, 90, 100] },
]

const FORECAST_MONTHS = ['5月', '6月', '7月', '8月', '9月', '10月']

export default function CostAnalyticsPage() {
  const [activeTab, setActiveTab] = useState<'overview' | 'forecast' | 'budgets'>('overview')
  const [modelBreakdown, setModelBreakdown] = useState(true)

  // Compute model-level costs
  const modelCosts = useMemo(() => {
    const totalTokens = DAILY_COST.reduce((s, d) => s + d.inputTokens + d.outputTokens, 0)
    return MODELS.map(m => {
      const share = 0.1 + Math.random() * 0.25
      const outTokens = Math.floor(totalTokens * share * 0.2)
      const inTokens = Math.floor(totalTokens * share * 0.8)
      return {
        ...m,
        cost: parseFloat(((inTokens / 1e6) * m.inputPrice + (outTokens / 1e6) * m.outputPrice).toFixed(2)),
        inputTokens: inTokens,
        outputTokens: outTokens,
        share: parseFloat((share * 100).toFixed(1)),
      }
    }).sort((a, b) => b.cost - a.cost)
  }, [])

  const totalCost = modelCosts.reduce((s, m) => s + m.cost, 0)
  const todayCost = DAILY_COST[DAILY_COST.length - 1]?.cost || 0
  const avgDaily = DAILY_COST.reduce((s, d) => s + d.cost, 0) / DAILY_COST.length
  const projectedMonthly = avgDaily * 30

  const costSeries = DAILY_COST.map(d => d.cost)
  const costLabels = DAILY_COST.map(d => d.date)

  return (
    <div>
      {/* ── Header ─────────────────────────────────────────────────── */}
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">成本分析</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">费用明细、趋势预测、预算告警</p>
      </div>

      {/* ── Tabs ───────────────────────────────────────────────────── */}
      <div className="flex gap-1 mb-5 bg-white border border-[var(--border)] rounded-lg p-1 w-fit">
        {([
          ['overview', '概览'],
          ['forecast', '预测'],
          ['budgets', '预算'],
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

      {/* ── Stat cards ─────────────────────────────────────────────── */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-5">
        <StatCard label="本月累计" value={`$${totalCost.toFixed(2)}`} color="var(--warning)" sub={`${DAILY_COST.length} 天`} />
        <StatCard label="今日花费" value={`$${todayCost.toFixed(2)}`} color="var(--accent)" sub={`日均 $${avgDaily.toFixed(2)}`} />
        <StatCard label="预计月度" value={`$${projectedMonthly.toFixed(2)}`} color={projectedMonthly > 400 ? 'var(--danger)' : 'var(--success)'} sub="基于 30 天平均" />
        <StatCard label="平均单价" value={`$${(totalCost / (DAILY_COST.reduce((s,d)=>s+d.inputTokens+d.outputTokens,0) / 1e6)).toFixed(2)}`} color="var(--text-secondary)" sub="/M tokens" />
      </div>

      {/* ── Cost trend ─────────────────────────────────────────────── */}
      {activeTab === 'overview' && (
        <>
          <div className="card mb-5">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-medium text-sm text-[var(--text)]">每日成本趋势（近 30 天）</h3>
              <label className="flex items-center gap-2 text-xs text-[var(--text-muted)] cursor-pointer">
                <input type="checkbox" checked={modelBreakdown} onChange={e => setModelBreakdown(e.target.checked)} />
                按模型拆分
              </label>
            </div>
            {modelBreakdown ? (
              <StackedBarChart dailyCost={DAILY_COST} models={MODELS} />
            ) : (
              <LineChart labels={costLabels} datasets={[{ label: 'Cost ($)', data: costSeries, color: 'var(--warning)', width: 2 }]} yAxisLabel="$" height={200} />
            )}
          </div>

          {/* Model cost table */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-3">模型成本分布</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                    <th className="text-left pb-2.5 font-medium">模型</th>
                    <th className="text-right pb-2.5 font-medium">占比</th>
                    <th className="text-right pb-2.5 font-medium">成本</th>
                    <th className="text-right pb-2.5 font-medium">Input $/M</th>
                    <th className="text-right pb-2.5 font-medium">Output $/M</th>
                    <th className="text-right pb-2.5 font-medium">Input Tokens</th>
                    <th className="text-right pb-2.5 font-medium">Output Tokens</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--border-light)]">
                  {modelCosts.map((m) => {
                    const pct = (m.cost / totalCost) * 100
                    return (
                      <tr key={m.name} className="hover:bg-[var(--bg-secondary)]">
                        <td className="py-2.5 pr-3">
                          <div className="flex items-center gap-2">
                            <span className="w-3 h-3 rounded-sm flex-shrink-0" style={{ background: m.color }} />
                            <span className="font-medium text-[var(--text)]">{m.name}</span>
                          </div>
                        </td>
                        <td className="py-2.5 pr-3">
                          <div className="flex items-center gap-2">
                            <div className="w-16 bg-[var(--bg-secondary)] rounded h-2 overflow-hidden">
                              <div className="h-full rounded" style={{ width: `${pct}%`, background: m.color }} />
                            </div>
                            <span className="text-[var(--text-muted)]">{pct.toFixed(1)}%</span>
                          </div>
                        </td>
                        <td className="py-2.5 pr-3 text-right font-medium text-[var(--warning)]">${m.cost.toFixed(2)}</td>
                        <td className="py-2.5 pr-3 text-right font-mono text-[var(--text-muted)]">${m.inputPrice.toFixed(2)}</td>
                        <td className="py-2.5 pr-3 text-right font-mono text-[var(--text-muted)]">${m.outputPrice.toFixed(2)}</td>
                        <td className="py-2.5 pr-3 text-right font-mono">{(m.inputTokens / 1000).toFixed(0)}K</td>
                        <td className="py-2.5 text-right font-mono">{(m.outputTokens / 1000).toFixed(0)}K</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            <div className="mt-3 pt-3 border-t border-[var(--border-light)] flex justify-between text-xs text-[var(--text-muted)]">
              <span>总成本</span>
              <span className="font-medium text-[var(--warning)]">${totalCost.toFixed(2)}</span>
            </div>
          </div>
        </>
      )}

      {/* ── Forecast ───────────────────────────────────────────────── */}
      {activeTab === 'forecast' && <ForecastTab avgDaily={avgDaily} projectedMonthly={projectedMonthly} />}

      {/* ── Budgets ────────────────────────────────────────────────── */}
      {activeTab === 'budgets' && <BudgetsTab budgets={BUDGETS} />}
    </div>
  )
}

function StatCard({ label, value, color, sub }: { label: string; value: string; color: string; sub: string }) {
  return (
    <div className="card py-4">
      <div className="text-[11px] font-medium text-[var(--text-muted)] uppercase tracking-wider mb-1">{label}</div>
      <div className="text-xl font-semibold" style={{ color }}>{value}</div>
      <div className="text-[11px] text-[var(--text-muted)] mt-0.5">{sub}</div>
    </div>
  )
}

// ── Stacked bar chart by model (SVG) ──────────────────────────────────────────
function StackedBarChart({ dailyCost, models }: { dailyCost: { date: string; cost: number }[]; models: { name: string; color: string }[] }) {
  const pad = 40
  const w = 800
  const h = 210
  const chartH = 180
  const maxV = Math.max(...dailyCost.map(d => d.cost))
  const bars = dailyCost.slice(-30)
  const barW = Math.max((w - pad - 10) / bars.length - 1, 2)

  const gridSteps = 4
  const gridLines = Array.from({ length: gridSteps + 1 }, (_, i) => {
    const v = (maxV * i) / gridSteps
    const y = 10 + chartH - (v / maxV) * chartH
    return { y, label: `$${v.toFixed(0)}` }
  })

  return (
    <div className="overflow-x-auto">
      <svg viewBox={`0 0 ${w} ${h}`} className="w-full min-w-[500px]" style={{ maxHeight: `${h}px` }}>
        {gridLines.map((g, i) => (
          <g key={i}>
            <line x1={pad} y1={g.y} x2={w - 10} y2={g.y} stroke="var(--border-light)" strokeWidth="0.5" />
            <text x={pad - 4} y={g.y + 4} textAnchor="end" fill="var(--text-muted)" fontSize="10">{g.label}</text>
          </g>
        ))}
        {bars.map((bar, i) => {
          const x = pad + (i / bars.length) * (w - pad - 10)
          let yAccum = 10 + chartH
          const segs = models.map((m, mi) => {
            const segH = (bar.cost / models.length * (0.5 + Math.random())) / maxV * chartH
            const y = yAccum - segH
            yAccum = y
            return <rect key={mi} x={x} y={y} width={barW} height={segH} fill={m.color} rx="0.5" opacity="0.85" />
          })
          return <g key={i}>{segs}</g>
        })}
        {/* x labels */}
        {bars.filter((_, i) => i % 5 === 0).map((b, i, a) => {
          const idx = bars.indexOf(b)
          const x = pad + (idx / bars.length) * (w - pad - 10)
          return <text key={i} x={x + barW / 2} y={h - 2} textAnchor="middle" fill="var(--text-muted)" fontSize="9">{b.date}</text>
        })}
      </svg>
    </div>
  )
}

// ── Forecast Tab ──────────────────────────────────────────────────────────────
function ForecastTab({ avgDaily, projectedMonthly }: { avgDaily: number; projectedMonthly: number }) {
  const forecast = FORECAST_MONTHS.map((_, i) => {
    const growth = 1 + i * 0.08
    const base = projectedMonthly * growth
    const optimistic = base * 0.85
    const pessimistic = base * 1.2
    return { month: FORECAST_MONTHS[i], base: parseFloat(base.toFixed(0)), optimistic: parseFloat(optimistic.toFixed(0)), pessimistic: parseFloat(pessimistic.toFixed(0)) }
  })

  return (
    <div className="space-y-4">
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">费用预测（未来 6 个月）</h3>
        <p className="text-xs text-[var(--text-muted)] mb-4">基于过去 30 天日均 ${avgDaily.toFixed(2)}，假设月增长 8%</p>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                <th className="text-left pb-2.5 font-medium">月份</th>
                <th className="text-right pb-2.5 font-medium">基准预测</th>
                <th className="text-right pb-2.5 font-medium text-[var(--success)]">乐观场景</th>
                <th className="text-right pb-2.5 font-medium text-[var(--danger)]">悲观场景</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border-light)]">
              {forecast.map((f) => (
                <tr key={f.month} className="hover:bg-[var(--bg-secondary)]">
                  <td className="py-2.5 pr-3 font-medium text-[var(--text)]">{f.month}</td>
                  <td className="py-2.5 pr-3 text-right font-mono text-[var(--warning)]">${f.base.toLocaleString()}</td>
                  <td className="py-2.5 pr-3 text-right font-mono text-[var(--success)]">${f.optimistic.toLocaleString()}</td>
                  <td className="py-2.5 text-right font-mono text-[var(--danger)]">${f.pessimistic.toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Bar chart of forecast */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">预测趋势图</h3>
        <BarChart labels={FORECAST_MONTHS} data={forecast.map(f => f.base)} color="var(--warning)" height={160} />
      </div>

      {/* Cost optimization tips */}
      <div className="card border-l-4 border-l-[var(--success)]">
        <h3 className="font-medium text-sm text-[var(--success)] mb-2">成本优化建议</h3>
        <ul className="text-xs text-[var(--text-secondary)] space-y-1.5">
          <li>· 将 15% 的批量推理任务迁移至 Flex Tier，预计节省 $42/月</li>
          <li>· DeepSeek-V3 替换部分 Qwen3.5 调用（同等质量，成本低 23%），预计节省 $78/月</li>
          <li>· 开启语义缓存后，重复 Query 命中率约 35%，预计节省 $120/月</li>
        </ul>
      </div>
    </div>
  )
}

// ── Budgets Tab ───────────────────────────────────────────────────────────────
function BudgetsTab({ budgets }: { budgets: Budget[] }) {
  return (
    <div className="space-y-4">
      {budgets.map(b => {
        const pct = (b.spent / b.limit) * 100
        const color = pct > 90 ? 'var(--danger)' : pct > 70 ? 'var(--warning)' : 'var(--success)'
        return (
          <div key={b.name} className="card">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h3 className="font-medium text-sm text-[var(--text)]">{b.name}</h3>
                <p className="text-xs text-[var(--text-muted)]">周期：{b.period === 'month' ? '月度' : '项目'}</p>
              </div>
              <button className="text-xs px-3 py-1.5 border border-[var(--border)] rounded-md text-[var(--text-secondary)] hover:border-[var(--accent)] transition-colors">编辑</button>
            </div>

            {/* Progress bar */}
            <div className="mb-3">
              <div className="flex justify-between text-xs mb-1">
                <span className="text-[var(--text-muted)]">${b.spent.toFixed(2)} / ${b.limit.toFixed(2)}</span>
                <span className="font-medium" style={{ color }}>{pct.toFixed(1)}%</span>
              </div>
              <div className="w-full bg-[var(--bg-secondary)] rounded h-4 overflow-hidden">
                <div className="h-full rounded transition-all" style={{ width: `${Math.min(pct, 100)}%`, background: color }} />
              </div>
            </div>

            {/* Alert thresholds */}
            <div className="text-xs text-[var(--text-muted)]">
              告警阈值：
              {b.alertThresholds.map(t => (
                <span key={t} className={`inline-block mr-2 ${pct >= t ? 'font-medium text-[var(--danger)]' : ''}`}>
                  {pct >= t ? `✓ ${t}%` : `${t}%`}
                </span>
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
