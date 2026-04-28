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
	"github.com/xincxiong/model-inference-platform/backend/internal/semcache"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"github.com/xincxiong/model-inference-platform/backend/internal/volcano"
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

	s3Client, err := store.NewS3(cfg.S3)
	if err != nil {
		logger.Warn("failed to init s3 client, falling back to db file storage", zap.Error(err))
	}

	semCache, err := semcache.New(context.Background(), semcache.Config{
		Enabled:         cfg.SemanticCache.Enabled,
		URI:             cfg.SemanticCache.URI,
		Table:           cfg.SemanticCache.Table,
		EmbeddingModel:  cfg.SemanticCache.EmbeddingModel,
		SimilarityLimit: cfg.SemanticCache.SimilarityLimit,
		TopK:            cfg.SemanticCache.TopK,
	}, logger)
	if err != nil {
		logger.Warn("semantic cache disabled: init failed", zap.Error(err))
	}

	s := &store.Store{DB: db, Redis: rdb, SemanticCache: semCache, S3: s3Client}
	store.SeedModels(context.Background(), db)

	hc := health.New(db, rdb)
	cb := circuitbreaker.New(circuitbreaker.DefaultConfig())

	hamiScheduler := hami.NewScheduler(logger)
	if hamiScheduler.IsEnabled() {
		logger.Info("HAMi scheduler enabled", zap.String("endpoint", os.Getenv("HAMI_SCHEDULER_ENDPOINT")))
	}

	wp := workerpool.New(rdb, logger, hamiScheduler)
	_ = wp

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
			zap.Int("output_tokens", ev.OutputTokens))
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	consumer.Start(ctx)

	engineRouter := engine.NewMultiEngineRouter(db, engine.DefaultRouterConfig(), logger)
	mr := modelrouter.New(db, engineRouter, logger)

	// ── Volcano Client ────────────────────────────────────────────────────
	volcanoClient, err := volcano.NewClient(logger, volcano.Config{
		Enabled:   cfg.Volcano.Enabled,
		Namespace: cfg.Volcano.Namespace,
	})
	if err != nil {
		logger.Warn("failed to create volcano client, fine-tuning will use simulation mode",
			zap.Error(err))
	}
	if volcanoClient != nil && volcanoClient.IsEnabled() {
		logger.Info("Volcano scheduler enabled",
			zap.String("namespace", cfg.Volcano.Namespace),
			zap.String("queue", cfg.Volcano.Queue))
	}

	r := router.SetupManagementRouter(cfg, s, mr, hc, cb, hamiScheduler, volcanoClient, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ManagementPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	go func() {
		logger.Info("management server starting", zap.Int("port", cfg.ManagementPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down management server...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}
	logger.Info("management server exited")
}
