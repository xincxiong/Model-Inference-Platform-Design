'use client'

import { METHOD_LABEL, SCENARIO_LABEL, STATUS_TAG } from '../constants'
import type { FineTuningJob } from '../types'

interface Props {
  jobs: FineTuningJob[]
  onRefresh: () => void
  onCancel: (id: string) => void
}

function formatTime(epochSec: number) {
  if (!epochSec) return '—'
  return new Date(epochSec * 1000).toLocaleString('zh-CN')
}

export function JobsList({ jobs, onRefresh, onCancel }: Props) {
  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-sm text-[var(--text)]">任务列表</h3>
        <button type="button" onClick={onRefresh} className="text-[11px] text-[var(--primary)] hover:underline">
          刷新
        </button>
      </div>

      {jobs.length === 0 ? (
        <p className="text-sm text-[var(--text-muted)] py-8 text-center">暂无任务，提交后会显示在这里</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-[var(--text-muted)] border-b border-[var(--border-light)]">
                <th className="pb-3 font-medium text-xs">ID</th>
                <th className="pb-3 font-medium text-xs">模型</th>
                <th className="pb-3 font-medium text-xs">方法</th>
                <th className="pb-3 font-medium text-xs">场景</th>
                <th className="pb-3 font-medium text-xs">算力</th>
                <th className="pb-3 font-medium text-xs">状态</th>
                <th className="pb-3 font-medium text-xs">输出模型</th>
                <th className="pb-3 font-medium text-xs">创建时间</th>
                <th className="pb-3 font-medium text-xs w-20"></th>
              </tr>
            </thead>
            <tbody>
              {jobs.map(j => (
                <tr key={j.id} className="border-b border-[var(--border-light)] hover:bg-[var(--bg-hover)] transition-colors">
                  <td className="py-3 pr-3 font-mono text-[11px] text-[var(--text-secondary)]">{j.id.slice(0, 14)}…</td>
                  <td className="py-3 pr-3 text-xs">{j.base_model}</td>
                  <td className="py-3 pr-3">
                    <span className={`tag ${['grpo','gspo','dapo','vapo','ppo','dpo'].includes(j.method) ? 'tag-purple' : 'tag-blue'} text-[10px]`}>
                      {METHOD_LABEL[j.method] || j.method}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-[11px] text-[var(--text-secondary)]">
                    {j.rollout_scenario ? (SCENARIO_LABEL[j.rollout_scenario] || j.rollout_scenario) : '—'}
                  </td>
                  <td className="py-3 pr-3 text-[11px] text-[var(--text-secondary)]">
                    {j.pool_id ? <span className="font-mono">{j.pool_id.slice(-8)} × {j.gpu_request}</span> : '—'}
                  </td>
                  <td className="py-3 pr-3">
                    <span className={`tag ${STATUS_TAG[j.status] || 'tag-gray'} text-[10px]`}>{j.status}</span>
                  </td>
                  <td className="py-3 pr-3 text-[var(--text-muted)] text-[11px] max-w-[160px] truncate">
                    {j.fine_tuned_model || '—'}
                  </td>
                  <td className="py-3 pr-3 text-[11px] text-[var(--text-muted)]">{formatTime(j.created_at)}</td>
                  <td className="py-3">
                    {(j.status === 'queued' || j.status === 'running') && (
                      <button type="button" className="btn-danger text-xs" onClick={() => onCancel(j.id)}>
                        取消
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
