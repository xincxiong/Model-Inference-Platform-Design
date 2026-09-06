'use client'

import { useState, useMemo } from 'react'
import { BASE_MODEL_OPTIONS } from '../constants'
import type { Dataset } from '../types'
import { DatasetPicker } from './dataset-picker'
import { ComputePoolSelector } from './compute-pool-selector'

interface Props {
  datasets: Dataset[]
  submitting: boolean
  poolId: string
  setPoolId: (v: string) => void
  gpuRequest: number
  setGpuRequest: (n: number) => void
  /**
   * Called on submit. Parent does the actual API call.
   * Receives the full payload shape FineTuningJobCreateRequest expects.
   */
  onSubmit: (payload: {
    model: string
    training_file: string
    validation_file?: string
    method: 'lora' | 'qlora' | 'full'
    hyperparameters: { epochs: number; learning_rate: string; batch_size: number }
    pool_id: string
    gpu_request: number
  }) => void
}

const QWEN_OPTIONS = BASE_MODEL_OPTIONS.filter(o => o.family === 'Qwen')

export function SFTForm({ datasets, submitting, poolId, setPoolId, gpuRequest, setGpuRequest, onSubmit }: Props) {
  const [baseModel, setBaseModel] = useState('deepseek-ai/DeepSeek-V3')
  const [trainingFile, setTrainingFile] = useState('file-demo-001')
  const [validationFile, setValidationFile] = useState('')
  const [sftMethod, setSftMethod] = useState<'lora' | 'qlora' | 'full'>('lora')
  const [epochs, setEpochs] = useState('3')
  const [lr, setLr] = useState('2e-4')
  const [batchSize, setBatchSize] = useState('8')

  const ready = useMemo(
    () => baseModel.trim().length > 0 && trainingFile.trim().length > 0 && poolId.length > 0 && gpuRequest > 0,
    [baseModel, trainingFile, poolId, gpuRequest],
  )

  return (
    <div className="space-y-4">
      {/* 基座模型 */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">基座模型</h3>
        <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">模型 ID</label>
        <input
          list="sft-base-model-options"
          value={baseModel}
          onChange={e => setBaseModel(e.target.value)}
          placeholder="例如：deepseek-ai/DeepSeek-V3 或 Qwen/Qwen3-72B"
        />
        <datalist id="sft-base-model-options">
          {BASE_MODEL_OPTIONS.map(o => (
            <option key={o.id} value={o.id}>{o.label}</option>
          ))}
        </datalist>
        <p className="text-[11px] text-[var(--text-muted)] mt-1.5">
          支持 HuggingFace 格式 ID 或平台内部已注册模型名称，也可直接选择下方推荐模型。
        </p>
        <div className="mt-4">
          <div className="flex items-center justify-between gap-2 mb-2">
            <p className="text-[11px] font-medium text-[var(--text-secondary)]">推荐可用模型</p>
            <span className="text-[10px] text-[var(--text-muted)]">已新增 Qwen 系列</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {BASE_MODEL_OPTIONS.map(o => {
              const active = baseModel === o.id
              return (
                <button
                  key={o.id}
                  type="button"
                  onClick={() => setBaseModel(o.id)}
                  className={`rounded-lg border px-3 py-2 text-left transition-all ${
                    active
                      ? 'border-[color:var(--primary)]/40 bg-[color:var(--primary)]/10 shadow-sm'
                      : 'border-[var(--border)] bg-white hover:border-[var(--primary)]/25 hover:bg-[var(--bg-hover)]'
                  }`}
                  title={o.desc}
                >
                  <div className="flex items-center gap-2">
                    <span className="rounded-full px-1.5 py-0.5 text-[10px] font-medium bg-[var(--bg-hover)] text-[var(--text-secondary)]">{o.family}</span>
                    <span className="text-xs">{o.label}</span>
                  </div>
                </button>
              )
            })}
          </div>
        </div>
      </div>

      {/* 数据集 */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="font-medium text-sm text-[var(--text)]">数据集</h3>
            <p className="text-xs text-[var(--text-muted)] mt-0.5">从数据集管理中选择，或手动填写文件 ID</p>
          </div>
          <span className="text-[11px] text-[var(--text-muted)] bg-[var(--bg-hover)] px-2 py-1 rounded">JSONL / parquet</span>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <DatasetPicker
            label="训练集"
            required
            datasets={datasets}
            value={trainingFile}
            onChange={setTrainingFile}
          />
          <DatasetPicker
            label="验证集"
            datasets={datasets}
            value={validationFile}
            onChange={setValidationFile}
            placeholder="file-xxxx 或 data/val.jsonl"
          />
        </div>
      </div>

      {/* 方法 + 超参 */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">微调方法</h3>
        <div className="grid gap-3 md:grid-cols-3 mb-4">
          {(['lora', 'qlora', 'full'] as const).map(m => (
            <button
              key={m}
              type="button"
              onClick={() => setSftMethod(m)}
              className={`p-3 rounded-lg border text-left transition-colors ${
                sftMethod === m
                  ? 'border-[var(--primary)] bg-[var(--primary)]/5'
                  : 'border-[var(--border-light)] bg-white hover:bg-[var(--bg-hover)]'
              }`}
            >
              <p className="text-sm font-medium text-[var(--text)]">{m.toUpperCase()}</p>
              <p className="text-[11px] text-[var(--text-muted)] mt-0.5">
                {m === 'lora' && '低秩适配器，仅训练少量参数'}
                {m === 'qlora' && '4-bit 量化 + LoRA，显存受限首选'}
                {m === 'full' && '全参数训练，效果最佳'}
              </p>
            </button>
          ))}
        </div>
        <p className="text-xs font-medium text-[var(--text-secondary)] mb-2">超参数</p>
        <div className="grid gap-4 md:grid-cols-3">
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Epochs</label>
            <input value={epochs} onChange={e => setEpochs(e.target.value)} placeholder="3" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Learning Rate</label>
            <input value={lr} onChange={e => setLr(e.target.value)} placeholder="2e-4" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Batch Size</label>
            <input value={batchSize} onChange={e => setBatchSize(e.target.value)} placeholder="8" />
          </div>
        </div>
      </div>

      {/* 算力资源（核心新增） */}
      <ComputePoolSelector
        value={poolId}
        onChange={setPoolId}
        gpuRequest={gpuRequest}
        onGpuRequestChange={setGpuRequest}
      />

      <button
        type="button"
        disabled={!ready || submitting}
        onClick={() => onSubmit({
          model: baseModel,
          training_file: trainingFile,
          validation_file: validationFile || undefined,
          method: sftMethod,
          hyperparameters: {
            epochs: Number(epochs) || 1,
            learning_rate: lr,
            batch_size: Number(batchSize) || 1,
          },
          pool_id: poolId,
          gpu_request: gpuRequest,
        })}
        className="btn-primary w-full py-2.5 disabled:opacity-50"
      >
        {submitting ? '提交中…' : '提交 SFT 微调任务'}
      </button>
    </div>
  )
}
