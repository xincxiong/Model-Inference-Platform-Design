package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xincxiong/model-inference-platform/backend/internal/config"
	"github.com/xincxiong/model-inference-platform/backend/internal/engine"
	"github.com/xincxiong/model-inference-platform/backend/internal/router"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, _ := zap.NewProduction()
	if cfg.GinMode == "debug" {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	db, err := store.NewPostgres(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect postgres", zap.Error(err))
	}
	defer db.Close()

	if err := store.RunMigrations(context.Background(), db); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	rdb, err := store.NewRedis(cfg.RedisURL)
	if err != nil {
		logger.Fatal("failed to connect redis", zap.Error(err))
	}
	defer rdb.Close()

	s := &store.Store{DB: db, Redis: rdb}
	eng := engine.NewMockEngine()

	store.SeedModels(context.Background(), db)

	r := router.Setup(cfg, s, eng, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.Int("port", cfg.ServerPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server exited")
}
