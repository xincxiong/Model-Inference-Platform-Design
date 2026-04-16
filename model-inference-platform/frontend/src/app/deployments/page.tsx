'use client'

import { useEffect, useState } from 'react'
import { listDeployments, createDeployment, patchDeployment, deleteDeployment, fetchModels } from '@/lib/api'

interface Deployment {
  id: string
  name: string
  description: string
  model_name: string
  billing_mode: 'token' | 'tpu' | 'unit'
  min_replicas: number
  max_replicas: number
  status: string
  endpoint: string
  created_at: string
  updated_at: string
}

interface ModelInfo {
  id: string
  name: string
  type: string
  provider: string
}

// ── 计费方式 ──────────────────────────────────────────────────────────────
const billingModes = [
  { id: 'token' as const, label: '按 Token 调用', badge: 'tag-blue', desc: '按实际消耗的输入/输出 Token 计费，适合低频或不规律调用' },
  { id: 'tpu'   as const, label: '按置备吐单元', badge: 'tag-purple', desc: '预留固定推理吞吐能力，适合稳定高并发业务' },
  { id: 'unit'  as const, label: '按模型单元',   badge: 'tag-amber',  desc: '独占单张 GPU 实例，适合对延迟敏感的生产场景' },
]

const statusConfig: Record<string, { label: string; cls: string }> = {
  provisioning: { label: '启动中', cls: 'tag-amber' },
  running:      { label: '运行中', cls: 'tag-green' },
  stopped:      { label: '已停止', cls: 'tag-blue'  },
  failed:       { label: '失败',   cls: 'tag-pink'  },
  deleting:     { label: '删除中', cls: 'tag-orange' },
}

const billingLabel: Record<string, string> = {
  token: '按 Token',
  tpu:   '按置备吐单元',
  unit:  '按模型单元',
}

// ── 空状态图标 ─────────────────────────────────────────────────────────────
function EmptyIllustration() {
  return (
    <svg width="96" height="72" viewBox="0 0 96 72" fill="none" xmlns="http://www.w3.org/2000/svg">
      <ellipse cx="48" cy="62" rx="32" ry="6" fill="#e8eaed" />
      <rect x="18" y="16" width="60" height="42" rx="8" fill="#f1f3f4" stroke="#dadce0" strokeWidth="1.5" />
      <rect x="26" y="24" width="20" height="3" rx="1.5" fill="#dadce0" />
      <rect x="26" y="31" width="44" height="2.5" rx="1.25" fill="#e8eaed" />
      <rect x="26" y="37" width="36" height="2.5" rx="1.25" fill="#e8eaed" />
      <rect x="26" y="43" width="28" height="2.5" rx="1.25" fill="#e8eaed" />
      <circle cx="66" cy="26" r="10" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5" strokeDasharray="3 2" />
      <path d="M62 26 h8 M66 22 v8" stroke="#1a73e8" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  )
}

// ── 步骤卡片（空状态下展示） ──────────────────────────────────────────────
const steps = [
  {
    num: 1, title: '选择模型',
    desc: '选择模型进行部署，平台支持部署系统预置模型或训练完成待部署的模型',
    svg: (
      <svg viewBox="0 0 120 80" fill="none" className="w-full h-full">
        <circle cx="60" cy="36" r="22" fill="#e8f0fe"/><circle cx="60" cy="36" r="14" fill="#c5d9fb"/>
        <circle cx="60" cy="36" r="7" fill="#1a73e8" opacity="0.7"/>
        <line x1="30" y1="36" x2="46" y2="36" stroke="#1a73e8" strokeWidth="2" strokeDasharray="3 2"/>
        <line x1="74" y1="36" x2="90" y2="36" stroke="#1a73e8" strokeWidth="2" strokeDasharray="3 2"/>
        <circle cx="26" cy="36" r="5" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5"/>
        <circle cx="94" cy="36" r="5" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5"/>
        <circle cx="60" cy="14" r="5" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5"/>
        <circle cx="60" cy="58" r="5" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5"/>
        <line x1="60" y1="19" x2="60" y2="30" stroke="#1a73e8" strokeWidth="2" strokeDasharray="3 2"/>
        <line x1="60" y1="42" x2="60" y2="53" stroke="#1a73e8" strokeWidth="2" strokeDasharray="3 2"/>
      </svg>
    ),
  },
  {
    num: 2, title: '按不同计费方式部署',
    desc: '根据选择的模型类型不同，可以选择按 Token 调用、按置备吐单元和按模型单元等计费方式进行部署',
    svg: (
      <svg viewBox="0 0 120 80" fill="none" className="w-full h-full">
        <rect x="20" y="20" width="80" height="40" rx="8" fill="#e8f0fe"/>
        <rect x="28" y="28" width="28" height="24" rx="4" fill="#c5d9fb"/>
        <rect x="64" y="28" width="28" height="24" rx="4" fill="#c5d9fb"/>
        <circle cx="42" cy="40" r="8" fill="#1a73e8" opacity="0.6"/>
        <path d="M40 40 l3 3 5-5" stroke="white" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
        <rect x="68" y="36" width="16" height="3" rx="1.5" fill="#1a73e8" opacity="0.5"/>
        <rect x="68" y="41" width="10" height="3" rx="1.5" fill="#1a73e8" opacity="0.3"/>
      </svg>
    ),
  },
  {
    num: 3, title: '调用模型服务',
    desc: '部署的模型服务正常运行后，可以通过 API 调用，支持模型运行监控和日志观测',
    svg: (
      <svg viewBox="0 0 120 80" fill="none" className="w-full h-full">
        <rect x="16" y="18" width="88" height="44" rx="6" fill="#e8f0fe"/>
        <rect x="22" y="24" width="76" height="32" rx="4" fill="#c5d9fb" opacity="0.6"/>
        <path d="M30 50 L42 38 L52 44 L64 32 L76 38 L86 28" stroke="#1a73e8" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" fill="none"/>
        <circle cx="42" cy="38" r="2.5" fill="#1a73e8"/>
        <circle cx="64" cy="32" r="2.5" fill="#1a73e8"/>
        <circle cx="86" cy="28" r="2.5" fill="#1a73e8"/>
      </svg>
    ),
  },
]

// ── 部署弹窗 ──────────────────────────────────────────────────────────────
function DeployModal({ models, onClose, onCreated }: {
  models: ModelInfo[]
  onClose: () => void
  onCreated: (d: Deployment) => void
}) {
  const [step, setStep] = useState<1 | 2 | 3>(1)
  const [selectedModel, setSelectedModel] = useState('')
  const [billing, setBilling] = useState<'token' | 'tpu' | 'unit'>('token')
  const [name, setName] = useState('')
  const [minR, setMinR] = useState(0)
  const [maxR, setMaxR] = useState(3)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const textModels = models.filter(m => m.type === 'text-to-text' || m.type === 'vision')

  async function handleDeploy() {
    if (!name.trim()) { setError('请填写部署名称'); return }
    setSubmitting(true); setError('')
    try {
      const d = await createDeployment({
        name: name.trim(),
        model_name: selectedModel,
        billing_mode: billing,
        min_replicas: minR,
        max_replicas: maxR,
      })
      onCreated(d)
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-white rounded-2xl shadow-2xl w-[560px] max-h-[88vh] flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-light)]">
          <div>
            <h2 className="text-base font-semibold text-[var(--text)]">部署新模型</h2>
            <p className="text-xs text-[var(--text-muted)] mt-0.5">按需单独部署模型，通过 API 使用模型推理服务</p>
          </div>
          <button onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-muted)] transition-colors">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        {/* Steps */}
        <div className="flex items-center gap-2 px-6 py-3 bg-[var(--bg-secondary)]">
          {(['选择模型', '计费方式', '配置确认'] as const).map((label, i) => {
            const s = i + 1; const active = step === s; const done = step > s
            return (
              <div key={s} className="flex items-center gap-2">
                <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-semibold ${done ? 'bg-[var(--success)] text-white' : active ? 'bg-[var(--accent)] text-white' : 'bg-[var(--border)] text-[var(--text-muted)]'}`}>
                  {done ? <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg> : s}
                </div>
                <span className={`text-xs font-medium ${active ? 'text-[var(--text)]' : 'text-[var(--text-muted)]'}`}>{label}</span>
                {i < 2 && <div className="w-8 h-px bg-[var(--border-light)] mx-1" />}
              </div>
            )
          })}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">
          {/* Step 1 */}
          {step === 1 && (
            <div className="space-y-2">
              <p className="text-sm font-medium text-[var(--text)] mb-3">选择要部署的模型</p>
              {textModels.length === 0 && <p className="text-sm text-[var(--text-muted)]">暂无可部署模型</p>}
              {textModels.map(m => (
                <div key={m.id} onClick={() => setSelectedModel(m.id)}
                  className={`flex items-center justify-between p-3.5 rounded-xl border cursor-pointer transition-all ${selectedModel === m.id ? 'border-[var(--accent)] bg-[var(--accent-light)]' : 'border-[var(--border-light)] hover:border-[var(--border)] hover:bg-[var(--bg-secondary)]'}`}>
                  <div className="flex items-center gap-3">
                    <div className={`w-9 h-9 rounded-lg flex items-center justify-center ${selectedModel === m.id ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-secondary)] text-[var(--text-muted)]'}`}>
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-[var(--text)]">{m.name}</p>
                      <p className="text-xs text-[var(--text-muted)]">{m.provider}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="tag tag-blue text-[11px]">{m.type === 'vision' ? '多模态' : '对话补全'}</span>
                    {selectedModel === m.id && <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Step 2 */}
          {step === 2 && (
            <div className="space-y-2">
              <p className="text-sm font-medium text-[var(--text)] mb-3">选择计费方式</p>
              {billingModes.map(b => (
                <div key={b.id} onClick={() => setBilling(b.id)}
                  className={`p-4 rounded-xl border cursor-pointer transition-all ${billing === b.id ? 'border-[var(--accent)] bg-[var(--accent-light)]' : 'border-[var(--border-light)] hover:border-[var(--border)] hover:bg-[var(--bg-secondary)]'}`}>
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium text-[var(--text)]">{b.label}</p>
                    {billing === b.id && <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>}
                  </div>
                  <p className="text-xs text-[var(--text-muted)] mt-1">{b.desc}</p>
                </div>
              ))}
            </div>
          )}

          {/* Step 3 */}
          {step === 3 && (
            <div className="space-y-5">
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">部署名称 *</label>
                <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. my-deepseek-v3" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最小副本数</label>
                  <input type="number" min={0} max={maxR} value={minR} onChange={e => setMinR(Number(e.target.value))} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最大副本数</label>
                  <input type="number" min={minR} max={20} value={maxR} onChange={e => setMaxR(Number(e.target.value))} />
                </div>
              </div>
              <div className="rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-light)] p-4 space-y-2">
                <p className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide">部署摘要</p>
                {[
                  ['模型', textModels.find(m => m.id === selectedModel)?.name ?? '—'],
                  ['计费方式', billingModes.find(b => b.id === billing)?.label ?? '—'],
                  ['副本范围', `${minR} ~ ${maxR}`],
                ].map(([k, v]) => (
                  <div key={k} className="flex items-center justify-between text-sm">
                    <span className="text-[var(--text-muted)]">{k}</span>
                    <span className="font-medium text-[var(--text)]">{v}</span>
                  </div>
                ))}
              </div>
              {error && <p className="text-xs text-[var(--danger)]">{error}</p>}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-[var(--border-light)]">
          <button onClick={() => step > 1 ? setStep((step - 1) as 1|2|3) : onClose()} className="btn-outline">
            {step === 1 ? '取消' : '上一步'}
          </button>
          <button
            onClick={() => step < 3 ? setStep((step + 1) as 2|3) : handleDeploy()}
            disabled={(step === 1 && !selectedModel) || submitting}
            className="btn-primary flex items-center gap-2"
          >
            {submitting ? (
              <><svg className="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>部署中...</>
            ) : step < 3 ? '下一步' : '确认部署'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── 主页面 ────────────────────────────────────────────────────────────────
export default function DeploymentsPage() {
  const [deployments, setDeployments] = useState<Deployment[]>([])
  const [models, setModels] = useState<ModelInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([listDeployments(), fetchModels()]).then(([dl, ml]) => {
      setDeployments(dl.data ?? [])
      setModels(ml.data ?? [])
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  async function handleDelete(id: string) {
    if (!confirm('确认删除该部署？')) return
    setDeletingId(id)
    try {
      await deleteDeployment(id)
      setDeployments(prev => prev.filter(d => d.id !== id))
    } finally {
      setDeletingId(null)
    }
  }

  async function handleStop(dep: Deployment) {
    const next = dep.status === 'running' ? 'stopped' : 'running'
    try {
      const updated = await patchDeployment(dep.id, { status: next })
      setDeployments(prev => prev.map(d => d.id === dep.id ? updated : d))
    } catch {}
  }

  const isEmpty = !loading && deployments.length === 0

  return (
    <div className="min-h-screen flex flex-col">
      {/* Top bar */}
      <div className="flex items-center justify-between px-8 py-5">
        <h1 className="text-[15px] font-semibold text-[var(--text)]">模型部署</h1>
        <div className="flex items-center gap-3">
          <button className="btn-outline flex items-center gap-1.5 text-sm py-2 px-4">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            使用指南
          </button>
          <button onClick={() => setShowModal(true)} className="btn-primary flex items-center gap-1.5 text-sm py-2 px-4">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            部署新模型
          </button>
        </div>
      </div>

      {/* Loading */}
      {loading && (
        <div className="flex-1 flex items-center justify-center">
          <div className="flex items-center gap-2 text-[var(--text-muted)] text-sm">
            <svg className="animate-spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
            加载中...
          </div>
        </div>
      )}

      {/* Empty state */}
      {isEmpty && (
        <div className="flex-1 flex flex-col items-center justify-center gap-10 pb-16">
          <div className="flex flex-col items-center gap-4 text-center">
            <EmptyIllustration />
            <div>
              <p className="text-[15px] font-semibold text-[var(--text)]">你还没有部署过模型</p>
              <p className="text-sm text-[var(--text-muted)] mt-1">按需求单独部署模型，通过 API 使用模型推理服务</p>
            </div>
          </div>
          <div className="flex items-stretch gap-5 max-w-[900px] w-full px-8">
            {steps.map(s => (
              <div key={s.num} className="flex-1 rounded-2xl border border-[var(--border-light)] bg-white overflow-hidden" style={{ boxShadow: 'var(--shadow-sm)' }}>
                <div className="px-5 pt-5 pb-3">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-lg font-bold text-[var(--accent)] opacity-60">{s.num}</span>
                    <p className="text-sm font-semibold text-[var(--text)]">{s.title}</p>
                  </div>
                  <p className="text-xs text-[var(--text-muted)] leading-relaxed">{s.desc}</p>
                </div>
                <div className="px-5 pb-5 h-28 flex items-end">
                  <div className="w-full h-full opacity-80">{s.svg}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Deployment list */}
      {!loading && deployments.length > 0 && (
        <div className="px-8 pb-8">
          <div className="bg-white rounded-2xl border border-[var(--border-light)] overflow-hidden" style={{ boxShadow: 'var(--shadow-sm)' }}>
            <table>
              <thead>
                <tr className="border-b border-[var(--border-light)]">
                  <th className="px-5 py-3.5 text-left">名称 / 模型</th>
                  <th className="px-5 py-3.5 text-left">计费方式</th>
                  <th className="px-5 py-3.5 text-left">副本</th>
                  <th className="px-5 py-3.5 text-left">状态</th>
                  <th className="px-5 py-3.5 text-left">Endpoint</th>
                  <th className="px-5 py-3.5 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {deployments.map(d => {
                  const st = statusConfig[d.status] ?? { label: d.status, cls: 'tag-blue' }
                  return (
                    <tr key={d.id} className="hover:bg-[var(--bg-secondary)] transition-colors">
                      <td className="px-5 py-4">
                        <p className="text-sm font-medium text-[var(--text)]">{d.name}</p>
                        <p className="text-xs text-[var(--text-muted)] mt-0.5">{d.model_name}</p>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-xs text-[var(--text-secondary)]">{billingLabel[d.billing_mode] ?? d.billing_mode}</span>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-sm text-[var(--text-secondary)]">{d.min_replicas} – {d.max_replicas}</span>
                      </td>
                      <td className="px-5 py-4">
                        <span className={`tag ${st.cls}`}>{st.label}</span>
                      </td>
                      <td className="px-5 py-4 max-w-[200px]">
                        <p className="text-xs text-[var(--text-muted)] truncate font-mono">{d.endpoint}</p>
                      </td>
                      <td className="px-5 py-4">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => handleStop(d)}
                            className="text-xs text-[var(--accent)] hover:underline"
                          >
                            {d.status === 'running' ? '停止' : '启动'}
                          </button>
                          <button
                            onClick={() => handleDelete(d.id)}
                            disabled={deletingId === d.id}
                            className="text-xs text-[var(--danger)] hover:underline disabled:opacity-50"
                          >
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {showModal && (
        <DeployModal
          models={models}
          onClose={() => setShowModal(false)}
          onCreated={d => setDeployments(prev => [d, ...prev])}
        />
      )}
    </div>
  )
}
