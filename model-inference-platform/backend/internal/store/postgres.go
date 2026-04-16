package store

import (
	"context"
	"fmt"

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
		// --- Text-to-Text ---
		{"deepseek-ai/DeepSeek-V4", "DeepSeek V4", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming",
			"DeepSeek 最新旗舰对话模型，采用 MoE 架构，拥有超过 600B 总参数量（37B 激活），在代码、数学推理与多轮对话场景综合表现业内领先，支持 1M Token 超长上下文。",
			1.0, 2.0, 80, 90.1, 1000000},
		{"deepseek-ai/DeepSeek-R1", "DeepSeek R1", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming",
			"DeepSeek 强化学习推理模型，采用 Chain-of-Thought 自我反思机制，在数学奥林匹克、代码竞赛及科学推理任务上达到 o1 级别水准，适合需要深度逐步推理的复杂问题。",
			0.55, 2.19, 60, 87.5, 65536},
		{"Qwen/Qwen3.5-72B", "Qwen 3.5 72B", "text-to-text", "Alibaba",
			"function_calling,json_mode,streaming",
			"阿里云千问 3.5 系列 72B 旗舰模型，在中英双语理解、代码生成与指令跟随方面表现出色，支持 128K 上下文，具备 Function Calling 和结构化输出能力。",
			0.9, 0.9, 65, 86.3, 131072},
		{"Qwen/Qwen3.5-35B-A3B", "Qwen 3.5 35B-A3B", "text-to-text", "Alibaba",
			"function_calling,json_mode,streaming",
			"Qwen3.5 系列 35B-A3B 原生视觉语言模型，基于混合架构设计，融合了线性注意力机制与稀疏混合专家模型，实现了更高的推理效率。该模型的综合表现接近 Qwen3.5-27B。",
			0.5, 1.2, 90, 84.8, 131072},
		{"THUDM/GLM-5-32B", "GLM-5 32B", "text-to-text", "Zhipu AI",
			"function_calling,json_mode,streaming",
			"智谱 GLM-5 系列 32B 模型，具备强大的中文理解与生成能力，支持工具调用、代码执行和结构化输出，适合企业级中文 NLP 场景与知识问答应用。",
			0.5, 0.5, 70, 82.0, 32768},
		{"meta-llama/Llama-4-70B", "Llama 4 70B", "text-to-text", "Meta",
			"function_calling,json_mode,streaming",
			"Meta 最新 Llama 4 系列 70B 开源模型，采用 MoE 架构显著提升推理效率，在英文基准测试中表现优异，支持多轮对话与 Function Calling，适合全球化开源部署场景。",
			0.8, 0.8, 55, 85.7, 131072},
		{"moonshot-ai/Kimi-K2.5", "Kimi K2.5", "text-to-text", "Moonshot AI",
			"function_calling,json_mode,streaming",
			"月之暗面 Kimi K2.5 模型，拥有业内领先的 256K 超长上下文窗口，擅长长文档理解、跨文档推理和多轮长对话，在法律、金融、科研等长文本密集型场景表现突出。",
			1.2, 2.5, 45, 88.0, 262144},
		{"bytedance/Doubao-2.0-Pro", "豆包 2.0 Pro", "text-to-text", "ByteDance",
			"function_calling,json_mode,streaming",
			"字节跳动豆包 2.0 Pro 模型，专为中文生产环境优化，在创意写作、内容生成与用户对话场景下具备极高的流畅度与指令跟随能力，支持 128K 上下文与工具调用。",
			0.6, 1.2, 75, 83.5, 131072},

		// --- Vision (多模态) ---
		{"Qwen/Qwen2.5-VL-72B", "Qwen2.5 VL 72B", "vision", "Alibaba",
			"vision,streaming",
			"阿里云千问 2.5 视觉语言旗舰模型，支持图像理解、OCR、图表解析与视频帧分析，在 OpenCompass 多模态榜单上位居开源前列，上下文支持混合图文输入。",
			1.5, 2.0, 40, 84.0, 32768},
		{"OpenGVLab/InternVL3-78B", "InternVL3 78B", "vision", "Shanghai AI Lab",
			"vision,streaming",
			"上海人工智能实验室 InternVL3 系列 78B 多模态大模型，在图像问答、视觉推理及医学影像理解等细分场景表现卓越，采用动态分辨率训练策略支持高清图像输入。",
			1.2, 1.8, 35, 82.5, 32768},

		// --- Embedding ---
		{"BAAI/bge-m3", "BGE-M3", "embedding", "BAAI",
			"multilingual",
			"北京智源研究院 BGE-M3 多语言稠密检索模型，支持 100+ 种语言，最大序列长度 8192 Token，同时兼容稠密、稀疏与多向量检索范式，是 RAG 场景的首选 Embedding 模型。",
			0.1, 0.0, 0, 0, 8192},
		{"jinaai/jina-embeddings-v3", "Jina Embeddings v3", "embedding", "Jina AI",
			"multilingual",
			"Jina AI 最新多语言嵌入模型，专为检索增强生成、语义搜索与文本相似度计算设计，支持任务自适应 LoRA 适配器，在 MTEB 多语言榜单上名列前茅。",
			0.1, 0.0, 0, 0, 8192},

		// --- Rerank ---
		{"BAAI/bge-reranker-v2-m3", "BGE Reranker v2", "rerank", "BAAI",
			"multilingual",
			"北京智源研究院 BGE 第二代多语言重排序模型，基于交叉编码器架构，能够精准评估查询与候选文档的相关性，适用于 RAG 检索管道中的二阶段精排。",
			0.1, 0.0, 0, 0, 8192},
		{"Qwen/Qwen3-Reranker", "Qwen3 Reranker", "rerank", "Alibaba",
			"multilingual",
			"阿里云千问 3 系列重排序专用模型，在中英双语检索任务中表现优异，支持指令引导的细粒度相关性打分，可显著提升 RAG 系统的最终回答质量。",
			0.1, 0.0, 0, 0, 8192},

		// --- Text-to-Image ---
		{"black-forest-labs/FLUX.1-dev", "FLUX.1 Dev", "text-to-image", "Black Forest Labs",
			"",
			"Black Forest Labs 出品的 FLUX.1 Dev 扩散模型，采用 Flow Matching 训练范式，在文本对齐、构图细节与艺术风格多样性方面超越同期主流文生图模型，适合高质量创意图像生成。",
			0.0, 0.03, 0, 0, 0},
		{"stabilityai/SDXL", "Stable Diffusion XL", "text-to-image", "Stability AI",
			"",
			"Stability AI 推出的 SDXL 模型，参数量达 3.5B，相比 SD 1.x / 2.x 大幅提升了图像质量与提示词遵循度，支持 1024×1024 原生分辨率输出，适合商业级图像创作工作流。",
			0.0, 0.02, 0, 0, 0},

		// --- Text-to-Video ---
		{"THUDM/CogVideoX-5B", "CogVideoX 5B", "text-to-video", "Zhipu AI",
			"",
			"智谱 CogVideoX 5B 文生视频模型，基于 3D VAE 与专家级 Transformer 架构，支持生成高一致性、高清晰度的短视频片段，在运动流畅性和场景理解方面表现出色。",
			0.0, 0.1, 0, 0, 0},

		// --- Speech ---
		{"FunAudioLLM/CosyVoice2-0.5B", "CosyVoice 2", "speech", "Alibaba",
			"tts,multilingual",
			"阿里云 CosyVoice 2 文字转语音模型，支持中英日韩等多语言零样本声音克隆，输出自然度媲美真人发音，延迟极低，适合实时语音对话、有声内容生成等场景。",
			0.015, 0.0, 0, 0, 0},
		{"FunAudioLLM/SenseVoice-Large", "SenseVoice Large", "speech", "Alibaba",
			"asr,multilingual",
			"阿里云 SenseVoice Large 自动语音识别模型，支持 50+ 种语言，具备情感识别与声学事件检测能力，在中文普通话和方言识别场景中准确率业内领先。",
			0.01, 0.0, 0, 0, 0},
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
