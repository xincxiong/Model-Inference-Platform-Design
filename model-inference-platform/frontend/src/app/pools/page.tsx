'use client'

import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import {
  listPoolSubscriptions,
  getPoolOverview,
  cancelPoolSubscription,
  renewPoolSubscription,
  listPoolInvoices,
  PoolSubscription,
  SubscriptionOverview,
  PoolInvoice,
} from '@/lib/api'

const STATUS_TAG: Record<string, string> = {
  active:    'tag-green',
  pending:   'tag-amber',
  overdue:   'tag-pink',
  suspended: 'tag-orange',
  expired:   'tag-gray',
  cancelled: 'tag-gray',
}
const STATUS_LABEL: Record<string, string> = {
  active:    '生效中',
  pending:   '待支付',
  overdue:   '欠费',
  suspended: '已冻结',
  expired:   '已到期',
  cancelled: '已取消',
}

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function daysUntil(s: string) {
  const ms = new Date(s).getTime() - Date.now()
  return Math.max(0, Math.ceil(ms / 86400000))
}

function formatCNY(n: number) {
  return '¥' + n.toLocaleString('zh-CN', { maximumFractionDigits: 0 })
}

export default function PoolsPage() {
  const [subs, setSubs] = useState<PoolSubscription[]>([])
  const [overview, setOverview] = useState<SubscriptionOverview | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [invoices, setInvoices] = useState<Record<string, PoolInvoice[]>>({})

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [s, o] = await Promise.all([listPoolSubscriptions(), getPoolOverview()])
      setSubs((s as { data: PoolSubscription[] }).data || [])
      setOverview(o)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const handleCancel = async (id: string, immediate: boolean) => {
    if (!confirm(immediate ? '立即取消此订阅？此操作不可撤销。' : '将在本订阅周期结束时取消，确认？')) return
    try {
      await cancelPoolSubscription(id, immediate)
      await load()
    } catch (e) {
      alert('取消失败：' + (e as Error).message)
    }
  }

  const handleRenew = async (id: string) => {
    try {
      await renewPoolSubscription(id)
      await load()
    } catch (e) {
      alert('续费失败：' + (e as Error).message)
    }
  }

  const toggleInvoices = async (id: string) => {
    if (expandedId === id) { setExpandedId(null); return }
    setExpandedId(id)
    if (!invoices[id]) {
      try {
        const r = await listPoolInvoices(id)
        setInvoices(prev => ({ ...prev, [id]: (r as { data: PoolInvoice[] }).data || [] }))
      } catch (e) {
        setInvoices(prev => ({ ...prev, [id]: [] }))
      }
    }
  }

  return (
    <div className="p-8 max-w-7xl">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-[var(--text)]">算力池</h1>
          <p className="text-sm text-[var(--text-muted)] mt-1">订阅算力容量，微调任务与端点按需使用</p>
        </div>
        <Link href="/pools/subscribe" className="btn-primary text-sm">+ 购买算力池</Link>
      </div>

      {/* Overview cards */}
      {overview && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="card">
            <p className="text-[11px] uppercase tracking-wider text-[var(--text-muted)]">生效订阅</p>
            <p className="text-2xl font-semibold mt-1.5 text-[var(--text)]">{overview.active_count}</p>
          </div>
          <div className="card">
            <p className="text-[11px] uppercase tracking-wider text-[var(--text-muted)]">月均花费</p>
            <p className="text-2xl font-semibold mt-1.5 text-[var(--text)]">{formatCNY(overview.monthly_spend)}</p>
          </div>
          <div className="card">
            <p className="text-[11px] uppercase tracking-wider text-[var(--text-muted)]">7 天内续费</p>
            <p className="text-2xl font-semibold mt-1.5 text-[var(--warning)]">{overview.upcoming_renewals}</p>
          </div>
          <div className="card">
            <p className="text-[11px] uppercase tracking-wider text-[var(--text-muted)]">累计节省</p>
            <p className="text-2xl font-semibold mt-1.5 text-emerald-600">{formatCNY(overview.total_savings)}</p>
            <p className="text-[10px] text-[var(--text-muted)] mt-0.5">相比按 GPU·小时</p>
          </div>
        </div>
      )}

      {/* Subscriptions list */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-medium text-sm text-[var(--text)]">我的订阅</h2>
          {subs.length > 0 && <span className="text-[11px] text-[var(--text-muted)]">{subs.length} 个订阅</span>}
        </div>

        {loading && <p className="text-sm text-[var(--text-muted)] py-8 text-center">加载中…</p>}
        {error && <p className="text-sm text-[var(--danger)] py-4">加载失败：{error}</p>}

        {!loading && !error && subs.length === 0 && (
          <div className="py-12 text-center">
            <p className="text-sm text-[var(--text-muted)] mb-3">还没有算力池订阅</p>
            <Link href="/pools/subscribe" className="btn-primary text-sm inline-block">立即购买</Link>
          </div>
        )}

        {!loading && subs.length > 0 && (
          <div className="space-y-3">
            {subs.map(s => {
              const days = daysUntil(s.end_at)
              const expiringSoon = s.status === 'active' && days <= 7
              const isExpanded = expandedId === s.id
              return (
                <div key={s.id} className="rounded-lg border border-[var(--border-light)] overflow-hidden">
                  <div className="p-4 flex items-center gap-4">
                    {/* GPU badge */}
                    <div className="flex-shrink-0 w-14 h-14 rounded-lg bg-gradient-to-br from-[var(--accent)]/15 to-[var(--accent)]/5 flex flex-col items-center justify-center">
                      <span className="text-[10px] font-semibold text-[var(--accent)]">{s.gpu_type}</span>
                      <span className="text-base font-bold text-[var(--text)]">×{s.gpu_count}</span>
                    </div>

                    {/* Main info */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <p className="text-sm font-medium text-[var(--text)] truncate">{s.sku_name || s.sku_id}</p>
                        <span className={`tag ${STATUS_TAG[s.status] || 'tag-gray'} text-[10px]`}>{STATUS_LABEL[s.status] || s.status}</span>
                        {s.trial && <span className="tag tag-cyan text-[10px]">试用</span>}
                        {s.auto_renew && s.status === 'active' && <span className="tag tag-blue text-[10px]">自动续费</span>}
                      </div>
                      <p className="text-[11px] text-[var(--text-muted)] mt-1">
                        {formatDate(s.start_at)} → {formatDate(s.end_at)}
                        {s.status === 'active' && <span className={`ml-2 ${expiringSoon ? 'text-[var(--warning)] font-medium' : ''}`}>
                          {days === 0 ? '今天到期' : `还剩 ${days} 天`}
                        </span>}
                      </p>
                    </div>

                    {/* Amount */}
                    <div className="text-right flex-shrink-0">
                      <p className="text-base font-semibold text-[var(--text)]">{formatCNY(s.total_amount)}</p>
                      <p className="text-[10px] text-[var(--text-muted)]">{s.currency}</p>
                    </div>

                    {/* Actions */}
                    <div className="flex gap-1.5 flex-shrink-0">
                      <button type="button" onClick={() => toggleInvoices(s.id)} className="btn-outline text-xs px-2.5 py-1">
                        {isExpanded ? '收起' : '账单'}
                      </button>
                      {s.status === 'active' && (
                        <>
                          <button type="button" onClick={() => handleRenew(s.id)} className="btn-outline text-xs px-2.5 py-1">续费</button>
                          <button type="button" onClick={() => handleCancel(s.id, false)} className="btn-danger text-xs px-2.5 py-1">取消</button>
                        </>
                      )}
                    </div>
                  </div>

                  {/* Invoices panel */}
                  {isExpanded && (
                    <div className="border-t border-[var(--border-light)] bg-[var(--bg-hover)]/30 px-4 py-3">
                      <p className="text-[11px] font-medium text-[var(--text-secondary)] mb-2">账单记录</p>
                      {(!invoices[s.id] || invoices[s.id].length === 0) && (
                        <p className="text-[11px] text-[var(--text-muted)]">暂无账单</p>
                      )}
                      {invoices[s.id] && invoices[s.id].length > 0 && (
                        <table className="w-full text-[11px]">
                          <thead>
                            <tr className="text-[var(--text-muted)]">
                              <th className="text-left font-medium pb-1">账期</th>
                              <th className="text-left font-medium pb-1">金额</th>
                              <th className="text-left font-medium pb-1">状态</th>
                              <th className="text-left font-medium pb-1">支付时间</th>
                            </tr>
                          </thead>
                          <tbody>
                            {invoices[s.id].map(inv => (
                              <tr key={inv.id} className="border-t border-[var(--border-light)]/50">
                                <td className="py-1.5 text-[var(--text)]">{formatDate(inv.period_start)} → {formatDate(inv.period_end)}</td>
                                <td className="py-1.5 font-medium">{formatCNY(inv.amount)}</td>
                                <td className="py-1.5"><span className={`tag ${inv.status === 'paid' ? 'tag-green' : 'tag-amber'} text-[10px]`}>{inv.status}</span></td>
                                <td className="py-1.5 text-[var(--text-muted)]">{inv.paid_at ? new Date(inv.paid_at).toLocaleString('zh-CN') : '—'}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
