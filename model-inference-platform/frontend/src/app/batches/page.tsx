'use client'

import { useEffect, useState, useRef } from 'react'
import {
  listBatches,
  createBatch,
  cancelBatch,
  listFiles,
  uploadFile,
} from '@/lib/api'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface BatchRequestCounts {
  total: number
  completed: number
  failed: number
}

interface Batch {
  id: string
  endpoint: string
  status: string
  input_file_id: string
  output_file_id?: string
  completion_window: string
  created_at: number
  in_progress_at?: number
  completed_at?: number
  cancelled_at?: number
  request_counts: BatchRequestCounts
  metadata?: Record<string, string>
}

interface FileObject {
  id: string
  filename: string
  purpose: string
  bytes: number
  created_at: number
  status: string
}

const STATUS_MAP: Record<string, { label: string; cls: string }> = {
  validating:  { label: '校验中',  cls: 'tag tag-blue' },
  in_progress: { label: '推理中',  cls: 'tag tag-blue' },
  completed:   { label: '已完成',  cls: 'tag tag-green' },
  failed:      { label: '失败',    cls: 'tag tag-red' },
  cancelled:   { label: '已取消',  cls: 'tag' },
  cancelling:  { label: '取消中',  cls: 'tag' },
  expired:     { label: '已过期',  cls: 'tag' },
}

function formatBytes(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

function formatDate(ts: number) {
  return new Date(ts * 1000).toLocaleString('zh-CN', { hour12: false })
}

// ── 空态插画 ───────────────────────────────────────────────────────────────
function EmptyIllustration() {
  return (
    <svg width="120" height="100" viewBox="0 0 120 100" fill="none" xmlns="http://www.w3.org/2000/svg">
      {/* 底部椭圆阴影 */}
      <ellipse cx="60" cy="88" rx="38" ry="7" fill="var(--border-light)" />
      {/* 主体六边形 */}
      <path d="M60 18 L88 34 L88 66 L60 82 L32 66 L32 34 Z" fill="var(--bg-secondary)" stroke="var(--border)" strokeWidth="1.5"/>
      {/* 内层六边形 */}
      <path d="M60 28 L80 39 L80 61 L60 72 L40 61 L40 39 Z" fill="var(--bg-hover)" stroke="var(--border-light)" strokeWidth="1"/>
      {/* 中心图标 */}
      <circle cx="60" cy="50" r="10" fill="var(--border)" opacity="0.5"/>
      <path d="M56 50 L59 53 L64 47" stroke="var(--bg)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
      {/* 装饰点 */}
      <circle cx="32" cy="34" r="3" fill="var(--primary)" opacity="0.3"/>
      <circle cx="88" cy="34" r="3" fill="var(--primary)" opacity="0.2"/>
      <circle cx="88" cy="66" r="3" fill="var(--primary)" opacity="0.3"/>
      <circle cx="32" cy="66" r="2" fill="var(--primary)" opacity="0.2"/>
    </svg>
  )
}

// ── 步骤插画组件 ───────────────────────────────────────────────────────────
function StepIllustration({ step }: { step: 1 | 2 | 3 }) {
  if (step === 1) return (
    <svg width="80" height="68" viewBox="0 0 80 68" fill="none">
      <rect x="14" y="8" width="44" height="52" rx="4" fill="var(--bg)" stroke="var(--border)" strokeWidth="1.5"/>
      <rect x="20" y="18" width="28" height="3" rx="1.5" fill="var(--border)"/>
      <rect x="20" y="25" width="22" height="3" rx="1.5" fill="var(--border-light)"/>
      <rect x="20" y="32" width="26" height="3" rx="1.5" fill="var(--border-light)"/>
      <rect x="20" y="39" width="18" height="3" rx="1.5" fill="var(--border-light)"/>
      <circle cx="60" cy="48" r="10" fill="var(--primary)" opacity="0.15"/>
      <path d="M56 48 L59 51 L64 45" stroke="var(--primary)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
  if (step === 2) return (
    <svg width="80" height="68" viewBox="0 0 80 68" fill="none">
      <rect x="10" y="12" width="48" height="44" rx="4" fill="var(--bg)" stroke="var(--border)" strokeWidth="1.5"/>
      <rect x="16" y="20" width="24" height="3" rx="1.5" fill="var(--border)"/>
      <rect x="16" y="27" width="18" height="3" rx="1.5" fill="var(--border-light)"/>
      <circle cx="58" cy="46" r="12" fill="var(--primary)" opacity="0.12"/>
      <line x1="58" y1="40" x2="58" y2="52" stroke="var(--primary)" strokeWidth="2.5" strokeLinecap="round"/>
      <line x1="52" y1="46" x2="64" y2="46" stroke="var(--primary)" strokeWidth="2.5" strokeLinecap="round"/>
    </svg>
  )
  return (
    <svg width="80" height="68" viewBox="0 0 80 68" fill="none">
      <rect x="8" y="10" width="50" height="48" rx="4" fill="var(--bg)" stroke="var(--border)" strokeWidth="1.5"/>
      <rect x="15" y="20" width="22" height="3" rx="1.5" fill="var(--border)"/>
      <rect x="15" y="28" width="30" height="2.5" rx="1.25" fill="var(--primary)" opacity="0.3"/>
      <rect x="15" y="34" width="20" height="2.5" rx="1.25" fill="var(--border-light)"/>
      <rect x="15" y="40" width="26" height="2.5" rx="1.25" fill="var(--border-light)"/>
      <circle cx="60" cy="46" r="11" fill="var(--primary)" opacity="0.12"/>
      <path d="M56 46 l3 3 l6-7" stroke="var(--primary)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

// ── 创建任务弹窗 ───────────────────────────────────────────────────────────
function CreateBatchModal({
  files,
  onClose,
  onCreated,
}: {
  files: FileObject[]
  onClose: () => void
  onCreated: () => void
}) {
  const [selectedFileId, setSelectedFileId] = useState('')
  const [endpoint, setEndpoint] = useState('/v1/chat/completions')
  const [completionWindow, setCompletionWindow] = useState('24h')
  const [creating, setCreating] = useState(false)
  const [err, setErr] = useState('')

  // 文件上传
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [localFiles, setLocalFiles] = useState<FileObject[]>(files)

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    setErr('')
    try {
      await uploadFile(file, 'batch')
      const res = await listFiles('batch')
      const updated: FileObject[] = res.data || []
      setLocalFiles(updated)
      if (updated.length > 0) setSelectedFileId(updated[updated.length - 1].id)
    } catch (ex: unknown) {
      setErr(ex instanceof Error ? ex.message : '上传失败')
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const handleCreate = async () => {
    if (!selectedFileId) { setErr('请先选择或上传输入文件'); return }
    setCreating(true)
    setErr('')
    try {
      await createBatch({ input_file_id: selectedFileId, endpoint, completion_window: completionWindow })
      onCreated()
      onClose()
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : '创建失败')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-[var(--bg)] rounded-2xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden">
        {/* 弹窗头部 */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border)]">
          <h3 className="font-semibold text-[var(--text)]">创建批量推理任务</h3>
          <button type="button" onClick={onClose} className="text-[var(--text-muted)] hover:text-[var(--text)] transition-colors">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>

        <div className="px-6 py-5 space-y-5">
          {err && (
            <div className="text-[var(--danger)] bg-[var(--danger-bg)] text-sm px-3 py-2.5 rounded-lg">{err}</div>
          )}

          {/* 输入文件 */}
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-2">
              输入文件 <span className="text-[var(--danger)]">*</span>
            </label>
            <div className="flex gap-2">
              <select
                className="flex-1"
                value={selectedFileId}
                onChange={e => setSelectedFileId(e.target.value)}
              >
                <option value="">— 选择已上传文件 —</option>
                {localFiles.map(f => (
                  <option key={f.id} value={f.id}>{f.filename}  ({f.id.slice(-8)})</option>
                ))}
              </select>
              <input ref={fileInputRef} type="file" accept=".jsonl,.json,.txt" onChange={handleUpload} className="hidden"/>
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading}
                className="btn-outline text-xs px-3 shrink-0"
              >
                {uploading ? (
                  <svg className="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                ) : (
                  <>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="inline mr-1">
                      <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
                    </svg>
                    上传
                  </>
                )}
              </button>
            </div>
            <p className="text-[11px] text-[var(--text-muted)] mt-1.5">
              每行一个 JSON 对象，需包含 <code className="bg-[var(--bg-hover)] px-1 rounded">custom_id</code>、<code className="bg-[var(--bg-hover)] px-1 rounded">method</code>、<code className="bg-[var(--bg-hover)] px-1 rounded">url</code>、<code className="bg-[var(--bg-hover)] px-1 rounded">body</code> 字段
            </p>
          </div>

          {/* 推理端点 */}
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-2">推理端点</label>
            <select value={endpoint} onChange={e => setEndpoint(e.target.value)}>
              <option value="/v1/chat/completions">/v1/chat/completions</option>
              <option value="/v1/completions">/v1/completions</option>
              <option value="/v1/embeddings">/v1/embeddings</option>
            </select>
          </div>

          {/* 完成窗口 */}
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-2">完成时间窗口</label>
            <div className="grid grid-cols-3 gap-2">
              {(['1h', '6h', '24h'] as const).map(w => (
                <button
                  key={w}
                  type="button"
                  onClick={() => setCompletionWindow(w)}
                  className={`py-2 rounded-lg text-sm font-medium border transition-colors ${
                    completionWindow === w
                      ? 'bg-[var(--primary)] text-white border-[var(--primary)]'
                      : 'bg-[var(--bg-secondary)] text-[var(--text-secondary)] border-[var(--border-light)] hover:border-[var(--primary)]/40'
                  }`}
                >
                  {w === '1h' ? '1 小时' : w === '6h' ? '6 小时' : '24 小时'}
                  {w === '24h' && <span className="block text-[10px] opacity-70">推荐</span>}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* 底部按钮 */}
        <div className="px-6 py-4 border-t border-[var(--border)] flex justify-end gap-3">
          <button type="button" onClick={onClose} className="btn-outline">取消</button>
          <button
            type="button"
            onClick={handleCreate}
            disabled={creating || !selectedFileId}
            className="btn-primary"
          >
            {creating ? (
              <span className="flex items-center gap-1.5">
                <svg className="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                提交中…
              </span>
            ) : '提交任务'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── 批次详情行 ────────────────────────────────────────────────────────────
function BatchRow({ batch, onCancel }: { batch: Batch; onCancel: (id: string) => void }) {
  const s = STATUS_MAP[batch.status] ?? { label: batch.status, cls: 'tag' }
  const progress = batch.request_counts?.total
    ? Math.round((batch.request_counts.completed / batch.request_counts.total) * 100)
    : 0
  const canCancel = batch.status === 'validating' || batch.status === 'in_progress'

  return (
    <div className="border border-[var(--border-light)] rounded-xl p-4 hover:bg-[var(--bg-hover)] transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2 flex-wrap">
            <span className="font-mono text-[11px] text-[var(--text-secondary)] bg-[var(--bg-secondary)] px-2 py-0.5 rounded truncate max-w-[240px]">
              {batch.id}
            </span>
            <span className={s.cls}>{s.label}</span>
          </div>
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-[var(--text-muted)]">
            <span className="flex items-center gap-1">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 16.92V19a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6A19.79 19.79 0 0 1 2.12 3.18 2 2 0 0 1 4.11 1h2.08a2 2 0 0 1 2 1.72"/></svg>
              {batch.endpoint}
            </span>
            <span>窗口 {batch.completion_window}</span>
            <span>创建 {formatDate(batch.created_at)}</span>
            {batch.completed_at && <span>完成 {formatDate(batch.completed_at)}</span>}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {batch.output_file_id && (
            <a
              href={`${API_BASE}/v1/files/${batch.output_file_id}/content`}
              download
              className="btn-outline text-xs"
            >
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="inline mr-1">
                <polyline points="8 17 12 21 16 17"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.88 18.09A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
              </svg>
              下载结果
            </a>
          )}
          {canCancel && (
            <button
              type="button"
              className="text-xs px-3 py-1.5 rounded-lg border border-[var(--danger)]/40 text-[var(--danger)] hover:bg-[var(--danger)]/5 transition-colors"
              onClick={() => onCancel(batch.id)}
            >
              取消
            </button>
          )}
        </div>
      </div>

      {/* 进度条 */}
      {(batch.status === 'in_progress' || batch.status === 'completed') && batch.request_counts?.total > 0 && (
        <div className="mt-3">
          <div className="flex justify-between text-[11px] text-[var(--text-muted)] mb-1.5">
            <span>
              {batch.request_counts.completed} / {batch.request_counts.total} 条完成
              {batch.request_counts.failed > 0 && (
                <span className="text-[var(--danger)] ml-2">{batch.request_counts.failed} 条失败</span>
              )}
            </span>
            <span className="font-medium">{progress}%</span>
          </div>
          <div className="h-1.5 bg-[var(--bg-hover)] rounded-full overflow-hidden">
            <div
              className="h-full rounded-full transition-all duration-700"
              style={{
                width: `${progress}%`,
                background: batch.status === 'completed' ? 'var(--success)' : 'var(--primary)',
              }}
            />
          </div>
        </div>
      )}
    </div>
  )
}

// ── 主页面 ────────────────────────────────────────────────────────────────
export default function BatchesPage() {
  const [batches, setBatches] = useState<Batch[]>([])
  const [files, setFiles] = useState<FileObject[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const load = async () => {
    try {
      const [batchRes, fileRes] = await Promise.all([listBatches(), listFiles('batch')])
      setBatches(batchRes.data || [])
      setFiles(fileRes.data || [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    refreshTimerRef.current = setInterval(() => void load(), 5000)
    return () => { if (refreshTimerRef.current) clearInterval(refreshTimerRef.current) }
  }, [])

  const handleCancel = async (id: string) => {
    try {
      await cancelBatch(id)
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '取消失败')
    }
  }

  const isEmpty = batches.length === 0

  return (
    <div className="min-h-full">
      {/* ── 顶部操作栏 ── */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h2 className="text-xl font-semibold text-[var(--text)]">批量推理</h2>
          <p className="text-xs text-[var(--text-muted)] mt-0.5">异步批量推理，享受 50% 折扣定价，不占用实时推理速率配额</p>
        </div>
        <div className="flex items-center gap-2">
          {/* 使用指南 */}
          <button
            type="button"
            className="flex items-center gap-1.5 px-3.5 py-2 rounded-lg border border-[var(--border)] text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] transition-colors"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
              <line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>
            </svg>
            使用指南
          </button>
          {/* 刷新 */}
          <button
            type="button"
            onClick={() => void load()}
            className="p-2 rounded-lg border border-[var(--border)] text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] transition-colors"
            title="刷新"
          >
            <svg
              width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"
              className={loading ? 'animate-spin' : ''}
            >
              <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
            </svg>
          </button>
          {/* 创建 */}
          <button
            type="button"
            onClick={() => setShowCreate(true)}
            className="btn-primary flex items-center gap-1.5"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            创建批量推理任务
          </button>
        </div>
      </div>

      {error && (
        <div className="text-[var(--danger)] bg-[var(--danger-bg)] text-sm px-4 py-3 rounded-lg mb-6">{error}</div>
      )}

      {isEmpty ? (
        /* ── 空态 ── */
        <div className="flex flex-col items-center justify-center py-16">
          <EmptyIllustration />
          <h3 className="mt-6 text-base font-medium text-[var(--text)]">你还没有批量推理任务</h3>
          <p className="mt-1.5 text-sm text-[var(--text-muted)]">请按照以下步骤去创建批量任务</p>

          {/* 三步引导卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-12 w-full max-w-3xl">
            {[
              {
                step: 1 as const,
                title: '准备数据文件',
                desc: '准备批量推理的数据文件，参考：',
                link: '批量模型推理文件格式说明',
              },
              {
                step: 2 as const,
                title: '创建批量推理任务',
                desc: '设置批量推理任务中包含的各项参数，提交推理目标',
                link: null,
              },
              {
                step: 3 as const,
                title: '管理批量推理任务',
                desc: '已提交的任务根据处理进度显示不同状态，已完成的任务可查询结果',
                link: null,
              },
            ].map(({ step, title, desc, link }) => (
              <div
                key={step}
                className="bg-[var(--bg-secondary)] rounded-2xl p-5 flex flex-col border border-[var(--border-light)] hover:border-[var(--primary)]/30 transition-colors"
              >
                <div className="flex items-center gap-2.5 mb-3">
                  <span className="w-6 h-6 rounded-full bg-[var(--primary)] text-white text-xs font-bold flex items-center justify-center shrink-0">
                    {step}
                  </span>
                  <h4 className="text-sm font-semibold text-[var(--text)]">{title}</h4>
                </div>
                <p className="text-xs text-[var(--text-muted)] leading-relaxed flex-1">
                  {desc}
                  {link && (
                    <button type="button" className="text-[var(--primary)] hover:underline ml-0.5">{link}</button>
                  )}
                </p>
                <div className="mt-5 flex justify-center">
                  <StepIllustration step={step} />
                </div>
              </div>
            ))}
          </div>
        </div>
      ) : (
        /* ── 任务列表 ── */
        <div className="space-y-3">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-[var(--text-muted)]">共 {batches.length} 个任务</span>
            <span className="text-[11px] text-[var(--text-muted)]">每 5s 自动刷新</span>
          </div>
          {batches.map(b => (
            <BatchRow key={b.id} batch={b} onCancel={handleCancel} />
          ))}
        </div>
      )}

      {/* 创建弹窗 */}
      {showCreate && (
        <CreateBatchModal
          files={files}
          onClose={() => setShowCreate(false)}
          onCreated={() => void load()}
        />
      )}
    </div>
  )
}
