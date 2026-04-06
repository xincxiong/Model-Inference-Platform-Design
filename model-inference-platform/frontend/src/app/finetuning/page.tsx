'use client'

import { useEffect, useState } from 'react'
import { listFineTuningJobs, createFineTuningJob, cancelFineTuningJob } from '@/lib/api'

interface FineTuningJob {
  id: string
  object: string
  model: string
  training_file?: string
  method: string
  status: string
  fine_tuned_model?: string
  created_at: number
  updated_at: number
  error?: { message: string }
}

export default function FinetuningPage() {
  const [jobs, setJobs] = useState<FineTuningJob[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [baseModel, setBaseModel] = useState('deepseek-ai/DeepSeek-V4')
  const [trainingFile, setTrainingFile] = useState('file-demo-001')
  const [method, setMethod] = useState('lora')

  const load = async () => {
    try {
      const res = await listFineTuningJobs()
      setJobs(res.data || [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const handleCreate = async () => {
    setError('')
    try {
      await createFineTuningJob({
        model: baseModel,
        training_file: trainingFile,
        method,
      })
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '创建失败')
    }
  }

  const handleCancel = async (id: string) => {
    try {
      await cancelFineTuningJob(id)
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '取消失败')
    }
  }

  if (loading && jobs.length === 0) {
    return (
      <div className="flex items-center gap-2 text-[var(--text-muted)]">
        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
        加载中…
      </div>
    )
  }

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">模型微调</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">
          OpenAI 风格 <code className="text-xs bg-[var(--bg-hover)] px-1 rounded">/v1/fine_tuning/jobs</code>（MVP：异步占位，约 1.5s 后标记为成功）
        </p>
      </div>

      {error && (
        <div className="text-[var(--danger)] bg-[var(--danger-bg)] text-sm px-4 py-3 rounded-lg mb-4">{error}</div>
      )}

      <div className="card mb-6">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">新建微调任务</h3>
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">基座模型 ID</label>
            <input value={baseModel} onChange={(e) => setBaseModel(e.target.value)} />
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">训练文件 ID（占位）</label>
            <input value={trainingFile} onChange={(e) => setTrainingFile(e.target.value)} />
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">方法</label>
            <select value={method} onChange={(e) => setMethod(e.target.value)}>
              <option value="lora">LoRA</option>
              <option value="full">Full</option>
            </select>
          </div>
        </div>
        <button type="button" onClick={handleCreate} className="btn-primary mt-4">
          提交任务
        </button>
      </div>

      <div className="card">
        <div className="flex justify-between items-center mb-4">
          <h3 className="font-medium text-sm text-[var(--text)]">任务列表</h3>
          <button type="button" className="btn-outline text-xs" onClick={() => { setLoading(true); load() }}>
            刷新
          </button>
        </div>
        {jobs.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-6 text-center">暂无任务</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b border-[var(--border-light)]">
                <th className="pb-3 pr-3">ID</th>
                <th className="pb-3 pr-3">基座模型</th>
                <th className="pb-3 pr-3">状态</th>
                <th className="pb-3 pr-3">产出模型</th>
                <th className="pb-3 w-24"></th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((j) => (
                <tr key={j.id} className="border-b border-[var(--border-light)]">
                  <td className="py-3 pr-3 font-mono text-[11px] text-[var(--text-secondary)]">{j.id}</td>
                  <td className="py-3 pr-3">{j.model}</td>
                  <td className="py-3 pr-3">
                    <span className="tag tag-blue">{j.status}</span>
                  </td>
                  <td className="py-3 pr-3 text-[var(--text-muted)] text-xs">
                    {j.fine_tuned_model || '—'}
                  </td>
                  <td className="py-3">
                    {(j.status === 'queued' || j.status === 'running') && (
                      <button type="button" className="btn-danger text-xs" onClick={() => handleCancel(j.id)}>
                        取消
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
