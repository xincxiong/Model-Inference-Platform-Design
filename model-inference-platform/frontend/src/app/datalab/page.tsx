'use client'

import { useState, useMemo } from 'react'

// ── Mock inference logs ──────────────────────────────────────────────────────
const MOCK_MODELS = ['deepseek-ai/DeepSeek-V3', 'Qwen/Qwen3.5-72B', 'GLM/GLM-5', 'meta/Llama-4-70B']
const MOCK_ROLES = ['system', 'user', 'assistant']
const MOCK_TOPICS = ['代码生成', '数学推理', '翻译', '摘要', '问答', '创意写作']

function randomLog(i: number) {
  const d = new Date(Date.now() - i * 120000)
  const inputTok = Math.floor(200 + Math.random() * 4000)
  const outputTok = Math.floor(50 + Math.random() * 2000)
  return {
    id: `log-${String(i).padStart(5, '0')}`,
    timestamp: d.toLocaleString('zh'),
    model: MOCK_MODELS[Math.floor(Math.random() * MOCK_MODELS.length)],
    topic: MOCK_TOPICS[Math.floor(Math.random() * MOCK_TOPICS.length)],
    inputTokens: inputTok,
    outputTokens: outputTok,
    latency: parseFloat((100 + Math.random() * 500).toFixed(0)),
    status: Math.random() > 0.05 ? '200' : '429',
    input: MOCK_TOPICS[Math.floor(Math.random() * MOCK_TOPICS.length)] + ' — ' + '这是一条推理请求的输入文本示例...'.slice(0, 30 + Math.floor(Math.random() * 40)),
    output: '这是模型返回的推理结果示例...'.slice(0, 30 + Math.floor(Math.random() * 60)),
  }
}

const MOCK_LOGS = Array.from({ length: 50 }, (_, i) => randomLog(i))

// ── Simulated dataset records ────────────────────────────────────────────────
const DATASET_COLUMNS = ['id', 'role', 'content', 'tokens', 'label', 'timestamp']

function randomRecord(i: number) {
  const role = MOCK_ROLES[Math.floor(Math.random() * MOCK_ROLES.length)]
  return {
    id: `rec-${String(i).padStart(4, '0')}`,
    role,
    content: role === 'system'
      ? '你是一个有帮助的助手。'
      : role === 'user'
        ? `请${MOCK_TOPICS[Math.floor(Math.random() * MOCK_TOPICS.length)]}：什么是机器学习？`
        : '机器学习是一种通过数据训练模型来实现预测或决策的人工智能技术...',
    tokens: Math.floor(10 + Math.random() * 500),
    label: Math.random() > 0.3 ? 'positive' : 'negative',
    timestamp: new Date(Date.now() - i * 300000).toLocaleString('zh'),
  }
}

const MOCK_DATASET = Array.from({ length: 30 }, (_, i) => randomRecord(i))

// ── Simulated saved datasets ─────────────────────────────────────────────────
const MOCK_DATASETS = [
  { id: 'ds-001', name: '推理日志 2026-04', rows: 142847, size: '847 MB', type: 'Lance', created: '2026-04-15' },
  { id: 'ds-002', name: '代码微调数据集', rows: 52400, size: '124 MB', type: 'JSONL', created: '2026-04-12' },
  { id: 'ds-003', name: '数学推理筛选集', rows: 8932, size: '23 MB', type: 'Lance', created: '2026-04-10' },
  { id: 'ds-004', name: '多语言翻译对', rows: 215000, size: '1.2 GB', type: 'Parquet', created: '2026-04-08' },
  { id: 'ds-005', name: 'RLHF 偏好数据', rows: 34200, size: '89 MB', type: 'JSONL', created: '2026-04-05' },
]

export default function DataLabPage() {
  const [activeView, setActiveView] = useState<'logs' | 'sql' | 'datasets'>('logs')
  const [sqlQuery, setSqlQuery] = useState('SELECT model, COUNT(*) as cnt, AVG(latency) as avg_latency\nFROM inference_logs\nWHERE status = 200\nGROUP BY model\nORDER BY cnt DESC\nLIMIT 20;')
  const [selectedDataset, setSelectedDataset] = useState<string | null>(null)

  return (
    <div>
      {/* ── Header ─────────────────────────────────────────────────── */}
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">数据实验室</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">推理日志浏览 · SQL 查询 · 数据集管理</p>
      </div>

      {/* ── View tabs ──────────────────────────────────────────────── */}
      <div className="flex gap-1 mb-5 bg-white border border-[var(--border)] rounded-lg p-1 w-fit">
        {([
          ['logs', '推理日志'],
          ['sql', 'SQL 查询'],
          ['datasets', '数据集'],
        ] as const).map(([key, label]) => (
          <button
            key={key}
            onClick={() => { setActiveView(key); setSelectedDataset(null) }}
            className={`text-xs px-4 py-1.5 rounded-md font-medium transition-colors ${
              activeView === key ? 'bg-[var(--accent)] text-white' : 'text-[var(--text-muted)] hover:text-[var(--text)]'
            }`}
          >{label}</button>
        ))}
      </div>

      {/* ── Inference Logs ─────────────────────────────────────────── */}
      {activeView === 'logs' && <LogsView />}

      {/* ── SQL Query ──────────────────────────────────────────────── */}
      {activeView === 'sql' && (
        <SqlView query={sqlQuery} setQuery={setSqlQuery} />
      )}

      {/* ── Datasets ───────────────────────────────────────────────── */}
      {activeView === 'datasets' && (
        <DatasetsView
          selectedDataset={selectedDataset}
          setSelectedDataset={setSelectedDataset}
        />
      )}
    </div>
  )
}

// ── Logs View ─────────────────────────────────────────────────────────────────
function LogsView() {
  const [page, setPage] = useState(1)
  const pageSize = 15
  const paged = MOCK_LOGS.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-medium text-sm text-[var(--text)]">推理日志（最近 50 条）</h3>
        <div className="flex gap-2 text-xs text-[var(--text-muted)]">
          <span className="tag tag-blue">Lance 格式</span>
          <span className="tag tag-green">向量化索引</span>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
              <th className="text-left pb-2.5 font-medium">时间</th>
              <th className="text-left pb-2.5 font-medium">模型</th>
              <th className="text-left pb-2.5 font-medium">主题</th>
              <th className="text-right pb-2.5 font-medium">Input</th>
              <th className="text-right pb-2.5 font-medium">Output</th>
              <th className="text-right pb-2.5 font-medium">延迟</th>
              <th className="text-center pb-2.5 font-medium">状态</th>
              <th className="text-left pb-2.5 font-medium">输入文本</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[var(--border-light)]">
            {paged.map((log) => (
              <tr key={log.id} className="hover:bg-[var(--bg-secondary)] transition-colors">
                <td className="py-2 pr-3 font-mono text-[var(--text-muted)] whitespace-nowrap">{log.timestamp}</td>
                <td className="py-2 pr-3 font-mono text-[var(--text)]">{log.model.split('/').pop()}</td>
                <td className="py-2 pr-3"><span className="tag tag-cyan">{log.topic}</span></td>
                <td className="py-2 pr-3 text-right font-mono">{log.inputTokens.toLocaleString()}</td>
                <td className="py-2 pr-3 text-right font-mono">{log.outputTokens.toLocaleString()}</td>
                <td className={`py-2 pr-3 text-right font-mono ${log.latency > 400 ? 'text-[var(--warning)]' : 'text-[var(--text)]'}`}>{log.latency}ms</td>
                <td className="py-2 pr-3 text-center">
                  <span className={`tag ${log.status === '200' ? 'tag-green' : 'tag-amber'}`}>{log.status}</span>
                </td>
                <td className="py-2 text-[var(--text-secondary)] truncate max-w-[200px]" title={log.input}>{log.input}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between mt-4 pt-3 border-t border-[var(--border-light)] text-xs">
        <span className="text-[var(--text-muted)]">第 {((page - 1) * pageSize) + 1}–{Math.min(page * pageSize, MOCK_LOGS.length)} 条 / 共 {MOCK_LOGS.length} 条</span>
        <div className="flex gap-1">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="px-3 py-1 rounded border border-[var(--border)] text-[var(--text-secondary)] disabled:opacity-40 hover:border-[var(--accent)]">上一页</button>
          <button onClick={() => setPage(p => Math.min(Math.ceil(MOCK_LOGS.length / pageSize), p + 1))} disabled={page >= Math.ceil(MOCK_LOGS.length / pageSize)} className="px-3 py-1 rounded border border-[var(--border)] text-[var(--text-secondary)] disabled:opacity-40 hover:border-[var(--accent)]">下一页</button>
        </div>
      </div>
    </div>
  )
}

// ── SQL View ──────────────────────────────────────────────────────────────────
function SqlView({ query, setQuery }: { query: string; setQuery: (q: string) => void }) {
  const [hasRun, setHasRun] = useState(false)

  const mockResults = [
    { model: 'DeepSeek-V3', cnt: 48723, avg_latency: '142ms' },
    { model: 'Qwen3.5-72B', cnt: 34102, avg_latency: '218ms' },
    { model: 'GLM-5', cnt: 21890, avg_latency: '167ms' },
    { model: 'Llama-4-70B', cnt: 18432, avg_latency: '195ms' },
  ]

  return (
    <div className="space-y-4">
      {/* Editor */}
      <div className="card p-0 overflow-hidden">
        <div className="flex items-center justify-between px-4 py-2 border-b border-[var(--border-light)] bg-[var(--bg-secondary)]">
          <span className="text-xs font-medium text-[var(--text-muted)]">SQL 查询编辑器</span>
          <button
            onClick={() => setHasRun(true)}
            className="text-xs px-4 py-1.5 bg-[var(--accent)] text-white rounded-md hover:bg-[var(--accent-hover)] transition-colors"
          >▶ 运行</button>
        </div>
        <textarea
          value={query}
          onChange={e => setQuery(e.target.value)}
          className="w-full font-mono text-xs p-4 border-0 resize-none focus:outline-none bg-white"
          rows={8}
          spellCheck={false}
        />
      </div>

      {/* Results */}
      {hasRun && (
        <div className="card">
          <h3 className="font-medium text-sm text-[var(--text)] mb-3">查询结果（{mockResults.length} 行）</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                  <th className="text-left pb-2.5 font-medium">Model</th>
                  <th className="text-right pb-2.5 font-medium">Count</th>
                  <th className="text-right pb-2.5 font-medium">Avg Latency</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--border-light)]">
                {mockResults.map((r) => (
                  <tr key={r.model} className="hover:bg-[var(--bg-secondary)]">
                    <td className="py-2 pr-3 font-mono text-[var(--text)]">{r.model}</td>
                    <td className="py-2 pr-3 text-right font-mono">{r.cnt.toLocaleString()}</td>
                    <td className="py-2 text-right font-mono text-[var(--warning)]">{r.avg_latency}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {!hasRun && (
        <div className="card text-center py-12">
          <p className="text-sm text-[var(--text-muted)]">编写查询并点击"运行"查看结果</p>
        </div>
      )}
    </div>
  )
}

// ── Datasets View ─────────────────────────────────────────────────────────────
function DatasetsView({ selectedDataset, setSelectedDataset }: { selectedDataset: string | null; setSelectedDataset: (id: string | null) => void }) {
  if (!selectedDataset) {
    return (
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-medium text-sm text-[var(--text)]">数据集列表</h3>
          <button className="text-xs px-4 py-1.5 bg-[var(--accent)] text-white rounded-md hover:bg-[var(--accent-hover)] transition-colors">+ 上传数据集</button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                <th className="text-left pb-2.5 font-medium">名称</th>
                <th className="text-right pb-2.5 font-medium">行数</th>
                <th className="text-right pb-2.5 font-medium">大小</th>
                <th className="text-center pb-2.5 font-medium">格式</th>
                <th className="text-left pb-2.5 font-medium">创建日期</th>
                <th className="text-center pb-2.5 font-medium">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border-light)]">
              {MOCK_DATASETS.map((ds) => (
                <tr key={ds.id} className="hover:bg-[var(--bg-secondary)] transition-colors">
                  <td className="py-2.5 pr-3">
                    <button onClick={() => setSelectedDataset(ds.id)} className="text-[var(--accent)] hover:underline font-medium text-left">{ds.name}</button>
                  </td>
                  <td className="py-2.5 pr-3 text-right font-mono">{ds.rows.toLocaleString()}</td>
                  <td className="py-2.5 pr-3 text-right font-mono text-[var(--text-muted)]">{ds.size}</td>
                  <td className="py-2.5 text-center">
                    <span className={`tag ${ds.type === 'Lance' ? 'tag-purple' : ds.type === 'JSONL' ? 'tag-blue' : 'tag-green'}`}>{ds.type}</span>
                  </td>
                  <td className="py-2.5 text-[var(--text-muted)]">{ds.created}</td>
                  <td className="py-2.5 text-center">
                    <button className="text-[var(--text-muted)] hover:text-[var(--accent)] transition-colors mr-2">导出</button>
                    <button className="text-[var(--danger)] hover:opacity-70 transition-colors">删除</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    )
  }

  const ds = MOCK_DATASETS.find(d => d.id === selectedDataset)!
  return (
    <div className="space-y-4">
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-3">
              <button onClick={() => setSelectedDataset(null)} className="text-xs text-[var(--accent)] hover:underline">← 返回列表</button>
              <h3 className="font-medium text-sm text-[var(--text)]">{ds.name}</h3>
            </div>
            <p className="text-xs text-[var(--text-muted)] mt-1">{ds.rows.toLocaleString()} 行 · {ds.size} · {ds.type} · {ds.created}</p>
          </div>
          <div className="flex gap-2">
            <button className="text-xs px-3 py-1.5 border border-[var(--border)] rounded-md text-[var(--text-secondary)] hover:border-[var(--accent)] transition-colors">导出 CSV</button>
            <button className="text-xs px-3 py-1.5 border border-[var(--border)] rounded-md text-[var(--text-secondary)] hover:border-[var(--accent)] transition-colors">导出 JSONL</button>
          </div>
        </div>
      </div>
      <div className="card">
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-[var(--text-muted)] border-b border-[var(--border)]">
                {DATASET_COLUMNS.map(c => (
                  <th key={c} className={`pb-2.5 font-medium ${c === 'content' ? 'text-left flex-1' : c !== 'id' ? 'text-right' : 'text-left'}`}>{c}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--border-light)]">
              {MOCK_DATASET.slice(0, 15).map((r) => (
                <tr key={r.id} className="hover:bg-[var(--bg-secondary)]">
                  <td className="py-2 pr-3 font-mono text-[var(--text-muted)]">{r.id}</td>
                  <td className="py-2 pr-3"><span className={`tag ${r.role === 'system' ? 'tag-purple' : r.role === 'user' ? 'tag-blue' : 'tag-green'}`}>{r.role}</span></td>
                  <td className="py-2 pr-3 max-w-[300px] truncate text-[var(--text-secondary)]">{r.content}</td>
                  <td className="py-2 pr-3 text-right font-mono">{r.tokens}</td>
                  <td className="py-2 pr-3 text-right"><span className={`tag ${r.label === 'positive' ? 'tag-green' : 'tag-amber'}`}>{r.label}</span></td>
                  <td className="py-2 text-right font-mono text-[var(--text-muted)]">{r.timestamp}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
