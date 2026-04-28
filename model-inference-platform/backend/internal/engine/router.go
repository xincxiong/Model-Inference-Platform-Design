// Package engine provides a multi-engine router that allows users to configure
// their preferred inference engine (vLLM, SGLang, or custom).
//
// Engine selection is controlled via:
//   1. Environment variable: INFERENCE_ENGINE=vllm|sglang|mock|custom
//   2. Per-model override in database: models.engine_type column
//   3. Dedicated endpoint configuration: dedicated_endpoints.engine_addr
//
// Default engine is vLLM when available; falls back to Mock when no real engine
// is configured (for local development).
package engine

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"go.uber.org/zap"
)

// EngineType identifies the inference engine backend.
type EngineType string

const (
	EngineVLLM   EngineType = "vllm"
	EngineSGLang EngineType = "sglang"
	EngineMock   EngineType = "mock"
	EngineCustom EngineType = "custom"
)

// MultiEngineRouter routes inference requests to the appropriate engine
// based on model configuration and user preferences.
type MultiEngineRouter struct {
	db      *pgxpool.Pool
	logger  *zap.Logger
	engines map[EngineType]Engine
	mu      sync.RWMutex

	// Default engine type (from env or config)
	defaultEngine EngineType

	// Custom engine addresses per model (from database or env)
	customEngines map[string]string // model_id -> engine_addr
}

// RouterConfig holds configuration for the multi-engine router.
type RouterConfig struct {
	// DefaultEngine is the fallback engine type.
	DefaultEngine EngineType

	// VLLM endpoint URL (from env: VLLM_ENDPOINT)
	VLLMEndpoint string

	// SGLang endpoint URL (from env: SGLANG_ENDPOINT)
	SGLangEndpoint string

	// Custom engine addresses (from env: CUSTOM_ENGINE_<MODEL_ID>=<URL>)
	CustomEngines map[string]string
}

// DefaultRouterConfig reads configuration from environment variables.
func DefaultRouterConfig() RouterConfig {
	engineType := EngineType(os.Getenv("INFERENCE_ENGINE"))
	if engineType == "" {
		engineType = EngineVLLM // default to vLLM
	}

	return RouterConfig{
		DefaultEngine:  engineType,
		VLLMEndpoint:   os.Getenv("VLLM_ENDPOINT"),
		SGLangEndpoint: os.Getenv("SGLANG_ENDPOINT"),
		CustomEngines:  parseCustomEnginesFromEnv(),
	}
}

// parseCustomEnginesFromEnv reads CUSTOM_ENGINE_<MODEL_ID> env vars.
func parseCustomEnginesFromEnv() map[string]string {
	result := make(map[string]string)
	for _, pair := range os.Environ() {
		if strings.HasPrefix(pair, "CUSTOM_ENGINE_") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				modelID := strings.TrimPrefix(parts[0], "CUSTOM_ENGINE_")
				result[modelID] = parts[1]
			}
		}
	}
	return result
}

// NewMultiEngineRouter creates a router with multiple engine backends.
func NewMultiEngineRouter(db *pgxpool.Pool, cfg RouterConfig, logger *zap.Logger) *MultiEngineRouter {
	router := &MultiEngineRouter{
		db:            db,
		logger:        logger,
		engines:       make(map[EngineType]Engine),
		defaultEngine: cfg.DefaultEngine,
		customEngines: cfg.CustomEngines,
	}

	// Initialize engines based on config
	if cfg.VLLMEndpoint != "" {
		router.engines[EngineVLLM] = NewVLLMClient(VLLMConfig{BaseURL: cfg.VLLMEndpoint}, logger)
		logger.Info("vllm engine initialized", zap.String("endpoint", cfg.VLLMEndpoint))
	}

	if cfg.SGLangEndpoint != "" {
		router.engines[EngineSGLang] = NewSGLangClient(SGLangConfig{BaseURL: cfg.SGLangEndpoint}, logger)
		logger.Info("sglang engine initialized", zap.String("endpoint", cfg.SGLangEndpoint))
	}

	// Always initialize Mock engine as fallback
	router.engines[EngineMock] = NewMockEngine()
	logger.Info("mock engine initialized (fallback)")

	// Log configuration
	logger.Info("multi-engine router configured",
		zap.String("default", string(router.defaultEngine)),
		zap.Int("engines", len(router.engines)),
		zap.Int("custom_engines", len(router.customEngines)))

	return router
}

// GetEngine returns the appropriate engine for a given model.
// Selection priority:
//   1. Custom engine address from env (CUSTOM_ENGINE_<MODEL_ID>)
//   2. Engine type from database (models.engine_type)
//   3. Default engine type
//   4. Mock engine (fallback)
func (r *MultiEngineRouter) GetEngine(ctx context.Context, modelID string) Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Check custom engine from env
	if addr, ok := r.customEngines[modelID]; ok {
		r.logger.Debug("using custom engine for model",
			zap.String("model", modelID),
			zap.String("addr", addr))
		// Create ad-hoc vLLM client for custom address
		return NewVLLMClient(VLLMConfig{BaseURL: addr}, r.logger)
	}

	// 2. Check engine_type from database
	var engineType string
	err := r.db.QueryRow(ctx,
		`SELECT engine_type FROM models WHERE id = $1`,
		modelID).Scan(&engineType)
	if err == nil && engineType != "" {
		et := EngineType(engineType)
		if eng, ok := r.engines[et]; ok {
			r.logger.Debug("using database-configured engine",
				zap.String("model", modelID),
				zap.String("engine", engineType))
			return eng
		}
	}

	// 3. Use default engine
	if eng, ok := r.engines[r.defaultEngine]; ok {
		r.logger.Debug("using default engine",
			zap.String("model", modelID),
			zap.String("engine", string(r.defaultEngine)))
		return eng
	}

	// 4. Fallback to Mock
	r.logger.Warn("no engine available, using mock",
		zap.String("model", modelID))
	return r.engines[EngineMock]
}

// GetEngineByType returns an engine by explicit type.
func (r *MultiEngineRouter) GetEngineByType(et EngineType) Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.engines[et]
}

// RegisterEngine adds a new engine backend dynamically.
func (r *MultiEngineRouter) RegisterEngine(et EngineType, eng Engine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.engines[et] = eng
	r.logger.Info("engine registered", zap.String("type", string(et)))
}

// SetCustomEngine sets a custom engine address for a specific model.
func (r *MultiEngineRouter) SetCustomEngine(modelID, addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.customEngines[modelID] = addr
	r.logger.Info("custom engine set",
		zap.String("model", modelID),
		zap.String("addr", addr))
}

// ChatCompletion routes to the appropriate engine.
func (r *MultiEngineRouter) ChatCompletion(ctx context.Context, req model.ChatCompletionRequest) (<-chan model.ChatCompletionChunk, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.ChatCompletion(ctx, req)
}

// Completion routes to the appropriate engine.
func (r *MultiEngineRouter) Completion(ctx context.Context, req model.CompletionRequest) (*model.CompletionResponse, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.Completion(ctx, req)
}

// Embedding routes to the appropriate engine.
func (r *MultiEngineRouter) Embedding(ctx context.Context, req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.Embedding(ctx, req)
}

// Rerank routes to the appropriate engine.
func (r *MultiEngineRouter) Rerank(ctx context.Context, req model.RerankRequest) (*model.RerankResponse, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.Rerank(ctx, req)
}

// ImageGeneration routes to the appropriate engine.
func (r *MultiEngineRouter) ImageGeneration(ctx context.Context, req model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.ImageGeneration(ctx, req)
}

func (r *MultiEngineRouter) VideoGeneration(ctx context.Context, req model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.VideoGeneration(ctx, req)
}

func (r *MultiEngineRouter) Transcription(ctx context.Context, req model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.Transcription(ctx, req)
}

func (r *MultiEngineRouter) Speech(ctx context.Context, req model.SpeechRequest) ([]byte, error) {
	eng := r.GetEngine(ctx, req.Model)
	return eng.Speech(ctx, req)
}

// Health checks all registered engines.
func (r *MultiEngineRouter) Health(ctx context.Context) map[EngineType]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[EngineType]bool)
	for et, eng := range r.engines {
		// Mock engine is always healthy
		if et == EngineMock {
			result[et] = true
			continue
		}

		// Check health for real engines
		if healthChecker, ok := eng.(interface{ Health(context.Context) error }); ok {
			result[et] = healthChecker.Health(ctx) == nil
		} else {
			result[et] = true // assume healthy if no health method
		}
	}
	return result
}

