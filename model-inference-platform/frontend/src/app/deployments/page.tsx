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

// ── 计费方案定义（含详细价格）─────────────────────────────────────────────
const TOKEN_PRICE_TIERS = [
  { label: 'DeepSeek-V3', input: 0.27, output: 1.10 },
  { label: 'DeepSeek-R1', input: 0.55, output: 2.19 },
  { label: 'Qwen2.5-72B', input: 0.42, output: 1.26 },
  { label: 'GLM-4-Air', input: 0.14, output: 0.14 },
  { label: '其他模型', input: 0.50, output: 2.00 },
]

const TPU_SPECS = [
  { name: '基础型', qps: 5, concur: 10, price: 0.50, tag: '适合开发测试' },
  { name: '标准型', qps: 20, concur: 50, price: 1.80, tag: '适合中等流量', recommend: true },
  { name: '高性能型', qps: 100, concur: 200, price: 6.50, tag: '适合高并发生产' },
  { name: '旗舰型', qps: 500, concur: 1000, price: 28.00, tag: '金融/实时业务' },
]

const GPU_SPECS = [
  { name: 'A10', mem: '24GB', price: 1.60, tag: '中等推理' },
  { name: 'L40S', mem: '48GB', price: 2.80, tag: '均衡性价比', recommend: true },
  { name: 'A100 80GB', mem: '80GB', price: 3.50, tag: '大模型首选' },
  { name: 'H100 80GB', mem: '80GB', price: 5.80, tag: '最高性能' },
]

interface BillingModeConfig {
  id: 'token' | 'tpu' | 'unit'
  label: string
  badge: string
  icon: JSX.Element
  tagline: string
  desc: string
  suitable: string
  priceUnit: string
  startFrom: string
}

const billingModes: BillingModeConfig[] = [
  {
    id: 'token',
    label: '按 Token 调用',
    badge: 'tag-blue',
    tagline: '调用多少付多少',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><circle cx="12" cy="12" r="10"/><path d="M12 8v4l3 3"/></svg>,
    desc: '按实际消耗的输入/输出 Token 计费，无需预付，不调用不产生费用。适合低频、不规律调用场景。',
    suitable: '开发/测试 · 低频调用 · 弹性业务',
    priceUnit: '元 / 百万 Token',
    startFrom: '¥0.14',
  },
  {
    id: 'tpu',
    label: '按置备吐单元',
    badge: 'tag-purple',
    tagline: '预留吞吐保障响应',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/><line x1="12" y1="12" x2="12" y2="16"/><line x1="10" y1="14" x2="14" y2="14"/></svg>,
    desc: '预购固定 QPS 吞吐能力，保障稳定延迟。超出额度后自动降速，适合并发稳定的在线业务。',
    suitable: '在线服务 · 稳定并发 · SLA 要求高',
    priceUnit: '元 / 小时（按规格）',
    startFrom: '¥0.50',
  },
  {
    id: 'unit',
    label: '按模型单元',
    badge: 'tag-amber',
    tagline: '独占 GPU 极致性能',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M15 2v2M9 2v2M2 9h2M2 15h2M22 9h-2M22 15h-2M15 22v-2M9 22v-2"/></svg>,
    desc: '独占单张/多张 GPU 实例，毫秒级冷启动、最低延迟，支持私有化数据隔离，适合对延迟敏感的生产核心链路。',
    suitable: '核心生产 · 延迟敏感 · 数据隔离',
    priceUnit: '元 / GPU·小时',
    startFrom: '¥1.60',
  },
]

const statusConfig: Record<string, { label: string; cls: string; dot: string }> = {
  provisioning: { label: '启动中', cls: 'tag-amber', dot: 'bg-amber-400' },
  running:      { label: '运行中', cls: 'tag-green', dot: 'bg-green-400' },
  stopped:      { label: '已停止', cls: 'tag-blue',  dot: 'bg-blue-400'  },
  failed:       { label: '失败',   cls: 'tag-pink',  dot: 'bg-red-400'   },
  deleting:     { label: '删除中', cls: 'tag-orange', dot: 'bg-orange-400' },
}

const billingLabel: Record<string, string> = {
  token: '按 Token',
  tpu:   '置备吐单元',
  unit:  '模型单元',
}

const billingBadge: Record<string, string> = {
  token: 'tag-blue',
  tpu:   'tag-purple',
  unit:  'tag-amber',
}

// Token 模式估算（百万 token 均价）
function estimateTokenCost(billing: 'token' | 'tpu' | 'unit', tpuSpec?: number, gpuSpec?: number, replicas?: number): { hourly?: number; note: string } {
  if (billing === 'token') return { note: '按实际 Token 用量计费，不调用不扣费' }
  if (billing === 'tpu') {
    const price = TPU_SPECS[tpuSpec ?? 1].price
    return { hourly: price, note: `¥${price}/时 · 超限自动降速` }
  }
  if (billing === 'unit') {
    const price = GPU_SPECS[gpuSpec ?? 1].price
    const r = replicas ?? 1
    return { hourly: price * r, note: `¥${price} × ${r} 副本 = ¥${(price * r).toFixed(2)}/时` }
  }
  return { note: '' }
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
    desc: '根据业务需求选择按 Token 调用、按置备吐单元或按模型单元计费，价格透明可预期',
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

// ── 模型类型图标 ──────────────────────────────────────────────────────────
const typeIcon: Record<string, JSX.Element> = {
  'text-to-text': <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>,
  'vision':       <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>,
}
const typeLabel: Record<string, string> = {
  'text-to-text': '对话补全',
  'vision':       '多模态',
}

// ── 部署弹窗 ──────────────────────────────────────────────────────────────
function DeployModal({ models, onClose, onCreated }: {
  models: ModelInfo[]
  onClose: () => void
  onCreated: (d: Deployment) => void
}) {
  const [step, setStep] = useState<1 | 2 | 3>(1)
  const [selectedModel, setSelectedModel] = useState('')
  const [billing, setBilling] = useState<'token' | 'tpu' | 'unit'>('token')
  const [tpuSpecIdx, setTpuSpecIdx] = useState(1)
  const [gpuSpecIdx, setGpuSpecIdx] = useState(1)
  const [name, setName] = useState('')
  const [minR, setMinR] = useState(0)
  const [maxR, setMaxR] = useState(3)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<'all' | 'text-to-text' | 'vision'>('all')

  const deployableModels = models.filter(m => m.type === 'text-to-text' || m.type === 'vision')
  const textModels = deployableModels.filter(m => {
    const matchType = typeFilter === 'all' || m.type === typeFilter
    const q = search.toLowerCase()
    const matchSearch = !q || m.name.toLowerCase().includes(q) || m.id.toLowerCase().includes(q) || m.provider.toLowerCase().includes(q)
    return matchType && matchSearch
  })

  const selectedModelInfo = deployableModels.find(m => m.id === selectedModel)
  const billingConfig = billingModes.find(b => b.id === billing)!
  const costEst = estimateTokenCost(billing, tpuSpecIdx, gpuSpecIdx, maxR)

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

  const canNext = step === 1 ? !!selectedModel : true

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-white rounded-2xl shadow-2xl w-[620px] max-h-[90vh] flex flex-col overflow-hidden">
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
                <div className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-semibold transition-all ${done ? 'bg-[var(--success)] text-white' : active ? 'bg-[var(--accent)] text-white' : 'bg-[var(--border)] text-[var(--text-muted)]'}`}>
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

          {/* ──── Step 1: 选择模型 ──── */}
          {step === 1 && (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <svg className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
                  <input value={search} onChange={e => setSearch(e.target.value)} placeholder="搜索模型名称、厂商…"
                    className="pl-8 pr-3 py-2 text-sm w-full rounded-lg border border-[var(--border-light)] bg-[var(--bg-secondary)] focus:outline-none focus:border-[var(--accent)] transition-colors" />
                </div>
                <div className="flex items-center gap-1 bg-[var(--bg-secondary)] rounded-lg border border-[var(--border-light)] p-0.5">
                  {([['all', '全部'], ['text-to-text', '对话'], ['vision', '多模态']] as const).map(([v, label]) => (
                    <button key={v} onClick={() => setTypeFilter(v)}
                      className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${typeFilter === v ? 'bg-white text-[var(--text)] shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >{label}</button>
                  ))}
                </div>
              </div>
              <p className="text-xs text-[var(--text-muted)]">
                共 <span className="font-semibold text-[var(--text)]">{deployableModels.length}</span> 个可部署模型，当前筛选 <span className="font-semibold text-[var(--text)]">{textModels.length}</span> 个
              </p>
              <div className="space-y-1.5 max-h-[340px] overflow-y-auto pr-0.5">
                {textModels.length === 0 && <div className="py-10 text-center text-sm text-[var(--text-muted)]">未找到匹配的模型</div>}
                {textModels.map(m => (
                  <div key={m.id} onClick={() => setSelectedModel(m.id)}
                    className={`flex items-center justify-between p-3 rounded-xl border cursor-pointer transition-all ${selectedModel === m.id ? 'border-[var(--accent)] bg-[var(--accent-light)]' : 'border-[var(--border-light)] hover:border-[var(--border)] hover:bg-[var(--bg-secondary)]'}`}>
                    <div className="flex items-center gap-3 min-w-0">
                      <div className={`w-8 h-8 rounded-lg flex-shrink-0 flex items-center justify-center ${selectedModel === m.id ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-secondary)] text-[var(--text-muted)]'}`}>
                        {typeIcon[m.type] ?? <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>}
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-[var(--text)] truncate">{m.name}</p>
                        <p className="text-xs text-[var(--text-muted)] truncate">{m.id}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0 ml-2">
                      <span className="text-[10px] text-[var(--text-muted)] bg-[var(--bg-secondary)] px-1.5 py-0.5 rounded">{m.provider}</span>
                      <span className={`tag text-[10px] ${m.type === 'vision' ? 'tag-purple' : 'tag-blue'}`}>{typeLabel[m.type]}</span>
                      {selectedModel === m.id && <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* ──── Step 2: 计费方式（含详细价格） ──── */}
          {step === 2 && (
            <div className="space-y-3">
              <div className="flex items-start gap-2 mb-1">
                <svg width="14" height="14" className="mt-0.5 flex-shrink-0 text-[var(--accent)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                <p className="text-xs text-[var(--text-muted)] leading-relaxed">选择计费方式后，展开可查看详细价格表。不同计费方式适用不同场景，选择前请确认业务需求。</p>
              </div>

              {billingModes.map(b => {
                const active = billing === b.id
                return (
                  <div key={b.id} className={`rounded-xl border transition-all overflow-hidden ${active ? 'border-[var(--accent)] shadow-sm' : 'border-[var(--border-light)] hover:border-[var(--border)]'}`}>
                    {/* 模式选择头 */}
                    <div
                      onClick={() => setBilling(b.id)}
                      className={`flex items-center gap-3 p-4 cursor-pointer ${active ? 'bg-[var(--accent-light)]' : 'hover:bg-[var(--bg-secondary)]'}`}
                    >
                      <div className={`w-9 h-9 rounded-lg flex-shrink-0 flex items-center justify-center ${active ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-secondary)] text-[var(--text-muted)]'}`}>
                        {b.icon}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-semibold text-[var(--text)]">{b.label}</span>
                          <span className={`tag text-[10px] ${b.badge}`}>{b.tagline}</span>
                        </div>
                        <p className="text-xs text-[var(--text-muted)] mt-0.5 truncate">{b.suitable}</p>
                      </div>
                      <div className="text-right flex-shrink-0">
                        <p className="text-xs text-[var(--text-muted)]">起步价</p>
                        <p className="text-sm font-bold text-[var(--text)]">{b.startFrom}</p>
                        <p className="text-[10px] text-[var(--text-muted)]">{b.priceUnit}</p>
                      </div>
                      <div className={`w-5 h-5 rounded-full border-2 flex-shrink-0 flex items-center justify-center transition-all ${active ? 'border-[var(--accent)] bg-[var(--accent)]' : 'border-[var(--border)]'}`}>
                        {active && <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>}
                      </div>
                    </div>

                    {/* 展开详细价格表 */}
                    {active && (
                      <div className="border-t border-[var(--border-light)] bg-white px-4 pb-4 pt-3 space-y-3">
                        <p className="text-xs font-medium text-[var(--text-secondary)]">{b.desc}</p>

                        {/* Token 模式：价格表 */}
                        {b.id === 'token' && (
                          <div>
                            <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-2">参考价格（元/百万 Token）</p>
                            <div className="rounded-lg overflow-hidden border border-[var(--border-light)]">
                              <table className="w-full text-xs">
                                <thead>
                                  <tr className="bg-[var(--bg-secondary)]">
                                    <th className="text-left px-3 py-2 text-[var(--text-muted)] font-medium">模型</th>
                                    <th className="text-right px-3 py-2 text-[var(--text-muted)] font-medium">输入</th>
                                    <th className="text-right px-3 py-2 text-[var(--text-muted)] font-medium">输出</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {TOKEN_PRICE_TIERS.map((t, i) => (
                                    <tr key={i} className="border-t border-[var(--border-light)]">
                                      <td className="px-3 py-2 text-[var(--text)]">{t.label}</td>
                                      <td className="px-3 py-2 text-right font-medium text-green-600">¥{t.input.toFixed(2)}</td>
                                      <td className="px-3 py-2 text-right font-medium text-orange-500">¥{t.output.toFixed(2)}</td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                            <p className="text-[10px] text-[var(--text-muted)] mt-1.5">* 实际价格以部署时模型定价为准，不调用不扣费</p>
                          </div>
                        )}

                        {/* TPU 模式：规格选择 */}
                        {b.id === 'tpu' && (
                          <div>
                            <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-2">选择吞吐规格</p>
                            <div className="grid grid-cols-2 gap-2">
                              {TPU_SPECS.map((spec, i) => (
                                <div key={i} onClick={() => setTpuSpecIdx(i)}
                                  className={`p-3 rounded-lg border cursor-pointer transition-all relative ${tpuSpecIdx === i ? 'border-[var(--accent)] bg-[var(--accent-light)]' : 'border-[var(--border-light)] hover:border-[var(--border)]'}`}>
                                  {spec.recommend && <span className="absolute -top-2 right-2 bg-[var(--accent)] text-white text-[9px] px-1.5 py-0.5 rounded-full">推荐</span>}
                                  <p className="text-xs font-semibold text-[var(--text)]">{spec.name}</p>
                                  <p className="text-[10px] text-[var(--text-muted)] mt-0.5">{spec.qps} QPS · {spec.concur} 并发</p>
                                  <p className="text-sm font-bold text-[var(--accent)] mt-1">¥{spec.price}<span className="text-[10px] font-normal text-[var(--text-muted)]">/时</span></p>
                                  <p className="text-[10px] text-[var(--text-muted)] mt-0.5">{spec.tag}</p>
                                </div>
                              ))}
                            </div>
                            <div className="mt-2 rounded-lg bg-purple-50 border border-purple-100 px-3 py-2 flex items-center justify-between">
                              <span className="text-xs text-purple-700">当前选择费率</span>
                              <span className="text-sm font-bold text-purple-700">¥{TPU_SPECS[tpuSpecIdx].price}/时 ≈ ¥{(TPU_SPECS[tpuSpecIdx].price * 24).toFixed(0)}/天</span>
                            </div>
                          </div>
                        )}

                        {/* GPU 单元模式：GPU 型号选择 */}
                        {b.id === 'unit' && (
                          <div>
                            <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-2">选择 GPU 型号</p>
                            <div className="grid grid-cols-2 gap-2">
                              {GPU_SPECS.map((spec, i) => (
                                <div key={i} onClick={() => setGpuSpecIdx(i)}
                                  className={`p-3 rounded-lg border cursor-pointer transition-all relative ${gpuSpecIdx === i ? 'border-[var(--accent)] bg-[var(--accent-light)]' : 'border-[var(--border-light)] hover:border-[var(--border)]'}`}>
                                  {spec.recommend && <span className="absolute -top-2 right-2 bg-[var(--accent)] text-white text-[9px] px-1.5 py-0.5 rounded-full">推荐</span>}
                                  <p className="text-xs font-semibold text-[var(--text)]">{spec.name}</p>
                                  <p className="text-[10px] text-[var(--text-muted)] mt-0.5">显存 {spec.mem}</p>
                                  <p className="text-sm font-bold text-[var(--accent)] mt-1">¥{spec.price}<span className="text-[10px] font-normal text-[var(--text-muted)]">/GPU·时</span></p>
                                  <p className="text-[10px] text-[var(--text-muted)] mt-0.5">{spec.tag}</p>
                                </div>
                              ))}
                            </div>
                            <div className="mt-2 rounded-lg bg-amber-50 border border-amber-100 px-3 py-2 flex items-center justify-between">
                              <span className="text-xs text-amber-700">单副本费率（副本数在下一步配置）</span>
                              <span className="text-sm font-bold text-amber-700">¥{GPU_SPECS[gpuSpecIdx].price}/GPU·时</span>
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {/* ──── Step 3: 配置确认 + 费用预估 ──── */}
          {step === 3 && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">部署名称 *</label>
                <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. my-deepseek-v3" />
              </div>

              {billing !== 'token' && (
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
              )}

              {/* 部署摘要 */}
              <div className="rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-light)] p-4 space-y-2.5">
                <p className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide">部署摘要</p>
                {[
                  ['模型', selectedModelInfo?.name ?? selectedModel],
                  ['计费方式', billingConfig.label],
                  ...(billing === 'tpu' ? [['TPU 规格', `${TPU_SPECS[tpuSpecIdx].name}（${TPU_SPECS[tpuSpecIdx].qps} QPS）`]] : []),
                  ...(billing === 'unit' ? [['GPU 型号', `${GPU_SPECS[gpuSpecIdx].name} ${GPU_SPECS[gpuSpecIdx].mem}`]] : []),
                  ...(billing !== 'token' ? [['副本范围', `${minR} ~ ${maxR}`]] : []),
                ].map(([k, v]) => (
                  <div key={k} className="flex items-center justify-between text-sm">
                    <span className="text-[var(--text-muted)]">{k}</span>
                    <span className="font-medium text-[var(--text)]">{v}</span>
                  </div>
                ))}
              </div>

              {/* 费用预估卡片 */}
              {billing === 'token' && (
                <div className="rounded-xl bg-blue-50 border border-blue-100 p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <svg width="14" height="14" className="text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                    <p className="text-xs font-semibold text-blue-700">费用说明：按量计费</p>
                  </div>
                  <p className="text-xs text-blue-600 leading-relaxed">仅在实际调用时产生费用，不调用不扣费。费用 = 消耗 Token 数 × 模型单价，不同模型价格不同，详见计费方式页面。</p>
                  <div className="mt-2 grid grid-cols-2 gap-2 text-center">
                    <div className="bg-white/70 rounded-lg py-2">
                      <p className="text-[10px] text-blue-500">最低起步（百万 token）</p>
                      <p className="text-sm font-bold text-blue-700">¥0.14</p>
                    </div>
                    <div className="bg-white/70 rounded-lg py-2">
                      <p className="text-[10px] text-blue-500">固定成本</p>
                      <p className="text-sm font-bold text-blue-700">¥0</p>
                    </div>
                  </div>
                </div>
              )}

              {billing === 'tpu' && (
                <div className="rounded-xl bg-purple-50 border border-purple-100 p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <svg width="14" height="14" className="text-purple-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
                    <p className="text-xs font-semibold text-purple-700">费用预估：置备吐单元</p>
                  </div>
                  <div className="grid grid-cols-3 gap-2 text-center">
                    <div className="bg-white/70 rounded-lg py-2">
                      <p className="text-[10px] text-purple-500">每小时</p>
                      <p className="text-sm font-bold text-purple-700">¥{TPU_SPECS[tpuSpecIdx].price}</p>
                    </div>
                    <div className="bg-white/70 rounded-lg py-2">
                      <p className="text-[10px] text-purple-500">预估日费</p>
                      <p className="text-sm font-bold text-purple-700">¥{(TPU_SPECS[tpuSpecIdx].price * 24).toFixed(1)}</p>
                    </div>
                    <div className="bg-white/70 rounded-lg py-2">
                      <p className="text-[10px] text-purple-500">预估月费</p>
                      <p className="text-sm font-bold text-purple-700">¥{(TPU_SPECS[tpuSpecIdx].price * 24 * 30).toFixed(0)}</p>
                    </div>
                  </div>
                  <p className="text-[10px] text-purple-500 mt-2">* 按实际运行时长计费，停止后不计费。超出 QPS 限额自动降速。</p>
                </div>
              )}

              {billing === 'unit' && (
                <div className="rounded-xl bg-amber-50 border border-amber-100 p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <svg width="14" height="14" className="text-amber-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/></svg>
                    <p className="text-xs font-semibold text-amber-700">费用预估：独占 GPU 实例</p>
                  </div>
                  <div className="space-y-1.5 mb-2 text-xs">
                    <div className="flex justify-between text-amber-700">
                      <span>GPU 单价</span>
                      <span className="font-semibold">¥{GPU_SPECS[gpuSpecIdx].price}/GPU·时</span>
                    </div>
                    <div className="flex justify-between text-amber-700">
                      <span>最大副本数</span>
                      <span className="font-semibold">{maxR} 个</span>
                    </div>
                    <div className="flex justify-between text-amber-700 border-t border-amber-200 pt-1.5">
                      <span className="font-semibold">最高每小时费用</span>
                      <span className="font-bold text-sm">¥{(GPU_SPECS[gpuSpecIdx].price * maxR).toFixed(2)}</span>
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-center">
                    <div className="bg-white/70 rounded-lg py-2">
                      <p className="text-[10px] text-amber-500">最少（min={minR} 副本）</p>
                      <p className="text-sm font-bold text-amber-700">{minR === 0 ? '¥0（缩容到 0）' : `¥${(GPU_SPECS[gpuSpecIdx].price * minR).toFixed(2)}/时`}</p>
                    </div>
                    <div className="bg-white/70 rounded-lg py-2">
                      <p className="text-[10px] text-amber-500">预估日费（均值 {Math.ceil((minR+maxR)/2)} 副本）</p>
                      <p className="text-sm font-bold text-amber-700">¥{(GPU_SPECS[gpuSpecIdx].price * ((minR + maxR) / 2) * 24).toFixed(0)}</p>
                    </div>
                  </div>
                  <p className="text-[10px] text-amber-500 mt-2">* 费用 = GPU 单价 × 当前运行副本数。min_replicas=0 时可缩容到 0，无请求时不计费。</p>
                </div>
              )}

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
            disabled={!canNext || submitting}
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

// ── 详情侧边栏 ─────────────────────────────────────────────────────────────
function DetailPanel({ dep, onClose, onUpdate, onDelete }: {
  dep: Deployment
  onClose: () => void
  onUpdate: (d: Deployment) => void
  onDelete: (id: string) => void
}) {
  const [scaleModal, setScaleModal] = useState(false)
  const [newMin, setNewMin] = useState(dep.min_replicas)
  const [newMax, setNewMax] = useState(dep.max_replicas)
  const [scaleSaving, setScaleSaving] = useState(false)
  const [copied, setCopied] = useState(false)

  const st = statusConfig[dep.status] ?? { label: dep.status, cls: 'tag-blue', dot: 'bg-blue-400' }
  const bm = billingModes.find(b => b.id === dep.billing_mode)

  function copyEndpoint() {
    navigator.clipboard.writeText(dep.endpoint || '')
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  async function saveScale() {
    setScaleSaving(true)
    try {
      const updated = await patchDeployment(dep.id, { min_replicas: newMin, max_replicas: newMax })
      onUpdate(updated)
      setScaleModal(false)
    } catch {} finally { setScaleSaving(false) }
  }

  async function toggleStatus() {
    const next = dep.status === 'running' ? 'stopped' : 'running'
    try {
      const updated = await patchDeployment(dep.id, { status: next })
      onUpdate(updated)
    } catch {}
  }

  return (
    <div className="fixed inset-0 z-40 flex">
      <div className="flex-1 bg-black/20" onClick={onClose} />
      <div className="w-[400px] bg-white border-l border-[var(--border-light)] flex flex-col overflow-y-auto" style={{ boxShadow: '-4px 0 24px rgba(0,0,0,0.08)' }}>

        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--border-light)]">
          <div className="flex items-center gap-2 min-w-0">
            <div className={`w-2 h-2 rounded-full ${st.dot} flex-shrink-0`} />
            <p className="font-semibold text-[var(--text)] truncate">{dep.name}</p>
          </div>
          <button onClick={onClose} className="w-7 h-7 flex items-center justify-center rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-muted)] transition-colors flex-shrink-0 ml-2">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <div className="flex-1 px-5 py-4 space-y-4">
          {/* 基础信息 */}
          <div className="space-y-2.5">
            <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide">基本信息</p>
            {[
              ['模型', dep.model_name],
              ['状态', null],
              ['创建时间', dep.created_at ? new Date(dep.created_at).toLocaleString('zh-CN') : '—'],
              ['更新时间', dep.updated_at ? new Date(dep.updated_at).toLocaleString('zh-CN') : '—'],
            ].map(([k, v]) => (
              <div key={String(k)} className="flex items-center justify-between text-sm">
                <span className="text-[var(--text-muted)] w-20 flex-shrink-0">{k}</span>
                {k === '状态' ? (
                  <span className={`tag ${st.cls}`}>{st.label}</span>
                ) : (
                  <span className="font-medium text-[var(--text)] text-right text-xs">{String(v)}</span>
                )}
              </div>
            ))}
          </div>

          {/* Endpoint */}
          {dep.endpoint && (
            <div>
              <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-2">API Endpoint</p>
              <div className="flex items-center gap-2 bg-[var(--bg-secondary)] rounded-lg px-3 py-2 border border-[var(--border-light)]">
                <code className="flex-1 text-xs text-[var(--text)] font-mono truncate">{dep.endpoint}</code>
                <button onClick={copyEndpoint} className="flex-shrink-0 text-[var(--text-muted)] hover:text-[var(--accent)] transition-colors">
                  {copied
                    ? <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="var(--success)" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>
                    : <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                  }
                </button>
              </div>
            </div>
          )}

          {/* 计费信息卡片 */}
          <div>
            <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-2">计费信息</p>
            {dep.billing_mode === 'token' && (
              <div className="rounded-xl bg-blue-50 border border-blue-100 p-4">
                <div className="flex items-center gap-2 mb-2">
                  <span className="tag tag-blue text-[10px]">按 Token 计费</span>
                  {dep.status === 'running' ? <span className="text-[10px] text-blue-500">计费中</span> : <span className="text-[10px] text-[var(--text-muted)]">暂无费用</span>}
                </div>
                <p className="text-xs text-blue-600">按实际消耗 Token 计费，不调用不扣费。费用随请求量弹性变化。</p>
                <div className="mt-2 grid grid-cols-2 gap-2 text-center text-[10px] text-blue-500">
                  <div className="bg-white/70 rounded py-1.5"><p>固定成本</p><p className="font-bold text-sm text-blue-700">¥0</p></div>
                  <div className="bg-white/70 rounded py-1.5"><p>最低价格</p><p className="font-bold text-sm text-blue-700">¥0.14/M</p></div>
                </div>
              </div>
            )}
            {dep.billing_mode === 'tpu' && (
              <div className="rounded-xl bg-purple-50 border border-purple-100 p-4">
                <div className="flex items-center gap-2 mb-2">
                  <span className="tag tag-purple text-[10px]">置备吐单元</span>
                  <span className="text-[10px] text-purple-500">{dep.status === 'running' ? '计费中' : '已停止'}</span>
                </div>
                {dep.status === 'running' ? (
                  <div className="grid grid-cols-2 gap-2 text-center text-[10px] text-purple-500">
                    <div className="bg-white/70 rounded py-1.5"><p>实例费率</p><p className="font-bold text-sm text-purple-700">¥{TPU_SPECS[1].price}/时</p></div>
                    <div className="bg-white/70 rounded py-1.5"><p>预估日费</p><p className="font-bold text-sm text-purple-700">¥{(TPU_SPECS[1].price * 24).toFixed(1)}</p></div>
                  </div>
                ) : (
                  <p className="text-xs text-[var(--text-muted)]">实例已停止，暂不产生费用。启动后按规格计费。</p>
                )}
              </div>
            )}
            {dep.billing_mode === 'unit' && (
              <div className="rounded-xl bg-amber-50 border border-amber-100 p-4">
                <div className="flex items-center gap-2 mb-2">
                  <span className="tag tag-amber text-[10px]">模型单元（GPU）</span>
                  <span className="text-[10px] text-amber-500">{dep.status === 'running' ? '计费中' : '已停止'}</span>
                </div>
                {dep.status === 'running' ? (
                  <>
                    <div className="grid grid-cols-2 gap-2 text-center text-[10px] text-amber-500 mb-2">
                      <div className="bg-white/70 rounded py-1.5"><p>GPU 单价</p><p className="font-bold text-sm text-amber-700">¥{GPU_SPECS[1].price}/GPU·时</p></div>
                      <div className="bg-white/70 rounded py-1.5"><p>当前副本</p><p className="font-bold text-sm text-amber-700">{dep.max_replicas} 个</p></div>
                    </div>
                    <div className="bg-white/70 rounded px-3 py-2 text-center">
                      <p className="text-[10px] text-amber-500">当前费率</p>
                      <p className="text-base font-bold text-amber-700">¥{(GPU_SPECS[1].price * dep.max_replicas).toFixed(2)}/时</p>
                      <p className="text-[10px] text-amber-500">≈ ¥{(GPU_SPECS[1].price * dep.max_replicas * 24).toFixed(0)}/天</p>
                    </div>
                  </>
                ) : (
                  <p className="text-xs text-[var(--text-muted)]">实例已停止，不产生费用。启动后按 GPU 数量计费。</p>
                )}
              </div>
            )}
          </div>

          {/* 副本配置 */}
          {dep.billing_mode !== 'token' && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <p className="text-[11px] font-semibold text-[var(--text-muted)] uppercase tracking-wide">副本配置</p>
                <button onClick={() => setScaleModal(true)} className="text-xs text-[var(--accent)] hover:underline">调整</button>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="bg-[var(--bg-secondary)] rounded-lg px-3 py-2.5 text-center border border-[var(--border-light)]">
                  <p className="text-[10px] text-[var(--text-muted)]">最小副本</p>
                  <p className="text-lg font-bold text-[var(--text)]">{dep.min_replicas}</p>
                </div>
                <div className="bg-[var(--bg-secondary)] rounded-lg px-3 py-2.5 text-center border border-[var(--border-light)]">
                  <p className="text-[10px] text-[var(--text-muted)]">最大副本</p>
                  <p className="text-lg font-bold text-[var(--text)]">{dep.max_replicas}</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="px-5 py-4 border-t border-[var(--border-light)] space-y-2">
          <button onClick={toggleStatus}
            className={`w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium transition-colors ${dep.status === 'running' ? 'bg-[var(--bg-secondary)] text-[var(--text)] hover:bg-[var(--bg-hover)] border border-[var(--border-light)]' : 'btn-primary'}`}>
            {dep.status === 'running' ? (
              <><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>停止实例</>
            ) : (
              <><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><polygon points="5 3 19 12 5 21 5 3"/></svg>启动实例</>
            )}
          </button>
          <button onClick={() => onDelete(dep.id)}
            className="w-full flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium text-[var(--danger)] hover:bg-red-50 border border-[var(--border-light)] transition-colors">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/></svg>
            删除部署
          </button>
        </div>
      </div>

      {/* 副本调整弹窗 */}
      {scaleModal && (
        <div className="absolute inset-0 flex items-center justify-center z-50 bg-black/30">
          <div className="bg-white rounded-2xl shadow-2xl w-[360px] p-6">
            <h3 className="text-base font-semibold text-[var(--text)] mb-4">调整副本数</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最小副本数</label>
                <input type="number" min={0} max={newMax} value={newMin} onChange={e => setNewMin(Number(e.target.value))} />
              </div>
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最大副本数</label>
                <input type="number" min={newMin} max={20} value={newMax} onChange={e => setNewMax(Number(e.target.value))} />
              </div>
              {dep.billing_mode === 'unit' && (
                <div className="rounded-lg bg-amber-50 border border-amber-100 p-3 text-xs text-amber-700 space-y-1">
                  <div className="flex justify-between"><span>调整后费率</span><span className="font-bold">¥{(GPU_SPECS[1].price * newMax).toFixed(2)}/时</span></div>
                  <div className="flex justify-between"><span>预估日费</span><span className="font-bold">¥{(GPU_SPECS[1].price * newMax * 24).toFixed(0)}</span></div>
                </div>
              )}
            </div>
            <div className="flex gap-2 mt-5">
              <button onClick={() => setScaleModal(false)} className="btn-outline flex-1">取消</button>
              <button onClick={saveScale} disabled={scaleSaving} className="btn-primary flex-1 flex items-center justify-center gap-1">
                {scaleSaving ? <svg className="animate-spin" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg> : null}
                保存
              </button>
            </div>
          </div>
        </div>
      )}
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
  const [selectedDep, setSelectedDep] = useState<Deployment | null>(null)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'running' | 'stopped' | 'provisioning' | 'failed'>('all')

  useEffect(() => {
    Promise.all([listDeployments(), fetchModels()]).then(([dl, ml]) => {
      setDeployments(dl.data ?? [])
      setModels(ml.models ?? [])
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  async function handleDelete(id: string) {
    if (!confirm('确认删除该部署？此操作不可恢复。')) return
    setDeletingId(id)
    try {
      await deleteDeployment(id)
      setDeployments(prev => prev.filter(d => d.id !== id))
      if (selectedDep?.id === id) setSelectedDep(null)
    } finally { setDeletingId(null) }
  }

  function handleUpdate(updated: Deployment) {
    setDeployments(prev => prev.map(d => d.id === updated.id ? updated : d))
    if (selectedDep?.id === updated.id) setSelectedDep(updated)
  }

  const runningCount = deployments.filter(d => d.status === 'running').length

  const filtered = deployments.filter(d => {
    const matchStatus = statusFilter === 'all' || d.status === statusFilter
    const q = search.toLowerCase()
    const matchSearch = !q || d.name.toLowerCase().includes(q) || d.model_name.toLowerCase().includes(q)
    return matchStatus && matchSearch
  })

  const isEmpty = !loading && deployments.length === 0

  return (
    <div className="min-h-screen flex flex-col">
      {/* Top bar */}
      <div className="flex items-center justify-between px-8 py-5">
        <div>
          <h1 className="text-[15px] font-semibold text-[var(--text)]">模型部署</h1>
          {!loading && deployments.length > 0 && (
            <p className="text-xs text-[var(--text-muted)] mt-0.5">
              共 <span className="font-semibold text-[var(--text)]">{deployments.length}</span> 个部署
              {runningCount > 0 && <span>，<span className="text-green-600 font-semibold">{runningCount}</span> 个运行中</span>}
            </p>
          )}
        </div>
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
            <button onClick={() => setShowModal(true)} className="btn-primary flex items-center gap-1.5 text-sm">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              部署第一个模型
            </button>
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

          {/* 计费方式说明卡片（空状态时展示） */}
          <div className="max-w-[900px] w-full px-8">
            <p className="text-xs font-semibold text-[var(--text-muted)] uppercase tracking-wide mb-3">计费方式说明</p>
            <div className="grid grid-cols-3 gap-4">
              {billingModes.map(b => (
                <div key={b.id} className="bg-white rounded-2xl border border-[var(--border-light)] p-4" style={{ boxShadow: 'var(--shadow-sm)' }}>
                  <div className="flex items-center gap-2 mb-2">
                    <div className="w-8 h-8 rounded-lg bg-[var(--bg-secondary)] flex items-center justify-center text-[var(--text-muted)]">{b.icon}</div>
                    <div>
                      <p className="text-xs font-semibold text-[var(--text)]">{b.label}</p>
                      <span className={`tag text-[9px] ${b.badge}`}>{b.tagline}</span>
                    </div>
                  </div>
                  <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mb-2">{b.desc}</p>
                  <div className="flex items-center justify-between text-[11px]">
                    <span className="text-[var(--text-muted)]">起步价</span>
                    <span className="font-bold text-[var(--text)]">{b.startFrom}<span className="font-normal text-[var(--text-muted)] ml-0.5">/{b.priceUnit.split('/')[1]}</span></span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Deployment list */}
      {!loading && deployments.length > 0 && (
        <div className="px-8 pb-8 space-y-3">
          {/* 搜索 + 筛选 */}
          <div className="flex items-center gap-3">
            <div className="relative w-64">
              <svg className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <input value={search} onChange={e => setSearch(e.target.value)} placeholder="搜索部署名称或模型…"
                className="pl-8 pr-3 py-2 text-sm w-full rounded-lg border border-[var(--border-light)] bg-white focus:outline-none focus:border-[var(--accent)] transition-colors" style={{ height: '36px' }} />
            </div>
            <div className="flex items-center gap-1 bg-white rounded-lg border border-[var(--border-light)] p-0.5">
              {([['all', '全部'], ['running', '运行中'], ['stopped', '已停止'], ['provisioning', '启动中']] as const).map(([v, label]) => (
                <button key={v} onClick={() => setStatusFilter(v)}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${statusFilter === v ? 'bg-[var(--accent)] text-white shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                >{label}</button>
              ))}
            </div>
            {filtered.length !== deployments.length && (
              <span className="text-xs text-[var(--text-muted)]">显示 {filtered.length} / {deployments.length}</span>
            )}
          </div>

          <div className="bg-white rounded-2xl border border-[var(--border-light)] overflow-hidden" style={{ boxShadow: 'var(--shadow-sm)' }}>
            <table>
              <thead>
                <tr className="border-b border-[var(--border-light)]">
                  <th className="px-5 py-3.5 text-left">名称 / 模型</th>
                  <th className="px-5 py-3.5 text-left">计费方式</th>
                  <th className="px-5 py-3.5 text-left">副本</th>
                  <th className="px-5 py-3.5 text-left">预估费率</th>
                  <th className="px-5 py-3.5 text-left">状态</th>
                  <th className="px-5 py-3.5 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 && (
                  <tr><td colSpan={6} className="py-12 text-center text-sm text-[var(--text-muted)]">未找到匹配的部署</td></tr>
                )}
                {filtered.map(d => {
                  const st = statusConfig[d.status] ?? { label: d.status, cls: 'tag-blue', dot: 'bg-blue-400' }
                  const isRunning = d.status === 'running'
                  // 费率显示
                  let rateCell = <span className="text-xs text-[var(--text-muted)]">—</span>
                  if (d.billing_mode === 'token') {
                    rateCell = <span className="text-xs text-blue-500">按量计费</span>
                  } else if (d.billing_mode === 'tpu' && isRunning) {
                    rateCell = <span className="text-xs font-medium text-purple-600">¥{TPU_SPECS[1].price}/时</span>
                  } else if (d.billing_mode === 'unit' && isRunning) {
                    const r = GPU_SPECS[1].price * d.max_replicas
                    rateCell = <div><p className="text-xs font-medium text-amber-600">¥{r.toFixed(2)}/时</p><p className="text-[10px] text-[var(--text-muted)]">≈¥{(r * 24).toFixed(0)}/天</p></div>
                  }

                  return (
                    <tr key={d.id} onClick={() => setSelectedDep(d)}
                      className="hover:bg-[var(--bg-secondary)] transition-colors cursor-pointer">
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-2">
                          <div className={`w-1.5 h-1.5 rounded-full ${st.dot} flex-shrink-0`} />
                          <div>
                            <p className="text-sm font-medium text-[var(--text)]">{d.name}</p>
                            <p className="text-xs text-[var(--text-muted)] mt-0.5">{d.model_name}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <span className={`tag text-[10px] ${billingBadge[d.billing_mode] ?? 'tag-blue'}`}>{billingLabel[d.billing_mode] ?? d.billing_mode}</span>
                      </td>
                      <td className="px-5 py-4">
                        {d.billing_mode === 'token' ? (
                          <span className="text-xs text-[var(--text-muted)]">弹性</span>
                        ) : (
                          <span className="text-sm text-[var(--text-secondary)]">{d.min_replicas} – {d.max_replicas}</span>
                        )}
                      </td>
                      <td className="px-5 py-4">{rateCell}</td>
                      <td className="px-5 py-4">
                        <span className={`tag ${st.cls}`}>{st.label}</span>
                      </td>
                      <td className="px-5 py-4" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-3">
                          <button onClick={() => setSelectedDep(d)} className="text-xs text-[var(--accent)] hover:underline">详情</button>
                          <button onClick={() => handleDelete(d.id)} disabled={deletingId === d.id}
                            className="text-xs text-[var(--danger)] hover:underline disabled:opacity-50">删除</button>
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
        <DeployModal models={models} onClose={() => setShowModal(false)} onCreated={d => setDeployments(prev => [d, ...prev])} />
      )}

      {selectedDep && (
        <DetailPanel
          dep={selectedDep}
          onClose={() => setSelectedDep(null)}
          onUpdate={handleUpdate}
          onDelete={handleDelete}
        />
      )}
    </div>
  )
}
