package modelrouter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xincxiong/model-inference-platform/backend/internal/engine"
	"go.uber.org/zap"
)

var (
	ErrModelNotFound    = errors.New("model not found or inactive")
	ErrModelTypeMismatch = errors.New("model type does not match this endpoint")
)

// ResolvedModel contains all routing information after resolving a model identifier.
type ResolvedModel struct {
	ID          string  // canonical model ID in DB (without flavor suffix)
	OriginalID  string  // what the user passed (may include -fast)
	Name        string
	ModelType   string  // text-to-text, vision, embedding, rerank, text-to-image, text-to-video, speech
	Provider    string
	Flavor      string  // "base" or "fast"
	InputPrice  float64
	OutputPrice float64
	MaxContext  int
	BackendKey  string  // which engine backend to route to ("shared", "dedicated:<endpoint_id>", etc.)
}

type modelCacheEntry struct {
	ID          string
	Name        string
	ModelType   string
	Provider    string
	InputPrice  float64
	OutputPrice float64
	MaxContext  int
}

// ModelRouter validates model requests and routes to the correct engine backend.
//
// Request flow:
//
//	Handler → ModelRouter.Resolve(modelID) → ResolvedModel
//	       → ModelRouter.ValidateType(resolved, allowedTypes)
//	       → ModelRouter.GetEngine(resolved) → engine.Engine
//	       → engine.ChatCompletion / Embedding / etc.
type ModelRouter struct {
	db            *pgxpool.Pool
	logger        *zap.Logger
	defaultEngine engine.Engine
	engines       map[string]engine.Engine // backend key → engine instance

	mu    sync.RWMutex
	cache map[string]*modelCacheEntry // model ID → cached info
}

// New creates a ModelRouter and performs the initial model cache load.
func New(db *pgxpool.Pool, defaultEngine engine.Engine, logger *zap.Logger) *ModelRouter {
	r := &ModelRouter{
		db:            db,
		logger:        logger,
		defaultEngine: defaultEngine,
		engines:       make(map[string]engine.Engine),
		cache:         make(map[string]*modelCacheEntry),
	}
	r.RefreshCache()
	go r.backgroundRefresh()
	return r
}

// RegisterEngine adds a named engine backend (e.g., "vllm-shared", "dedicated:ep_xxx").
func (r *ModelRouter) RegisterEngine(key string, eng engine.Engine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.engines[key] = eng
}

// Resolve parses the model identifier, validates existence, and returns routing info.
//
// Parsing rules:
//   - "deepseek-ai/DeepSeek-V4"       → base flavor, ID = "deepseek-ai/DeepSeek-V4"
//   - "deepseek-ai/DeepSeek-V4-fast"  → fast flavor, ID = "deepseek-ai/DeepSeek-V4"
//   - "ep_xxx:deepseek-ai/DeepSeek-V4" → dedicated endpoint routing (Phase 2)
func (r *ModelRouter) Resolve(modelID string) (*ResolvedModel, error) {
	if modelID == "" {
		return nil, fmt.Errorf("%w: empty model identifier", ErrModelNotFound)
	}

	original := modelID

	// Phase 2: dedicated endpoint routing_key detection
	backendKey := "shared"
	if strings.HasPrefix(modelID, "ep_") {
		parts := strings.SplitN(modelID, ":", 2)
		if len(parts) == 2 {
			backendKey = "dedicated:" + parts[0]
			modelID = parts[1]
		}
	}

	// Flavor parsing: strip -fast suffix
	flavor := "base"
	if strings.HasSuffix(modelID, "-fast") {
		flavor = "fast"
		modelID = strings.TrimSuffix(modelID, "-fast")
	}

	// Cache lookup
	r.mu.RLock()
	entry, ok := r.cache[modelID]
	r.mu.RUnlock()

	if !ok {
		// Cache miss — try a direct DB lookup (handles newly added models)
		entry = r.loadFromDB(modelID)
		if entry == nil {
			return nil, fmt.Errorf("%w: %s", ErrModelNotFound, original)
		}
	}

	return &ResolvedModel{
		ID:          entry.ID,
		OriginalID:  original,
		Name:        entry.Name,
		ModelType:   entry.ModelType,
		Provider:    entry.Provider,
		Flavor:      flavor,
		InputPrice:  entry.InputPrice,
		OutputPrice: entry.OutputPrice,
		MaxContext:  entry.MaxContext,
		BackendKey:  backendKey,
	}, nil
}

// ValidateType checks that the resolved model's type is among the allowed types for an endpoint.
func (r *ModelRouter) ValidateType(resolved *ResolvedModel, allowedTypes []string) error {
	for _, t := range allowedTypes {
		if resolved.ModelType == t {
			return nil
		}
	}
	return fmt.Errorf("%w: model '%s' is type '%s', but this endpoint requires one of %v",
		ErrModelTypeMismatch, resolved.OriginalID, resolved.ModelType, allowedTypes)
}

// GetEngine returns the engine backend for a resolved model.
//
// Routing priority:
//  1. Dedicated endpoint engine (if BackendKey starts with "dedicated:")
//  2. Named engine matching BackendKey
//  3. Default engine (shared pool / mock)
func (r *ModelRouter) GetEngine(resolved *ResolvedModel) engine.Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if eng, ok := r.engines[resolved.BackendKey]; ok {
		return eng
	}
	return r.defaultEngine
}

// RefreshCache reloads all active models from the database into the in-memory cache.
func (r *ModelRouter) RefreshCache() {
	rows, err := r.db.Query(context.Background(),
		`SELECT id, name, model_type, provider, input_price, output_price, max_context
		 FROM models WHERE status = 'active'`)
	if err != nil {
		r.logger.Error("model cache refresh failed", zap.Error(err))
		return
	}
	defer rows.Close()

	newCache := make(map[string]*modelCacheEntry)
	for rows.Next() {
		var e modelCacheEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.ModelType, &e.Provider, &e.InputPrice, &e.OutputPrice, &e.MaxContext); err != nil {
			continue
		}
		newCache[e.ID] = &e
	}

	r.mu.Lock()
	r.cache = newCache
	r.mu.Unlock()

	r.logger.Info("model cache refreshed", zap.Int("count", len(newCache)))
}

func (r *ModelRouter) loadFromDB(modelID string) *modelCacheEntry {
	var e modelCacheEntry
	err := r.db.QueryRow(context.Background(),
		`SELECT id, name, model_type, provider, input_price, output_price, max_context
		 FROM models WHERE id = $1 AND status = 'active'`, modelID).
		Scan(&e.ID, &e.Name, &e.ModelType, &e.Provider, &e.InputPrice, &e.OutputPrice, &e.MaxContext)
	if err != nil {
		return nil
	}

	// Warm the cache
	r.mu.Lock()
	r.cache[e.ID] = &e
	r.mu.Unlock()

	return &e
}

func (r *ModelRouter) backgroundRefresh() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		r.RefreshCache()
	}
}
