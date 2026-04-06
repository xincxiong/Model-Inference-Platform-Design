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
		input_price NUMERIC(12,6) NOT NULL DEFAULT 0,
		output_price NUMERIC(12,6) NOT NULL DEFAULT 0,
		max_context INT NOT NULL DEFAULT 4096,
		speed NUMERIC(8,1) NOT NULL DEFAULT 0,
		quality_score NUMERIC(5,1) NOT NULL DEFAULT 0,
		features TEXT NOT NULL DEFAULT '',
		status VARCHAR(20) NOT NULL DEFAULT 'active'
	);

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
		status VARCHAR(32) NOT NULL DEFAULT 'queued',
		fine_tuned_model VARCHAR(128),
		error_message TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_fine_tuning_jobs_user ON fine_tuning_jobs(user_id, created_at DESC);
	`

	_, err := db.Exec(ctx, migration)
	return err
}

// SeedModels populates the models table with platform-supported models
// covering all 7 model types defined in the product plan.
func SeedModels(ctx context.Context, db *pgxpool.Pool) {
	models := []struct {
		ID, Name, Type, Provider, Features string
		InPrice, OutPrice, Speed, Quality  float64
		MaxCtx                             int
	}{
		// --- Text-to-Text ---
		{"deepseek-ai/DeepSeek-V4", "DeepSeek V4", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming", 1.0, 2.0, 80, 90.1, 1000000},
		{"deepseek-ai/DeepSeek-R1", "DeepSeek R1", "text-to-text", "DeepSeek",
			"function_calling,json_mode,streaming", 0.55, 2.19, 60, 87.5, 65536},
		{"Qwen/Qwen3.5-72B", "Qwen 3.5 72B", "text-to-text", "Alibaba",
			"function_calling,json_mode,streaming", 0.9, 0.9, 65, 86.3, 131072},
		{"THUDM/GLM-5-32B", "GLM-5 32B", "text-to-text", "Zhipu AI",
			"function_calling,json_mode,streaming", 0.5, 0.5, 70, 82.0, 32768},
		{"meta-llama/Llama-4-70B", "Llama 4 70B", "text-to-text", "Meta",
			"function_calling,json_mode,streaming", 0.8, 0.8, 55, 85.7, 131072},
		{"moonshot-ai/Kimi-K2.5", "Kimi K2.5", "text-to-text", "Moonshot AI",
			"function_calling,json_mode,streaming", 1.2, 2.5, 45, 88.0, 262144},
		{"bytedance/Doubao-2.0-Pro", "豆包 2.0 Pro", "text-to-text", "ByteDance",
			"function_calling,json_mode,streaming", 0.6, 1.2, 75, 83.5, 131072},

		// --- Vision (多模态) ---
		{"Qwen/Qwen2.5-VL-72B", "Qwen2.5 VL 72B", "vision", "Alibaba",
			"vision,streaming", 1.5, 2.0, 40, 84.0, 32768},
		{"OpenGVLab/InternVL3-78B", "InternVL3 78B", "vision", "Shanghai AI Lab",
			"vision,streaming", 1.2, 1.8, 35, 82.5, 32768},

		// --- Embedding ---
		{"BAAI/bge-m3", "BGE-M3", "embedding", "BAAI",
			"multilingual", 0.1, 0.0, 0, 0, 8192},
		{"jinaai/jina-embeddings-v3", "Jina Embeddings v3", "embedding", "Jina AI",
			"multilingual", 0.1, 0.0, 0, 0, 8192},

		// --- Rerank ---
		{"BAAI/bge-reranker-v2-m3", "BGE Reranker v2", "rerank", "BAAI",
			"multilingual", 0.1, 0.0, 0, 0, 8192},
		{"Qwen/Qwen3-Reranker", "Qwen3 Reranker", "rerank", "Alibaba",
			"multilingual", 0.1, 0.0, 0, 0, 8192},

		// --- Text-to-Image ---
		{"black-forest-labs/FLUX.1-dev", "FLUX.1 Dev", "text-to-image", "Black Forest Labs",
			"", 0.0, 0.03, 0, 0, 0},
		{"stabilityai/SDXL", "Stable Diffusion XL", "text-to-image", "Stability AI",
			"", 0.0, 0.02, 0, 0, 0},

		// --- Text-to-Video ---
		{"THUDM/CogVideoX-5B", "CogVideoX 5B", "text-to-video", "Zhipu AI",
			"", 0.0, 0.1, 0, 0, 0},

		// --- Speech ---
		{"FunAudioLLM/CosyVoice2-0.5B", "CosyVoice 2", "speech", "Alibaba",
			"tts,multilingual", 0.015, 0.0, 0, 0, 0},
		{"FunAudioLLM/SenseVoice-Large", "SenseVoice Large", "speech", "Alibaba",
			"asr,multilingual", 0.01, 0.0, 0, 0, 0},
	}

	for _, m := range models {
		_, _ = db.Exec(ctx,
			`INSERT INTO models (id, name, model_type, provider, input_price, output_price, max_context, speed, quality_score, features)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			 ON CONFLICT (id) DO UPDATE SET
			   name=EXCLUDED.name, model_type=EXCLUDED.model_type, provider=EXCLUDED.provider,
			   input_price=EXCLUDED.input_price, output_price=EXCLUDED.output_price,
			   max_context=EXCLUDED.max_context, speed=EXCLUDED.speed,
			   quality_score=EXCLUDED.quality_score, features=EXCLUDED.features`,
			m.ID, m.Name, m.Type, m.Provider, m.InPrice, m.OutPrice, m.MaxCtx, m.Speed, m.Quality, m.Features)
	}
}
