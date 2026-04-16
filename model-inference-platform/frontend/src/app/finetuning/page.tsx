'use client'

import { useEffect, useState } from 'react'
import { listFineTuningJobs, createFineTuningJob, cancelFineTuningJob, listDatasets } from '@/lib/api'

// ─── Types ────────────────────────────────────────────────────────────────────

interface FineTuningJob {
  id: string
  object: string
  model: string
  training_file?: string
  method: string
  status: string
  rollout_scenario?: string
  reward_config?: Record<string, unknown>
  fine_tuned_model?: string
  created_at: number
  updated_at: number
  error?: { message: string }
}

interface Dataset {
  id: string
  name: string
  description?: string
  num_rows: number
  size_bytes: number
}

type TrainingMode = 'sft' | 'rl'
type RLMethod = 'grpo' | 'gspo' | 'dapo' | 'vapo' | 'ppo' | 'dpo'

// ─── Constants ────────────────────────────────────────────────────────────────

const ROLLOUT_SCENARIOS = [
  {
    value: 'chat',
    label: '对话场景',
    desc: '通用多轮对话 RLHF，reward 基于偏好对比或评分模型',
    badge: 'RLHF · GRPO',
    icon: (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
    ),
  },
  {
    value: 'code',
    label: '代码生成',
    desc: '沙箱执行 reward，测试用例通过率 / 编译正确性驱动优化（参考 TritonForge）',
    badge: 'Sandbox · Exec',
    icon: (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
      </svg>
    ),
  },
  {
    value: 'tool_call',
    label: '工具调用 (MCP)',
    desc: 'Function Calling / MCP 工具路由，reward 基于工具选择精度与参数匹配',
    badge: 'MCP · FC',
    icon: (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>
      </svg>
    ),
  },
  {
    value: 'agent_single',
    label: 'Agent 单轮执行',
    desc: '单次决策输出 Action，reward 基于任务完成度（参考 P1 物理推理）',
    badge: 'Single-turn',
    icon: (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
      </svg>
    ),
  },
  {
    value: 'agent_multi',
    label: 'Agent 多轮执行',
    desc: '多步骤 Agentic loop，环境反馈驱动（参考 OpenClaw-RL / ArenaRL）',
    badge: 'Multi-turn · Env',
    icon: (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/>
      </svg>
    ),
  },
  {
    value: 'math_reasoning',
    label: '数学 / 推理',
    desc: '可验证答案的推理任务，binary reward，适合 RLVE / 自适应难度课程（参考 P1）',
    badge: 'Verifiable · RLVE',
    icon: (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/>
        <line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/>
      </svg>
    ),
  },
  {
    value: 'custom',
    label: '自定义场景',
    desc: '提供 SGLang rollout 脚本或外部 verifier 服务，完全可扩展',
    badge: 'SGLang · Custom',
    icon: (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="3"/><path d="M19.07 4.93a10 10 0 0 1 0 14.14M4.93 4.93a10 10 0 0 0 0 14.14"/>
      </svg>
    ),
  },
]

const REWARD_FUNCTIONS = [
  { value: 'accuracy',            label: '正确性 (Accuracy)',           desc: '与 ground-truth 对比，最常用的可验证 reward' },
  { value: 'format',              label: '格式合规 (Format)',            desc: '检验输出是否符合指定格式（JSON / Markdown / XML 等）' },
  { value: 'length_penalty',      label: '长度惩罚 (Length Penalty)',   desc: '惩罚过短 / 过长的冗余回复' },
  { value: 'tool_call_correctness', label: '工具调用精度 (Tool Accuracy)', desc: 'Function name + 参数匹配度，MCP 场景专用' },
  { value: 'code_execution',      label: '代码执行通过率 (Code Exec)',   desc: '沙箱执行并检查测试用例，支持编译反馈循环' },
  { value: 'task_completion',     label: '任务完成度 (Task Completion)', desc: 'Agent loop 最终目标达成率' },
  { value: 'hindsight_hints',     label: 'Hindsight Hints',              desc: '后验蒸馏：将成功轨迹的"提示"作为奖励信号' },
  { value: 'custom_model',        label: '自定义奖励模型',               desc: '调用外部 reward model endpoint（HTTP verifier 服务）' },
]

const STATUS_TAG: Record<string, string> = {
  queued: 'tag-gray', running: 'tag-blue', succeeded: 'tag-green',
  failed: 'tag-red', cancelled: 'tag-gray',
}

const METHOD_LABEL: Record<string, string> = {
  lora: 'LoRA', qlora: 'QLoRA', full: 'Full FT',
  grpo: 'GRPO', gspo: 'GSPO', dapo: 'DAPO', vapo: 'VAPO', ppo: 'PPO', dpo: 'DPO',
}

const SCENARIO_LABEL: Record<string, string> = {
  chat: '对话', code: '代码', tool_call: 'MCP/工具',
  agent_single: 'Agent 单轮', agent_multi: 'Agent 多轮',
  math_reasoning: '数学推理', custom: '自定义',
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function FinetuningPage() {
  const [mode, setMode] = useState<TrainingMode>('sft')
  const [jobs, setJobs] = useState<FineTuningJob[]>([])
  const [datasets, setDatasets] = useState<Dataset[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // Dataset picker mode: 'pick' = select from list, 'manual' = type file ID
  const [sftTrainPickMode, setSftTrainPickMode] = useState<'pick' | 'manual'>('pick')
  const [sftValPickMode, setSftValPickMode] = useState<'pick' | 'manual'>('pick')
  const [rlTrainPickMode, setRlTrainPickMode] = useState<'pick' | 'manual'>('pick')
  const [rlValPickMode, setRlValPickMode] = useState<'pick' | 'manual'>('pick')
  const [promptDatasetPickMode, setPromptDatasetPickMode] = useState<'pick' | 'manual'>('pick')

  // SFT fields
  const [baseModel, setBaseModel] = useState('deepseek-ai/DeepSeek-V3')
  const [trainingFile, setTrainingFile] = useState('file-demo-001')
  const [validationFile, setValidationFile] = useState('')
  const [sftMethod, setSftMethod] = useState<'lora' | 'qlora' | 'full'>('lora')
  const [epochs, setEpochs] = useState('3')
  const [lr, setLr] = useState('2e-4')
  const [batchSize, setBatchSize] = useState('8')

  // RL — base
  const [rlModel, setRlModel] = useState('deepseek-ai/DeepSeek-V3')
  const [rlMethod, setRlMethod] = useState<RLMethod>('grpo')
  const [rolloutScenario, setRolloutScenario] = useState('chat')
  const [rlTrainingFile, setRlTrainingFile] = useState('')
  const [rlValidationFile, setRlValidationFile] = useState('')

  // RL — slime: Training Engine (Megatron-LM)
  const [tensorParallel, setTensorParallel] = useState('2')
  const [pipelineParallel, setPipelineParallel] = useState('1')
  const [dataParallel, setDataParallel] = useState('4')
  const [trainSteps, setTrainSteps] = useState('1000')
  const [saveSteps, setSaveSteps] = useState('100')

  // RL — slime: Rollout Engine (SGLang)
  const [rolloutSamples, setRolloutSamples] = useState('512')
  const [rolloutTurns, setRolloutTurns] = useState('1')
  const [rolloutTemp, setRolloutTemp] = useState('0.7')
  const [sgMemFraction, setSgMemFraction] = useState('0.8')
  const [asyncRollout, setAsyncRollout] = useState(true)
  const [customRolloutScript, setCustomRolloutScript] = useState('')

  // RL — slime: Data Buffer
  const [bufferSize, setBufferSize] = useState('2048')
  const [promptDataset, setPromptDataset] = useState('')

  // Reward config
  const [rewardConfigTab, setRewardConfigTab] = useState<'builtin' | 'endpoint' | 'code'>('builtin')
  const [rewardFns, setRewardFns] = useState<string[]>(['accuracy', 'format'])
  const [rewardWeights, setRewardWeights] = useState<Record<string, string>>({ accuracy: '1.0', format: '0.5' })
  const [rewardEndpoint, setRewardEndpoint] = useState('')
  const [rewardCode, setRewardCode] = useState(`def reward_fn(solution: str, ground_truth: str, **kwargs) -> float:
    """
    自定义奖励函数
    Args:
        solution:     模型生成的回答
        ground_truth: 标准答案（来自数据集 answer 字段）
        **kwargs:     其他上下文（prompt、metadata 等）
    Returns:
        float: 奖励分数，建议范围 [-1, 1] 或 [0, 1]
    """
    # TODO: 实现你的 reward 逻辑
    if solution.strip() == ground_truth.strip():
        return 1.0
    return 0.0
`)
  const [klCoeff, setKlCoeff] = useState('0.01')
  const [clipRange, setClipRange] = useState('0.2')
  const [advNorm, setAdvNorm] = useState(true)

  const load = async () => {
    try {
      const [jobRes, dsRes] = await Promise.all([listFineTuningJobs(), listDatasets()])
      setJobs(jobRes.data || [])
      setDatasets(dsRes.data || [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [])

  const toggleRewardFn = (v: string) => {
    setRewardFns(prev => prev.includes(v) ? prev.filter(x => x !== v) : [...prev, v])
  }

  const handleCreate = async () => {
    setError('')
    setSubmitting(true)
    try {
      if (mode === 'sft') {
        await createFineTuningJob({
          model: baseModel,
          training_file: trainingFile,
          validation_file: validationFile || undefined,
          method: sftMethod,
          hyperparameters: { epochs: Number(epochs), learning_rate: lr, batch_size: Number(batchSize) },
        })
      } else {
        const weightsMap: Record<string, number> = {}
        rewardFns.forEach(fn => { weightsMap[fn] = parseFloat(rewardWeights[fn] || '1') })
        await createFineTuningJob({
          model: rlModel,
          training_file: rlTrainingFile || undefined,
          validation_file: rlValidationFile || undefined,
          method: rlMethod,
          rollout_scenario: rolloutScenario,
          rollout_config: {
            num_turns: Number(rolloutTurns),
            num_samples: Number(rolloutSamples),
            temperature: parseFloat(rolloutTemp),
            async_rollout: asyncRollout,
            sglang_mem_fraction_static: parseFloat(sgMemFraction),
            custom_rollout_script: customRolloutScript || undefined,
          },
          engine_config: {
            training_engine: 'megatron',
            rollout_engine: 'sglang',
            tensor_model_parallel_size: Number(tensorParallel),
            pipeline_model_parallel_size: Number(pipelineParallel),
            data_parallel_size: Number(dataParallel),
            train_steps: Number(trainSteps),
            save_steps: Number(saveSteps),
          },
          data_buffer_config: {
            buffer_size: Number(bufferSize),
            prompt_dataset: promptDataset || rlTrainingFile || undefined,
          },
          reward_config: {
            reward_functions: rewardConfigTab === 'endpoint' ? [] : rewardFns,
            weights: weightsMap,
            reward_model_endpoint: rewardConfigTab === 'endpoint' ? (rewardEndpoint || undefined) : undefined,
            reward_code: rewardConfigTab === 'code' ? (rewardCode || undefined) : undefined,
            kl_coeff: parseFloat(klCoeff),
            clip_range: parseFloat(clipRange),
            advantage_norm: advNorm,
          },
        })
      }
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleCancel = async (id: string) => {
    try {
      await cancelFineTuningJob(id)
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '取消失败')
    }
  }

  if (loading && jobs.length === 0) {
    return (
      <div className="flex items-center gap-2 text-[var(--text-muted)]">
        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
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

      {/* ─── Config Flow Overview (above tabs) ─────────────────── */}
      <div className="mb-5 p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-hover)]">
        <div className="flex items-center gap-2 mb-3">
          <p className="text-[11px] font-medium text-[var(--text-muted)] uppercase tracking-wider">配置流程</p>
          {mode === 'rl' && (
            <span className="text-[10px] font-bold text-[var(--primary)] bg-[color:var(--primary)]/10 px-1.5 py-0.5 rounded font-mono">RL</span>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-0">
          {(mode === 'sft' ? [
            {
              step: '1', title: '选择基座模型', desc: '指定要微调的基础模型 ID',
              done: baseModel.trim().length > 0,
            },
            {
              step: '2', title: '添加训练数据', desc: '上传或指定训练文件 ID',
              done: trainingFile.trim().length > 0,
            },
            {
              step: '3', title: '选择微调方法', desc: 'LoRA · QLoRA · Full FT',
              done: true,
            },
            {
              step: '4', title: '配置超参数', desc: 'Epochs · LR · Batch Size',
              done: epochs.trim().length > 0 && lr.trim().length > 0,
            },
            {
              step: '5', title: '提交任务', desc: '点击"提交微调任务"',
              done: false,
            },
          ] : [
            {
              step: '1', title: '选择基座模型', desc: '指定参考模型 ID',
              done: rlModel.trim().length > 0,
            },
            {
              step: '2', title: '选择 RL 算法', desc: 'GRPO · GSPO · DAPO · VAPO · PPO · DPO',
              done: true,
            },
            {
              step: '3', title: 'Training Engine', desc: 'Megatron-LM 并行策略',
              done: tensorParallel.trim().length > 0 && trainSteps.trim().length > 0,
            },
            {
              step: '4', title: 'Rollout Engine', desc: 'SGLang 场景 + Data Buffer',
              done: rolloutScenario.length > 0 && rolloutSamples.trim().length > 0,
            },
            {
              step: '5', title: 'Reward 配置', desc: '奖励函数 + KL 超参',
              done: rewardFns.length > 0,
            },
            {
              step: '6', title: '提交任务', desc: `提交 ${rlMethod.toUpperCase()} 后训练任务`,
              done: false,
            },
          ]).map((item, idx, arr) => (
            <div key={item.step} className="flex items-center">
              <div className={`group/step relative flex items-center gap-2 px-3 py-2 rounded-lg transition-colors ${
                item.done
                  ? 'bg-[color:var(--primary)]/10 text-[var(--primary)]'
                  : 'text-[var(--text-muted)] hover:bg-[var(--bg-hover)]'
              }`}>
                {/* 步骤编号 / 完成勾 */}
                <span className={`flex items-center justify-center w-5 h-5 rounded-full text-[10px] font-bold shrink-0 ${
                  item.done
                    ? 'bg-[var(--primary)] text-white'
                    : 'bg-[var(--bg)] border border-[var(--border)] text-[var(--text-muted)]'
                }`}>
                  {item.done ? (
                    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                  ) : item.step}
                </span>
                {/* 步骤标题 */}
                <span className="text-[11px] font-medium leading-tight whitespace-nowrap">{item.title}</span>
                {/* ❓ hover 触发 tooltip */}
                <span className="text-[10px] opacity-40 group-hover/step:opacity-80 transition-opacity cursor-default leading-none select-none">?</span>
                {/* Tooltip 气泡 */}
                <div className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50
                  opacity-0 group-hover/step:opacity-100 transition-opacity duration-150">
                  <div className="bg-[var(--text)] text-[var(--bg)] text-[10px] leading-snug rounded-md px-2.5 py-1.5 whitespace-nowrap shadow-lg">
                    {item.desc}
                  </div>
                  <div className="w-2 h-2 bg-[var(--text)] rotate-45 mx-auto -mt-1"/>
                </div>
              </div>
              {idx < arr.length - 1 && (
                <span className="mx-0.5 text-[var(--border)] shrink-0 text-base leading-none select-none">›</span>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Mode tabs */}
      <div className="flex gap-1 p-1 bg-[var(--bg-hover)] rounded-lg w-fit mb-6">
        {([
          { id: 'sft', label: '监督微调 (SFT)', sub: 'LoRA · QLoRA · Full FT' },
          { id: 'rl',  label: '强化学习后训练 (RL)', sub: 'GRPO · GSPO · DAPO · VAPO · PPO · DPO' },
        ] as const).map(tab => (
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

      {/* ─── SFT Panel ─────────────────────────────────────────── */}
      {mode === 'sft' && (
        <div className="space-y-4 mb-6">

          {/* 模型选择 */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-4">基座模型</h3>
            <div>
              <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">模型 ID</label>
              <input value={baseModel} onChange={e => setBaseModel(e.target.value)} placeholder="deepseek-ai/DeepSeek-V3" />
              <p className="text-[11px] text-[var(--text-muted)] mt-1.5">支持 HuggingFace 格式 ID 或平台内部已注册模型名称</p>
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
              {/* 训练集 */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-medium text-[var(--text-secondary)]">
                    训练集 <span className="text-[var(--danger)]">*</span>
                  </label>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setSftTrainPickMode('pick')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${sftTrainPickMode === 'pick' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >选择</button>
                    <button
                      type="button"
                      onClick={() => setSftTrainPickMode('manual')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${sftTrainPickMode === 'manual' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >手动</button>
                  </div>
                </div>
                {sftTrainPickMode === 'pick' ? (
                  <>
                    <select
                      value={trainingFile}
                      onChange={e => setTrainingFile(e.target.value)}
                    >
                      <option value="">— 选择数据集 —</option>
                      {datasets.map(ds => (
                        <option key={ds.id} value={ds.id}>
                          {ds.name}{ds.num_rows > 0 ? ` (${ds.num_rows.toLocaleString()} 行)` : ''}
                        </option>
                      ))}
                    </select>
                    {trainingFile && (() => {
                      const ds = datasets.find(d => d.id === trainingFile)
                      return ds ? (
                        <p className="text-[11px] text-[var(--primary)] mt-1 flex items-center gap-1">
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                          {ds.description || ds.name} · {ds.num_rows > 0 ? `${ds.num_rows.toLocaleString()} 行` : '空'}
                        </p>
                      ) : null
                    })()}
                    {datasets.length === 0 && (
                      <p className="text-[11px] text-[var(--text-muted)] mt-1">
                        暂无数据集，请先在<a href="/datasets" className="text-[var(--primary)] mx-0.5 hover:underline">数据集页面</a>创建
                      </p>
                    )}
                  </>
                ) : (
                  <>
                    <input
                      value={trainingFile}
                      onChange={e => setTrainingFile(e.target.value)}
                      placeholder="file-xxxx 或 data/train.jsonl"
                    />
                    <p className="text-[11px] text-[var(--text-muted)] mt-1">从「数据集」页面上传后获取文件 ID</p>
                  </>
                )}
              </div>
              {/* 验证集 */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-medium text-[var(--text-secondary)]">
                    验证集 <span className="text-[var(--text-muted)]">（可选）</span>
                  </label>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setSftValPickMode('pick')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${sftValPickMode === 'pick' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >选择</button>
                    <button
                      type="button"
                      onClick={() => setSftValPickMode('manual')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${sftValPickMode === 'manual' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >手动</button>
                  </div>
                </div>
                {sftValPickMode === 'pick' ? (
                  <>
                    <select
                      value={validationFile}
                      onChange={e => setValidationFile(e.target.value)}
                    >
                      <option value="">— 不使用验证集 —</option>
                      {datasets.map(ds => (
                        <option key={ds.id} value={ds.id}>
                          {ds.name}{ds.num_rows > 0 ? ` (${ds.num_rows.toLocaleString()} 行)` : ''}
                        </option>
                      ))}
                    </select>
                    {validationFile && (() => {
                      const ds = datasets.find(d => d.id === validationFile)
                      return ds ? (
                        <p className="text-[11px] text-[var(--primary)] mt-1 flex items-center gap-1">
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                          {ds.description || ds.name} · {ds.num_rows > 0 ? `${ds.num_rows.toLocaleString()} 行` : '空'}
                        </p>
                      ) : null
                    })()}
                  </>
                ) : (
                  <>
                    <input
                      value={validationFile}
                      onChange={e => setValidationFile(e.target.value)}
                      placeholder="file-xxxx 或 data/val.jsonl"
                    />
                    <p className="text-[11px] text-[var(--text-muted)] mt-1">用于训练过程中的 eval loss 监控</p>
                  </>
                )}
              </div>
            </div>
            <div className="mt-4 pt-4 border-t border-[var(--border-light)] flex items-center gap-3">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-[var(--text-muted)] shrink-0">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
                <line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/>
              </svg>
              <p className="text-[11px] text-[var(--text-muted)]">
                数据格式：每行一个 JSON 对象，需包含 <span className="font-mono bg-[var(--bg-hover)] px-1 rounded">messages</span> 字段（OpenAI Chat 格式）或 <span className="font-mono bg-[var(--bg-hover)] px-1 rounded">prompt</span> / <span className="font-mono bg-[var(--bg-hover)] px-1 rounded">completion</span> 字段。
                <a href="#" className="text-[var(--primary)] ml-1 hover:underline">查看格式文档</a>
              </p>
            </div>
          </div>

          {/* 训练配置 */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-4">训练配置</h3>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">微调方法</label>
                <select value={sftMethod} onChange={e => setSftMethod(e.target.value as typeof sftMethod)}>
                  <option value="lora">LoRA — 参数高效，显存低</option>
                  <option value="qlora">QLoRA — 量化 LoRA，极低显存</option>
                  <option value="full">Full Fine-tuning — 全参数更新</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">训练轮数 (Epochs)</label>
                <input type="number" min="1" max="100" value={epochs} onChange={e => setEpochs(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">学习率 (LR)</label>
                <input value={lr} onChange={e => setLr(e.target.value)} placeholder="2e-4" />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Batch Size</label>
                <input type="number" min="1" value={batchSize} onChange={e => setBatchSize(e.target.value)} />
              </div>
            </div>
            <button type="button" onClick={handleCreate} disabled={submitting} className="btn-primary mt-5">
              {submitting ? '提交中…' : '提交微调任务'}
            </button>
          </div>

        </div>
      )}

      {/* ─── RL Panel ──────────────────────────────────────────── */}
      {mode === 'rl' && (
        <div className="space-y-4 mb-6">

          {/* ① 基础配置 */}
          <div className="card">
            <div className="flex items-center gap-2 mb-4">
              <span className="text-xs font-bold text-[var(--primary)] bg-[color:var(--primary)]/10 px-2 py-0.5 rounded">RL</span>
              <h3 className="font-medium text-sm text-[var(--text)]">基础配置</h3>
            </div>
            <div className="grid gap-4 md:grid-cols-3">
              <div className="md:col-span-2">
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">基座 / 参考模型 ID</label>
                <input value={rlModel} onChange={e => setRlModel(e.target.value)} placeholder="deepseek-ai/DeepSeek-V3" />
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
              </div>
            </div>
          </div>

          {/* ① 数据集 */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="font-medium text-sm text-[var(--text)]">数据集</h3>
                <p className="text-xs text-[var(--text-muted)] mt-0.5">
                  从数据集管理中选择，或手动填写文件 ID
                </p>
              </div>
              <span className="text-[11px] text-[var(--text-muted)] bg-[var(--bg-hover)] px-2 py-1 rounded">JSONL / parquet</span>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              {/* 训练/Prompt 数据集 */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-medium text-[var(--text-secondary)]">
                    训练 / Prompt 数据集 <span className="text-[var(--danger)]">*</span>
                  </label>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setRlTrainPickMode('pick')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${rlTrainPickMode === 'pick' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >选择</button>
                    <button
                      type="button"
                      onClick={() => setRlTrainPickMode('manual')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${rlTrainPickMode === 'manual' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >手动</button>
                  </div>
                </div>
                {rlTrainPickMode === 'pick' ? (
                  <>
                    <select
                      value={rlTrainingFile}
                      onChange={e => setRlTrainingFile(e.target.value)}
                    >
                      <option value="">— 选择数据集 —</option>
                      {datasets.map(ds => (
                        <option key={ds.id} value={ds.id}>
                          {ds.name}{ds.num_rows > 0 ? ` (${ds.num_rows.toLocaleString()} 行)` : ''}
                        </option>
                      ))}
                    </select>
                    {rlTrainingFile && (() => {
                      const ds = datasets.find(d => d.id === rlTrainingFile)
                      return ds ? (
                        <p className="text-[11px] text-[var(--primary)] mt-1 flex items-center gap-1">
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                          {ds.description || ds.name} · {ds.num_rows > 0 ? `${ds.num_rows.toLocaleString()} 行` : '空'}
                        </p>
                      ) : null
                    })()}
                    {datasets.length === 0 && (
                      <p className="text-[11px] text-[var(--text-muted)] mt-1">
                        暂无数据集，请先在<a href="/datasets" className="text-[var(--primary)] mx-0.5 hover:underline">数据集页面</a>创建
                      </p>
                    )}
                  </>
                ) : (
                  <>
                    <input
                      value={rlTrainingFile}
                      onChange={e => setRlTrainingFile(e.target.value)}
                      placeholder="file-xxxx 或 data/prompts.jsonl"
                    />
                    <p className="text-[11px] text-[var(--text-muted)] mt-1">
                      {rlMethod === 'dpo' ? '配对偏好数据：每行含 prompt / chosen / rejected' : 'Prompt 列表，用于 SGLang Rollout 采样生成轨迹'}
                    </p>
                  </>
                )}
              </div>
              {/* 验证集 */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-medium text-[var(--text-secondary)]">
                    验证集 <span className="text-[var(--text-muted)]">（可选）</span>
                  </label>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setRlValPickMode('pick')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${rlValPickMode === 'pick' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >选择</button>
                    <button
                      type="button"
                      onClick={() => setRlValPickMode('manual')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${rlValPickMode === 'manual' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >手动</button>
                  </div>
                </div>
                {rlValPickMode === 'pick' ? (
                  <>
                    <select
                      value={rlValidationFile}
                      onChange={e => setRlValidationFile(e.target.value)}
                    >
                      <option value="">— 不使用验证集 —</option>
                      {datasets.map(ds => (
                        <option key={ds.id} value={ds.id}>
                          {ds.name}{ds.num_rows > 0 ? ` (${ds.num_rows.toLocaleString()} 行)` : ''}
                        </option>
                      ))}
                    </select>
                    {rlValidationFile && (() => {
                      const ds = datasets.find(d => d.id === rlValidationFile)
                      return ds ? (
                        <p className="text-[11px] text-[var(--primary)] mt-1 flex items-center gap-1">
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                          {ds.description || ds.name} · {ds.num_rows > 0 ? `${ds.num_rows.toLocaleString()} 行` : '空'}
                        </p>
                      ) : null
                    })()}
                  </>
                ) : (
                  <>
                    <input
                      value={rlValidationFile}
                      onChange={e => setRlValidationFile(e.target.value)}
                      placeholder="file-xxxx 或 data/val.jsonl"
                    />
                    <p className="text-[11px] text-[var(--text-muted)] mt-1">训练中定期评估 reward / win-rate</p>
                  </>
                )}
              </div>
            </div>
            <div className="mt-4 pt-4 border-t border-[var(--border-light)] flex items-center gap-3">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-[var(--text-muted)] shrink-0">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
                <line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>
              </svg>
              <p className="text-[11px] text-[var(--text-muted)]">
                {rlMethod === 'dpo'
                  ? <>DPO 格式：<span className="font-mono bg-[var(--bg-hover)] px-1 rounded">&#123;"prompt":…,"chosen":…,"rejected":…&#125;</span></>
                  : <>Prompt 格式：<span className="font-mono bg-[var(--bg-hover)] px-1 rounded">&#123;"prompt":…&#125;</span> 或含 <span className="font-mono bg-[var(--bg-hover)] px-1 rounded">messages</span> 字段，slime Data Buffer 自动分发至 SGLang Rollout</>
                }
                <a href="#" className="text-[var(--primary)] ml-1 hover:underline">查看格式文档</a>
              </p>
            </div>
          </div>

          {/* ② Training Engine — Megatron-LM */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="font-medium text-sm text-[var(--text)]">Training Engine</h3>
                <p className="text-xs text-[var(--text-muted)] mt-0.5">
                  基于 <span className="font-mono">Megatron-LM</span>，配置并行策略与训练步数
                </p>
              </div>
              <span className="text-[11px] text-[var(--text-muted)] bg-[var(--bg-hover)] px-2 py-1 rounded font-mono">megatron-lm</span>
            </div>
            <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-5">
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Tensor Parallel</label>
                <input type="number" min="1" value={tensorParallel} onChange={e => setTensorParallel(e.target.value)}
                  placeholder="--tensor-model-parallel-size" />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Pipeline Parallel</label>
                <input type="number" min="1" value={pipelineParallel} onChange={e => setPipelineParallel(e.target.value)}
                  placeholder="--pipeline-model-parallel-size" />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Data Parallel</label>
                <input type="number" min="1" value={dataParallel} onChange={e => setDataParallel(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Train Steps</label>
                <input type="number" min="1" value={trainSteps} onChange={e => setTrainSteps(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Save Steps</label>
                <input type="number" min="1" value={saveSteps} onChange={e => setSaveSteps(e.target.value)} />
              </div>
            </div>
          </div>

          {/* ③ Rollout Engine — SGLang */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="font-medium text-sm text-[var(--text)]">Rollout Engine</h3>
                <p className="text-xs text-[var(--text-muted)] mt-0.5">
                  基于 <span className="font-mono">SGLang</span>，生成策略样本并计算即时奖励
                </p>
              </div>
              <div className="flex items-center gap-2">
                <label className="text-xs text-[var(--text-muted)] cursor-pointer flex items-center gap-1.5">
                  <input type="checkbox" checked={asyncRollout} onChange={e => setAsyncRollout(e.target.checked)} className="accent-[var(--primary)]" />
                  Async Rollout
                </label>
                <span className="text-[11px] text-[var(--text-muted)] bg-[var(--bg-hover)] px-2 py-1 rounded font-mono">sglang</span>
              </div>
            </div>

            {/* Rollout scenario */}
            <p className="text-xs font-medium text-[var(--text-secondary)] mb-3">Rollout 场景</p>
            <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4 mb-4">
              {ROLLOUT_SCENARIOS.map(s => (
                <button
                  key={s.value}
                  type="button"
                  onClick={() => setRolloutScenario(s.value)}
                  className={`text-left p-3 rounded-lg border transition-all ${
                    rolloutScenario === s.value
                      ? 'border-[var(--primary)] bg-[color:var(--primary)]/5'
                      : 'border-[var(--border)] hover:border-[var(--border-hover)] bg-[var(--bg)]'
                  }`}
                >
                  <div className={`flex items-center gap-1.5 mb-1 font-medium text-xs ${
                    rolloutScenario === s.value ? 'text-[var(--primary)]' : 'text-[var(--text)]'
                  }`}>
                    <span className={rolloutScenario === s.value ? 'text-[var(--primary)]' : 'text-[var(--text-muted)]'}>
                      {s.icon}
                    </span>
                    {s.label}
                  </div>
                  <p className="text-[10px] text-[var(--text-muted)] leading-relaxed mb-1.5">{s.desc}</p>
                  <span className="text-[9px] font-mono bg-[var(--bg-hover)] px-1.5 py-0.5 rounded text-[var(--text-secondary)]">
                    {s.badge}
                  </span>
                </button>
              ))}
            </div>

            {/* Rollout params */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 pt-4 border-t border-[var(--border-light)]">
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">采样样本数 (Samples / Step)</label>
                <input type="number" min="32" step="32" value={rolloutSamples} onChange={e => setRolloutSamples(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                  {rolloutScenario === 'agent_multi' ? '最大 Turns 数' : 'Rollout Turns'}
                </label>
                <input type="number" min="1" value={rolloutTurns} onChange={e => setRolloutTurns(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">采样温度 (Temperature)</label>
                <input type="number" min="0" max="2" step="0.05" value={rolloutTemp} onChange={e => setRolloutTemp(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                  SGLang <span className="font-mono text-[10px]">--mem-fraction-static</span>
                </label>
                <input type="number" min="0.1" max="1" step="0.05" value={sgMemFraction} onChange={e => setSgMemFraction(e.target.value)} />
              </div>
            </div>

            {/* Custom rollout script */}
            {rolloutScenario === 'custom' && (
              <div className="mt-4 pt-4 border-t border-[var(--border-light)]">
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                  自定义 Rollout 脚本路径 / URL <span className="text-[var(--text-muted)]">（SGLang 插件路径或外部 verifier 服务）</span>
                </label>
                <input
                  value={customRolloutScript}
                  onChange={e => setCustomRolloutScript(e.target.value)}
                  placeholder="examples/custom_rollout.py 或 http://verifier-service:8080"
                />
              </div>
            )}
          </div>

          {/* ④ Data Buffer */}
          <div className="card">
            <h3 className="font-medium text-sm text-[var(--text)] mb-1">Data Buffer</h3>
            <p className="text-xs text-[var(--text-muted)] mb-4">
              slime 统一数据缓冲区，连接 Training 与 Rollout，管理 prompt 初始化与异步数据流
            </p>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Buffer Size</label>
                <input type="number" min="128" step="128" value={bufferSize} onChange={e => setBufferSize(e.target.value)} />
              </div>
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-medium text-[var(--text-secondary)]">
                    Prompt 数据集
                    <span className="text-[var(--text-muted)] font-normal ml-1">（留空则复用上方训练集）</span>
                  </label>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setPromptDatasetPickMode('pick')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${promptDatasetPickMode === 'pick' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >选择</button>
                    <button
                      type="button"
                      onClick={() => setPromptDatasetPickMode('manual')}
                      className={`text-[10px] px-1.5 py-0.5 rounded transition-colors ${promptDatasetPickMode === 'manual' ? 'bg-[var(--primary)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text)]'}`}
                    >手动</button>
                  </div>
                </div>
                {promptDatasetPickMode === 'pick' ? (
                  <>
                    <select
                      value={promptDataset}
                      onChange={e => setPromptDataset(e.target.value)}
                    >
                      <option value="">— 复用上方训练集 —</option>
                      {datasets.map(ds => (
                        <option key={ds.id} value={ds.id}>
                          {ds.name}{ds.num_rows > 0 ? ` (${ds.num_rows.toLocaleString()} 行)` : ''}
                        </option>
                      ))}
                    </select>
                    {promptDataset && (() => {
                      const ds = datasets.find(d => d.id === promptDataset)
                      return ds ? (
                        <p className="text-[11px] text-[var(--primary)] mt-1 flex items-center gap-1">
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                          {ds.description || ds.name} · {ds.num_rows > 0 ? `${ds.num_rows.toLocaleString()} 行` : '空'}
                        </p>
                      ) : null
                    })()}
                    {!promptDataset && rlTrainingFile && (() => {
                      const ds = datasets.find(d => d.id === rlTrainingFile)
                      return ds ? (
                        <p className="text-[11px] text-[var(--text-muted)] mt-1 flex items-center gap-1">
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                          将使用训练集：{ds.name}
                        </p>
                      ) : null
                    })()}
                  </>
                ) : (
                  <>
                    <input value={promptDataset} onChange={e => setPromptDataset(e.target.value)} placeholder="data/prompts.jsonl 或 dataset ID" />
                    <p className="text-[11px] text-[var(--text-muted)] mt-1">留空则自动复用上方训练集</p>
                  </>
                )}
              </div>
            </div>
          </div>

          {/* ⑤ Reward Configuration */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="font-medium text-sm text-[var(--text)]">Reward Configuration</h3>
                <p className="text-xs text-[var(--text-muted)] mt-0.5">配置奖励来源：内置函数、外部服务或自定义代码</p>
              </div>
              {/* Tab switcher */}
              <div className="flex items-center gap-1 bg-[var(--bg-hover)] p-0.5 rounded-lg">
                {([['builtin', '内置函数'], ['endpoint', '外部服务'], ['code', '代码注入']] as const).map(([tab, label]) => (
                  <button
                    key={tab}
                    type="button"
                    onClick={() => setRewardConfigTab(tab)}
                    className={`text-[11px] px-2.5 py-1 rounded-md transition-all ${
                      rewardConfigTab === tab
                        ? 'bg-[var(--card)] text-[var(--text)] shadow-sm font-medium'
                        : 'text-[var(--text-muted)] hover:text-[var(--text)]'
                    }`}
                  >{label}</button>
                ))}
              </div>
            </div>

            {/* Tab: 内置函数 */}
            {rewardConfigTab === 'builtin' && (
              <div className="flex flex-wrap gap-2">
                {REWARD_FUNCTIONS.filter(fn => fn.value !== 'custom_model').map(fn => {
                  const active = rewardFns.includes(fn.value)
                  return (
                    <button
                      key={fn.value}
                      type="button"
                      title={fn.desc}
                      onClick={() => toggleRewardFn(fn.value)}
                      className={`group flex items-center gap-2 px-3 py-2 rounded-lg border text-xs transition-all ${
                        active
                          ? 'border-[var(--primary)] bg-[color:var(--primary)]/8 text-[var(--primary)]'
                          : 'border-[var(--border)] text-[var(--text-muted)] hover:border-[var(--primary)]/50 hover:text-[var(--text)]'
                      }`}
                    >
                      <span className={`w-3.5 h-3.5 rounded-sm border flex items-center justify-center shrink-0 transition-colors ${
                        active ? 'bg-[var(--primary)] border-[var(--primary)]' : 'border-[var(--border)]'
                      }`}>
                        {active && <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3.5"><polyline points="20 6 9 17 4 12"/></svg>}
                      </span>
                      <span className="font-medium">{fn.label}</span>
                      {active && (
                        <span className="flex items-center gap-1 ml-1 pl-2 border-l border-[var(--primary)]/30">
                          <span className="text-[var(--text-muted)] text-[10px]">w</span>
                          <input
                            type="number"
                            min="0" max="10" step="0.1"
                            value={rewardWeights[fn.value] ?? '1.0'}
                            onChange={e => { e.stopPropagation(); setRewardWeights(prev => ({ ...prev, [fn.value]: e.target.value })) }}
                            onClick={e => e.stopPropagation()}
                            className="w-12 text-[11px] text-center bg-transparent border-none outline-none p-0 text-[var(--primary)]"
                          />
                        </span>
                      )}
                    </button>
                  )
                })}
                {rewardFns.length === 0 && (
                  <p className="text-[11px] text-[var(--text-muted)] py-2">请至少选择一个奖励函数</p>
                )}
              </div>
            )}

            {/* Tab: 外部服务 */}
            {rewardConfigTab === 'endpoint' && (
              <div className="space-y-3">
                <div>
                  <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">
                    Reward Model Endpoint <span className="text-[var(--danger)]">*</span>
                  </label>
                  <input
                    value={rewardEndpoint}
                    onChange={e => setRewardEndpoint(e.target.value)}
                    placeholder="http://reward-model-service:8080/score"
                  />
                  <p className="text-[11px] text-[var(--text-muted)] mt-1">
                    POST <span className="font-mono bg-[var(--bg-hover)] px-1 rounded">&#123;"solution": "…", "ground_truth": "…"&#125;</span> → 返回 <span className="font-mono bg-[var(--bg-hover)] px-1 rounded">&#123;"score": 0.95&#125;</span>
                  </p>
                </div>
                <div className="p-3 rounded-lg bg-[var(--bg-hover)] text-[11px] text-[var(--text-muted)] flex gap-2">
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="shrink-0 mt-0.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                  <span>外部服务模式下内置函数不生效，所有 reward 信号来自该 endpoint。确保服务在训练节点内网可达。</span>
                </div>
              </div>
            )}

            {/* Tab: 代码注入 */}
            {rewardConfigTab === 'code' && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-xs font-medium text-[var(--text-secondary)]">
                    Python Reward 函数 <span className="font-mono text-[10px] text-[var(--text-muted)] ml-1">reward_fn(solution, ground_truth, **kwargs) → float</span>
                  </label>
                  <button
                    type="button"
                    onClick={() => setRewardCode(`def reward_fn(solution: str, ground_truth: str, **kwargs) -> float:
    """
    自定义奖励函数
    Args:
        solution:     模型生成的回答
        ground_truth: 标准答案（来自数据集 answer 字段）
        **kwargs:     其他上下文（prompt、metadata 等）
    Returns:
        float: 奖励分数，建议范围 [-1, 1] 或 [0, 1]
    """
    # TODO: 实现你的 reward 逻辑
    if solution.strip() == ground_truth.strip():
        return 1.0
    return 0.0
`)}
                    className="text-[10px] text-[var(--text-muted)] hover:text-[var(--text)] flex items-center gap-1"
                  >
                    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 .49-4.95"/></svg>
                    重置
                  </button>
                </div>
                <div className="relative rounded-lg overflow-hidden border border-[var(--border)] bg-[#1e1e2e]">
                  <div className="flex items-center justify-between px-3 py-1.5 bg-[#181825] border-b border-[#313244]">
                    <div className="flex items-center gap-1.5">
                      <span className="w-2.5 h-2.5 rounded-full bg-[#f38ba8]"/>
                      <span className="w-2.5 h-2.5 rounded-full bg-[#fab387]"/>
                      <span className="w-2.5 h-2.5 rounded-full bg-[#a6e3a1]"/>
                    </div>
                    <span className="text-[10px] text-[#6c7086] font-mono">reward_fn.py</span>
                  </div>
                  <textarea
                    value={rewardCode}
                    onChange={e => setRewardCode(e.target.value)}
                    spellCheck={false}
                    rows={14}
                    className="w-full bg-transparent font-mono text-[12px] text-[#cdd6f4] px-4 py-3 resize-y outline-none border-none leading-relaxed"
                    style={{ tabSize: 4 }}
                  />
                </div>
                <div className="mt-2 p-3 rounded-lg bg-[var(--bg-hover)] text-[11px] text-[var(--text-muted)] flex gap-2">
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="shrink-0 mt-0.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                  <span>代码将在沙箱中执行，每条样本调用一次。可 import 标准库，不可访问网络。内置函数在代码模式下仍会叠加计算（权重由上方「内置函数」Tab 控制）。</span>
                </div>
              </div>
            )}

            {/* Policy optimization hyperparams */}
            {rlMethod !== 'dpo' && (
              <div className="mt-4 pt-4 border-t border-[var(--border-light)]">
                <p className="text-xs font-medium text-[var(--text-secondary)] mb-3">
                  {rlMethod.toUpperCase()} 策略优化超参数
                </p>
                <div className="grid gap-4 md:grid-cols-3">
                  <div>
                    <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">KL Penalty 系数</label>
                    <input type="number" min="0" max="1" step="0.001" value={klCoeff} onChange={e => setKlCoeff(e.target.value)} />
                  </div>
                  {rlMethod === 'ppo' && (
                    <div>
                      <label className="text-xs font-medium text-[var(--text-secondary)] block mb-1.5">Clip Range (ε)</label>
                      <input type="number" min="0.05" max="0.5" step="0.01" value={clipRange} onChange={e => setClipRange(e.target.value)} />
                    </div>
                  )}
                  <div className="flex items-center gap-2 pt-5">
                    <input
                      type="checkbox"
                      id="adv-norm"
                      checked={advNorm}
                      onChange={e => setAdvNorm(e.target.checked)}
                      className="accent-[var(--primary)]"
                    />
                    <label htmlFor="adv-norm" className="text-xs text-[var(--text-secondary)] cursor-pointer">
                      Advantage Normalization
                    </label>
                  </div>
                </div>
              </div>
            )}
          </div>

          <button type="button" onClick={handleCreate} disabled={submitting} className="btn-primary">
            {submitting ? '提交中…' : `提交 ${rlMethod.toUpperCase()} 后训练任务`}
          </button>
        </div>
      )}

      {/* ─── Job List ───────────────────────────────────────────── */}
      <div className="card">
        <div className="flex justify-between items-center mb-4">
          <h3 className="font-medium text-sm text-[var(--text)]">任务列表</h3>
          <button
            type="button"
            className="btn-outline text-xs"
            onClick={() => { setLoading(true); void load() }}
          >
            刷新
          </button>
        </div>

        {jobs.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] py-6 text-center">暂无任务</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b border-[var(--border-light)] text-[var(--text-muted)] text-xs">
                <th className="pb-3 pr-3 font-medium">ID</th>
                <th className="pb-3 pr-3 font-medium">基座模型</th>
                <th className="pb-3 pr-3 font-medium">方法</th>
                <th className="pb-3 pr-3 font-medium">场景</th>
                <th className="pb-3 pr-3 font-medium">状态</th>
                <th className="pb-3 pr-3 font-medium">产出模型</th>
                <th className="pb-3 w-20"></th>
              </tr>
            </thead>
            <tbody>
              {jobs.map(j => (
                <tr key={j.id} className="border-b border-[var(--border-light)] hover:bg-[var(--bg-hover)] transition-colors">
                  <td className="py-3 pr-3 font-mono text-[11px] text-[var(--text-secondary)]">{j.id}</td>
                  <td className="py-3 pr-3 text-xs">{j.model}</td>
                  <td className="py-3 pr-3">
                    <span className={`tag ${['grpo','gspo','dapo','vapo','ppo','dpo'].includes(j.method) ? 'tag-purple' : 'tag-blue'}`}>
                      {METHOD_LABEL[j.method] || j.method}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-[11px] text-[var(--text-secondary)]">
                    {j.rollout_scenario ? (SCENARIO_LABEL[j.rollout_scenario] || j.rollout_scenario) : '—'}
                  </td>
                  <td className="py-3 pr-3">
                    <span className={`tag ${STATUS_TAG[j.status] || 'tag-gray'}`}>{j.status}</span>
                  </td>
                  <td className="py-3 pr-3 text-[var(--text-muted)] text-[11px] max-w-[160px] truncate">
                    {j.fine_tuned_model || '—'}
                  </td>
                  <td className="py-3">
                    {(j.status === 'queued' || j.status === 'running') && (
                      <button type="button" className="btn-danger text-xs" onClick={() => handleCancel(j.id)}>
                        取消
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
