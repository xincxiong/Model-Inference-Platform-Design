'use client'

import { useState, useMemo } from 'react'
import { BASE_MODEL_OPTIONS, ROLLOUT_SCENARIOS, DEFAULT_REWARD_CODE } from '../constants'
import type { Dataset, RLMethod } from '../types'
import { DatasetPicker } from './dataset-picker'
import { ComputePoolSelector } from './compute-pool-selector'
import { RewardConfig } from './reward-config'

interface Props {
  datasets: Dataset[]
  submitting: boolean
  poolId: string
  setPoolId: (v: string) => void
  gpuRequest: number
  setGpuRequest: (n: number) => void
  rolloutPoolId: string
  setRolloutPoolId: (v: string) => void
  rolloutGpuRequest: number
  setRolloutGpuRequest: (n: number) => void
  onSubmit: (payload: Record<string, unknown>) => void
}

export function RLForm({
  datasets, submitting,
  poolId, setPoolId, gpuRequest, setGpuRequest,
  rolloutPoolId, setRolloutPoolId, rolloutGpuRequest, setRolloutGpuRequest,
  onSubmit,
}: Props) {
  // Base
  const [rlModel, setRlModel] = useState('deepseek-ai/DeepSeek-V3')
  const [rlMethod, setRlMethod] = useState<RLMethod>('grpo')
  const [rolloutScenario, setRolloutScenario] = useState('chat')
  const [rlTrainingFile, setRlTrainingFile] = useState('')
  const [rlValidationFile, setRlValidationFile] = useState('')

  // Training Engine (Megatron-LM)
  const [tensorParallel, setTensorParallel] = useState('2')
  const [pipelineParallel, setPipelineParallel] = useState('1')
  const [dataParallel, setDataParallel] = useState('4')
  const [trainSteps, setTrainSteps] = useState('1000')
  const [saveSteps, setSaveSteps] = useState('100')

  // Rollout Engine (SGLang)
  const [rolloutSamples, setRolloutSamples] = useState('512')
  const [rolloutTurns, setRolloutTurns] = useState('1')
  const [rolloutTemp, setRolloutTemp] = useState('0.7')
  const [sgMemFraction, setSgMemFraction] = useState('0.8')
  const [asyncRollout, setAsyncRollout] = useState(true)
  const [customRolloutScript, setCustomRolloutScript] = useState('')

  // Data Buffer
  const [bufferSize, setBufferSize] = useState('2048')
  const [promptDataset, setPromptDataset] = useState('')

  // Reward
  const [rewardFns, setRewardFns] = useState<string[]>(['accuracy', 'format'])
  const [rewardWeights, setRewardWeights] = useState<Record<string, string>>({ accuracy: '1.0', format: '0.5' })
  const [rewardEndpoint, setRewardEndpoint] = useState('')
  const [rewardCode, setRewardCode] = useState(DEFAULT_REWARD_CODE)
  const [klCoeff, setKlCoeff] = useState('0.01')
  const [clipRange, setClipRange] = useState('0.2')
  const [advNorm, setAdvNorm] = useState(true)

  const parallelTopology = useMemo(() => ({
    tp: Number(tensorParallel) || 1,
    pp: Number(pipelineParallel) || 1,
    dp: Number(dataParallel) || 1,
  }), [tensorParallel, pipelineParallel, dataParallel])

  const ready = useMemo(
    () => rlModel.trim().length > 0 && poolId.length > 0 && gpuRequest > 0,
    [rlModel, poolId, gpuRequest],
  )

  const weightsMap: Record<string, number> = {}
  rewardFns.forEach(fn => { weightsMap[fn] = parseFloat(rewardWeights[fn] || '1') })

  return (
    <div className="space-y-4">
      {/* 基础配置 */}
      <div className="card">
        <div className="flex items-center gap-2 mb-4">
          <span className="text-xs font-bold text-[var(--primary)] bg-[color:var(--primary)]/10 px-2 py-0.5 rounded">RL</span>
          <h3 className="font-medium text-sm text-[var(--text)]">基础配置</h3>
        </div>
        <div className="grid gap-4 md:grid-cols-3">
          <div className="md:col-span-2">
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">基座 / 参考模型 ID</label>
            <input
              list="rl-base-model-options"
              value={rlModel}
              onChange={e => setRlModel(e.target.value)}
              placeholder="例如：deepseek-ai/DeepSeek-R1 或 Qwen/Qwen3-72B"
            />
            <datalist id="rl-base-model-options">
              {BASE_MODEL_OPTIONS.map(o => (
                <option key={o.id} value={o.id}>{o.label}</option>
              ))}
            </datalist>
            <div className="mt-2 flex flex-wrap gap-2">
              {BASE_MODEL_OPTIONS.filter(o => o.family === 'Qwen').map(o => {
                const active = rlModel === o.id
                return (
                  <button
                    key={o.id}
                    type="button"
                    onClick={() => setRlModel(o.id)}
                    className={`rounded-full border px-2.5 py-1 text-[11px] transition-colors ${
                      active
                        ? 'border-emerald-300 bg-emerald-50 text-emerald-700'
                        : 'border-[var(--border)] bg-white text-[var(--text-muted)] hover:border-emerald-200 hover:text-[var(--text)]'
                    }`}
                  >{o.label}</button>
                )
              })}
            </div>
          </div>
          <div>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">RL 算法</label>
            <select value={rlMethod} onChange={e => setRlMethod(e.target.value as RLMethod)}>
              <option value="grpo">GRPO — Group Relative Policy Optimization</option>
              <option value="gspo">GSPO — Group Sequence Policy Optimization</option>
              <option value="dapo">DAPO — Decoupled Advantage Policy Optimization</option>
              <option value="vapo">VAPO — Value-Augmented Policy Optimization</option>
              <option value="ppo">PPO — Proximal Policy Optimization</option>
              <option value="dpo">DPO — Direct Preference Optimization</option>
            </select>
            <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5 mt-3">Rollout 场景</label>
            <select value={rolloutScenario} onChange={e => setRolloutScenario(e.target.value)}>
              {ROLLOUT_SCENARIOS.map(s => (
                <option key={s.value} value={s.value}>{s.label} — {s.desc.slice(0, 30)}…</option>
              ))}
            </select>
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
            label="训练 / Prompt 数据集"
            required
            datasets={datasets}
            value={rlTrainingFile}
            onChange={setRlTrainingFile}
            helperText={rlMethod === 'dpo'
              ? '配对偏好数据：每行含 prompt / chosen / rejected'
              : 'Prompt 列表，用于 SGLang Rollout 采样生成轨迹'}
          />
          <DatasetPicker
            label="验证集"
            datasets={datasets}
            value={rlValidationFile}
            onChange={setRlValidationFile}
          />
        </div>
      </div>

      {/* Training Engine */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">Training Engine（Megatron-LM）</h3>
        <div className="grid gap-4 md:grid-cols-3">
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Tensor Parallel</label>
            <input value={tensorParallel} onChange={e => setTensorParallel(e.target.value)} placeholder="2" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Pipeline Parallel</label>
            <input value={pipelineParallel} onChange={e => setPipelineParallel(e.target.value)} placeholder="1" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Data Parallel</label>
            <input value={dataParallel} onChange={e => setDataParallel(e.target.value)} placeholder="4" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">训练步数</label>
            <input value={trainSteps} onChange={e => setTrainSteps(e.target.value)} placeholder="1000" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">保存步数</label>
            <input value={saveSteps} onChange={e => setSaveSteps(e.target.value)} placeholder="100" />
          </div>
        </div>
      </div>

      {/* Rollout Engine */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">Rollout Engine（SGLang）</h3>
        <div className="grid gap-4 md:grid-cols-3">
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">采样数 / Prompt</label>
            <input value={rolloutSamples} onChange={e => setRolloutSamples(e.target.value)} placeholder="512" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">对话轮数</label>
            <input value={rolloutTurns} onChange={e => setRolloutTurns(e.target.value)} placeholder="1" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Temperature</label>
            <input value={rolloutTemp} onChange={e => setRolloutTemp(e.target.value)} placeholder="0.7" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">SGLang Mem Fraction</label>
            <input value={sgMemFraction} onChange={e => setSgMemFraction(e.target.value)} placeholder="0.8" />
          </div>
          <div className="flex items-end">
            <label className="flex items-center gap-2 text-xs">
              <input type="checkbox" checked={asyncRollout} onChange={e => setAsyncRollout(e.target.checked)} />
              异步 Rollout（Data Buffer 解耦）
            </label>
          </div>
          <div className="md:col-span-3">
            <label className="text-xs text-[var(--text-muted)] block mb-1">自定义 Rollout 脚本（custom 场景）</label>
            <input
              value={customRolloutScript}
              onChange={e => setCustomRolloutScript(e.target.value)}
              placeholder="examples/custom_rollout.py 或 http://verifier-service:8080"
            />
          </div>
        </div>
      </div>

      {/* Data Buffer */}
      <div className="card">
        <h3 className="font-medium text-sm text-[var(--text)] mb-4">Data Buffer（异步解耦）</h3>
        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Buffer 大小</label>
            <input value={bufferSize} onChange={e => setBufferSize(e.target.value)} placeholder="2048" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)] block mb-1">Prompt 数据集（可选，默认用训练集）</label>
            <input
              value={promptDataset}
              onChange={e => setPromptDataset(e.target.value)}
              placeholder="data/prompts.jsonl 或 dataset ID"
            />
          </div>
        </div>
      </div>

      {/* Reward */}
      <RewardConfig
        rewardFns={rewardFns}
        setRewardFns={setRewardFns}
        rewardWeights={rewardWeights}
        setRewardWeights={setRewardWeights}
        rewardEndpoint={rewardEndpoint}
        setRewardEndpoint={setRewardEndpoint}
        rewardCode={rewardCode}
        setRewardCode={setRewardCode}
        klCoeff={klCoeff}
        setKlCoeff={setKlCoeff}
        clipRange={clipRange}
        setClipRange={setClipRange}
        advNorm={advNorm}
        setAdvNorm={setAdvNorm}
        method={rlMethod}
      />

      {/* 算力资源（核心新增，带 rollout 选项）*/}
      <ComputePoolSelector
        value={poolId}
        onChange={setPoolId}
        gpuRequest={gpuRequest}
        onGpuRequestChange={setGpuRequest}
        parallelTopology={parallelTopology}
        rollout={{
          value: rolloutPoolId,
          onChange: setRolloutPoolId,
          gpuRequest: rolloutGpuRequest,
          onGpuRequestChange: setRolloutGpuRequest,
        }}
      />

      <button
        type="button"
        disabled={!ready || submitting}
        onClick={() => onSubmit({
          model: rlModel,
          training_file: rlTrainingFile || undefined,
          validation_file: rlValidationFile || undefined,
          method: rlMethod,
          rollout_scenario: rolloutScenario,
          rollout_config: {
            num_turns: Number(rolloutTurns) || 1,
            num_samples: Number(rolloutSamples) || 1,
            temperature: parseFloat(rolloutTemp) || 0.7,
            async_rollout: asyncRollout,
            sglang_mem_fraction_static: parseFloat(sgMemFraction) || 0.8,
            custom_rollout_script: customRolloutScript || undefined,
          },
          engine_config: {
            training_engine: 'megatron',
            rollout_engine: 'sglang',
            tensor_model_parallel_size: Number(tensorParallel) || 1,
            pipeline_model_parallel_size: Number(pipelineParallel) || 1,
            data_parallel_size: Number(dataParallel) || 1,
            train_steps: Number(trainSteps) || 1,
            save_steps: Number(saveSteps) || 1,
          },
          data_buffer_config: {
            buffer_size: Number(bufferSize) || 1,
            prompt_dataset: promptDataset || rlTrainingFile || undefined,
          },
          reward_config: {
            reward_functions: rewardFns,
            weights: weightsMap,
            reward_model_endpoint: rewardEndpoint || undefined,
            reward_code: rewardCode,
            kl_coeff: parseFloat(klCoeff) || 0,
            clip_range: parseFloat(clipRange) || 0,
            advantage_norm: advNorm,
          },
          pool_id: poolId,
          gpu_request: gpuRequest,
          rollout_pool_id: rolloutPoolId || undefined,
        })}
        className="btn-primary w-full py-2.5 disabled:opacity-50"
      >
        {submitting ? '提交中…' : `提交 ${rlMethod.toUpperCase()} 训练任务`}
      </button>
    </div>
  )
}
