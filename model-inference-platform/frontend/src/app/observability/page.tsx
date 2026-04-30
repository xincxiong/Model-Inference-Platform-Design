'use client'

import { useState, useEffect, useMemo } from 'react'
import { LineChart, BarChart } from '@/components/charts'

// ── Simulated observability data ─────────────────────────────────────────────
function generateTimeSeries(points: number, min: number, max: number, spikeAt?: number) {
  const data: number[] = []
  for (let i = 0; i < points; i++) {
    let v = min + Math.random() * (max - min)
    if (spikeAt !== undefined && Math.abs(i - spikeAt) < 3) v *= 2.5
    data.push(parseFloat(v.toFixed(1)))
  }
  return data
}

function generateLabels(count: number, unit: 'min' | 'hour' = 'min') {
  const labels: string[] = []
  const now = Date.now()
  for (let i = count - 1; i >= 0; i--) {
    const d = new Date(now - i * (unit === 'min' ? 60000 : 3600000))
    labels.push(d.toLocaleTimeString('zh', { hour: '2-digit', minute: '2-digit' }))
  }
  return labels
}

interface MetricCard {
  label: string
  value: string
  sub?: string
  color: string
  trend?: 'up' | 'down' | 'stable'
}

const MOCK_METRICS: MetricCard[] = [
  { label: 'RPM', value: '8,247', sub: 'P99: 12ms', color: 'var(--accent)', trend: 'up' },
  { label: 'TPM', value: '14.2M', sub: 'Input 9.1M / Output 5.1M', color: 'var(--success)', trend: 'up' },
  { label: '平均 TTFT', value: '142ms', sub: 'P99: 380ms', color: 'var(--warning)', trend: 'down' },
  { label: '错误率', value: '0.12%', sub: '429: 12 · 500: 3', color: 'var(--danger)', trend: 'stable' },
  { label: 'GPU 利用率', value: '73.4%', sub: 'H100: 82% · A100: 61%', color: 'var(--accent)', trend: 'stable' },
  { label: '活跃副本', value: '24 / 32', sub: '缩至零: 2', color: 'var(--success)', trend: 'stable' },
]

const MODEL_LIST = ['deepseek-ai/DeepSeek-V3', 'Qwen/Qwen3.5-72B', 'GLM/GLM-5', 'meta/Llama-4-70B', 'moonshot/Kimi-K2.5', '字节/豆包2.0']
const ENDPOINT_LIST = ['ep-ds-v3-prod', 'ep-qwen-chat', 'ep-glm-vision', 'ep-llama-fast', 'ep-kimi-agent']
const STATUS_CODES = [
  { code: '200', count: 487234, pct: 97.4 },
  { code: '206', count: 4521, pct: 0.9 },
  { code: '429', count: 3842, pct: 0.8 },
  { code: '500', count: 1203, pct: 0.2 },
  { code: '400', count: 3401, pct: 0.7 },
]

export default function ObservabilityPage() {
  const [timeRange, setTimeRange] = useState<'15m' | '1h' | '6h' | '24h'>('1h')
  const [activeTab, setActiveTab] = useState<'latency' | 'throughput' | 'errors'>('latency')
  const [modelFilter, setModelFilter] = useState('all')

  const pointCount = timeRange === '15m' ? 15 : timeRange === '1h' ? 60 : timeRange === '6h' ? 72 : 48
  const labelUnit: 'min' | 'hour' = timeRange === '24h' ? 'hour' : 'min'
  const labels = useMemo(() => generateLabels(pointCount, labelUnit), [pointCount, labelUnit])

  const latencyP50 = useMemo(() => generateTimeSeries(pointCount, 80, 160), [pointCount])
  const latencyP90 = useMemo(() => generateTimeSeries(pointCount, 180, 320, 45), [pointCount])
  const latencyP99 = useMemo(() => generateTimeSeries(pointCount, 250, 500, 45), [pointCount])

  const rpmSeries = useMemo(() => generateTimeSeries(pointCount, 5000, 12000, 45), [pointCount])
  const tpmSeries = useMemo(() => generateTimeSeries(pointCount, 8, 20), [pointCount])

  const errorRate = useMemo(() => generateTimeSeries(pointCount, 0.05, 0.3, 45), [pointCount])

  return (
    <div>
      {/* ── Header ─────────────────────────────────────────────────── */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h2 className="text-xl font-semibold text-[var(--text)]">可观测性仪表盘</h2>
          <p className="text-sm text-[var(--text-muted)] mt-1">实时监控推理延迟、吞吐量、错误率和 GPU 利用率</p>
        </div>
        <div className="flex items-center gap-2">
          <span className="inline-block w-2 h-2 rounded-full bg-[var(--success)] animate-pulse" />
          <span className="text-xs text-[var(--text-muted)]">Live</span>
          <select
            value={timeRange}
            onChange={e => setTimeRange(e.target.value as typeof timeRange)}
            className="text-xs px-2 py-1.5 rounded-md border border-[var(--border)] bg-white"
          >
            <option value="15m">最近 15 分钟</option>
            <option value="1h">最近 1 小时</option>
            <option value="6h">最近 6 小时</option>
            <option value="24h">最近 24 小时</option>
          </select>
        </div>
      </div>

      {/* ── Model filter ───────────────────────────────────────────── */}
      <div className="flex items-center gap-2 mb-5 overflow-x-auto">
        <button
          onClick={() => setModelFilter('all')}
          className={`text-xs px-3 py-1.5 rounded-md whitespace-nowrap transition-colors ${
            modelFilter === 'all' ? 'bg-[var(--accent)] text-white' : 'bg-white border border-[var(--border)] text-[var(--text-secondary)] hover:border-[var(--accent)]'
          }`}
        >全部</button>
        {MODEL_LIST.map(m => (
          <button
            key={m}
            onClick={() => setModelFilter(m)}
            className={`text-xs px-3 py-1.5 rounded-md whitespace-nowrap transition-colors ${
              modelFilter === m ? 'bg-[var(--accent)] text-white' : 'bg-white border border-[var(--border)] text-[var(--text-secondary)] hover:border-[var(--accent)]'
            }`}
          >{m.split('/').pop()}</button>
        ))}
      </div>

      {/* ── 6 metric cards ─────────────────────────────────────────── */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3 mb-5">
        {MOCK_METRICS.map((m) => (
          <div key={m.label} className="card py-4">
            <div className="text-[11px] font-medium text-[var(--text-muted)] uppercase tracking-wider mb-1">{m.label}</div>
            <div className="text-xl font-semibold" style={{ color: m.color }}>{m.value}</div>
            <div className="text-[11px] text-[var(--text-muted)] mt-0.5">{m.sub}</div>
          </div>
        ))}
      </div>

      {/* ── Tab selector ───────────────────────────────────────────── */}
      <div className="flex gap-1 mb-4 bg-white border border-[var(--border)] rounded-lg p-1 w-fit">
        {([
          ['latency', '延迟 (TTFT/TPS)'],
          ['throughput', '吞吐量 (RPM/TPM)'],
          ['errors', '错误率'],
        ] as const).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setActiveTab(key)}
            className={`text-xs px-4 py-1.5 rounded-md font-medium transition-colors ${
              activeTab === key ? 'bg-[var(--accent)] text-white' : 'text-[var(--text-muted)] hover:text-[var(--text)]'
            }`}
          >{label}</button>
        ))}
      </div>

      {/* ── Main chart ─────────────────────────────────────────────── */}
      <div className="card mb-5">
        {activeTab === 'latency' && (
          <>
            <h3 className="font-medium text-sm text-[var(--text)] mb-4">推理延迟 (P50 / P90 / P99)</h3>
            <LineChart
              labels={labels}
              datasets={[
                { label: 'P50', data: latencyP50, color: 'var(--success)', width: 1.5 },
                { label: 'P90', data: latencyP90, color: 'var(--warning)', width: 1.5 },
                { label: 'P99', data: latencyP99, color: 'var(--danger)', width: 1.5 },
              ]}
              yAxisLabel="ms"
              height={200}
            />
          </>
        )}
        {activeTab === 'throughput' && (
          <>
            <h3 className="font-medium text-sm text-[var(--text)] mb-4">请求吞吐 (RPM / TPM)</h3>
            <LineChart
              labels={labels}
              datasets={[
                { label: 'RPM', data: rpmSeries, color: 'var(--accent)', width: 2 },
              ]}
              yAxisLabel="req/min"
              height={200}
            />
            <div className="mt-4 pt-4 border-t border-[var(--border-light)]">
              <h4 className="text-xs font-medium text-[var(--text-muted)] mb-2">TPM (百万 Token/分)</h4>
              <LineChart
                labels={labels}
                datasets={[
                  { label: 'TPM', data: tpmSeries, color: 'var(--success)', width: 2 },
                ]}
                yAxisLabel="M tok/min"
                height={120}
              />
            </div>
          </>
        )}
        {activeTab === 'errors' && (
          <>
            <h3 className="font-medium text-sm text-[var(--text)] mb-4">错误率 (%)</h3>
            <LineChart
              labels={labels}
              datasets={[
                { label: 'Error Rate %', data: errorRate, color: 'var(--danger)', width: 2 },
              ]}
              yAxisLabel="%"
              height={200}
            />
            <div className="mt-4 pt-4 border-t border-[var(--border-light)]">
              <h4 className="text-xs font-medium text-[var(--text-muted)] mb-3">状态码分布</h4>
              <div className="space-y-2">
                {STATUS_CODES.map((s) => (
                  <div key={s.code} className="flex items-center gap-3 text-xs">
                    <span className={`w-12 font-mono font-medium ${
                      s.code.startsWith('2') ? 'text-[var(--success)]' : s.code === '429' ? 'text-[var(--warning)]' : 'text-[var(--danger)]'
                    }`}>{s.code}</span>
                    <div className="flex-1 bg-[var(--bg-secondary)] rounded h-3 overflow-hidden">
                      <div className="h-full rounded transition-all" style={{
                        width: `${s.pct * 10}%`,
                        minWidth: '2px',
                        background: s.code.startsWith('2') ? 'var(--success)' : s.code === '429' ? 'var(--warning)' : 'var(--danger)',
                      }} />
                    </div>
                    <span className="w-16 text-right text-[var(--text-muted)]">{s.count.toLocaleString()}</span>
                    <span className="w-12 text-right text-[var(--text-muted)]">{s.pct}%</span>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}
      </div>

      {/* ── GPU Utilization + Endpoint Status ──────────────────────── */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-5">
        <div className="card">
          <h3 className="font-medium text-sm text-[var(--text)] mb-3">GPU 节点利用率</h3>
          <div className="space-y-2.5">
            {[
              { name: 'gpu-node-h100-01', type: 'H100 80GB', util: 82, mem: '71.2/80 GB' },
              { name: 'gpu-node-h100-02', type: 'H100 80GB', util: 78, mem: '68.4/80 GB' },
              { name: 'gpu-node-a100-01', type: 'A100 80GB', util: 65, mem: '52.1/80 GB' },
              { name: 'gpu-node-a100-02', type: 'A100 40GB', util: 43, mem: '28.6/40 GB' },
              { name: 'gpu-node-l40s-01', type: 'L40S 48GB', util: 91, mem: '44.3/48 GB' },
            ].map((node) => (
              <div key={node.name} className="flex items-center gap-3">
                <div className="w-36 text-xs font-mono text-[var(--text-secondary)] truncate">{node.name}</div>
                <span className="tag text-[10px]" style={{
                  background: node.util > 85 ? 'var(--warning-bg)' : node.util > 60 ? 'var(--success-bg)' : 'var(--bg-secondary)',
                  color: node.util > 85 ? 'var(--warning)' : node.util > 60 ? 'var(--success)' : 'var(--text-muted)',
                }}>{node.type}</span>
                <div className="flex-1 bg-[var(--bg-secondary)] rounded h-3 overflow-hidden">
                  <div className="h-full rounded transition-all" style={{
                    width: `${node.util}%`,
                    background: node.util > 85 ? 'var(--warning)' : node.util > 60 ? 'var(--success)' : 'var(--accent)',
                  }} />
                </div>
                <span className="text-xs font-medium w-10 text-right">{node.util}%</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <h3 className="font-medium text-sm text-[var(--text)] mb-3">端点状态</h3>
          <div className="space-y-2">
            {ENDPOINT_LIST.map((ep, i) => {
              const running = i < 3
              const replicas = running ? `${Math.floor(Math.random() * 3) + 1}/${Math.floor(Math.random() * 2) + 2}` : '0/2'
              return (
                <div key={ep} className="flex items-center justify-between text-xs py-1.5 border-b border-[var(--border-light)] last:border-0">
                  <div className="flex items-center gap-2">
                    <span className={`inline-block w-2 h-2 rounded-full ${running ? 'bg-[var(--success)]' : 'bg-[var(--text-muted)]'}`} />
                    <span className="font-mono text-[var(--text)]">{ep}</span>
                  </div>
                  <span className={`tag ${running ? 'tag-green' : 'tag-purple'}`}>
                    {running ? `running ${replicas}` : 'scaled-to-0'}
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      </div>

      {/* ── Footer ─────────────────────────────────────────────────── */}
      <div className="flex items-center justify-between text-xs text-[var(--text-muted)]">
        <span>Prometheus · Grafana · OpenTelemetry</span>
        <span>数据刷新: 15s · 上次更新: {new Date().toLocaleTimeString('zh')}</span>
      </div>
    </div>
  )
}
