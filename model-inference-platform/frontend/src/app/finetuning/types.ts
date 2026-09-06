// Shared types for the finetuning page. Plain TS — no React imports.

export type TrainingMode = 'sft' | 'rl'

export type RLMethod = 'grpo' | 'gspo' | 'dapo' | 'vapo' | 'ppo' | 'dpo'

export type SFTMethod = 'lora' | 'qlora' | 'full'

export interface Dataset {
  id: string
  name: string
  description: string
  num_rows: number
}

export interface FineTuningJob {
  id: string
  base_model: string
  training_file?: string
  method: string
  rollout_scenario?: string
  status: string
  fine_tuned_model?: string
  pool_id?: string
  gpu_request?: number
  rollout_pool_id?: string
  created_at: number
}

export interface FlowItem {
  step: string
  title: string
  desc: string
  done: boolean
}

export interface BaseModelOption {
  id: string
  label: string
  family: string
  desc: string
}
