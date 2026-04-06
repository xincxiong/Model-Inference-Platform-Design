'use client'

import { useEffect, useState } from 'react'
import {
  fetchDedicatedTemplates,
  listDedicatedEndpoints,
  createDedicatedEndpoint,
  deleteDedicatedEndpoint,
  patchDedicatedEndpoint,
} from '@/lib/api'

interface Template {
  model_name: string
  name: string
  model_type: string
  provider: string
}

interface DedicatedEndpoint {
  id: string
  name: string
  model_name: string
  routing_key: string
  routing_prefix: string
  status: string
  gpu_type: string
  region: string
  min_replicas: number
  max_replicas: number
  created_at: string
}

export default function EndpointsPage() {
  const [templates, setTemplates] = useState<Template[]>([])
  const [endpoints, setEndpoints] = useState<DedicatedEndpoint[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [name, setName] = useState('')
  const [modelName, setModelName] = useState('')
  const [gpuType, setGpuType] = useState('A100-80GB')
  const [region, setRegion] = useState('cn-east-1')
  const [minRep, setMinRep] = useState(0)
  const [maxRep, setMaxRep] = useState(2)

  const load = () => {
    Promise.all([fetchDedicatedTemplates(), listDedicatedEndpoints()])
      .then(([t, e]) => {
        setTemplates(t.templates || [])
        setEndpoints(e.data || [])
        if (!modelName && (t.templates || []).length > 0) {
          setModelName(t.templates[0].model_name)
        }
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [])

  const handleCreate = async () => {
    if (!name || !modelName) return
    setError('')
    try {
      await createDedicatedEndpoint({
        name,
        model_name: modelName,
        gpu_type: gpuType,
        region,
        min_replicas: minRep,
        max_replicas: maxRep,
      })
      setName('')
      load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '创建失败')
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('删除该专属端点？')) return
    try {
      await deleteDedicatedEndpoint(id)
      load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '删除失败')
    }
  }

  const toggleStopped = async (ep: DedicatedEndpoint) => {
    const next = ep.status === 'stopped' ? 'running' : 'stopped'
    try {
      await patchDedicatedEndpoint(ep.id, { status: next })
      load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '更新失败')
    }
  }

  if (loading) {
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
        <h2 className="text-xl font-semibold text-[var(--text)]">专属端点</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">
          创建后使用 routing_key 作为 OpenAI 请求中的 <code className="text-xs bg-[var(--bg-hover)] px-1 rounded">model</code>，例如{' '}
          <code className="text-xs">ep_ab12cd34:deepseek-ai/DeepSeek-V4</code>
        </p>
      </div>

      {error && (
        <div className="text-[var(--danger)] bg-[var(--danger-bg)] text-sm px-4 py-3 rounded-lg mb-4">{error}</div>
      )}

      <div className="card mb-6">
        <h3 className="font-medium text-sm text-[var(--text)] mb-3">新建端点</h3>
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">名称</label>
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="生产环境对话" />
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">基座模型</label>
            <select value={modelName} onChange={(e) => setModelName(e.target.value)}>
              {templates.map((t) => (
                <option key={t.model_name} value={t.model_name}>
                  {t.name} ({t.model_type})
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">GPU 类型</label>
            <input value={gpuType} onChange={(e) => setGpuType(e.target.value)} />
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">区域</label>
            <input value={region} onChange={(e) => setRegion(e.target.value)} />
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">最小副本</label>
            <input type="number" min={0} value={minRep} onChange={(e) => setMinRep(Number(e.target.value))} />
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">最大副本</label>
            <input type="number" min={1} value={maxRep} onChange={(e) => setMaxRep(Number(e.target.value))} />
          </div>
        </div>
        <button type="button" onClick={handleCreate} className="btn-primary mt-4">
          创建
        </button>
      </div>

      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">我的端点</h3>
        {endpoints.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-6 text-center">暂无端点</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm min-w-[720px]">
              <thead>
                <tr className="text-left border-b border-[var(--border-light)]">
                  <th className="pb-3 pr-3">名称</th>
                  <th className="pb-3 pr-3">routing_key</th>
                  <th className="pb-3 pr-3">状态</th>
                  <th className="pb-3 pr-3">GPU / 区域</th>
                  <th className="pb-3 w-40">操作</th>
                </tr>
              </thead>
              <tbody>
                {endpoints.map((ep) => (
                  <tr key={ep.id} className="border-b border-[var(--border-light)]">
                    <td className="py-3 pr-3 font-medium text-[var(--text)]">{ep.name}</td>
                    <td className="py-3 pr-3">
                      <code className="text-[11px] break-all text-[var(--text-secondary)]">{ep.routing_key}</code>
                      <button
                        type="button"
                        className="ml-2 text-xs text-[var(--accent)]"
                        onClick={() => navigator.clipboard.writeText(ep.routing_key)}
                      >
                        复制
                      </button>
                    </td>
                    <td className="py-3 pr-3">
                      <span className={`tag ${ep.status === 'running' ? 'tag-green' : 'tag-amber'}`}>{ep.status}</span>
                    </td>
                    <td className="py-3 pr-3 text-[var(--text-muted)]">
                      {ep.gpu_type} · {ep.region}
                    </td>
                    <td className="py-3 flex flex-wrap gap-2">
                      <button type="button" className="btn-outline text-xs" onClick={() => toggleStopped(ep)}>
                        {ep.status === 'stopped' ? '启动' : '停止'}
                      </button>
                      <button type="button" className="btn-danger text-xs" onClick={() => handleDelete(ep.id)}>
                        删除
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
