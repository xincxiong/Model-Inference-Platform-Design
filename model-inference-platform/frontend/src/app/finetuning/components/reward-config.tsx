'use client'

import { useState } from 'react'
import { REWARD_FUNCTIONS } from '../constants'

interface Props {
  rewardFns: string[]
  setRewardFns: (v: string[]) => void
  rewardWeights: Record<string, string>
  setRewardWeights: (w: Record<string, string>) => void
  rewardEndpoint: string
  setRewardEndpoint: (v: string) => void
  rewardCode: string
  setRewardCode: (v: string) => void
  klCoeff: string
  setKlCoeff: (v: string) => void
  clipRange: string
  setClipRange: (v: string) => void
  advNorm: boolean
  setAdvNorm: (v: boolean) => void
  /** RL method — controls whether KL / clip ranges are shown. */
  method: string
}

export function RewardConfig(p: Props) {
  const [tab, setTab] = useState<'builtin' | 'endpoint' | 'code'>('builtin')

  const toggle = (v: string) => {
    p.setRewardFns(p.rewardFns.includes(v) ? p.rewardFns.filter(x => x !== v) : [...p.rewardFns, v])
  }

  const showKlpClip = ['ppo', 'grpo', 'gspo', 'dapo', 'vapo'].includes(p.method)

  return (
    <div className="card">
      <h3 className="font-medium text-sm text-[var(--text)] mb-4">Reward 配置</h3>

      {/* Tab switcher */}
      <div className="flex gap-1 p-1 bg-[var(--bg-hover)] rounded-lg w-fit mb-4">
        {(['builtin', 'endpoint', 'code'] as const).map(t => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={`px-3 py-1 rounded text-[11px] font-medium transition-colors ${
              tab === t ? 'bg-white text-[var(--text)] shadow-sm' : 'text-[var(--text-muted)] hover:text-[var(--text)]'
            }`}
          >
            {t === 'builtin' ? '内置函数' : t === 'endpoint' ? '外部服务' : '代码注入'}
          </button>
        ))}
      </div>

      {/* Tab: 内置 */}
      {tab === 'builtin' && (
        <div className="space-y-2">
          {REWARD_FUNCTIONS.filter(rf => rf.value !== 'custom_model').map(rf => {
            const active = p.rewardFns.includes(rf.value)
            return (
              <label key={rf.value} className="flex items-start gap-3 p-2.5 rounded-lg hover:bg-[var(--bg-hover)] cursor-pointer">
                <input
                  type="checkbox"
                  checked={active}
                  onChange={() => toggle(rf.value)}
                  className="mt-0.5"
                />
                <div className="flex-1">
                  <p className="text-sm text-[var(--text)]">{rf.label}</p>
                  <p className="text-[11px] text-[var(--text-muted)]">{rf.desc}</p>
                </div>
                {active && (
                  <div className="flex items-center gap-1">
                    <span className="text-[10px] text-[var(--text-muted)]">权重</span>
                    <input
                      type="number"
                      step="0.1"
                      min="0"
                      className="w-16 text-xs"
                      value={p.rewardWeights[rf.value] || '1.0'}
                      onChange={e => p.setRewardWeights({ ...p.rewardWeights, [rf.value]: e.target.value })}
                    />
                  </div>
                )}
              </label>
            )
          })}
        </div>
      )}

      {/* Tab: 外部服务 */}
      {tab === 'endpoint' && (
        <div>
          <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
            Reward Model HTTP Endpoint
          </label>
          <input
            value={p.rewardEndpoint}
            onChange={e => p.setRewardEndpoint(e.target.value)}
            placeholder="http://reward-model-service:8080/score"
          />
          <p className="text-[11px] text-[var(--text-muted)] mt-1.5">
            平台将以 POST 方式发送 {`{prompt, response, ground_truth}`}，期望返回 {`{score: float}`}
          </p>
        </div>
      )}

      {/* Tab: 代码注入 */}
      {tab === 'code' && (
        <div>
          <p className="text-[11px] text-[var(--text-muted)] mb-2">
            自定义 reward 函数 Python 代码。每条样本调用一次，返回 float 分数。
          </p>
          <textarea
            value={p.rewardCode}
            onChange={e => p.setRewardCode(e.target.value)}
            rows={10}
            className="font-mono text-xs"
          />
        </div>
      )}

      {/* KL / clip 超参 */}
      {showKlpClip && (
        <div className="mt-4 pt-4 border-t border-[var(--border-light)] space-y-3">
          <p className="text-xs font-medium text-[var(--text-secondary)]">{p.method.toUpperCase()} 策略优化超参数</p>
          <div className="grid gap-3 md:grid-cols-3">
            <div>
              <label className="text-xs text-[var(--text-muted)] block mb-1">KL 惩罚系数</label>
              <input value={p.klCoeff} onChange={e => p.setKlCoeff(e.target.value)} />
            </div>
            <div>
              <label className="text-xs text-[var(--text-muted)] block mb-1">Clip Range</label>
              <input value={p.clipRange} onChange={e => p.setClipRange(e.target.value)} />
            </div>
            <div className="flex items-end">
              <label className="flex items-center gap-2 text-xs">
                <input
                  type="checkbox"
                  checked={p.advNorm}
                  onChange={e => p.setAdvNorm(e.target.checked)}
                />
                优势归一化
              </label>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

