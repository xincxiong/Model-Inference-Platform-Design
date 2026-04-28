package store

import (
    "github.com/xincxiong/model-inference-platform/backend/internal/config"
    "github.com/xincxiong/model-inference-platform/backend/internal/storage"
)

// NewS3 creates an S3 storage client when enabled; returns nil when disabled.
func NewS3(cfg config.S3Config) (*storage.S3Store, error) {
    if !cfg.Enabled {
        return nil, nil
    }
    return storage.NewS3Store(storage.S3Config{
        Endpoint:     cfg.Endpoint,
        Bucket:       cfg.Bucket,
        Region:       cfg.Region,
        AccessKey:    cfg.AccessKey,
        SecretKey:    cfg.SecretKey,
        UsePathStyle: cfg.UsePathStyle,
    })
}
