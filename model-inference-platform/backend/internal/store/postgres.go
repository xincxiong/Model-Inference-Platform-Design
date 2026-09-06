package store

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	config.MaxConns = 20
	config.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func RunMigrations(ctx context.Context, db *pgxpool.Pool) error {
	migration := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS api_keys (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		key_hash VARCHAR(64) NOT NULL UNIQUE,
		key_prefix VARCHAR(12) NOT NULL,
		service_tier VARCHAR(20) NOT NULL DEFAULT 'default',
		expires_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);

	CREATE TABLE IF NOT EXISTS models (
		id VARCHAR(128) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		model_type VARCHAR(50) NOT NULL DEFAULT 'text-to-text',
		provider VARCHAR(128) NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		input_price NUMERIC(12,6) NOT NULL DEFAULT 0,
		output_price NUMERIC(12,6) NOT NULL DEFAULT 0,
		max_context INT NOT NULL DEFAULT 4096,
		speed NUMERIC(8,1) NOT NULL DEFAULT 0,
		quality_score NUMERIC(5,1) NOT NULL DEFAULT 0,
		features TEXT NOT NULL DEFAULT '',
		status VARCHAR(20) NOT NULL DEFAULT 'active'
	);
	ALTER TABLE models ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
	ALTER TABLE models ADD COLUMN IF NOT EXISTS speed NUMERIC(8,1) NOT NULL DEFAULT 0;
	ALTER TABLE models ADD COLUMN IF NOT EXISTS quality_score NUMERIC(5,1) NOT NULL DEFAULT 0;
	ALTER TABLE models ADD COLUMN IF NOT EXISTS features TEXT NOT NULL DEFAULT '';
	ALTER TABLE models ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
	-- engine_type: vllm | sglang | mock | custom (user-configurable)
	ALTER TABLE models ADD COLUMN IF NOT EXISTS engine_type VARCHAR(32) NOT NULL DEFAULT '';
	-- engine_addr: custom engine endpoint URL (for engine_type='custom')
	ALTER TABLE models ADD COLUMN IF NOT EXISTS engine_addr TEXT NOT NULL DEFAULT '';

	CREATE TABLE IF NOT EXISTS responses (
		id VARCHAR(64) PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		model VARCHAR(128) NOT NULL,
		input JSONB NOT NULL DEFAULT '[]',
		output JSONB NOT NULL DEFAULT '[]',
		output_text TEXT NOT NULL DEFAULT '',
		input_tokens INT NOT NULL DEFAULT 0,
		output_tokens INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_responses_user ON responses(user_id, created_at DESC);

	CREATE TABLE IF NOT EXISTS billing_accounts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		balance NUMERIC(14,6) NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS usage_records (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		api_key_id UUID NOT NULL,
		model VARCHAR(128) NOT NULL,
		input_tokens INT NOT NULL DEFAULT 0,
		output_tokens INT NOT NULL DEFAULT 0,
		cost NUMERIC(14,8) NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_usage_user_date ON usage_records(user_id, created_at DESC);

	CREATE TABLE IF NOT EXISTS promo_codes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		code VARCHAR(64) UNIQUE NOT NULL,
		amount NUMERIC(14,6) NOT NULL,
		used BOOLEAN NOT NULL DEFAULT FALSE,
		used_by_id UUID REFERENCES users(id),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Seed a default admin user if none exists
	INSERT INTO users (id, email, name)
	VALUES ('00000000-0000-0000-0000-000000000001', 'admin@example.com', 'Admin')
	ON CONFLICT (email) DO NOTHING;

	INSERT INTO billing_accounts (user_id, balance)
	VALUES ('00000000-0000-0000-0000-000000000001', 100.0)
	ON CONFLICT (user_id) DO NOTHING;

	-- Seed a default dev API key (sk-dev-key-00000000)
	-- SHA-256 of "sk-dev-key-00000000" = 5b9c3f6e8a2d1047c8e5f3b6a9d2c1e4f7b8a3d6c9e2f5b8a1d4c7e0f3b6a9d
	INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, service_tier)
	VALUES (
		'00000000-0000-0000-0000-000000000002',
		'00000000-0000-0000-0000-000000000001',
		'Default Dev Key',
		encode(sha256('sk-dev-key-00000000'::bytea), 'hex'),
		'sk-dev-key',
		'default'
	)
	ON CONFLICT (id) DO NOTHING;

	-- Seed a sample promo code
	INSERT INTO promo_codes (code, amount)
	VALUES ('WELCOME50', 50.0)
	ON CONFLICT (code) DO NOTHING;

	CREATE TABLE IF NOT EXISTS dedicated_endpoints (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		model_name VARCHAR(128) NOT NULL,
		flavor_name VARCHAR(32) NOT NULL DEFAULT 'base',
		gpu_type VARCHAR(64) NOT NULL,
		gpu_count INT NOT NULL DEFAULT 1,
		region VARCHAR(64) NOT NULL DEFAULT 'cn-east-1',
		min_replicas INT NOT NULL DEFAULT 0,
		max_replicas INT NOT NULL DEFAULT 4,
		scaling_policy JSONB NOT NULL DEFAULT '{}',
		routing_prefix VARCHAR(48) NOT NULL UNIQUE,
		status VARCHAR(32) NOT NULL DEFAULT 'running',
		current_replicas INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_dedicated_endpoints_user ON dedicated_endpoints(user_id);
	-- idempotent HAMi GPU virtualization column additions
	ALTER TABLE dedicated_endpoints ADD COLUMN IF NOT EXISTS gpu_memory_mib INT NOT NULL DEFAULT 0;
	ALTER TABLE dedicated_endpoints ADD COLUMN IF NOT EXISTS gpu_cores INT NOT NULL DEFAULT 0;
	ALTER TABLE dedicated_endpoints ADD COLUMN IF NOT EXISTS scheduler_policy VARCHAR(32) NOT NULL DEFAULT 'binpack';
	ALTER TABLE dedicated_endpoints ADD COLUMN IF NOT EXISTS topology_aware BOOLEAN NOT NULL DEFAULT FALSE;
	ALTER TABLE dedicated_endpoints ADD COLUMN IF NOT EXISTS hard_isolation BOOLEAN NOT NULL DEFAULT FALSE;
	ALTER TABLE dedicated_endpoints ADD COLUMN IF NOT EXISTS scheduled_node VARCHAR(255) NOT NULL DEFAULT '';
	ALTER TABLE dedicated_endpoints ADD COLUMN IF NOT EXISTS physical_gpu_id INT NOT NULL DEFAULT 0;

	CREATE TABLE IF NOT EXISTS fine_tuning_jobs (
		id VARCHAR(64) PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		base_model VARCHAR(128) NOT NULL,
		training_file VARCHAR(128) NOT NULL DEFAULT '',
		method VARCHAR(32) NOT NULL DEFAULT 'lora',
		hyperparameters JSONB NOT NULL DEFAULT '{}',
		rollout_scenario VARCHAR(32) NOT NULL DEFAULT '',
		rollout_config JSONB NOT NULL DEFAULT '{}',
		reward_config JSONB NOT NULL DEFAULT '{}',
		status VARCHAR(32) NOT NULL DEFAULT 'queued',
		fine_tuned_model VARCHAR(128),
		error_message TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_fine_tuning_jobs_user ON fine_tuning_jobs(user_id, created_at DESC);
	-- idempotent column additions for existing databases
	ALTER TABLE fine_tuning_jobs ADD COLUMN IF NOT EXISTS rollout_scenario VARCHAR(32) NOT NULL DEFAULT '';
	ALTER TABLE fine_tuning_jobs ADD COLUMN IF NOT EXISTS rollout_config JSONB NOT NULL DEFAULT '{}';
	ALTER TABLE fine_tuning_jobs ADD COLUMN IF NOT EXISTS reward_config JSONB NOT NULL DEFAULT '{}';

	-- Files table (OpenAI-compatible /v1/files)
	CREATE TABLE IF NOT EXISTS files (
		id VARCHAR(64) PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		filename VARCHAR(512) NOT NULL,
		purpose VARCHAR(64) NOT NULL DEFAULT 'batch',
		bytes BIGINT NOT NULL DEFAULT 0,
		checksum VARCHAR(64) NOT NULL DEFAULT '',
		content BYTEA NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_files_user ON files(user_id, created_at DESC);

	-- Batches table (OpenAI-compatible /v1/batches)
	CREATE TABLE IF NOT EXISTS batches (
		id VARCHAR(64) PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		input_file_id VARCHAR(64) NOT NULL,
		endpoint VARCHAR(128) NOT NULL,
		completion_window VARCHAR(16) NOT NULL DEFAULT '24h',
		status VARCHAR(32) NOT NULL DEFAULT 'validating',
		output_file_id VARCHAR(64),
		error_file_id VARCHAR(64),
		request_counts JSONB NOT NULL DEFAULT '{"total":0,"completed":0,"failed":0}',
		metadata JSONB NOT NULL DEFAULT '{}',
		cancelled_at TIMESTAMPTZ,
		cancelling_at TIMESTAMPTZ,
		completed_at TIMESTAMPTZ,
		expired_at TIMESTAMPTZ,
		failed_at TIMESTAMPTZ,
		in_progress_at TIMESTAMPTZ,
		expires_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_batches_user ON batches(user_id, created_at DESC);

	-- Datasets table (/v1/datasets)
	CREATE TABLE IF NOT EXISTS datasets (
		id VARCHAR(64) PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		file_id VARCHAR(64),
		num_rows BIGINT NOT NULL DEFAULT 0,
		size_bytes BIGINT NOT NULL DEFAULT 0,
		metadata JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_datasets_user ON datasets(user_id, created_at DESC);

	-- Deployments table (/v1/deployments)
	-- billing_mode: token | tpu | unit
	CREATE TABLE IF NOT EXISTS deployments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		model_name VARCHAR(128) NOT NULL,
		billing_mode VARCHAR(16) NOT NULL DEFAULT 'token',
		min_replicas INT NOT NULL DEFAULT 0,
		max_replicas INT NOT NULL DEFAULT 3,
		status VARCHAR(32) NOT NULL DEFAULT 'provisioning',
		endpoint TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_deployments_user ON deployments(user_id, created_at DESC);

	-- Model versions table (multi-version co-existence + A/B traffic split + smooth rollout)
	CREATE TABLE IF NOT EXISTS model_versions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		model_id VARCHAR(128) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
		version VARCHAR(64) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		changelog TEXT NOT NULL DEFAULT '',
		backend_addr TEXT NOT NULL DEFAULT '',
		traffic_pct INT NOT NULL DEFAULT 0 CHECK (traffic_pct >= 0 AND traffic_pct <= 100),
		status VARCHAR(32) NOT NULL DEFAULT 'inactive',
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		created_by UUID REFERENCES users(id) ON DELETE SET NULL,
		activated_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE (model_id, version)
	);
	CREATE INDEX IF NOT EXISTS idx_model_versions_model ON model_versions(model_id);
	-- idempotent column additions (safe on existing DB)
	ALTER TABLE model_versions ADD COLUMN IF NOT EXISTS changelog TEXT NOT NULL DEFAULT '';

	-- Org members table (/api/members)
	-- Lightweight invite-based membership; org_owner_id = the user who owns the workspace
	CREATE TABLE IF NOT EXISTS org_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		org_owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		invite_email VARCHAR(255) NOT NULL,
		role VARCHAR(16) NOT NULL DEFAULT 'member',
		status VARCHAR(16) NOT NULL DEFAULT 'pending',
		invited_by UUID NOT NULL REFERENCES users(id),
		joined_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_org_members_owner ON org_members(org_owner_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_org_members_unique ON org_members(org_owner_id, invite_email);

	-- ─── pool_skus: 算力池商品目录（包年包月 SKU） ──────────────────────────
	CREATE TABLE IF NOT EXISTS pool_skus (
		id VARCHAR(64) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		gpu_type VARCHAR(64) NOT NULL,
		gpu_count INT NOT NULL,
		region VARCHAR(32) NOT NULL DEFAULT 'cn-east-1',
		sharing_mode VARCHAR(32) NOT NULL DEFAULT 'shared-fifo',
		term VARCHAR(16) NOT NULL,
		term_months INT NOT NULL,
		hourly_list_price NUMERIC(12,4) NOT NULL,
		term_price NUMERIC(12,4) NOT NULL,
		discount_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
		sla_class VARCHAR(32) NOT NULL DEFAULT 'standard',
		active BOOLEAN NOT NULL DEFAULT TRUE,
		sort_order INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_pool_skus_active ON pool_skus(active, sort_order);

	-- ─── pool_subscriptions: 用户订阅记录 ──────────────────────────────────
	CREATE TABLE IF NOT EXISTS pool_subscriptions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		sku_id VARCHAR(64) NOT NULL REFERENCES pool_skus(id),
		pool_id VARCHAR(64),
		total_amount NUMERIC(12,4) NOT NULL,
		currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
		start_at TIMESTAMPTZ NOT NULL,
		end_at TIMESTAMPTZ NOT NULL,
		auto_renew BOOLEAN NOT NULL DEFAULT FALSE,
		status VARCHAR(16) NOT NULL DEFAULT 'pending',
		payment_status VARCHAR(16) NOT NULL DEFAULT 'pending',
		renewed_from_id UUID REFERENCES pool_subscriptions(id),
		trial BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_pool_subs_user ON pool_subscriptions(user_id);
	CREATE INDEX IF NOT EXISTS idx_pool_subs_status ON pool_subscriptions(status);
	CREATE INDEX IF NOT EXISTS idx_pool_subs_end ON pool_subscriptions(end_at);

	-- ─── pool_invoices: 订阅账单 ───────────────────────────────────────────
	CREATE TABLE IF NOT EXISTS pool_invoices (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		subscription_id UUID NOT NULL REFERENCES pool_subscriptions(id) ON DELETE CASCADE,
		period_start TIMESTAMPTZ NOT NULL,
		period_end TIMESTAMPTZ NOT NULL,
		amount NUMERIC(12,4) NOT NULL,
		status VARCHAR(16) NOT NULL DEFAULT 'pending',
		due_at TIMESTAMPTZ NOT NULL,
		paid_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_pool_invoices_sub ON pool_invoices(subscription_id);
	CREATE INDEX IF NOT EXISTS idx_pool_invoices_status ON pool_invoices(status);

	-- ─── compute_pools: 算力池实例 (从订阅创建的运营资源) ──────────────────────
	CREATE TABLE IF NOT EXISTS compute_pools (
		id VARCHAR(64) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		gpu_type VARCHAR(64) NOT NULL,
		gpu_count INT NOT NULL,
		region VARCHAR(32) NOT NULL DEFAULT 'cn-east-1',
		sharing_mode VARCHAR(32) NOT NULL DEFAULT 'shared-fifo',
		subscription_id UUID NOT NULL REFERENCES pool_subscriptions(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		volcano_queue VARCHAR(128) NOT NULL DEFAULT '',
		scheduler_policy VARCHAR(32) NOT NULL DEFAULT 'binpack',
		hard_isolation BOOLEAN NOT NULL DEFAULT FALSE,
		status VARCHAR(16) NOT NULL DEFAULT 'active',
		used_gpu INT NOT NULL DEFAULT 0,
		service_start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		service_end_at TIMESTAMPTZ,
		sla_class VARCHAR(32) NOT NULL DEFAULT 'standard',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_compute_pools_user ON compute_pools(user_id);
	CREATE INDEX IF NOT EXISTS idx_compute_pools_sub ON compute_pools(subscription_id);
	-- 一个订阅最多一个 active 池 (re-purchase 会先 archive 旧的)
	CREATE UNIQUE INDEX IF NOT EXISTS idx_compute_pools_sub_active ON compute_pools(subscription_id) WHERE status = 'active';

	-- ─── fine_tuning_jobs: 增加算力资源字段 ──────────────────────────────────
	ALTER TABLE fine_tuning_jobs ADD COLUMN IF NOT EXISTS pool_id VARCHAR(64);
	ALTER TABLE fine_tuning_jobs ADD COLUMN IF NOT EXISTS gpu_request INT NOT NULL DEFAULT 0;
	ALTER TABLE fine_tuning_jobs ADD COLUMN IF NOT EXISTS rollout_pool_id VARCHAR(64);
	CREATE INDEX IF NOT EXISTS idx_fine_tuning_jobs_pool ON fine_tuning_jobs(pool_id) WHERE pool_id IS NOT NULL;

	-- ─── Cleanup: drop stale used_gpu column (P1 E1) ────────────────────────
	-- Previously used as a stored counter but never updated. Capacity is now
	-- computed live via SUM(gpu_request) FILTER (WHERE status IN active set).
	ALTER TABLE compute_pools DROP COLUMN IF EXISTS used_gpu;
	`

	_, err := db.Exec(ctx, migration)
	return err
}

// SeedModels populates the models table with platform-supported models
// covering all 7 model types defined in the product plan.
func SeedModels(ctx context.Context, db *pgxpool.Pool) {
	models := []struct {
		ID, Name, Type, Provider, Features, Description string
		InPrice, OutPrice, Speed, Quality               float64
		MaxCtx                                          int
	}{
		// ── Text-to-Text ─────────────────────────────────────────────────────
		{"deepseek-ai/DeepSeek-V3", "DeepSeek V3", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming",
			"DeepSeek V3 旗舰对话模型，采用 MoE 架构（671B 总参数 / 37B 激活），在代码、数学推理与多轮对话综合表现业内领先，支持 128K 上下文与工具调用。",
			0.27, 1.10, 82, 91.5, 131072},
		{"deepseek-ai/DeepSeek-R1", "DeepSeek R1", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming",
			"DeepSeek 强化学习推理模型，Chain-of-Thought 自我反思机制，在数学、代码竞赛及科学推理任务上达到 o1 级水准，适合深度逐步推理的复杂问题。",
			0.55, 2.19, 60, 87.5, 65536},
		{"deepseek-ai/DeepSeek-R1-0528", "DeepSeek R1 0528", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming",
			"DeepSeek R1 2025-05 更新版，进一步强化数学与代码推理能力，AIME 2025 得分超越 o3 mini，同等参数量下推理准确率大幅提升。",
			0.55, 2.19, 58, 89.2, 65536},
		{"Qwen/Qwen3-72B", "Qwen3 72B", "text-to-text", "Alibaba",
			"function_calling,json_mode,streaming,thinking",
			"阿里云千问 3 系列 72B 旗舰模型，支持混合思考模式（Thinking/Non-thinking 动态切换），在中英双语理解、代码生成与指令跟随方面表现出色，支持 128K 上下文。",
			0.4, 1.2, 68, 88.0, 131072},
		{"Qwen/Qwen3-30B-A3B", "Qwen3 30B-A3B", "text-to-text", "Alibaba",
			"function_calling,json_mode,streaming,thinking",
			"Qwen3 MoE 高效推理模型（30B 总参数 / 3B 激活），推理速度快、成本低，综合能力接近 72B Dense 模型，适合高并发在线推理场景。",
			0.22, 0.88, 95, 85.5, 131072},
		{"Qwen/Qwen3-8B", "Qwen3 8B", "text-to-text", "Alibaba",
			"function_calling,json_mode,streaming,thinking",
			"Qwen3 轻量化 8B 模型，支持混合思考模式，在同等参数量级中综合能力领先，适合边缘部署与低延迟推理场景。",
			0.06, 0.06, 120, 78.5, 131072},
		{"meta-llama/Llama-3.3-70B-Instruct", "Llama 3.3 70B Instruct", "text-to-text", "Meta",
			"function_calling,json_mode,streaming",
			"Meta Llama 3.3 系列 70B 指令微调模型，在英文基准上达到或超越 Llama 3.1 405B，支持多轮对话与 Function Calling，是性价比最高的开源英文模型之一。",
			0.59, 0.79, 68, 87.0, 131072},
		{"meta-llama/Llama-4-Scout-17B-16E", "Llama 4 Scout 17B-16E", "text-to-text", "Meta",
			"function_calling,json_mode,streaming,vision",
			"Meta Llama 4 Scout 原生多模态 MoE 模型（17B × 16 专家），支持图文混合输入，10M Token 超长上下文，在效率与能力之间取得极佳平衡，适合多模态 RAG 场景。",
			0.17, 0.17, 110, 84.0, 10000000},
		{"meta-llama/Llama-4-Maverick-17B-128E", "Llama 4 Maverick 17B-128E", "text-to-text", "Meta",
			"function_calling,json_mode,streaming,vision",
			"Meta Llama 4 Maverick 旗舰 MoE 模型（17B × 128 专家），多模态能力强劲，在图像理解与长文本推理方面对标 GPT-4o，支持 1M Token 上下文。",
			0.27, 0.85, 75, 87.5, 1000000},
		{"mistralai/Mistral-Small-3.1-24B", "Mistral Small 3.1 24B", "text-to-text", "Mistral AI",
			"function_calling,json_mode,streaming,vision",
			"Mistral Small 3.1 24B 轻量多模态模型，支持图文输入与 128K 上下文，Apache 2.0 完全开源，在同等参数量中推理速度与质量兼顾，适合本地私有化部署。",
			0.1, 0.3, 115, 81.0, 131072},
		{"mistralai/Mistral-Large-2411", "Mistral Large 2411", "text-to-text", "Mistral AI",
			"function_calling,json_mode,streaming",
			"Mistral Large 2411 旗舰模型，128K 上下文，在代码、推理与多语言任务上综合性能位居开源前列，支持 Function Calling 与 System Prompt 精细控制。",
			2.0, 6.0, 50, 86.5, 131072},
		{"google/Gemma-3-27B-IT", "Gemma 3 27B IT", "text-to-text", "Google",
			"function_calling,json_mode,streaming,vision",
			"Google Gemma 3 系列 27B 指令微调模型，原生支持图文多模态输入，128K 上下文，Apache 2.0 开源，在多项学术基准上超越同级开源模型。",
			0.3, 0.7, 85, 83.5, 131072},
		{"THUDM/GLM-4-32B-0520", "GLM-4 32B 0520", "text-to-text", "Zhipu AI",
			"function_calling,json_mode,streaming,thinking",
			"智谱 GLM-4 32B 2025 更新版，支持 Think 深度推理模式，强化了中文理解与代码执行能力，适合企业级中文 NLP 场景、智能客服与 Agent 任务。",
			0.5, 0.5, 72, 83.0, 131072},
		{"moonshot-ai/Kimi-K2", "Kimi K2", "text-to-text", "Moonshot AI",
			"function_calling,json_mode,streaming",
			"月之暗面 Kimi K2 MoE 模型（1T 总参数 / 32B 激活），在 Agentic 任务、工具调用与代码生成场景综合表现优异，完全开源，支持 128K 上下文。",
			0.6, 2.5, 55, 89.5, 131072},
		{"bytedance/Doubao-Pro-32K", "豆包 Pro 32K", "text-to-text", "ByteDance",
			"function_calling,json_mode,streaming",
			"字节跳动豆包 Pro 系列，专为中文生产环境优化，在创意写作、内容生成与用户对话场景下具备极高流畅度与指令跟随能力，支持 32K 上下文与工具调用。",
			0.8, 1.6, 78, 83.5, 32768},
		{"microsoft/Phi-4", "Phi-4 14B", "text-to-text", "Microsoft",
			"function_calling,json_mode,streaming",
			"微软 Phi-4 14B 小型语言模型，在数学推理与 STEM 知识问答上表现超越同参数量级模型，适合资源受限的边缘设备部署与教育类应用场景。",
			0.07, 0.14, 130, 80.5, 16384},
		{"nvidia/Llama-3.1-Nemotron-70B-Instruct", "Nemotron 70B Instruct", "text-to-text", "NVIDIA",
			"function_calling,json_mode,streaming",
			"NVIDIA 基于 Llama 3.1 微调的 Nemotron 70B 对话模型，通过 RLHF 强化人类偏好对齐，在 MT-Bench 对话评测中得分居开源模型前列。",
			0.35, 0.4, 60, 85.0, 131072},

		// ── Code ─────────────────────────────────────────────────────────────
		{"deepseek-ai/DeepSeek-Coder-V2-Instruct", "DeepSeek Coder V2", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming",
			"DeepSeek Coder V2 236B MoE 代码专用模型，支持 338 种编程语言，在 HumanEval、SWE-bench 等代码基准上超越 GPT-4 Turbo，适合代码补全、调试与 Code Review 场景。",
			0.14, 0.28, 70, 90.5, 131072},
		{"Qwen/Qwen2.5-Coder-32B-Instruct", "Qwen2.5 Coder 32B", "text-to-text", "Alibaba",
			"function_calling,json_mode,streaming",
			"阿里云千问 2.5 代码专用 32B 模型，在 HumanEval 上达到 92.7%，支持代码解释、单元测试生成与多语言翻译，适合 IDE 插件与代码 Agent 场景。",
			0.12, 0.12, 85, 89.0, 131072},

		// ── Vision (多模态) ────────────────────────────────────────────────────
		{"Qwen/Qwen2.5-VL-72B-Instruct", "Qwen2.5 VL 72B", "vision", "Alibaba",
			"vision,streaming",
			"阿里云千问 2.5 视觉语言旗舰模型，支持图像理解、OCR、图表解析与视频帧分析，在 OpenCompass 多模态榜单上位居开源前列，支持混合图文输入。",
			1.5, 2.0, 40, 84.0, 32768},
		{"Qwen/Qwen2.5-VL-7B-Instruct", "Qwen2.5 VL 7B", "vision", "Alibaba",
			"vision,streaming",
			"Qwen2.5-VL 轻量化 7B 多模态模型，支持图像与视频理解，适合对延迟和成本敏感的视觉问答与文档解析场景，可本地部署。",
			0.35, 0.5, 90, 76.0, 32768},
		{"OpenGVLab/InternVL3-78B", "InternVL3 78B", "vision", "Shanghai AI Lab",
			"vision,streaming",
			"上海人工智能实验室 InternVL3 系列 78B 多模态大模型，在图像问答、视觉推理及医学影像理解等细分场景表现卓越，采用动态分辨率训练支持高清图像输入。",
			1.2, 1.8, 35, 82.5, 32768},
		{"OpenGVLab/InternVL3-8B", "InternVL3 8B", "vision", "Shanghai AI Lab",
			"vision,streaming",
			"InternVL3 系列 8B 轻量多模态模型，支持高分辨率图像输入与视频理解，在同参数量多模态基准上名列前茅，适合低成本私有化多模态应用。",
			0.1, 0.1, 100, 74.5, 32768},
		{"meta-llama/Llama-3.2-90B-Vision-Instruct", "Llama 3.2 90B Vision", "vision", "Meta",
			"vision,streaming",
			"Meta Llama 3.2 90B 视觉语言模型，支持图像理解与文本交织输入，在 VQA、图表问答等视觉推理任务上表现优异，完全开源可私有化部署。",
			0.9, 0.9, 38, 80.0, 131072},

		// ── Embedding ─────────────────────────────────────────────────────────
		{"BAAI/bge-m3", "BGE-M3", "embedding", "BAAI",
			"multilingual",
			"北京智源研究院 BGE-M3 多语言稠密检索模型，支持 100+ 种语言，最大序列长度 8192 Token，兼容稠密、稀疏与多向量检索范式，是 RAG 场景首选 Embedding 模型。",
			0.1, 0.0, 0, 0, 8192},
		{"BAAI/bge-large-zh-v1.5", "BGE Large ZH v1.5", "embedding", "BAAI",
			"",
			"BGE 中文大型 Embedding 模型，专为中文语义检索优化，在 CMTEB 中文嵌入榜单上长期排名前列，适合中文知识库与企业搜索场景。",
			0.05, 0.0, 0, 0, 512},
		{"jinaai/jina-embeddings-v3", "Jina Embeddings v3", "embedding", "Jina AI",
			"multilingual",
			"Jina AI 多语言嵌入模型，支持任务自适应 LoRA 适配器，在 MTEB 多语言榜单上名列前茅，专为 RAG、语义搜索与文本相似度计算设计。",
			0.1, 0.0, 0, 0, 8192},
		{"Qwen/Qwen3-Embedding-8B", "Qwen3 Embedding 8B", "embedding", "Alibaba",
			"multilingual",
			"阿里云千问 3 系列文本嵌入模型，在 MTEB 榜单上刷新多项中英文检索记录，支持可变维度输出（256 / 512 / 1024 / 2048 / 4096），适合大规模向量检索场景。",
			0.05, 0.0, 0, 0, 32768},

		// ── Rerank ────────────────────────────────────────────────────────────
		{"BAAI/bge-reranker-v2-m3", "BGE Reranker v2", "rerank", "BAAI",
			"multilingual",
			"北京智源研究院 BGE 第二代多语言重排序模型，基于交叉编码器架构，能精准评估查询与候选文档的相关性，适用于 RAG 检索管道中的二阶段精排。",
			0.1, 0.0, 0, 0, 8192},
		{"Qwen/Qwen3-Reranker-8B", "Qwen3 Reranker 8B", "rerank", "Alibaba",
			"multilingual",
			"阿里云千问 3 系列重排序专用模型，在中英双语检索任务中表现优异，支持指令引导的细粒度相关性打分，可显著提升 RAG 系统回答质量。",
			0.1, 0.0, 0, 0, 32768},

		// ── Text-to-Image ─────────────────────────────────────────────────────
		{"black-forest-labs/FLUX.1-dev", "FLUX.1 Dev", "text-to-image", "Black Forest Labs",
			"",
			"Black Forest Labs FLUX.1 Dev 扩散模型，采用 Flow Matching 训练范式，在文本对齐、构图细节与艺术风格多样性方面超越同期主流文生图模型，适合高质量创意图像生成。",
			0.0, 0.03, 0, 0, 0},
		{"black-forest-labs/FLUX.1-schnell", "FLUX.1 Schnell", "text-to-image", "Black Forest Labs",
			"",
			"FLUX.1 Schnell 快速推理版本，Apache 2.0 完全开源，生成速度比 Dev 版快 10 倍以上，适合实时预览、批量生成与低延迟应用场景。",
			0.0, 0.01, 0, 0, 0},
		{"stabilityai/stable-diffusion-3.5-large", "Stable Diffusion 3.5 Large", "text-to-image", "Stability AI",
			"",
			"Stability AI SD 3.5 Large 8B 扩散 Transformer 模型，采用多模态扩散 Transformer（MMDiT）架构，在文本渲染、构图准确度与照片真实感方面大幅超越 SDXL。",
			0.0, 0.04, 0, 0, 0},

		// ── Text-to-Video ─────────────────────────────────────────────────────
		{"THUDM/CogVideoX-5B", "CogVideoX 5B", "text-to-video", "Zhipu AI",
			"",
			"智谱 CogVideoX 5B 文生视频模型，基于 3D VAE 与专家级 Transformer 架构，支持生成高一致性短视频，在运动流畅性和场景理解方面表现出色。",
			0.0, 0.1, 0, 0, 0},
		{"Wan-AI/Wan2.1-T2V-14B", "Wan2.1 T2V 14B", "text-to-video", "Alibaba",
			"",
			"阿里通义万象 Wan2.1 文生视频 14B 旗舰模型，支持 1080P 高清视频生成，在物理一致性、镜头运动与场景复杂度上达到商业级水准，完全开源。",
			0.0, 0.15, 0, 0, 0},
		{"hpcai-tech/Open-Sora-v2", "Open-Sora v2", "text-to-video", "HPC-AI Tech",
			"",
			"Open-Sora v2 开源文生视频模型，支持 720P 多时长视频生成，采用 Causal VAE 与 Diffusion Transformer 架构，完全复现商业 Sora 能力，可本地私有化部署。",
			0.0, 0.08, 0, 0, 0},

		// ── Speech ────────────────────────────────────────────────────────────
		{"FunAudioLLM/CosyVoice2-0.5B", "CosyVoice 2", "speech", "Alibaba",
			"tts,multilingual",
			"阿里云 CosyVoice 2 文字转语音模型，支持中英日韩多语言零样本声音克隆，输出自然度媲美真人，延迟极低，适合实时语音对话与有声内容生成。",
			0.015, 0.0, 0, 0, 0},
		{"FunAudioLLM/SenseVoice-Large", "SenseVoice Large", "speech", "Alibaba",
			"asr,multilingual",
			"阿里云 SenseVoice Large 自动语音识别模型，支持 50+ 种语言，具备情感识别与声学事件检测能力，中文普通话和方言识别准确率业内领先。",
			0.01, 0.0, 0, 0, 0},
		{"openai/whisper-large-v3", "Whisper Large v3", "speech", "OpenAI",
			"asr,multilingual",
			"OpenAI Whisper Large v3 开源语音识别模型，支持 99 种语言自动检测与转录，在多语言 ASR 基准上全面领先，完全开源可本地部署，适合字幕生成与会议记录场景。",
			0.006, 0.0, 0, 0, 0},
	}

	for _, m := range models {
		_, _ = db.Exec(ctx,
			`INSERT INTO models (id, name, model_type, provider, description, input_price, output_price, max_context, speed, quality_score, features)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			 ON CONFLICT (id) DO UPDATE SET
			   name=EXCLUDED.name, model_type=EXCLUDED.model_type, provider=EXCLUDED.provider,
			   description=EXCLUDED.description,
			   input_price=EXCLUDED.input_price, output_price=EXCLUDED.output_price,
			   max_context=EXCLUDED.max_context, speed=EXCLUDED.speed,
			   quality_score=EXCLUDED.quality_score, features=EXCLUDED.features`,
			m.ID, m.Name, m.Type, m.Provider, m.Description, m.InPrice, m.OutPrice, m.MaxCtx, m.Speed, m.Quality, m.Features)
	}
}

// SeedPoolSKUs populates the pool_skus table with sample monthly/quarterly/yearly plans
// across the platform's GPU catalog. Idempotent: re-running is a no-op.
func SeedPoolSKUs(ctx context.Context, db *pgxpool.Pool) {
	// Hourly list prices mirror endpoints page's GPU_OPTIONS for consistency.
	type skuSeed struct {
		ID, Name, Desc, GPUType, Region, SharingMode, Term string
		GPUCount, TermMonths                                 int
		HourlyPrice                                          float64
		DiscountPct                                          float64
		SLAClass                                             string
		SortOrder                                            int
	}
	seeds := []skuSeed{
		// A100 80GB × 4 - shared
		{"sku-a100-80-4-monthly", "A100 80GB × 4 共享池 月付", "4 张 A100 80GB，共享 FIFO 模式，适合中小团队 SFT/RL 实验。", "A100-80GB", "cn-east-1", "shared-fifo", "monthly", 4, 1, 14.0, 10, "standard", 10},
		{"sku-a100-80-4-quarterly", "A100 80GB × 4 共享池 季付", "4 张 A100 80GB，3 个月订阅，额外 15% 折扣。", "A100-80GB", "cn-east-1", "shared-fifo", "quarterly", 4, 3, 14.0, 25, "standard", 11},
		{"sku-a100-80-4-yearly", "A100 80GB × 4 共享池 年付", "4 张 A100 80GB，12 个月订阅，30% 折扣。", "A100-80GB", "cn-east-1", "shared-fifo", "yearly", 4, 12, 14.0, 30, "standard", 12},

		// H100 80GB × 8 - shared
		{"sku-h100-80-8-monthly", "H100 80GB × 8 共享池 月付", "8 张 H100 80GB，Hopper 架构，70B+ 模型微调首选。", "H100-80GB", "cn-east-1", "shared-fifo", "monthly", 8, 1, 46.4, 10, "standard", 20},
		{"sku-h100-80-8-quarterly", "H100 80GB × 8 共享池 季付", "8 张 H100 80GB，3 个月订阅，25% 折扣。", "H100-80GB", "cn-east-1", "shared-fifo", "quarterly", 8, 3, 46.4, 25, "enhanced", 21},
		{"sku-h100-80-8-yearly", "H100 80GB × 8 共享池 年付", "8 张 H100 80GB，12 个月订阅，40% 折扣。", "H100-80GB", "cn-east-1", "shared-fifo", "yearly", 8, 12, 46.4, 40, "enhanced", 22},

		// H100 80GB × 8 - exclusive (大模型团队)
		{"sku-h100-80-8-excl-monthly", "H100 80GB × 8 独占池 月付", "8 张 H100 80GB 独占一任务，GRPO/VAPO 全参微调专用。", "H100-80GB", "cn-east-1", "exclusive", "monthly", 8, 1, 46.4, 5, "enhanced", 30},
		{"sku-h100-80-8-excl-yearly", "H100 80GB × 8 独占池 年付", "8 张 H100 80GB 独占，12 个月订阅，35% 折扣。", "H100-80GB", "cn-east-1", "exclusive", "yearly", 8, 12, 46.4, 35, "enhanced", 31},

		// L40S 48GB × 4 - 性价比
		{"sku-l40s-48-4-monthly", "L40S 48GB × 4 共享池 月付", "4 张 L40S 48GB，推理优化型，性价比极高。", "L40S-48GB", "cn-east-1", "shared-fifo", "monthly", 4, 1, 6.4, 10, "standard", 40},
		{"sku-l40s-48-4-yearly", "L40S 48GB × 4 共享池 年付", "4 张 L40S 48GB，年付 30% 折扣。", "L40S-48GB", "cn-east-1", "shared-fifo", "yearly", 4, 12, 6.4, 30, "standard", 41},
	}

	for _, s := range seeds {
		hoursInTerm := float64(s.TermMonths) * 30 * 24
		listTotal := s.HourlyPrice * float64(s.GPUCount) * hoursInTerm
		termPrice := listTotal * (1 - s.DiscountPct/100)
		_, err := db.Exec(ctx, `
			INSERT INTO pool_skus (id, name, description, gpu_type, gpu_count, region, sharing_mode, term, term_months,
				hourly_list_price, term_price, discount_pct, sla_class, active, sort_order)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,TRUE,$14)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, description=EXCLUDED.description,
				hourly_list_price=EXCLUDED.hourly_list_price, term_price=EXCLUDED.term_price,
				discount_pct=EXCLUDED.discount_pct, sla_class=EXCLUDED.sla_class,
				active=EXCLUDED.active, sort_order=EXCLUDED.sort_order`,
			s.ID, s.Name, s.Desc, s.GPUType, s.GPUCount, s.Region, s.SharingMode, s.Term, s.TermMonths,
			s.HourlyPrice, termPrice, s.DiscountPct, s.SLAClass, s.SortOrder)
		if err != nil {
			// Log to stderr but don't block startup; seed is best-effort.
			fmt.Fprintf(os.Stderr, "seed pool_sku %s: %v\n", s.ID, err)
		}
	}
}
