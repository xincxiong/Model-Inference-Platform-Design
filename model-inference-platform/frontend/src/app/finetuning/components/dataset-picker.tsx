'use client'

import { useState } from 'react'
import type { Dataset } from '../types'

interface Props {
  label: string
  required?: boolean
  datasets: Dataset[]
  value: string
  onChange: (v: string) => void
  placeholder?: string
  helperText?: string
}

/**
 * Dataset picker — replaces the 4× copy-pasted "pick vs manual" toggle
 * that was duplicated for SFT train / SFT val / RL train / RL val.
 */
export function DatasetPicker({
  label, required, datasets, value, onChange, placeholder, helperText,
}: Props) {
  const [mode, setMode] = useState<'pick' | 'manual'>('pick')
  const selected = value ? datasets.find(d => d.id === value) : null

  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <label className="text-xs font-medium text-[var(--text-secondary)]">
          {label}
          {required && <span className="text-[var(--danger)] ml-0.5">*</span>}
          {!required && <span className="text-[var(--text-muted)] ml-0.5">（可选）</span>}
        </label>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => setMode('pick')}
            className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${mode === 'pick' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
          >选择</button>
          <button
            type="button"
            onClick={() => setMode('manual')}
            className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${mode === 'manual' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
          >手动</button>
        </div>
      </div>

      {mode === 'pick' ? (
        <>
          <select value={value} onChange={e => onChange(e.target.value)}>
            <option value="">— {required ? '选择数据集' : '不使用'} —</option>
            {datasets.map(ds => (
              <option key={ds.id} value={ds.id}>
                {ds.name}{ds.num_rows > 0 ? ` (${ds.num_rows.toLocaleString()} 行)` : ''}
              </option>
            ))}
          </select>
          {selected && (
            <p className="text-[11px] text-[var(--primary)] mt-1 flex items-center gap-1">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
              {selected.description || selected.name} · {selected.num_rows > 0 ? `${selected.num_rows.toLocaleString()} 行` : '空'}
            </p>
          )}
          {datasets.length === 0 && (
            <p className="text-[11px] text-[var(--text-muted)] mt-1">
              暂无数据集，请先在<a href="/datasets" className="text-[var(--primary)] mx-0.5 hover:underline">数据集页面</a>创建
            </p>
          )}
        </>
      ) : (
        <>
          <input
            value={value}
            onChange={e => onChange(e.target.value)}
            placeholder={placeholder || 'file-xxxx 或 data/train.jsonl'}
          />
          <p className="text-[11px] text-[var(--text-muted)] mt-1">
            {helperText || '从「数据集」页面上传后获取文件 ID'}
          </p>
        </>
      )}
    </div>
  )
}
