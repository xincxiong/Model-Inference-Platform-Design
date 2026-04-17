'use client'

import { useEffect, useState, useMemo } from 'react'
import {
  fetchDedicatedTemplates,
  listDedicatedEndpoints,
  createDedicatedEndpoint,
  deleteDedicatedEndpoint,
  patchDedicatedEndpoint,
} from '@/lib/api'

// ── Types ──────────────────────────────────────────────────────────────────
interface Template {
  model_name: string
  name: string
  model_type: string
  provider: string
}

interface DedicatedEndpoint {
  id: string
  name: string
  description: string
  model_name: string
  flavor_name: string
  gpu_type: string
  gpu_count: number
  region: string
  min_replicas: number
  max_replicas: number
  current_replicas: number
  routing_key: string
  routing_prefix: string
  status: string
  created_at: string
  updated_at: string
}

// ── Constants ──────────────────────────────────────────────────────────────
const GPU_OPTIONS = [
  { value: 'A100-80GB',    label: 'NVIDIA A100 80GB',  badge: 'tag-purple', desc: '旗舰级训练/推理，适合 70B+ 大模型',   pricePerHour: 3.50 },
  { value: 'A100-40GB',    label: 'NVIDIA A100 40GB',  badge: 'tag-blue',   desc: '通用高性能，适合 7B–40B 模型',       pricePerHour: 2.20 },
  { value: 'H100-80GB',    label: 'NVIDIA H100 80GB',  badge: 'tag-amber',  desc: 'Hopper 架构，速度比 A100 快 3x',    pricePerHour: 5.80 },
  { value: 'L40S-48GB',    label: 'NVIDIA L40S 48GB',  badge: 'tag-green',  desc: '推理优化型，性价比极高',              pricePerHour: 1.60 },
  { value: 'RTX4090-24GB', label: 'NVIDIA RTX 4090',   badge: 'tag-pink',   desc: '消费级旗舰，适合轻量模型',           pricePerHour: 0.90 },
]

const REGION_OPTIONS = [
  { value: 'cn-east-1',  label: '华东一区（上海）',  flag: '🇨🇳' },
  { value: 'cn-south-1', label: '华南一区（广州）',  flag: '🇨🇳' },
  { value: 'cn-north-1', label: '华北一区（北京）',  flag: '🇨🇳' },
  { value: 'ap-east-1',  label: '亚太东南（香港）', flag: '🇭🇰' },
  { value: 'us-west-1',  label: '美西（圣何塞）',   flag: '🇺🇸' },
]

const statusConfig: Record<string, { label: string; cls: string }> = {
  running:      { label: '运行中', cls: 'tag-green'  },
  stopped:      { label: '已停止', cls: 'tag-blue'   },
  provisioning: { label: '启动中', cls: 'tag-amber'  },
  error:        { label: '异常',   cls: 'tag-pink'   },
  deleting:     { label: '删除中', cls: 'tag-orange' },
}

const typeLabel: Record<string, string> = {
  'text-to-text': '对话补全',
  'vision':       '多模态',
}

// 计算端点每小时费率
function calcHourlyRate(gpuType: string, gpuCount: number, replicas: number): number {
  const gpu = GPU_OPTIONS.find(g => g.value === gpuType)
  if (!gpu) return 0
  return gpu.pricePerHour * gpuCount * replicas
}

// ── Empty Illustration ────────────────────────────────────────────────────
function EmptyIllustration() {
  return (
    <svg width="96" height="72" viewBox="0 0 96 72" fill="none" xmlns="http://www.w3.org/2000/svg">
      <ellipse cx="48" cy="62" rx="32" ry="6" fill="#e8eaed" />
      <rect x="22" y="14" width="52" height="38" rx="7" fill="#f1f3f4" stroke="#dadce0" strokeWidth="1.5" />
      <rect x="30" y="22" width="16" height="3" rx="1.5" fill="#dadce0" />
      <rect x="30" y="29" width="36" height="2.5" rx="1.25" fill="#e8eaed" />
      <rect x="30" y="35" width="28" height="2.5" rx="1.25" fill="#e8eaed" />
      <circle cx="68" cy="20" r="9" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5" />
      <path d="M64 20 h8 M68 16 v8" stroke="#1a73e8" strokeWidth="1.8" strokeLinecap="round" />
      <path d="M44 50 L52 50" stroke="#1a73e8" strokeWidth="1.5" strokeLinecap="round" strokeDasharray="3 2" />
    </svg>
  )
}

// ── Create Modal (3-step) ─────────────────────────────────────────────────
function CreateModal({ templates, onClose, onCreated }: {
  templates: Template[]
  onClose: () => void
  onCreated: (ep: DedicatedEndpoint) => void
}) {
  const [step, setStep] = useState<1 | 2 | 3>(1)
  const [selectedModel, setSelectedModel] = useState('')
  const [gpuType, setGpuType] = useState('A100-80GB')
  const [gpuCount, setGpuCount] = useState(1)
  const [region, setRegion] = useState('cn-east-1')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [minRep, setMinRep] = useState(0)
  const [maxRep, setMaxRep] = useState(2)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<'all' | 'text-to-text' | 'vision'>('all')

  const filteredTemplates = templates.filter(t => {
    const matchType = typeFilter === 'all' || t.model_type === typeFilter
    const q = search.toLowerCase()
    const matchSearch = !q || t.name.toLowerCase().includes(q) || t.model_name.toLowerCase().includes(q) || t.provider.toLowerCase().includes(q)
    return matchType && matchSearch
  })

  const selectedTemplate = templates.find(t => t.model_name === selectedModel)
  const selectedGPU = GPU_OPTIONS.find(g => g.value === gpuType)
  const selectedRegion = REGION_OPTIONS.find(r => r.value === region)

  // 费用预估：按最小副本和最大副本
  const minCostPerHour = calcHourlyRate(gpuType, gpuCount, Math.max(minRep, 1))
  const maxCostPerHour = calcHourlyRate(gpuType, gpuCount, maxRep)

  async function handleCreate() {
    if (!name.trim()) { setError('请填写端点名称'); return }
    setSubmitting(true); setError('')
    try {
      const ep = await createDedicatedEndpoint({
        name: name.trim(),
        description: description.trim(),
        model_name: selectedModel,
        gpu_type: gpuType,
        gpu_count: gpuCount,
        region,
        min_replicas: minRep,
        max_replicas: maxRep,
      })
      onCreated(ep)
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-white rounded-2xl shadow-2xl w-[600px] max-h-[90vh] flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-light)]">
          <div>
            <h2 className="text-base font-semibold text-[var(--text)]">新建专属端点</h2>
            <p className="text-xs text-[var(--text-muted)] mt-0.5">独占 GPU 实例，低延迟专属推理服务 · 按 GPU·小时计费</p>
          </div>
          <button onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-muted)] transition-colors">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        {/* Steps */}
        <div className="flex items-center gap-2 px-6 py-3 bg-[var(--bg-secondary)]">
          {(['选择模型', 'GPU 配置', '端点配置'] as const).map((label, i) => {
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
          {/* Step 1: 选择模型 */}
          {step === 1 && (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <svg className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
                  <input
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                    placeholder="搜索模型名称、厂商…"
                    className="pl-8 pr-3 py-2 text-sm w-full rounded-lg border border-[var(--border-light)] bg-[var(--bg-secondary)] focus:outline-none focus:border-[var(--accent)] transition-colors"
                  />
                </div>
                <div className="flex items-center gap-1 bg-[var(--bg-secondary)] rounded-lg border border-[var(--border-light)] p-0.5">
                  {([['all', '全部'], ['text-to-text', '对话'], ['vision', '多模态']] as const).map(([v, label]) => (
                    <button
                      key={v}
                      onClick={() => setTypeFilter(v)}
                      className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${typeFilter === v ? 'bg-white text-[var(--text)] shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >{label}</button>
                  ))}
                </div>
              </div>
              <p className="text-xs text-[var(--text-muted)]">
                共 <span className="font-semibold text-[var(--text)]">{templates.length}</span> 个可用模型，当前筛选 <span className="font-semibold text-[var(--text)]">{filteredTemplates.length}</span> 个
              </p>
              <div className="space-y-1.5 max-h-[320px] overflow-y-auto pr-0.5">
                {filteredTemplates.length === 0 && (
                  <div className="py-10 text-center text-sm text-[var(--text-muted)]">未找到匹配的模型</div>
                )}
                {filteredTemplates.map(t => (
                  <div key={t.model_name} onClick={() => setSelectedModel(t.model_name)}
                    className={`flex items-center justify-between p-3 rounded-xl border cursor-pointer transition-all ${selectedModel === t.model_name ? 'border-[var(--accent)] bg-[var(--accent-light)]' : 'border-[var(--border-light)] hover:border-[var(--border)] hover:bg-[var(--bg-secondary)]'}`}>
                    <div className="flex items-center gap-3 min-w-0">
                      <div className={`w-8 h-8 rounded-lg flex-shrink-0 flex items-center justify-center ${selectedModel === t.model_name ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-secondary)] text-[var(--text-muted)]'}`}>
                        {t.model_type === 'vision'
                          ? <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                          : <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                        }
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-[var(--text)] truncate">{t.name}</p>
                        <p className="text-xs text-[var(--text-muted)] truncate">{t.model_name}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0 ml-2">
                      <span className="text-[10px] text-[var(--text-muted)] bg-[var(--bg-secondary)] px-1.5 py-0.5 rounded">{t.provider}</span>
                      <span className={`tag text-[10px] ${t.model_type === 'vision' ? 'tag-purple' : 'tag-blue'}`}>{typeLabel[t.model_type] ?? t.model_type}</span>
                      {selectedModel === t.model_name && <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Step 2: GPU 配置 */}
          {step === 2 && (
            <div className="space-y-4">
              <div>
                <p className="text-sm font-medium text-[var(--text)] mb-3">选择 GPU 类型</p>
                <div className="space-y-2">
                  {GPU_OPTIONS.map(g => (
                    <div key={g.value} onClick={() => setGpuType(g.value)}
                      className={`p-3.5 rounded-xl border cursor-pointer transition-all ${gpuType === g.value ? 'border-[var(--accent)] bg-[var(--accent-light)]' : 'border-[var(--border-light)] hover:border-[var(--border)] hover:bg-[var(--bg-secondary)]'}`}>
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2.5">
                          <span className={`tag text-[10px] ${g.badge}`}>{g.value}</span>
                          <span className="text-sm font-medium text-[var(--text)]">{g.label}</span>
                        </div>
                        <div className="flex items-center gap-3">
                          <span className="text-xs font-semibold text-[var(--warning)]">¥{g.pricePerHour.toFixed(2)}<span className="font-normal text-[var(--text-muted)]">/GPU·时</span></span>
                          {gpuType === g.value && <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>}
                        </div>
                      </div>
                      <p className="text-xs text-[var(--text-muted)] mt-1">{g.desc}</p>
                    </div>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">GPU 数量</label>
                  <input type="number" min={1} max={8} value={gpuCount} onChange={e => setGpuCount(Math.max(1, Math.min(8, Number(e.target.value))))} />
                  <p className="text-xs text-[var(--text-muted)] mt-1">1 ~ 8 张</p>
                </div>
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">部署区域</label>
                  <select value={region} onChange={e => setRegion(e.target.value)} className="w-full rounded-lg border border-[var(--border-light)] bg-white px-3 py-2 text-sm focus:outline-none focus:border-[var(--accent)]">
                    {REGION_OPTIONS.map(r => (
                      <option key={r.value} value={r.value}>{r.flag} {r.label}</option>
                    ))}
                  </select>
                </div>
              </div>

              {/* 费用小提示 */}
              <div className="flex items-start gap-2.5 rounded-xl bg-amber-50 border border-amber-100 px-4 py-3">
                <svg className="flex-shrink-0 mt-0.5" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#d97706" strokeWidth="2" strokeLinecap="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                <p className="text-xs text-amber-800 leading-relaxed">
                  <span className="font-semibold">按 GPU·小时计费</span>：端点运行期间持续扣费，停止后不计费。
                  当前选择：{selectedGPU?.label} × {gpuCount} 张，单副本费率 <span className="font-semibold">¥{(selectedGPU?.pricePerHour ?? 0) * gpuCount}/时</span>。
                </p>
              </div>
            </div>
          )}

          {/* Step 3: 端点配置 */}
          {step === 3 && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">端点名称 *</label>
                <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. prod-deepseek-v3" />
              </div>
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">描述（可选）</label>
                <input value={description} onChange={e => setDescription(e.target.value)} placeholder="用于生产环境的专属推理端点…" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最小副本数</label>
                  <input type="number" min={0} max={maxRep} value={minRep} onChange={e => setMinRep(Math.max(0, Math.min(maxRep, Number(e.target.value))))} />
                  <p className="text-xs text-[var(--text-muted)] mt-1">0 = 自动缩容到 0（冷启动延迟约 30s）</p>
                </div>
                <div>
                  <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最大副本数</label>
                  <input type="number" min={Math.max(minRep, 1)} max={16} value={maxRep} onChange={e => setMaxRep(Math.max(Math.max(minRep, 1), Number(e.target.value)))} />
                  <p className="text-xs text-[var(--text-muted)] mt-1">高并发时自动扩容上限</p>
                </div>
              </div>

              {/* 部署摘要 */}
              <div className="rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-light)] p-4 space-y-2">
                <p className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-1">部署摘要</p>
                {[
                  ['基座模型', selectedTemplate?.name ?? selectedModel],
                  ['GPU 配置', `${selectedGPU?.label ?? gpuType} × ${gpuCount}`],
                  ['区域', selectedRegion ? `${selectedRegion.flag} ${selectedRegion.label}` : region],
                  ['副本范围', `${minRep} ~ ${maxRep}`],
                ].map(([k, v]) => (
                  <div key={k} className="flex items-center justify-between text-sm">
                    <span className="text-[var(--text-muted)]">{k}</span>
                    <span className="font-medium text-[var(--text)]">{v}</span>
                  </div>
                ))}
                <div className="border-t border-[var(--border-light)] pt-2 mt-2 space-y-1">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-[var(--text-muted)]">最低费用</span>
                    <span className="font-semibold text-[var(--warning)]">
                      {minRep === 0 ? '¥0.00/时（缩容到 0 时）' : `¥${minCostPerHour.toFixed(2)}/时`}
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-[var(--text-muted)]">最高费用</span>
                    <span className="font-semibold text-[var(--danger)]">¥{maxCostPerHour.toFixed(2)}/时</span>
                  </div>
                  <p className="text-[10px] text-[var(--text-muted)] pt-1">按实际运行副本数 × GPU 数量 × 单价实时结算，停止端点即停止计费</p>
                </div>
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
            onClick={() => step < 3 ? setStep((step + 1) as 2|3) : handleCreate()}
            disabled={(step === 1 && !selectedModel) || submitting}
            className="btn-primary flex items-center gap-2"
          >
            {submitting ? (
              <><svg className="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>创建中...</>
            ) : step < 3 ? '下一步' : '确认创建'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── Scale Modal ───────────────────────────────────────────────────────────
function ScaleModal({ ep, onClose, onUpdated }: {
  ep: DedicatedEndpoint
  onClose: () => void
  onUpdated: (ep: DedicatedEndpoint) => void
}) {
  const [minRep, setMinRep] = useState(ep.min_replicas)
  const [maxRep, setMaxRep] = useState(ep.max_replicas)
  const [currentRep, setCurrentRep] = useState(ep.current_replicas)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const gpu = GPU_OPTIONS.find(g => g.value === ep.gpu_type)
  const hourlyMin = calcHourlyRate(ep.gpu_type, ep.gpu_count, Math.max(minRep, 1))
  const hourlyMax = calcHourlyRate(ep.gpu_type, ep.gpu_count, maxRep)
  const hourlyNow = calcHourlyRate(ep.gpu_type, ep.gpu_count, currentRep)

  async function handleSave() {
    if (maxRep < minRep) { setError('最大副本数不能小于最小副本数'); return }
    if (currentRep < minRep || currentRep > maxRep) { setError('当前副本数需在最小和最大之间'); return }
    setSaving(true); setError('')
    try {
      const updated = await patchDedicatedEndpoint(ep.id, {
        min_replicas: minRep,
        max_replicas: maxRep,
        current_replicas: currentRep,
      })
      onUpdated(updated)
      onClose()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-white rounded-2xl shadow-2xl w-[440px] overflow-hidden">
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-light)]">
          <div>
            <h2 className="text-base font-semibold text-[var(--text)]">调整副本数</h2>
            <p className="text-xs text-[var(--text-muted)] mt-0.5">{ep.name}</p>
          </div>
          <button onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-muted)]">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <div className="px-6 py-5 space-y-4">
          {/* 副本滑块/输入 */}
          <div className="space-y-3">
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-medium text-[var(--text-secondary)]">当前副本数（立即生效）</label>
                <span className="text-sm font-bold text-[var(--accent)]">{currentRep}</span>
              </div>
              <input
                type="range" min={minRep} max={maxRep} value={currentRep}
                onChange={e => setCurrentRep(Number(e.target.value))}
                className="w-full accent-[var(--accent)]"
              />
              <div className="flex justify-between text-[10px] text-[var(--text-muted)] mt-0.5">
                <span>{minRep}</span><span>{maxRep}</span>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最小副本数</label>
                <input type="number" min={0} max={maxRep} value={minRep}
                  onChange={e => { const v = Number(e.target.value); setMinRep(v); if (currentRep < v) setCurrentRep(v) }} />
              </div>
              <div>
                <label className="block text-xs font-medium text-[var(--text-secondary)] mb-1.5">最大副本数</label>
                <input type="number" min={Math.max(minRep, 1)} max={16} value={maxRep}
                  onChange={e => { const v = Number(e.target.value); setMaxRep(v); if (currentRep > v) setCurrentRep(v) }} />
              </div>
            </div>
          </div>

          {/* 费用影响预览 */}
          <div className="rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-light)] p-3.5">
            <p className="text-xs font-semibold text-[var(--text-secondary)] mb-2.5">费用影响（{gpu?.label ?? ep.gpu_type} × {ep.gpu_count}）</p>
            <div className="space-y-1.5">
              <div className="flex items-center justify-between text-xs">
                <span className="text-[var(--text-muted)]">当前副本费率</span>
                <span className="font-semibold text-[var(--warning)]">¥{hourlyNow.toFixed(2)}/时</span>
              </div>
              <div className="flex items-center justify-between text-xs">
                <span className="text-[var(--text-muted)]">弹性范围费率</span>
                <span className="font-medium text-[var(--text-secondary)]">
                  {minRep === 0 ? '¥0' : `¥${hourlyMin.toFixed(2)}`} ~ ¥{hourlyMax.toFixed(2)}/时
                </span>
              </div>
              <div className="flex items-center justify-between text-xs">
                <span className="text-[var(--text-muted)]">预估日费用（当前副本）</span>
                <span className="font-medium text-[var(--text)]">¥{(hourlyNow * 24).toFixed(2)}</span>
              </div>
            </div>
          </div>

          {error && <p className="text-xs text-[var(--danger)]">{error}</p>}
        </div>

        <div className="flex items-center justify-between px-6 py-4 border-t border-[var(--border-light)]">
          <button onClick={onClose} className="btn-outline">取消</button>
          <button onClick={handleSave} disabled={saving} className="btn-primary flex items-center gap-2">
            {saving && <svg className="animate-spin" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>}
            保存更改
          </button>
        </div>
      </div>
    </div>
  )
}

// ── Detail Panel (Drawer) ─────────────────────────────────────────────────
function DetailPanel({ ep, onClose, onStatusChange, onScaleClick }: {
  ep: DedicatedEndpoint
  onClose: () => void
  onStatusChange: (id: string, status: string) => void
  onScaleClick: () => void
}) {
  const [copied, setCopied] = useState<string | null>(null)
  const [togglingStatus, setTogglingStatus] = useState(false)
  const [codeTab, setCodeTab] = useState<'curl' | 'python'>('curl')

  const copy = (text: string, key: string) => {
    navigator.clipboard.writeText(text)
    setCopied(key)
    setTimeout(() => setCopied(null), 1500)
  }

  const curlExample = `curl https://api.inference.example.com/v1/chat/completions \\
  -H "Authorization: Bearer sk-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${ep.routing_key}",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`

  const pythonExample = `from openai import OpenAI

client = OpenAI(
    api_key="sk-...",
    base_url="https://api.inference.example.com/v1"
)

response = client.chat.completions.create(
    model="${ep.routing_key}",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)`

  const handleToggle = async () => {
    setTogglingStatus(true)
    const next = ep.status === 'running' ? 'stopped' : 'running'
    try {
      await patchDedicatedEndpoint(ep.id, { status: next })
      onStatusChange(ep.id, next)
    } catch {}
    finally { setTogglingStatus(false) }
  }

  const st = statusConfig[ep.status] ?? { label: ep.status, cls: 'tag-blue' }
  const gpu = GPU_OPTIONS.find(g => g.value === ep.gpu_type)
  const hourlyRate = calcHourlyRate(ep.gpu_type, ep.gpu_count, ep.current_replicas)
  const dailyEstimate = hourlyRate * 24
  const createdDate = ep.created_at ? new Date(ep.created_at).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : '—'

  return (
    <div className="fixed inset-0 z-40 flex justify-end">
      <div className="absolute inset-0 bg-black/20" onClick={onClose} />
      <div className="relative w-[480px] h-full bg-white border-l border-[var(--border-light)] flex flex-col shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-start justify-between px-6 py-5 border-b border-[var(--border-light)]">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <h3 className="text-base font-semibold text-[var(--text)]">{ep.name}</h3>
              <span className={`tag ${st.cls}`}>{st.label}</span>
            </div>
            {ep.description && <p className="text-xs text-[var(--text-muted)]">{ep.description}</p>}
            <p className="text-[10px] text-[var(--text-muted)] mt-0.5">创建于 {createdDate}</p>
          </div>
          <button onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-[var(--bg-hover)] text-[var(--text-muted)] ml-2 flex-shrink-0">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto">
          {/* Billing Card */}
          <div className="px-6 py-4 border-b border-[var(--border-light)]">
            <p className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-3">计费信息</p>
            <div className="rounded-xl bg-gradient-to-br from-amber-50 to-orange-50 border border-amber-100 p-4">
              <div className="flex items-center gap-2 mb-3">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#d97706" strokeWidth="2" strokeLinecap="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
                <span className="text-xs font-semibold text-amber-800">按 GPU·小时实时计费</span>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div className="text-center">
                  <p className="text-[10px] text-amber-700 mb-1">GPU 单价</p>
                  <p className="text-sm font-bold text-amber-900">¥{(gpu?.pricePerHour ?? 0).toFixed(2)}</p>
                  <p className="text-[9px] text-amber-600">/GPU·时</p>
                </div>
                <div className="text-center border-x border-amber-100">
                  <p className="text-[10px] text-amber-700 mb-1">当前费率</p>
                  <p className="text-sm font-bold text-orange-700">¥{hourlyRate.toFixed(2)}</p>
                  <p className="text-[9px] text-amber-600">/时（{ep.current_replicas} 副本）</p>
                </div>
                <div className="text-center">
                  <p className="text-[10px] text-amber-700 mb-1">预估日费</p>
                  <p className="text-sm font-bold text-orange-700">¥{dailyEstimate.toFixed(2)}</p>
                  <p className="text-[9px] text-amber-600">/天</p>
                </div>
              </div>
              <div className="mt-3 pt-3 border-t border-amber-100">
                <div className="flex items-center justify-between text-[10px] text-amber-700">
                  <span>计费公式：GPU 单价 × GPU 数量（{ep.gpu_count}）× 运行副本数（{ep.current_replicas}）</span>
                  {ep.status === 'stopped' && <span className="font-semibold text-green-700">⏸ 已停止，不计费</span>}
                </div>
              </div>
            </div>
          </div>

          {/* Routing Key */}
          <div className="px-6 py-4 border-b border-[var(--border-light)]">
            <p className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-2">Routing Key</p>
            <div className="flex items-center gap-2 bg-[var(--bg-secondary)] rounded-xl border border-[var(--border-light)] px-3 py-2.5">
              <code className="flex-1 text-xs font-mono text-[var(--text)] break-all">{ep.routing_key}</code>
              <button onClick={() => copy(ep.routing_key, 'routing_key')} className="flex-shrink-0 flex items-center gap-1 text-xs text-[var(--accent)] hover:text-[var(--accent-hover)] transition-colors">
                {copied === 'routing_key'
                  ? <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>
                  : <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                }
                {copied === 'routing_key' ? '已复制' : '复制'}
              </button>
            </div>
            <p className="text-xs text-[var(--text-muted)] mt-1.5">在 API 请求的 <code className="bg-[var(--bg-hover)] px-1 rounded text-[10px]">model</code> 字段使用此 Key 路由到专属实例</p>
          </div>

          {/* Info Grid */}
          <div className="px-6 py-4 border-b border-[var(--border-light)]">
            <div className="flex items-center justify-between mb-3">
              <p className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide">配置信息</p>
              <button
                onClick={onScaleClick}
                className="flex items-center gap-1 text-xs text-[var(--accent)] hover:text-[var(--accent-hover)] font-medium transition-colors"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg>
                调整副本
              </button>
            </div>
            <div className="grid grid-cols-2 gap-3">
              {[
                ['基座模型', ep.model_name],
                ['GPU 类型', ep.gpu_type],
                ['GPU 数量', String(ep.gpu_count)],
                ['部署区域', REGION_OPTIONS.find(r => r.value === ep.region)?.label ?? ep.region],
                ['副本范围', `${ep.min_replicas} ~ ${ep.max_replicas}`],
                ['当前副本', String(ep.current_replicas)],
              ].map(([k, v]) => (
                <div key={k} className="bg-[var(--bg-secondary)] rounded-xl p-3">
                  <p className="text-[10px] text-[var(--text-muted)] mb-0.5">{k}</p>
                  <p className="text-xs font-medium text-[var(--text)] truncate" title={v}>{v}</p>
                </div>
              ))}
            </div>
          </div>

          {/* API Code Example */}
          <div className="px-6 py-4">
            <p className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wide mb-3">API 调用示例</p>
            <div className="rounded-xl border border-[var(--border-light)] overflow-hidden">
              <div className="flex items-center border-b border-[var(--border-light)] bg-[var(--bg-secondary)] px-1 pt-1">
                {(['curl', 'python'] as const).map(tab => (
                  <button key={tab} onClick={() => setCodeTab(tab)}
                    className={`px-4 py-2 text-xs font-medium rounded-t-lg transition-all ${codeTab === tab ? 'bg-white text-[var(--text)] shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text)]'}`}>
                    {tab === 'curl' ? 'cURL' : 'Python'}
                  </button>
                ))}
                <div className="flex-1" />
                <button onClick={() => copy(codeTab === 'curl' ? curlExample : pythonExample, 'code')}
                  className="flex items-center gap-1 text-xs text-[var(--accent)] px-3 py-1.5 hover:text-[var(--accent-hover)] transition-colors">
                  {copied === 'code'
                    ? <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><polyline points="20 6 9 17 4 12"/></svg>
                    : <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                  }
                  {copied === 'code' ? '已复制' : '复制代码'}
                </button>
              </div>
              <pre className="p-4 text-[11px] font-mono bg-[#1e1e2e] overflow-x-auto leading-relaxed" style={{ color: '#cdd6f4' }}>
                <code>{codeTab === 'curl' ? curlExample : pythonExample}</code>
              </pre>
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="px-6 py-4 border-t border-[var(--border-light)] flex items-center gap-3">
          <button
            onClick={handleToggle}
            disabled={togglingStatus || !['running', 'stopped'].includes(ep.status)}
            className={`flex-1 flex items-center justify-center gap-1.5 text-sm py-2 px-4 rounded-lg font-medium transition-all disabled:opacity-50 ${ep.status === 'running' ? 'btn-outline' : 'btn-primary'}`}
          >
            {togglingStatus
              ? <svg className="animate-spin" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
              : ep.status === 'running'
                ? <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>
                : <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polygon points="5 3 19 12 5 21 5 3"/></svg>
            }
            {ep.status === 'running' ? '停止端点' : '启动端点'}
          </button>
          <button onClick={onScaleClick} className="btn-outline flex items-center gap-1.5 text-sm py-2 px-4">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg>
            调整副本
          </button>
        </div>
      </div>
    </div>
  )
}

// ── Steps Guide for Empty State ───────────────────────────────────────────
const guideSteps = [
  {
    num: 1, title: '选择基座模型',
    desc: '从平台预置的对话或多模态模型中选择，支持 DeepSeek、Qwen、Llama 等主流开源模型',
    svg: (
      <svg viewBox="0 0 120 80" fill="none" className="w-full h-full">
        <circle cx="60" cy="36" r="22" fill="#e8f0fe"/><circle cx="60" cy="36" r="14" fill="#c5d9fb"/>
        <circle cx="60" cy="36" r="7" fill="#1a73e8" opacity="0.7"/>
        <line x1="30" y1="36" x2="46" y2="36" stroke="#1a73e8" strokeWidth="2" strokeDasharray="3 2"/>
        <line x1="74" y1="36" x2="90" y2="36" stroke="#1a73e8" strokeWidth="2" strokeDasharray="3 2"/>
        <circle cx="26" cy="36" r="5" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5"/>
        <circle cx="94" cy="36" r="5" fill="#e8f0fe" stroke="#1a73e8" strokeWidth="1.5"/>
      </svg>
    ),
  },
  {
    num: 2, title: '配置 GPU 资源',
    desc: '选择 GPU 型号（A100/H100/L40S）、数量与区域，按 GPU·小时计费，停止即停止扣费',
    svg: (
      <svg viewBox="0 0 120 80" fill="none" className="w-full h-full">
        <rect x="20" y="24" width="80" height="32" rx="6" fill="#e8f0fe"/>
        <rect x="28" y="30" width="24" height="20" rx="3" fill="#c5d9fb"/>
        <rect x="60" y="30" width="32" height="9" rx="2" fill="#c5d9fb"/>
        <rect x="60" y="42" width="20" height="8" rx="2" fill="#c5d9fb" opacity="0.6"/>
        <circle cx="40" cy="40" r="6" fill="#1a73e8" opacity="0.6"/>
        <path d="M37 40 l2.5 2.5 4-4" stroke="white" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
        <rect x="64" y="33" width="20" height="3" rx="1.5" fill="#1a73e8" opacity="0.5"/>
        <rect x="64" y="45" width="12" height="3" rx="1.5" fill="#1a73e8" opacity="0.3"/>
      </svg>
    ),
  },
  {
    num: 3, title: '通过 routing_key 调用',
    desc: '端点创建后会生成唯一 routing_key，作为 OpenAI 兼容 API 的 model 字段路由到专属实例',
    svg: (
      <svg viewBox="0 0 120 80" fill="none" className="w-full h-full">
        <rect x="16" y="20" width="88" height="40" rx="6" fill="#e8f0fe"/>
        <rect x="22" y="26" width="76" height="28" rx="4" fill="#c5d9fb" opacity="0.5"/>
        <rect x="28" y="32" width="30" height="3" rx="1.5" fill="#1a73e8" opacity="0.6"/>
        <rect x="28" y="39" width="46" height="2.5" rx="1.25" fill="#1a73e8" opacity="0.3"/>
        <rect x="28" y="45" width="22" height="2.5" rx="1.25" fill="#1a73e8" opacity="0.2"/>
        <circle cx="86" cy="38" r="8" fill="#1a73e8" opacity="0.15" stroke="#1a73e8" strokeWidth="1.2"/>
        <path d="M83 38 l4 0 M85 36 l2 2 -2 2" stroke="#1a73e8" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
      </svg>
    ),
  },
]

// ── Main Page ──────────────────────────────────────────────────────────────
export default function EndpointsPage() {
  const [templates, setTemplates] = useState<Template[]>([])
  const [endpoints, setEndpoints] = useState<DedicatedEndpoint[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [selectedEp, setSelectedEp] = useState<DedicatedEndpoint | null>(null)
  const [scalingEp, setScalingEp] = useState<DedicatedEndpoint | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')

  useEffect(() => {
    Promise.all([fetchDedicatedTemplates(), listDedicatedEndpoints()])
      .then(([t, e]) => {
        setTemplates(t.templates ?? [])
        setEndpoints(e.data ?? [])
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const filteredEndpoints = useMemo(() => {
    return endpoints.filter(ep => {
      const matchStatus = statusFilter === 'all' || ep.status === statusFilter
      const q = search.toLowerCase()
      const matchSearch = !q || ep.name.toLowerCase().includes(q) || ep.model_name.toLowerCase().includes(q) || ep.routing_key?.toLowerCase().includes(q)
      return matchStatus && matchSearch
    })
  }, [endpoints, search, statusFilter])

  // 总费率（所有运行中端点）
  const totalHourlyRate = useMemo(() => {
    return endpoints
      .filter(ep => ep.status === 'running')
      .reduce((sum, ep) => sum + calcHourlyRate(ep.gpu_type, ep.gpu_count, ep.current_replicas), 0)
  }, [endpoints])

  async function handleDelete(id: string) {
    if (!confirm('确认删除该专属端点？此操作不可撤销。')) return
    setDeletingId(id)
    try {
      await deleteDedicatedEndpoint(id)
      setEndpoints(prev => prev.filter(ep => ep.id !== id))
      if (selectedEp?.id === id) setSelectedEp(null)
    } finally {
      setDeletingId(null)
    }
  }

  function handleStatusChange(id: string, status: string) {
    setEndpoints(prev => prev.map(ep => ep.id === id ? { ...ep, status } : ep))
    setSelectedEp(prev => prev?.id === id ? { ...prev, status } : prev)
  }

  function handleEndpointUpdated(updated: DedicatedEndpoint) {
    setEndpoints(prev => prev.map(ep => ep.id === updated.id ? updated : ep))
    setSelectedEp(prev => prev?.id === updated.id ? updated : prev)
  }

  const isEmpty = !loading && endpoints.length === 0

  return (
    <div className="min-h-screen flex flex-col">
      {/* Top bar */}
      <div className="flex items-center justify-between px-8 py-5">
        <div>
          <h1 className="text-[15px] font-semibold text-[var(--text)]">专属端点</h1>
          {endpoints.length > 0 && totalHourlyRate > 0 && (
            <p className="text-xs text-[var(--text-muted)] mt-0.5">
              当前运行费率：<span className="font-semibold text-amber-600">¥{totalHourlyRate.toFixed(2)}/时</span>
              <span className="ml-1 text-[var(--text-muted)]">（预估 ¥{(totalHourlyRate * 24).toFixed(2)}/天）</span>
            </p>
          )}
        </div>
        <div className="flex items-center gap-3">
          <button className="btn-outline flex items-center gap-1.5 text-sm py-2 px-4">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            使用文档
          </button>
          <button onClick={() => setShowModal(true)} className="btn-primary flex items-center gap-1.5 text-sm py-2 px-4">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            新建端点
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
              <p className="text-[15px] font-semibold text-[var(--text)]">你还没有专属端点</p>
              <p className="text-sm text-[var(--text-muted)] mt-1">创建专属 GPU 实例，获得独占推理资源与最低延迟</p>
            </div>
          </div>
          <div className="flex items-stretch gap-5 max-w-[900px] w-full px-8">
            {guideSteps.map(s => (
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

      {/* Endpoint list */}
      {!loading && endpoints.length > 0 && (
        <div className="px-8 pb-8">
          {/* Search & Filter bar */}
          <div className="flex items-center gap-3 mb-4">
            <div className="relative w-64">
              <svg className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <input
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="搜索端点名称、模型…"
                className="pl-8 pr-3 py-2 text-sm w-full rounded-lg border border-[var(--border-light)] bg-white focus:outline-none focus:border-[var(--accent)] transition-colors"
              />
            </div>
            <div className="flex items-center gap-1 bg-white rounded-lg border border-[var(--border-light)] p-0.5">
              {[['all', '全部'], ['running', '运行中'], ['stopped', '已停止'], ['provisioning', '启动中'], ['error', '异常']].map(([v, label]) => (
                <button
                  key={v}
                  onClick={() => setStatusFilter(v)}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${statusFilter === v ? 'bg-[var(--bg-secondary)] text-[var(--text)] shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                >{label}</button>
              ))}
            </div>
            <span className="text-xs text-[var(--text-muted)] ml-auto">
              {filteredEndpoints.length} / {endpoints.length} 个端点
            </span>
          </div>

          <div className="bg-white rounded-2xl border border-[var(--border-light)] overflow-hidden" style={{ boxShadow: 'var(--shadow-sm)' }}>
            <table>
              <thead>
                <tr className="border-b border-[var(--border-light)]">
                  <th className="px-5 py-3.5 text-left">名称 / 模型</th>
                  <th className="px-5 py-3.5 text-left">Routing Key</th>
                  <th className="px-5 py-3.5 text-left">GPU / 区域</th>
                  <th className="px-5 py-3.5 text-left">副本</th>
                  <th className="px-5 py-3.5 text-left">费率</th>
                  <th className="px-5 py-3.5 text-left">状态</th>
                  <th className="px-5 py-3.5 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {filteredEndpoints.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-5 py-10 text-center text-sm text-[var(--text-muted)]">没有符合条件的端点</td>
                  </tr>
                ) : filteredEndpoints.map(ep => {
                  const st = statusConfig[ep.status] ?? { label: ep.status, cls: 'tag-blue' }
                  const rate = calcHourlyRate(ep.gpu_type, ep.gpu_count, ep.current_replicas)
                  return (
                    <tr
                      key={ep.id}
                      className={`hover:bg-[var(--bg-secondary)] transition-colors cursor-pointer ${selectedEp?.id === ep.id ? 'bg-[var(--accent-light)]' : ''}`}
                      onClick={() => setSelectedEp(selectedEp?.id === ep.id ? null : ep)}
                    >
                      <td className="px-5 py-4">
                        <p className="text-sm font-medium text-[var(--text)]">{ep.name}</p>
                        <p className="text-xs text-[var(--text-muted)] mt-0.5 truncate max-w-[160px]">{ep.model_name}</p>
                      </td>
                      <td className="px-5 py-4 max-w-[180px]">
                        <div className="flex items-center gap-1.5">
                          <code className="text-xs font-mono text-[var(--text-secondary)] truncate">{ep.routing_key}</code>
                          <button
                            onClick={e => { e.stopPropagation(); navigator.clipboard.writeText(ep.routing_key) }}
                            className="flex-shrink-0 text-[var(--accent)] hover:text-[var(--accent-hover)] transition-colors"
                            title="复制 routing_key"
                          >
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                          </button>
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <p className="text-sm text-[var(--text-secondary)]">{ep.gpu_type} × {ep.gpu_count}</p>
                        <p className="text-xs text-[var(--text-muted)] mt-0.5">{REGION_OPTIONS.find(r => r.value === ep.region)?.label ?? ep.region}</p>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-sm text-[var(--text-secondary)]">{ep.min_replicas} – {ep.max_replicas}</span>
                        <p className="text-xs text-[var(--text-muted)] mt-0.5">当前 {ep.current_replicas}</p>
                      </td>
                      <td className="px-5 py-4">
                        {ep.status === 'running' && rate > 0 ? (
                          <div>
                            <p className="text-sm font-medium text-amber-600">¥{rate.toFixed(2)}/时</p>
                            <p className="text-xs text-[var(--text-muted)] mt-0.5">≈¥{(rate * 24).toFixed(2)}/天</p>
                          </div>
                        ) : (
                          <span className="text-sm text-[var(--text-muted)]">—</span>
                        )}
                      </td>
                      <td className="px-5 py-4">
                        <span className={`tag ${st.cls}`}>{st.label}</span>
                      </td>
                      <td className="px-5 py-4">
                        <div className="flex items-center justify-end gap-2" onClick={e => e.stopPropagation()}>
                          <button
                            onClick={e => { e.stopPropagation(); setScalingEp(ep) }}
                            className="text-xs text-[var(--text-secondary)] hover:text-[var(--text)] hover:underline"
                          >
                            Scale
                          </button>
                          <button
                            onClick={() => setSelectedEp(selectedEp?.id === ep.id ? null : ep)}
                            className="text-xs text-[var(--accent)] hover:underline"
                          >
                            详情
                          </button>
                          <button
                            onClick={() => handleDelete(ep.id)}
                            disabled={deletingId === ep.id}
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

      {/* Modals */}
      {showModal && (
        <CreateModal
          templates={templates}
          onClose={() => setShowModal(false)}
          onCreated={ep => { setEndpoints(prev => [ep, ...prev]); setSelectedEp(ep) }}
        />
      )}

      {scalingEp && (
        <ScaleModal
          ep={scalingEp}
          onClose={() => setScalingEp(null)}
          onUpdated={updated => { handleEndpointUpdated(updated); setScalingEp(null) }}
        />
      )}

      {selectedEp && (
        <DetailPanel
          ep={selectedEp}
          onClose={() => setSelectedEp(null)}
          onStatusChange={handleStatusChange}
          onScaleClick={() => { setScalingEp(selectedEp) }}
        />
      )}
    </div>
  )
}
