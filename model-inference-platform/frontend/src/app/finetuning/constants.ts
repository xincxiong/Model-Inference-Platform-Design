// Shared constants for the finetuning page.

import type { BaseModelOption, RLMethod } from './types'

export const BASE_MODEL_OPTIONS: BaseModelOption[] = [
  { id: 'deepseek-ai/DeepSeek-V3', label: 'DeepSeek V3', family: 'DeepSeek', desc: '通用旗舰对话模型，适合通用 SFT 与指令微调。' },
  { id: 'deepseek-ai/DeepSeek-R1', label: 'DeepSeek R1', family: 'DeepSeek', desc: '强化推理模型，适合数学、代码与复杂推理场景。' },
  { id: 'deepseek-ai/DeepSeek-Coder-V2-Instruct', label: 'DeepSeek Coder V2', family: 'DeepSeek', desc: '代码专用模型，适合代码生成、补全与修复。' },
  { id: 'Qwen/Qwen3-72B', label: 'Qwen3 72B', family: 'Qwen', desc: 'Qwen 系列旗舰通用模型，适合高质量指令微调。' },
  { id: 'Qwen/Qwen3-30B-A3B', label: 'Qwen3 30B-A3B', family: 'Qwen', desc: 'MoE 高效模型，兼顾效果与推理成本。' },
  { id: 'Qwen/Qwen3-8B', label: 'Qwen3 8B', family: 'Qwen', desc: '轻量化模型，适合低成本实验和快速验证。' },
  { id: 'Qwen/Qwen2.5-Coder-32B-Instruct', label: 'Qwen2.5 Coder 32B', family: 'Qwen', desc: '代码模型，适合工程类训练与评测。' },
  { id: 'Qwen/Qwen2.5-VL-7B-Instruct', label: 'Qwen2.5 VL 7B', family: 'Qwen', desc: '轻量多模态模型，适合图文理解训练。' },
  { id: 'Qwen/Qwen2.5-VL-72B-Instruct', label: 'Qwen2.5 VL 72B', family: 'Qwen', desc: '高性能多模态模型，适合复杂视觉问答场景。' },
] as unknown as BaseModelOption[]

export const ROLLOUT_SCENARIOS = [
  { value: 'chat', label: '对话场景', desc: '通用对话 RLHF，基于人类偏好打分' },
  { value: 'code', label: '代码生成', desc: '沙箱执行验证 reward（TritonForge）' },
  { value: 'tool_call', label: '工具调用 (MCP)', desc: 'MCP / Function-calling 工具对齐' },
  { value: 'agent_single', label: 'Agent 单轮执行', desc: '单轮 Agent 任务执行（P1 物理推理）' },
  { value: 'agent_multi', label: 'Agent 多轮执行', desc: '多轮 Agent 轨迹优化（OpenClaw-RL / ArenaRL）' },
  { value: 'math_reasoning', label: '数学 / 推理', desc: '可验证答案的推理任务，binary reward' },
  { value: 'custom', label: '自定义场景', desc: '自定义 SGLang rollout 脚本' },
]

export const REWARD_FUNCTIONS = [
  { value: 'accuracy', label: '正确性 (Accuracy)', desc: '与 ground-truth 对比，最常用的可验证 reward' },
  { value: 'format', label: '格式合规 (Format)', desc: '检验输出是否符合指定格式（JSON / Markdown / XML 等）' },
  { value: 'length_penalty', label: '长度惩罚 (Length Penalty)', desc: '惩罚过短 / 过长的冗余回复' },
  { value: 'tool_call_correctness', label: '工具调用精度 (Tool Accuracy)', desc: 'Function name + 参数匹配度，MCP 场景专用' },
  { value: 'code_execution', label: '代码执行通过率 (Code Exec)', desc: '沙箱执行并检查测试用例，支持编译反馈循环' },
  { value: 'task_completion', label: '任务完成度 (Task Completion)', desc: 'Agent loop 最终目标达成率' },
  { value: 'hindsight_hints', label: 'Hindsight Hints', desc: '后验蒸馏：将成功轨迹的"提示"作为奖励信号' },
  { value: 'custom_model', label: '自定义奖励模型', desc: '调用外部 reward model endpoint（HTTP verifier 服务）' },
]

export const METHOD_LABEL: Record<string, string> = {
  lora: 'LoRA', qlora: 'QLoRA', full: 'Full',
  grpo: 'GRPO', gspo: 'GSPO', dapo: 'DAPO', vapo: 'VAPO', ppo: 'PPO', dpo: 'DPO',
}

export const SCENARIO_LABEL: Record<string, string> = {
  chat: '通用对话', code: '代码生成', tool_call: '工具调用',
  agent_single: 'Agent 单轮', agent_multi: 'Agent 多轮',
  math_reasoning: '数学推理', custom: '自定义',
}

export const STATUS_TAG: Record<string, string> = {
  validating: 'tag-blue', queued: 'tag-amber', running: 'tag-green',
  succeeded: 'tag-green', failed: 'tag-pink', cancelled: 'tag-gray',
}

export const SHARING_LABEL: Record<string, string> = {
  exclusive: '独占',
  'shared-fifo': '共享 FIFO',
}

export const validMethods = {
  lora: true, qlora: true, full: true,
  grpo: true, gspo: true, dapo: true, vapo: true, ppo: true, dpo: true,
} as const

export const validRolloutScenarios = {
  chat: true, code: true, tool_call: true,
  agent_single: true, agent_multi: true, math_reasoning: true, custom: true,
} as const

export function isRLMethod(m: string): boolean {
  const rl: RLMethod[] = ['grpo', 'gspo', 'dapo', 'vapo', 'ppo', 'dpo']
  return rl.includes(m as RLMethod)
}

export const DEFAULT_REWARD_CODE = `def reward_fn(solution: str, ground_truth: str, **kwargs) -> float:
    """
    自定义奖励函数
    solution: 模型生成的答案
    ground_truth: 数据集中的标准答案
    返回 float 分数
    """
    return 1.0 if solution.strip() == ground_truth.strip() else 0.0
`
