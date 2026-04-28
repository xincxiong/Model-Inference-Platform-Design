package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL    string `mapstructure:"DATABASE_URL"`
	RedisURL       string `mapstructure:"REDIS_URL"`
	ServerPort     int    `mapstructure:"SERVER_PORT"`
	ManagementPort int    `mapstructure:"MANAGEMENT_PORT"`
	GinMode        string `mapstructure:"GIN_MODE"`
	FrontendURL    string `mapstructure:"FRONTEND_URL"`
	Volcano        VolcanoConfig
	S3             S3Config
	SemanticCache  SemanticCacheConfig
}

type VolcanoConfig struct {
	Enabled                bool   `mapstructure:"VOLCANO_ENABLED"`
	Namespace              string `mapstructure:"VOLCANO_NAMESPACE"`
	Queue                  string `mapstructure:"VOLCANO_QUEUE"`
	JobImage               string `mapstructure:"VOLCANO_JOB_IMAGE"`
	SchedulerPolicy        string `mapstructure:"VOLCANO_SCHEDULER_POLICY"`
	PriorityClass          string `mapstructure:"VOLCANO_PRIORITY_CLASS"`
	TTLSecondsAfterFinish  int    `mapstructure:"VOLCANO_TTL_SECONDS"`
	MinAvailableOverride   int    `mapstructure:"VOLCANO_MIN_AVAILABLE"`
}

type S3Config struct {
	Enabled      bool   `mapstructure:"S3_ENABLED"`
	Bucket       string `mapstructure:"S3_BUCKET"`
	Region       string `mapstructure:"S3_REGION"`
	Endpoint     string `mapstructure:"S3_ENDPOINT"`
	AccessKey    string `mapstructure:"S3_ACCESS_KEY"`
	SecretKey    string `mapstructure:"S3_SECRET_KEY"`
	UsePathStyle bool   `mapstructure:"S3_PATH_STYLE"`
}

type SemanticCacheConfig struct {
	Enabled         bool    `mapstructure:"SEM_CACHE_ENABLED"`
	URI             string  `mapstructure:"LANCEDB_URI"`
	Table           string  `mapstructure:"LANCEDB_TABLE"`
	EmbeddingModel  string  `mapstructure:"SEM_CACHE_EMBED_MODEL"`
	SimilarityLimit float64 `mapstructure:"SEM_CACHE_THRESHOLD"`
	TopK            int     `mapstructure:"SEM_CACHE_TOPK"`
}

func Load() (*Config, error) {
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("MANAGEMENT_PORT", 8081)
	viper.SetDefault("GIN_MODE", "debug")
	viper.SetDefault("DATABASE_URL", "postgres://inference:inference123@localhost:5432/inference_platform?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")

	// Volcano defaults
	viper.SetDefault("VOLCANO_ENABLED", false)
	viper.SetDefault("VOLCANO_NAMESPACE", "inference-platform")
	viper.SetDefault("VOLCANO_QUEUE", "inference-platform-queue")
	viper.SetDefault("VOLCANO_JOB_IMAGE", "pytorch/pytorch:2.1.0-cuda12.1-cudnn8-runtime")
	viper.SetDefault("VOLCANO_SCHEDULER_POLICY", "spread")
	viper.SetDefault("VOLCANO_PRIORITY_CLASS", "")
	viper.SetDefault("VOLCANO_TTL_SECONDS", 604800) // 7 天
	viper.SetDefault("VOLCANO_MIN_AVAILABLE", 0)

	// S3 defaults
	viper.SetDefault("S3_ENABLED", false)
	viper.SetDefault("S3_BUCKET", "")
	viper.SetDefault("S3_REGION", "us-east-1")
	viper.SetDefault("S3_ENDPOINT", "")
	viper.SetDefault("S3_ACCESS_KEY", "")
	viper.SetDefault("S3_SECRET_KEY", "")
	viper.SetDefault("S3_PATH_STYLE", true)

	// Semantic cache defaults
	viper.SetDefault("SEM_CACHE_ENABLED", false)
	viper.SetDefault("LANCEDB_URI", "./lancedb")
	viper.SetDefault("LANCEDB_TABLE", "semantic_cache")
	viper.SetDefault("SEM_CACHE_EMBED_MODEL", "text-embedding-3-small")
	viper.SetDefault("SEM_CACHE_THRESHOLD", 0.85)
	viper.SetDefault("SEM_CACHE_TOPK", 3)

	viper.AutomaticEnv()

	cfg := &Config{}
	cfg.DatabaseURL = viper.GetString("DATABASE_URL")
	cfg.RedisURL = viper.GetString("REDIS_URL")
	cfg.ServerPort = viper.GetInt("SERVER_PORT")
	cfg.ManagementPort = viper.GetInt("MANAGEMENT_PORT")
	cfg.GinMode = viper.GetString("GIN_MODE")
	cfg.FrontendURL = viper.GetString("FRONTEND_URL")
	cfg.Volcano = VolcanoConfig{
		Enabled:               viper.GetBool("VOLCANO_ENABLED"),
		Namespace:             viper.GetString("VOLCANO_NAMESPACE"),
		Queue:                 viper.GetString("VOLCANO_QUEUE"),
		JobImage:              viper.GetString("VOLCANO_JOB_IMAGE"),
		SchedulerPolicy:       viper.GetString("VOLCANO_SCHEDULER_POLICY"),
		PriorityClass:         viper.GetString("VOLCANO_PRIORITY_CLASS"),
		TTLSecondsAfterFinish: viper.GetInt("VOLCANO_TTL_SECONDS"),
		MinAvailableOverride:  viper.GetInt("VOLCANO_MIN_AVAILABLE"),
	}
	cfg.S3 = S3Config{
		Enabled:      viper.GetBool("S3_ENABLED"),
		Bucket:       viper.GetString("S3_BUCKET"),
		Region:       viper.GetString("S3_REGION"),
		Endpoint:     viper.GetString("S3_ENDPOINT"),
		AccessKey:    viper.GetString("S3_ACCESS_KEY"),
		SecretKey:    viper.GetString("S3_SECRET_KEY"),
		UsePathStyle: viper.GetBool("S3_PATH_STYLE"),
	}
	cfg.SemanticCache = SemanticCacheConfig{
		Enabled:         viper.GetBool("SEM_CACHE_ENABLED"),
		URI:             viper.GetString("LANCEDB_URI"),
		Table:           viper.GetString("LANCEDB_TABLE"),
		EmbeddingModel:  viper.GetString("SEM_CACHE_EMBED_MODEL"),
		SimilarityLimit: viper.GetFloat64("SEM_CACHE_THRESHOLD"),
		TopK:            viper.GetInt("SEM_CACHE_TOPK"),
	}

	return cfg, nil
}
