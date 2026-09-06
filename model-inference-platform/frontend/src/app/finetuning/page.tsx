'use client'

import { useEffect, useState, useMemo } from 'react'
import {
  listFineTuningJobs,
  createFineTuningJob,
  cancelFineTuningJob,
  listDatasets,
} from '@/lib/api'
import type { Dataset, FineTuningJob, FlowItem } from './types'
import { ConfigFlow } from './components/config-flow'
import { JobsList } from './components/jobs-list'
import { SFTForm } from './components/sft-form'
import { RLForm } from './components/rl-form'

export default function FinetuningPage() {
  // Mode + shared compute-pool state (used by both SFT and RL forms)
  const [mode, setMode] = useState<'sft' | 'rl'>('sft')
  const [poolId, setPoolId] = useState('')
  const [gpuRequest, setGpuRequest] = useState(4)
  const [rolloutPoolId, setRolloutPoolId] = useState('')
  const [rolloutGpuRequest, setRolloutGpuRequest] = useState(4)

  // Shared lists
  const [jobs, setJobs] = useState<FineTuningJob[]>([])
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const load = async () => {
    try {
      const [j, d] = await Promise.all([
        listFineTuningJobs(),
        listDatasets(),
      ])
      const jobList = ((j as { data?: FineTuningJob[] }).data) || []
      const dsList = ((d as { data?: Dataset[] }).data) || []
      setJobs(jobList)
      setDatasets(dsList)
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [])

  const handleCreate = async (payload: Record<string, unknown>) => {
    setError('')
    setSubmitting(true)
    try {
      await createFineTuningJob(payload)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleCancel = async (id: string) => {
    if (!confirm('确认取消此任务？')) return
    try {
      await cancelFineTuningJob(id)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : '取消失败')
    }
  }

  const flowItems: FlowItem[] = useMemo(() => mode === 'sft'
    ? [
        { step: '1', title: '基座模型', desc: '选择要微调的基础模型', done: true },
        { step: '2', title: '训练数据', desc: '选择训练/验证数据集', done: datasets.length > 0 },
        { step: '3', title: '微调方法', desc: 'LoRA / QLoRA / Full FT', done: true },
        { step: '4', title: '算力资源', desc: '选择算力池并申请 GPU', done: poolId.length > 0 && gpuRequest > 0 },
        { step: '5', title: '提交', desc: '检查配置后提交微调任务', done: false },
      ]
    : [
        { step: '1', title: '基座 + RL 算法', desc: '选择模型与 GRPO/DPO 等', done: true },
        { step: '2', title: '数据集', desc: 'Prompt 配对偏好 / 验证', done: datasets.length > 0 },
        { step: '3', title: 'Training Engine', desc: 'Megatron-LM 并行拓扑 + 步数', done: true },
        { step: '4', title: 'Rollout + Reward', desc: '场景 + 内置/外置/代码奖励', done: true },
        { step: '5', title: '算力资源', desc: '训练池 + 可选 Rollout 池', done: poolId.length > 0 && gpuRequest > 0 },
        { step: '6', title: '提交', desc: '检查后提交 RL 训练任务', done: false },
      ], [mode, datasets.length, poolId, gpuRequest])

  if (loading && jobs.length === 0) {
    return (
      <div className="flex items-center gap-2 text-[var(--text-muted)] p-8">
        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 12h4z"/>
        </svg>
        加载中…
      </div>
    )
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--text)]">模型微调</h2>
        <p className="text-sm text-[var(--text-muted)] mt-1">
          监督微调（SFT）与强化学习后训练（Megatron-LM × SGLang）
        </p>
      </div>

      {error && (
        <div className="text-[var(--danger)] bg-[var(--danger-bg)] text-sm px-4 py-3 rounded-lg mb-4">{error}</div>
      )}

      {/* Mode tabs */}
      <div className="flex gap-1 p-1 bg-[var(--bg-hover)] rounded-lg w-fit mb-4">
        {([
          { id: 'sft' as const, label: '监督微调 (SFT)', sub: 'LoRA · QLoRA · Full FT' },
          { id: 'rl' as const,  label: '强化学习后训练 (RL)', sub: 'GRPO · GSPO · DAPO · VAPO · PPO · DPO' },
        ]).map(tab => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setMode(tab.id)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-all text-left ${
              mode === tab.id
                ? 'bg-[var(--bg)] text-[var(--text)] shadow-sm'
                : 'text-[var(--text-muted)] hover:text-[var(--text)]'
            }`}
          >
            <div>{tab.label}</div>
            <div className={`text-[10px] mt-0.5 ${mode === tab.id ? 'text-[var(--text-secondary)]' : 'text-[var(--text-muted)]'}`}>
              {tab.sub}
            </div>
          </button>
        ))}
      </div>

      {/* Config flow overview */}
      <ConfigFlow items={flowItems} />

      {/* Active form */}
      {mode === 'sft' ? (
        <SFTForm
          datasets={datasets}
          submitting={submitting}
          poolId={poolId}
          setPoolId={setPoolId}
          gpuRequest={gpuRequest}
          setGpuRequest={setGpuRequest}
          onSubmit={handleCreate}
        />
      ) : (
        <RLForm
          datasets={datasets}
          submitting={submitting}
          poolId={poolId}
          setPoolId={setPoolId}
          gpuRequest={gpuRequest}
          setGpuRequest={setGpuRequest}
          rolloutPoolId={rolloutPoolId}
          setRolloutPoolId={setRolloutPoolId}
          rolloutGpuRequest={rolloutGpuRequest}
          setRolloutGpuRequest={setRolloutGpuRequest}
          onSubmit={handleCreate}
        />
      )}

      {/* Jobs list */}
      <div className="mt-6">
        <JobsList jobs={jobs} onRefresh={load} onCancel={handleCancel} />
      </div>
    </div>
  )
}
