'use client'

import type { FlowItem } from '../types'

interface Props {
  items: FlowItem[]
}

export function ConfigFlow({ items }: Props) {
  const completed = items.filter(i => i.done).length
  const currentIdx = Math.max(items.findIndex(i => !i.done), 0)

  return (
    <div className="card mb-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-medium text-sm text-[var(--text)]">配置流程</h3>
        <span className="text-[11px] text-[var(--text-muted)]">
          已完成 <span className="text-[var(--text)] font-medium">{completed}</span> / {items.length}
        </span>
      </div>
      <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }}>
        {items.map((item, idx) => {
          const isCurrent = idx === currentIdx
          return (
            <div
              key={item.step}
              className={`rounded-lg p-3 border transition-colors ${
                item.done
                  ? 'border-emerald-200 bg-emerald-50/40'
                  : isCurrent
                    ? 'border-[var(--primary)]/30 bg-[var(--primary)]/5'
                    : 'border-[var(--border-light)] bg-white'
              }`}
            >
              <div className="flex items-center gap-2 mb-1.5">
                <span className={`w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-medium ${
                  item.done ? 'bg-emerald-500 text-white' : isCurrent ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)]'
                }`}>
                  {item.done ? '✓' : item.step}
                </span>
                <p className="text-xs font-medium text-[var(--text)]">{item.title}</p>
              </div>
              <p className="text-[11px] text-[var(--text-muted)] leading-relaxed">{item.desc}</p>
            </div>
          )
        })}
      </div>
    </div>
  )
}
