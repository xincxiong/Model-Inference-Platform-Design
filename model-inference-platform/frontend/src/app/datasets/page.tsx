'use client'

import { useEffect, useState } from 'react'
import {
  listDatasets,
  createDataset,
  deleteDataset,
  getDatasetContent,
  queryDataset,
  listFiles,
} from '@/lib/api'

interface Dataset {
  id: string
  name: string
  description: string
  file_id?: string
  num_rows: number
  size_bytes: number
  metadata?: Record<string, string>
  created_at: number
  updated_at: number
}

interface FileObject {
  id: string
  filename: string
  bytes: number
}

function formatBytes(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

function formatDate(ts: number) {
  return new Date(ts * 1000).toLocaleString('zh-CN', { hour12: false })
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export default function DatasetsPage() {
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [files, setFiles] = useState<FileObject[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Create form
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newFileId, setNewFileId] = useState('')
  const [creating, setCreating] = useState(false)

  // Detail view
  const [selected, setSelected] = useState<Dataset | null>(null)
  const [activeTab, setActiveTab] = useState<'preview' | 'query'>('preview')
  const [contentRows, setContentRows] = useState<unknown[]>([])
  const [contentTotal, setContentTotal] = useState(0)
  const [contentPage, setContentPage] = useState(1)
  const [contentLoading, setContentLoading] = useState(false)

  // Query
  const [filterInput, setFilterInput] = useState('')
  const [queryResults, setQueryResults] = useState<unknown[]>([])
  const [querying, setQuerying] = useState(false)

  const load = async () => {
    try {
      const [dsRes, fileRes] = await Promise.all([listDatasets(), listFiles()])
      setDatasets(dsRes.data || [])
      setFiles(fileRes.data || [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [])

  const loadContent = async (ds: Dataset, page = 1) => {
    setContentLoading(true)
    try {
      const res = await getDatasetContent(ds.id, page, 20)
      setContentRows(res.data || [])
      setContentTotal(res.total || 0)
      setContentPage(page)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载内容失败')
    } finally {
      setContentLoading(false)
    }
  }

  const handleSelect = async (ds: Dataset) => {
    setSelected(ds)
    setActiveTab('preview')
    setQueryResults([])
    setFilterInput('')
    await loadContent(ds, 1)
  }

  const handleCreate = async () => {
    if (!newName.trim()) {
      setError('数据集名称不能为空')
      return
    }
    setCreating(true)
    setError('')
    try {
      await createDataset({ name: newName, description: newDesc, file_id: newFileId || undefined })
      setNewName('')
      setNewDesc('')
      setNewFileId('')
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('确认删除该数据集？')) return
    setError('')
    try {
      await deleteDataset(id)
      if (selected?.id === id) setSelected(null)
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '删除失败')
    }
  }

  const handleQuery = async () => {
    if (!selected) return
    setQuerying(true)
    setError('')
    try {
      const res = await queryDataset(selected.id, filterInput || undefined, 50)
      setQueryResults(res.data || [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '查询失败')
    } finally {
      setQuerying(false)
    }
  }

  if (loading && datasets.length === 0) {
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
        <h2 className="text-xl font-semibold text-[var(--text)]">数据集</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">
          管理训练数据集，支持内容预览、过滤查询与导出
        </p>
      </div>

      {error && (
        <div className="text-[var(--danger)] bg-[var(--danger-bg)] text-sm px-4 py-3 rounded-lg mb-4">{error}</div>
      )}

      <div className="flex gap-5">
        {/* Left: list + create */}
        <div className="w-80 shrink-0 space-y-4">
          {/* Create */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-3">新建数据集</h3>
            <div className="space-y-2.5">
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1">数据集名称</label>
                <input
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="my-training-data"
                  className="text-sm"
                />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1">描述（可选）</label>
                <input
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  placeholder="数据集说明"
                  className="text-sm"
                />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1">关联文件（可选）</label>
                <select value={newFileId} onChange={(e) => setNewFileId(e.target.value)} className="text-sm">
                  <option value="">— 不关联文件 —</option>
                  {files.map((f) => (
                    <option key={f.id} value={f.id}>{f.filename}</option>
                  ))}
                </select>
              </div>
            </div>
            <button
              type="button"
              onClick={handleCreate}
              disabled={creating || !newName.trim()}
              className="btn-primary text-sm mt-3 w-full"
            >
              {creating ? '创建中…' : '创建数据集'}
            </button>
          </div>

          {/* Dataset list */}
          <div className="card">
            <div className="flex justify-between items-center mb-3">
              <h3 className="font-medium text-sm text-[var(--text)]">数据集列表</h3>
              <button type="button" className="btn-outline text-xs" onClick={() => void load()}>刷新</button>
            </div>
            {datasets.length === 0 ? (
              <p className="text-sm text-[var(--text-muted)] py-4 text-center">暂无数据集</p>
            ) : (
              <div className="space-y-2">
                {datasets.map((ds) => (
                  <div
                    key={ds.id}
                    className={`rounded-lg border p-3 cursor-pointer transition-colors ${
                      selected?.id === ds.id
                        ? 'border-[var(--accent)] bg-[var(--accent-bg)]'
                        : 'border-[var(--border-light)] hover:bg-[var(--bg-hover)]'
                    }`}
                    onClick={() => void handleSelect(ds)}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="font-medium text-sm text-[var(--text)] truncate">{ds.name}</p>
                        <p className="text-xs text-[var(--text-muted)] mt-0.5">
                          {ds.num_rows > 0 ? `${ds.num_rows.toLocaleString()} 行` : '空数据集'} · {formatBytes(ds.size_bytes)}
                        </p>
                      </div>
                      <button
                        type="button"
                        className="text-[var(--text-muted)] hover:text-[var(--danger)] text-xs shrink-0"
                        onClick={(e) => { e.stopPropagation(); void handleDelete(ds.id) }}
                      >
                        删除
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right: detail panel */}
        <div className="flex-1 min-w-0">
          {!selected ? (
            <div className="card h-64 flex items-center justify-center text-[var(--text-muted)]">
              <div className="text-center">
                <svg className="mx-auto mb-3 opacity-30" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v6c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/><path d="M3 11v6c0 1.66 4.03 3 9 3s9-1.34 9-3v-6"/>
                </svg>
                <p className="text-sm">从左侧选择数据集查看详情</p>
              </div>
            </div>
          ) : (
            <div className="card">
              {/* Header */}
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="font-semibold text-[var(--text)]">{selected.name}</h3>
                  {selected.description && (
                    <p className="text-sm text-[var(--text-muted)] mt-0.5">{selected.description}</p>
                  )}
                  <div className="flex gap-3 mt-1 text-xs text-[var(--text-muted)]">
                    <span>{selected.num_rows > 0 ? `${selected.num_rows.toLocaleString()} 行` : '空数据集'}</span>
                    <span>{formatBytes(selected.size_bytes)}</span>
                    <span>创建于 {formatDate(selected.created_at)}</span>
                  </div>
                </div>
                <div className="flex gap-2">
                  <a
                    href={`${API_URL}/v1/datasets/${selected.id}/export?format=jsonl`}
                    download
                    className="btn-outline text-xs"
                  >
                    导出 JSONL
                  </a>
                  <a
                    href={`${API_URL}/v1/datasets/${selected.id}/export?format=csv`}
                    download
                    className="btn-outline text-xs"
                  >
                    导出 CSV
                  </a>
                </div>
              </div>

              {/* Tabs */}
              <div className="flex gap-1 mb-4 border-b border-[var(--border-light)]">
                {(['preview', 'query'] as const).map((t) => (
                  <button
                    key={t}
                    type="button"
                    onClick={() => setActiveTab(t)}
                    className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                      activeTab === t
                        ? 'border-[var(--accent)] text-[var(--accent)]'
                        : 'border-transparent text-[var(--text-muted)] hover:text-[var(--text)]'
                    }`}
                  >
                    {t === 'preview' ? '内容预览' : '过滤查询'}
                  </button>
                ))}
              </div>

              {activeTab === 'preview' && (
                <div>
                  {contentLoading ? (
                    <div className="flex items-center gap-2 text-[var(--text-muted)] py-6 justify-center">
                      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                      加载内容…
                    </div>
                  ) : contentRows.length === 0 ? (
                    <p className="text-sm text-[var(--text-muted)] py-6 text-center">暂无数据（需关联文件）</p>
                  ) : (
                    <>
                      <div className="space-y-2 mb-4">
                        {contentRows.map((row, i) => (
                          <div key={i} className="bg-[var(--bg-secondary)] rounded-lg p-3 text-xs font-mono text-[var(--text-secondary)] overflow-x-auto whitespace-pre-wrap break-all">
                            {typeof row === 'string' ? row : JSON.stringify(row, null, 2)}
                          </div>
                        ))}
                      </div>
                      <div className="flex items-center justify-between text-xs text-[var(--text-muted)]">
                        <span>共 {contentTotal} 行，每页 20</span>
                        <div className="flex gap-2">
                          <button
                            type="button"
                            className="btn-outline text-xs"
                            disabled={contentPage <= 1}
                            onClick={() => void loadContent(selected, contentPage - 1)}
                          >
                            上一页
                          </button>
                          <span className="self-center">第 {contentPage} 页</span>
                          <button
                            type="button"
                            className="btn-outline text-xs"
                            disabled={contentPage * 20 >= contentTotal}
                            onClick={() => void loadContent(selected, contentPage + 1)}
                          >
                            下一页
                          </button>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              )}

              {activeTab === 'query' && (
                <div>
                  <div className="flex gap-2 mb-4">
                    <input
                      value={filterInput}
                      onChange={(e) => setFilterInput(e.target.value)}
                      placeholder="过滤条件，格式: field=value（如 role=user）"
                      className="flex-1 text-sm"
                      onKeyDown={(e) => e.key === 'Enter' && void handleQuery()}
                    />
                    <button
                      type="button"
                      onClick={handleQuery}
                      disabled={querying}
                      className="btn-primary text-sm"
                    >
                      {querying ? '查询中…' : '查询'}
                    </button>
                  </div>
                  {queryResults.length === 0 && !querying ? (
                    <p className="text-sm text-[var(--text-muted)] py-6 text-center">
                      输入过滤条件后点击查询，留空则返回所有数据（最多 50 条）
                    </p>
                  ) : (
                    <div className="space-y-2">
                      <p className="text-xs text-[var(--text-muted)] mb-2">共 {queryResults.length} 条结果</p>
                      {queryResults.map((row, i) => (
                        <div key={i} className="bg-[var(--bg-secondary)] rounded-lg p-3 text-xs font-mono text-[var(--text-secondary)] overflow-x-auto whitespace-pre-wrap break-all">
                          {JSON.stringify(row, null, 2)}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
