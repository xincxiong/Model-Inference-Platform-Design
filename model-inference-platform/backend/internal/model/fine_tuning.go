package model

import "time"

// ─── Request / Response ────────────────────────────────────────────────────

// FineTuningJobCreateRequest matches POST /v1/fine_tuning/jobs
// method: lora | qlora | full | grpo | gspo | dapo | vapo | ppo | dpo
type FineTuningJobCreateRequest struct {
	Model           string                 `json:"model" binding:"required"`
	TrainingFile    string                 `json:"training_file"`
	ValidationFile  string                 `json:"validation_file"`
	Method          string                 `json:"method"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`

	// RL Post-Training fields (slime framework)
	RolloutScenario  string                 `json:"rollout_scenario,omitempty"`
	RolloutConfig    map[string]interface{} `json:"rollout_config,omitempty"`
	RewardConfig     *RewardConfig          `json:"reward_config,omitempty"`

	// slime: Training Engine (Megatron-LM) + Rollout Engine (SGLang) config
	EngineConfig     *EngineConfig          `json:"engine_config,omitempty"`

	// slime: Data Buffer config
	DataBufferConfig *DataBufferConfig      `json:"data_buffer_config,omitempty"`
}

// RolloutScenario constants (aligned with slime examples)
const (
	RolloutChat        = "chat"           // 对话场景 (RLHF/GRPO)
	RolloutCode        = "code"           // 代码生成，沙箱执行 reward (TritonForge)
	RolloutToolCall    = "tool_call"      // 工具调用 MCP/Function-calling
	RolloutAgentSingle = "agent_single"   // Agent 单轮执行 (P1 物理推理)
	RolloutAgentMulti  = "agent_multi"    // Agent 多轮执行 (OpenClaw-RL / ArenaRL)
	RolloutMathReasoning = "math_reasoning" // 数学/推理，可验证 binary reward (RLVE)
	RolloutCustom      = "custom"         // 自定义 SGLang rollout 脚本
)

// EngineConfig configures the slime training/rollout engines.
// Training engine: Megatron-LM; Rollout engine: SGLang.
type EngineConfig struct {
	TrainingEngine             string `json:"training_engine,omitempty"` // "megatron"
	RolloutEngine              string `json:"rollout_engine,omitempty"`  // "sglang"
	TensorModelParallelSize    int    `json:"tensor_model_parallel_size,omitempty"`
	PipelineModelParallelSize  int    `json:"pipeline_model_parallel_size,omitempty"`
	DataParallelSize           int    `json:"data_parallel_size,omitempty"`
	TrainSteps                 int    `json:"train_steps,omitempty"`
	SaveSteps                  int    `json:"save_steps,omitempty"`
}

// DataBufferConfig configures the slime Data Buffer that bridges Training and Rollout.
type DataBufferConfig struct {
	BufferSize    int    `json:"buffer_size,omitempty"`
	PromptDataset string `json:"prompt_dataset,omitempty"`
}

// RewardConfig defines how RL reward is computed.
type RewardConfig struct {
	// Built-in reward functions (comma-separated list or array)
	// e.g. ["accuracy", "format", "length_penalty", "tool_call_correctness"]
	RewardFunctions []string `json:"reward_functions"`

	// Weights for each reward function. Keys match RewardFunctions items.
	Weights map[string]float64 `json:"weights,omitempty"`

	// Custom reward model endpoint (optional)
	RewardModelEndpoint string `json:"reward_model_endpoint,omitempty"`

	// KL penalty coefficient (for PPO / GRPO)
	KLCoeff float64 `json:"kl_coeff,omitempty"`

	// ClipRange for PPO
	ClipRange float64 `json:"clip_range,omitempty"`

	// AdvantageNorm normalizes advantages before update
	AdvantageNorm bool `json:"advantage_norm,omitempty"`

	// ExtraParams for any additional reward config
	ExtraParams map[string]interface{} `json:"extra_params,omitempty"`
}

// ─── DB Row ─────────────────────────────────────────────────────────────────

type FineTuningJobRow struct {
	ID              string
	UserID          string
	BaseModel       string
	TrainingFile    string
	Method          string
	Hyperparameters map[string]interface{}
	RolloutScenario string
	RolloutConfig   map[string]interface{}
	RewardConfig    map[string]interface{}
	Status          string
	FineTunedModel  *string
	ErrorMessage    *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ─── API Response ───────────────────────────────────────────────────────────

type FineTuningJob struct {
	ID              string                 `json:"id"`
	Object          string                 `json:"object"`
	Model           string                 `json:"model"`
	TrainingFile    string                 `json:"training_file,omitempty"`
	Method          string                 `json:"method"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	RolloutScenario string                 `json:"rollout_scenario,omitempty"`
	RolloutConfig   map[string]interface{} `json:"rollout_config,omitempty"`
	RewardConfig    map[string]interface{} `json:"reward_config,omitempty"`
	Status          string                 `json:"status"`
	FineTunedModel  *string                `json:"fine_tuned_model,omitempty"`
	Error           *FineTuningJobError    `json:"error,omitempty"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
}

type FineTuningJobError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type FineTuningJobListResponse struct {
	Object string          `json:"object"`
	Data   []FineTuningJob `json:"data"`
}
