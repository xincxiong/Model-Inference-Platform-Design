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
}

func Load() (*Config, error) {
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("MANAGEMENT_PORT", 8081)
	viper.SetDefault("GIN_MODE", "debug")
	viper.SetDefault("DATABASE_URL", "postgres://inference:inference123@localhost:5432/inference_platform?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")

	viper.AutomaticEnv()

	cfg := &Config{}
	cfg.DatabaseURL = viper.GetString("DATABASE_URL")
	cfg.RedisURL = viper.GetString("REDIS_URL")
	cfg.ServerPort = viper.GetInt("SERVER_PORT")
	cfg.ManagementPort = viper.GetInt("MANAGEMENT_PORT")
	cfg.GinMode = viper.GetString("GIN_MODE")
	cfg.FrontendURL = viper.GetString("FRONTEND_URL")

	return cfg, nil
}
