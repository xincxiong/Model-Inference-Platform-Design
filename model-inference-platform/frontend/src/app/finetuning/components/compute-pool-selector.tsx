'use client'

import { useEffect, useState, useMemo } from 'react'
import Link from 'next/link'
import { listComputePools, type ComputePool } from '@/lib/api'
import { SHARING_LABEL } from '../constants'

interface Props {
  /** Currently selected pool id (controlled). */
  value: string
  /** Called when the user picks a pool. */
  onChange: (poolId: string) => void
  /** Requested GPU count (controlled). */
  gpuRequest: number
  /** Called when user changes gpu request. */
  onGpuRequestChange: (n: number) => void
  /** Hard upper bound (defaults to selected pool's gpu_count). */
  maxGpu?: number
  /**
   * If true, show a secondary rollout pool dropdown (RL only).
   * Falls back to the training pool when empty.
   */
  rollout?: {
    value: string
    onChange: (poolId: string) => void
    gpuRequest: number
    onGpuRequestChange: (n: number) => void
  }
  /** Whether parallel topology fields exist; shown for validation hint. */
  parallelTopology?: { tp: number; pp: number; dp: number }
}

function daysUntil(iso?: string) {
  if (!iso) return null
  const ms = new Date(iso).getTime() - Date.now()
  return Math.max(0, Math.ceil(ms / 86400000))
}

export function ComputePoolSelector({
  value, onChange,
  gpuRequest, onGpuRequestChange,
  maxGpu, rollout, parallelTopology,
}: Props) {
  const [pools, setPools] = useState<ComputePool[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true
    setLoading(true)
    listComputePools()
      .then(r => {
        if (!alive) return
        const list = ((r as { data?: ComputePool[] }).data) || []
        // Only active pools with active subscriptions are usable
        setPools(list.filter(p => p.status === 'active' && p.subscription_status === 'active'))
      })
      .catch(e => alive && setError((e as Error).message))
      .finally(() => alive && setLoading(false))
    return () => { alive = false }
  }, [])

  const selected = useMemo(() => pools.find(p => p.id === value) || null, [pools, value])

  // Effective pool cap (used for both training and rollout)
  const cap = selected?.gpu_count ?? maxGpu ?? 8
  const free = selected ? Math.max(0, selected.gpu_count - selected.used_gpu) : cap

  // TP*PP*DP validation hint
  const topoInvalid = (() => {
    if (!parallelTopology) return false
    const { tp, pp, dp } = parallelTopology
    if (tp <= 0 || pp <= 0 || dp <= 0) return false
    return tp * pp * dp > gpuRequest
  })()

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-sm text-[var(--text)]">算力资源</h3>
        <Link href="/pools/subscribe" className="text-[11px] text-[var(--primary)] hover:underline">
          管理订阅 →
        </Link>
      </div>

      {error && <p className="text-[11px] text-[var(--danger)] mb-2">{error}</p>}

      {loading ? (
        <p className="text-[11px] text-[var(--text-muted)]">加载池列表…</p>
      ) : pools.length === 0 ? (
        <div className="rounded-lg border border-dashed border-[var(--border)] p-4 text-center">
          <p className="text-sm text-[var(--text-muted)] mb-2">还没有可用的算力池</p>
          <Link href="/pools/subscribe" className="btn-primary text-xs inline-block">立即购买</Link>
        </div>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                训练池 <span className="text-[var(--danger)]">*</span>
              </label>
              <select
                value={value}
                onChange={e => onChange(e.target.value)}
              >
                <option value="">— 选择算力池 —</option>
                {pools.map(p => (
                  <option key={p.id} value={p.id}>
                    {p.name} · {p.gpu_type} × {p.gpu_count} · {SHARING_LABEL[p.sharing_mode] || p.sharing_mode}
                  </option>
                ))}
              </select>
              {selected && (
                <div className="mt-2 text-[11px] text-[var(--text-muted)] space-y-0.5">
                  <p>
                    订阅到期：
                    {selected.subscription_end_at
                      ? ` ${new Date(selected.subscription_end_at).toLocaleDateString('zh-CN')}（剩 ${daysUntil(selected.subscription_end_at) ?? '?'} 天）`
                      : '—'}
                  </p>
                  <p>
                    当前占用：<span className="text-[var(--text)]">{selected.used_gpu}/{selected.gpu_count}</span>
                    {selected.sharing_mode === 'shared-fifo' && ` · 可用 ${free} 张`}
                    {selected.sharing_mode === 'exclusive' && selected.used_gpu > 0 && ' · 独占池繁忙中'}
                  </p>
                </div>
              )}
            </div>
            <div>
              <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                申请卡数 <span className="text-[var(--danger)]">*</span>
              </label>
              <input
                type="number"
                min={1}
                max={cap}
                value={gpuRequest}
                onChange={e => onGpuRequestChange(Math.max(1, Math.min(cap, Number(e.target.value) || 1)))}
              />
              <p className="text-[11px] text-[var(--text-muted)] mt-1.5">
                {selected
                  ? `池容量 ${selected.gpu_count} 张${selected.sharing_mode === 'exclusive' ? '（独占模式，一次性占用）' : '（共享模式，可与其他任务并发）'}`
                  : `建议不超过池容量 ${cap}`}
              </p>
              {topoInvalid && (
                <p className="text-[11px] text-[var(--warning)] mt-1">
                  ⚠ TP × PP × DP 超过申请卡数，请调整并行拓扑或增加申请数
                </p>
              )}
            </div>
          </div>

          {/* Rollout pool (RL only) */}
          {rollout && (
            <div className="grid gap-4 md:grid-cols-2 mt-4 pt-4 border-t border-[var(--border-light)]">
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                  Rollout 池
                  <span className="text-[var(--text-muted)]">（可选，不选则用训练池）</span>
                </label>
                <select
                  value={rollout.value}
                  onChange={e => rollout.onChange(e.target.value)}
                >
                  <option value="">— 复用训练池 —</option>
                  {pools.map(p => (
                    <option key={p.id} value={p.id}>
                      {p.name} · {p.gpu_type} × {p.gpu_count}
                    </option>
                  ))}
                </select>
                <p className="text-[11px] text-[var(--text-muted)] mt-1.5">
                  Rollout 引擎（SGLang）通常可用低成本 GPU
                </p>
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                  Rollout 申请卡数
                </label>
                <input
                  type="number"
                  min={1}
                  max={cap}
                  value={rollout.gpuRequest}
                  onChange={e => rollout.onGpuRequestChange(Math.max(1, Math.min(cap, Number(e.target.value) || 1)))}
                />
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
