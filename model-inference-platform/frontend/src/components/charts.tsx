'use client'

// ── Lightweight SVG Line Chart (no external deps) ─────────────────────────────
export function LineChart({
  labels,
  datasets,
  yAxisLabel,
  height = 180,
}: {
  labels: string[]
  datasets: { label: string; data: number[]; color: string; width?: number }[]
  yAxisLabel?: string
  height?: number
}) {
  const allValues = datasets.flatMap(d => d.data)
  const minV = Math.min(...allValues)
  const maxV = Math.max(...allValues)
  const range = maxV - minV || 1
  const pad = 40
  const w = 800
  const h = height + 30
  const chartH = height - 20

  const toY = (v: number) => 10 + chartH - ((v - minV) / range) * chartH
  const toX = (i: number) => pad + (i / (labels.length - 1)) * (w - pad - 10)

  // grid lines
  const gridSteps = 4
  const gridLines = Array.from({ length: gridSteps + 1 }, (_, i) => {
    const v = minV + (range * i) / gridSteps
    const y = toY(v)
    return { y, label: v.toFixed(v > 100 ? 0 : 1) }
  })

  return (
    <div className="overflow-x-auto">
      <svg viewBox={`0 0 ${w} ${h}`} className="w-full min-w-[500px]" style={{ maxHeight: `${h}px` }}>
        {/* grid */}
        {gridLines.map((g, i) => (
          <g key={i}>
            <line x1={pad} y1={g.y} x2={w - 10} y2={g.y} stroke="var(--border-light)" strokeWidth="0.5" />
            <text x={pad - 4} y={g.y + 4} textAnchor="end" fill="var(--text-muted)" fontSize="10" fontFamily="monospace">{g.label}</text>
          </g>
        ))}

        {/* x labels */}
        {labels.filter((_, i) => i % Math.ceil(labels.length / 6) === 0).map((l, i, a) => {
          const origIdx = labels.indexOf(l)
          return <text key={i} x={toX(origIdx)} y={h - 2} textAnchor="middle" fill="var(--text-muted)" fontSize="9" fontFamily="monospace">{l}</text>
        })}

        {/* lines */}
        {datasets.map((ds, di) => {
          const pathD = ds.data.map((v, i) => `${i === 0 ? 'M' : 'L'} ${toX(i)} ${toY(v)}`).join(' ')
          const areaD = pathD + ` L ${toX(ds.data.length - 1)} ${toY(minV)} L ${toX(0)} ${toY(minV)} Z`
          return (
            <g key={di}>
              <defs>
                <linearGradient id={`grad-${di}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={ds.color} stopOpacity="0.12" />
                  <stop offset="100%" stopColor={ds.color} stopOpacity="0" />
                </linearGradient>
              </defs>
              <path d={areaD} fill={`url(#grad-${di})`} />
              <path d={pathD} fill="none" stroke={ds.color} strokeWidth={ds.width || 1.5} strokeLinejoin="round" />
            </g>
          )
        })}

        {/* legend */}
        {datasets.map((ds, i) => {
          const lx = pad + i * 120
          return (
            <g key={i}>
              <line x1={lx} y1={2} x2={lx + 16} y2={2} stroke={ds.color} strokeWidth={ds.width || 1.5} />
              <text x={lx + 20} y={6} fill="var(--text-secondary)" fontSize="10">{ds.label}</text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

// ── Lightweight SVG Bar Chart ─────────────────────────────────────────────────
export function BarChart({
  labels,
  data,
  color = 'var(--accent)',
  height = 160,
}: {
  labels: string[]
  data: number[]
  color?: string
  height?: number
}) {
  const pad = 40
  const w = 800
  const h = height + 30
  const chartH = height - 20
  const maxV = Math.max(...data, 0.001)
  const barW = Math.max((w - pad - 10) / data.length - 2, 3)

  const gridSteps = 4
  const gridLines = Array.from({ length: gridSteps + 1 }, (_, i) => {
    const v = (maxV * i) / gridSteps
    const y = 10 + chartH - (v / maxV) * chartH
    return { y, label: v.toFixed(v > 100 ? 0 : 1) }
  })

  return (
    <div className="overflow-x-auto">
      <svg viewBox={`0 0 ${w} ${h}`} className="w-full min-w-[500px]" style={{ maxHeight: `${h}px` }}>
        {gridLines.map((g, i) => (
          <g key={i}>
            <line x1={pad} y1={g.y} x2={w - 10} y2={g.y} stroke="var(--border-light)" strokeWidth="0.5" />
            <text x={pad - 4} y={g.y + 4} textAnchor="end" fill="var(--text-muted)" fontSize="10" fontFamily="monospace">{g.label}</text>
          </g>
        ))}
        {data.map((v, i) => {
          const x = pad + (i / data.length) * (w - pad - 10)
          const barH = (v / maxV) * chartH
          const y = 10 + chartH - barH
          return (
            <g key={i}>
              <rect x={x} y={y} width={barW} height={barH} rx="1" fill={color} opacity="0.8" />
              {labels.length <= 30 && i % Math.ceil(labels.length / 8) === 0 && (
                <text x={x + barW / 2} y={h - 2} textAnchor="middle" fill="var(--text-muted)" fontSize="8" fontFamily="monospace">{labels[i]}</text>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}
