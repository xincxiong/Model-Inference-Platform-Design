package storage

import (
    "bytes"
    "context"
    "io"
    "time"

    minio "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds runtime configuration for S3-compatible storage.
type S3Config struct {
    Endpoint     string
    Bucket       string
    Region       string
    AccessKey    string
    SecretKey    string
    UsePathStyle bool
}

// S3Store wraps a MinIO (S3-compatible) client.
type S3Store struct {
    client *minio.Client
    cfg    S3Config
}

// NewS3Store creates a new S3 store.
func NewS3Store(cfg S3Config) (*S3Store, error) {
    cli, err := minio.New(cfg.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
        Region: cfg.Region,
        Secure: !isHTTP(cfg.Endpoint),
        BucketLookup: func() minio.BucketLookupType {
            if cfg.UsePathStyle {
                return minio.BucketLookupPath
            }
            return minio.BucketLookupAuto
        }(),
    })
    if err != nil {
        return nil, err
    }

    return &S3Store{client: cli, cfg: cfg}, nil
}

// Upload uploads content to S3 with the given object key and content type.
func (s *S3Store) Upload(ctx context.Context, key string, contentType string, data []byte) error {
    reader := bytes.NewReader(data)
    _, err := s.client.PutObject(ctx, s.cfg.Bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
        ContentType:  contentType,
        SendContentMd5: true,
    })
    return err
}

// Download retrieves the object content from S3.
func (s *S3Store) Download(ctx context.Context, key string) ([]byte, string, error) {
    obj, err := s.client.GetObject(ctx, s.cfg.Bucket, key, minio.GetObjectOptions{})
    if err != nil {
        return nil, "", err
    }
    defer obj.Close()
    buf, err := io.ReadAll(obj)
    if err != nil {
        return nil, "", err
    }
    stat, err := obj.Stat()
    if err != nil {
        return buf, "application/octet-stream", nil
    }
    return buf, stat.ContentType, nil
}

// Delete removes an object from S3.
func (s *S3Store) Delete(ctx context.Context, key string) error {
    return s.client.RemoveObject(ctx, s.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

// HealthCheck performs a simple bucket existence check.
func (s *S3Store) HealthCheck(ctx context.Context) error {
    exists, err := s.client.BucketExists(ctx, s.cfg.Bucket)
    if err != nil {
        return err
    }
    if !exists {
        return s.client.MakeBucket(ctx, s.cfg.Bucket, minio.MakeBucketOptions{Region: s.cfg.Region})
    }
    return nil
}

// isHTTP detects plain HTTP endpoint.
func isHTTP(endpoint string) bool {
    return len(endpoint) >= 7 && endpoint[:7] == "http://"
}

// PresignDownload returns a presigned URL for download with expiry.
func (s *S3Store) PresignDownload(ctx context.Context, key string, expires time.Duration) (string, error) {
    url, err := s.client.PresignedGetObject(ctx, s.cfg.Bucket, key, expires, nil)
    if err != nil {
        return "", err
    }
    return url.String(), nil
}
