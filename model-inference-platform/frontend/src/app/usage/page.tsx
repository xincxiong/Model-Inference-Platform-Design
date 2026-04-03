'use client'

import { useState, useEffect } from 'react'
import { fetchUsage, redeemPromo } from '@/lib/api'

interface DailyUsage {
  date: string
  input_tokens: number
  output_tokens: number
  cost: number
}

interface UsageData {
  balance: number
  total_spent: number
  total_tokens: number
  daily_breakdown: DailyUsage[]
}

export default function UsagePage() {
  const [usage, setUsage] = useState<UsageData | null>(null)
  const [promoCode, setPromoCode] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

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

  if (!usage) return (
    <div className="flex items-center gap-2 text-[var(--text-muted)]">
      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
      Loading...
    </div>
  )

  const breakdown = usage.daily_breakdown || []
  const maxCost = Math.max(...breakdown.map(d => d.cost), 0.001)

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">用量统计</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">查看账户余额、消费记录和每日用量</p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-3 gap-4 mb-5">
        <div className="card text-center">
          <div className="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">余额</div>
          <div className="text-3xl font-semibold text-[var(--success)]">${usage.balance.toFixed(4)}</div>
        </div>
        <div className="card text-center">
          <div className="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">累计消费</div>
          <div className="text-3xl font-semibold text-[var(--warning)]">${usage.total_spent.toFixed(4)}</div>
        </div>
        <div className="card text-center">
          <div className="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">累计 Tokens</div>
          <div className="text-3xl font-semibold text-[var(--accent)]">{usage.total_tokens.toLocaleString()}</div>
        </div>
      </div>

      {/* Promo code */}
      <div className="card mb-5">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">Promo Code</h3>
        <div className="flex gap-3">
          <input value={promoCode} onChange={(e) => setPromoCode(e.target.value)} placeholder="输入促销码" />
          <button onClick={handleRedeem} className="btn-primary whitespace-nowrap">Redeem</button>
        </div>
        {message && <p className="text-[var(--success)] text-sm mt-2 font-medium">{message}</p>}
        {error && <p className="text-[var(--danger)] text-sm mt-2">{error}</p>}
      </div>

      {/* Daily breakdown */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">Daily Usage (Last 30 days)</h3>
        {breakdown.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-4 text-center">暂无用量数据</p>
        ) : (
          <div className="space-y-2.5">
            {breakdown.map((d) => (
              <div key={d.date} className="flex items-center gap-3 text-sm">
                <span className="w-24 text-[var(--text-muted)] text-xs font-mono">{d.date}</span>
                <div className="flex-1 bg-[var(--bg-secondary)] rounded-full h-5 overflow-hidden">
                  <div
                    className="h-full bg-[var(--accent)] rounded-full transition-all"
                    style={{ width: `${(d.cost / maxCost) * 100}%`, minWidth: '2px' }}
                  />
                </div>
                <span className="w-24 text-right text-xs font-medium">${d.cost.toFixed(6)}</span>
                <span className="w-28 text-right text-[var(--text-muted)] text-xs">
                  {(d.input_tokens + d.output_tokens).toLocaleString()} tok
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
