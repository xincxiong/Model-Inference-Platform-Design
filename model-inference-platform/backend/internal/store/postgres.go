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
	`

	_, err := db.Exec(ctx, migration)
	return err
}

func SeedModels(ctx context.Context, db *pgxpool.Pool) {
	models := []struct {
		ID, Name, Type, Provider string
		InPrice, OutPrice        float64
		MaxCtx                   int
	}{
		{"deepseek-ai/DeepSeek-V4", "DeepSeek V4", "text-to-text", "DeepSeek", 1.0, 2.0, 131072},
		{"deepseek-ai/DeepSeek-R1", "DeepSeek R1", "text-to-text", "DeepSeek", 0.55, 2.19, 65536},
		{"Qwen/Qwen3.5-72B", "Qwen 3.5 72B", "text-to-text", "Alibaba", 0.9, 0.9, 131072},
		{"THUDM/GLM-5-32B", "GLM-5 32B", "text-to-text", "Zhipu AI", 0.5, 0.5, 32768},
		{"meta-llama/Llama-4-70B", "Llama 4 70B", "text-to-text", "Meta", 0.8, 0.8, 131072},
		{"BAAI/bge-large-zh-v1.5", "BGE Large ZH", "embedding", "BAAI", 0.1, 0.0, 8192},
		{"BAAI/bge-reranker-v2-m3", "BGE Reranker v2", "rerank", "BAAI", 0.1, 0.0, 8192},
		{"black-forest-labs/FLUX.1-dev", "FLUX.1 Dev", "text-to-image", "Black Forest Labs", 0.0, 0.03, 0},
	}

	for _, m := range models {
		_, _ = db.Exec(ctx,
			`INSERT INTO models (id, name, model_type, provider, input_price, output_price, max_context)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING`,
			m.ID, m.Name, m.Type, m.Provider, m.InPrice, m.OutPrice, m.MaxCtx)
	}
}
