package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/xincxiong/model-inference-platform/backend/internal/semcache"
)

type Store struct {
	DB            *pgxpool.Pool
	Redis         *redis.Client
	SemanticCache *semcache.Cache
	S3            interface{ // minimal interface for upload/download; nil when disabled
		Upload(ctx context.Context, key string, contentType string, data []byte) error
		Download(ctx context.Context, key string) ([]byte, string, error)
		Delete(ctx context.Context, key string) error
	}
}
