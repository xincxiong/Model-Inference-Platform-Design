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

	"github.com/xincxiong/model-inference-platform/backend/internal/circuitbreaker"
	"github.com/xincxiong/model-inference-platform/backend/internal/config"
	"github.com/xincxiong/model-inference-platform/backend/internal/engine"
	"github.com/xincxiong/model-inference-platform/backend/internal/hami"
	"github.com/xincxiong/model-inference-platform/backend/internal/health"
	"github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
	"github.com/xincxiong/model-inference-platform/backend/internal/queue"
	"github.com/xincxiong/model-inference-platform/backend/internal/router"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"github.com/xincxiong/model-inference-platform/backend/internal/workerpool"
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

	store.SeedModels(context.Background(), db)

	// ── Health checker ────────────────────────────────────────────────────
	hc := health.New(db, rdb)

	// ── Circuit breaker manager ───────────────────────────────────────────
	cb := circuitbreaker.New(circuitbreaker.DefaultConfig())

	// ── Worker pool ───────────────────────────────────────────────────────
	wp := workerpool.New(rdb, logger)
	_ = wp // pool is available for future engine integration

	// ── Kafka queue ───────────────────────────────────────────────────────
	var kafkaBrokers []string
	if b := os.Getenv("KAFKA_BROKERS"); b != "" {
		kafkaBrokers = []string{b}
	}
	producer := queue.NewProducer(kafkaBrokers, "inference-events", logger)
	defer producer.Close()

	consumer := queue.NewConsumer(producer, logger)
	consumer.Register(queue.EventTypeUsage, func(msg queue.Message) error {
		ev, err := queue.UnmarshalUsageEvent(msg)
		if err != nil {
			return err
		}
		logger.Debug("usage event consumed",
			zap.String("request_id", ev.RequestID),
			zap.String("model", ev.ModelID),
			zap.Int("input_tokens", ev.InputTokens),
			zap.Int("output_tokens", ev.OutputTokens),
		)
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	consumer.Start(ctx)

	// ── HAMi Scheduler ────────────────────────────────────────────────────
	// Reads HAMI_ENABLED and HAMI_SCHEDULER_ENDPOINT from environment.
	// In local dev (HAMI_ENABLED=false), uses stub mode: scheduling calls
	// succeed immediately without connecting to a real HAMi instance.
	// In production Kubernetes (HAMI_ENABLED=true), connects to HAMi extender.
	hamiScheduler := hami.NewScheduler(logger)
	if hamiScheduler.IsEnabled() {
		logger.Info("HAMi scheduler enabled",
			zap.String("endpoint", os.Getenv("HAMI_SCHEDULER_ENDPOINT")))
	}

	// ── Inference engine (Multi-engine router) ─────────────────────────────
	// Supports vLLM, SGLang, Mock, and custom engines.
	// Configuration via environment variables:
	//   - INFERENCE_ENGINE=vllm|sglang|mock|custom (default: vllm)
	//   - VLLM_ENDPOINT=http://localhost:8000
	//   - SGLANG_ENDPOINT=http://localhost:30000
	//   - CUSTOM_ENGINE_<MODEL_ID>=<URL> (per-model override)
	//
	// When no real engine endpoint is configured, falls back to Mock.
	engineRouter := engine.NewMultiEngineRouter(db, engine.DefaultRouterConfig(), logger)

	// ── Model router ──────────────────────────────────────────────────────
	mr := modelrouter.New(db, engineRouter, logger)

	r := router.Setup(cfg, s, mr, hc, cb, hamiScheduler, logger)

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

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}
	logger.Info("server exited")
}
