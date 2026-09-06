'use client'

import { useEffect, useState, useMemo } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { listPoolSKUs, purchasePoolSubscription, PoolSKU } from '@/lib/api'

const GPU_BADGE: Record<string, string> = {
  'A100-80GB':  'tag-purple',
  'A100-40GB':  'tag-blue',
  'H100-80GB':  'tag-amber',
  'L40S-48GB':  'tag-green',
  'RTX4090-24GB': 'tag-pink',
}

const TERM_LABEL: Record<string, string> = {
  monthly:   '月付',
  quarterly: '季付',
  yearly:    '年付',
}
const SHARING_LABEL: Record<string, string> = {
  exclusive:    '独占',
  'shared-fifo': '共享',
}

function formatCNY(n: number) {
  return '¥' + n.toLocaleString('zh-CN', { maximumFractionDigits: 0 })
}

export default function SubscribePage() {
  const router = useRouter()
  const [skus, setSkus] = useState<PoolSKU[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [gpuFilter, setGpuFilter] = useState<string>('all')
  const [sharingFilter, setSharingFilter] = useState<string>('all')
  const [submitting, setSubmitting] = useState<string | null>(null)
  const [trialSku, setTrialSku] = useState<string | null>(null)

  useEffect(() => {
    listPoolSKUs()
      .then(r => setSkus((r as { data: PoolSKU[] }).data || []))
      .catch(e => setError((e as Error).message))
      .finally(() => setLoading(false))
  }, [])

  // Group SKUs by gpu_type + gpu_count + sharing_mode for the comparison card
  const groups = useMemo(() => {
    const map = new Map<string, { variants: PoolSKU[]; gpu_type: string; gpu_count: number; sharing_mode: string }>()
    skus.forEach(s => {
      const key = `${s.gpu_type}|${s.gpu_count}|${s.sharing_mode}`
      const existing = map.get(key)
      if (existing) {
        existing.variants.push(s)
      } else {
        map.set(key, { variants: [s], gpu_type: s.gpu_type, gpu_count: s.gpu_count, sharing_mode: s.sharing_mode })
      }
    })
    return Array.from(map.values())
  }, [skus])

  const filtered = groups.filter(g =>
    (gpuFilter === 'all' || g.gpu_type === gpuFilter) &&
    (sharingFilter === 'all' || g.sharing_mode === sharingFilter)
  )

  const gpuOptions = useMemo(() => Array.from(new Set(skus.map(s => s.gpu_type))).sort(), [skus])

  const handlePurchase = async (sku: PoolSKU, trial: boolean) => {
    const confirmMsg = trial
      ? `确认开启 7 天免费试用「${sku.name}」？试用期内可随时取消。`
      : `确认购买「${sku.name}」${TERM_LABEL[sku.term]}？将立即扣费 ${formatCNY(sku.term_price)}。`
    if (!confirm(confirmMsg)) return
    setSubmitting(sku.id)
    setTrialSku(trial ? sku.id : null)
    try {
      await purchasePoolSubscription(sku.id, true, trial)
      router.push('/pools')
    } catch (e) {
      alert('购买失败：' + (e as Error).message)
      setSubmitting(null)
    }
  }

  return (
    <div className="p-8 max-w-7xl">
      <div className="flex items-center gap-2 text-xs text-[var(--text-muted)] mb-2">
        <Link href="/pools" className="hover:text-[var(--text)]">算力池</Link>
        <span>/</span>
        <span className="text-[var(--text)]">购买算力</span>
      </div>

      <div className="mb-6">
        <h1 className="text-xl font-semibold text-[var(--text)]">购买算力池</h1>
        <p className="text-sm text-[var(--text-muted)] mt-1">
          包年包月订阅算力容量，池内任务按需使用不额外计费。年付最高可省 40%。
        </p>
      </div>

      <div className="card mb-4">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-xs text-[var(--text-muted)]">GPU 类型</span>
            <div className="flex gap-1">
              {(['all', ...gpuOptions] as const).map(g => (
                <button key={g} type="button" onClick={() => setGpuFilter(g)}
                  className={`px-2.5 py-1 rounded text-[11px] font-medium transition-colors ${
                    gpuFilter === g ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'
                  }`}>
                  {g === 'all' ? '全部' : g}
                </button>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-[var(--text-muted)]">使用模式</span>
            <div className="flex gap-1">
              {(['all', 'exclusive', 'shared-fifo'] as const).map(m => (
                <button key={m} type="button" onClick={() => setSharingFilter(m)}
                  className={`px-2.5 py-1 rounded text-[11px] font-medium transition-colors ${
                    sharingFilter === m ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'
                  }`}>
                  {m === 'all' ? '全部' : SHARING_LABEL[m] || m}
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>

      {loading && <p className="text-sm text-[var(--text-muted)] py-8 text-center">加载中…</p>}
      {error && <p className="text-sm text-[var(--danger)] py-4">加载失败：{error}</p>}

      {!loading && !error && (
        <div className="space-y-4">
          {filtered.length === 0 && (
            <p className="text-sm text-[var(--text-muted)] py-8 text-center">没有匹配的套餐</p>
          )}
          {filtered.map(g => {
            const sortedVariants = [...g.variants].sort((a, b) => a.term_months - b.term_months)
            const monthlyTerm = sortedVariants.find(v => v.term === 'monthly')
            const maxDiscount = Math.max(...sortedVariants.map(x => x.discount_pct))
            return (
              <div key={`${g.gpu_type}-${g.gpu_count}-${g.sharing_mode}`} className="card">
                <div className="flex items-start gap-4 mb-4">
                  <div className="flex-shrink-0 w-16 h-16 rounded-lg bg-gradient-to-br from-[var(--accent)]/15 to-[var(--accent)]/5 flex flex-col items-center justify-center">
                    <span className="text-[10px] font-semibold text-[var(--accent)]">{g.gpu_type}</span>
                    <span className="text-base font-bold text-[var(--text)]">×{g.gpu_count}</span>
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold text-base text-[var(--text)]">{g.gpu_type} × {g.gpu_count}</h3>
                      <span className={`tag ${GPU_BADGE[g.gpu_type] || 'tag-blue'} text-[10px]`}>{g.gpu_type}</span>
                      <span className="tag tag-cyan text-[10px]">{SHARING_LABEL[g.sharing_mode] || g.sharing_mode}</span>
                      {g.variants.some(v => v.sla_class === 'enhanced') && <span className="tag tag-amber text-[10px]">SLA 增强</span>}
                    </div>
                    <p className="text-xs text-[var(--text-muted)] mt-1">华东一区 · 池内任务共享容量，停止使用不退还</p>
                  </div>
                </div>

                <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${sortedVariants.length}, minmax(0, 1fr))` }}>
                  {sortedVariants.map(v => {
                    const isBest = v.discount_pct === maxDiscount && maxDiscount > 0
                    return (
                      <div key={v.id} className={`relative rounded-lg border p-4 transition-all ${
                        isBest ? 'border-[var(--accent)]/40 bg-[var(--accent)]/5 shadow-sm' : 'border-[var(--border-light)] bg-white'
                      }`}>
                        {isBest && <span className="absolute -top-2 left-3 px-1.5 py-0.5 rounded text-[10px] font-medium bg-[var(--accent)] text-white">最划算</span>}
                        <p className="text-xs font-medium text-[var(--text)]">{TERM_LABEL[v.term]}</p>
                        <p className="text-2xl font-bold text-[var(--text)] mt-1">{formatCNY(v.term_price)}</p>
                        <p className="text-[10px] text-[var(--text-muted)] mt-0.5">
                          月均 {formatCNY(v.term_price / v.term_months)}
                        </p>
                        {v.discount_pct > 0 ? (
                          <p className="text-[11px] text-emerald-600 font-medium mt-1.5">省 {v.discount_pct}%</p>
                        ) : (
                          <p className="text-[11px] text-[var(--text-muted)] mt-1.5">基准价</p>
                        )}
                        {monthlyTerm && v.id !== monthlyTerm.id && (
                          <p className="text-[10px] text-[var(--text-muted)] mt-0.5">
                            比月付省 {formatCNY(monthlyTerm.term_price * v.term_months - v.term_price)}
                          </p>
                        )}
                        <div className="mt-3 space-y-1.5">
                          <button type="button" onClick={() => handlePurchase(v, false)} disabled={submitting === v.id}
                            className="btn-primary w-full text-xs py-1.5 disabled:opacity-50">
                            {submitting === v.id && trialSku !== v.id ? '处理中…' : '立即购买'}
                          </button>
                          <button type="button" onClick={() => handlePurchase(v, true)} disabled={submitting === v.id}
                            className="btn-outline w-full text-xs py-1.5 disabled:opacity-50">
                            {submitting === v.id && trialSku === v.id ? '处理中…' : '免费试用 7 天'}
                          </button>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
