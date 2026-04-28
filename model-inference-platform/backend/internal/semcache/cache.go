package semcache

import (
    "context"
    "math"
    "sync"
    "time"

    "go.uber.org/zap"
)

// Config mirrors the SemanticCacheConfig in config package to avoid circular deps.
type Config struct {
    Enabled         bool
    URI             string // reserved for future LanceDB persistence
    Table           string
    EmbeddingModel  string
    SimilarityLimit float64
    TopK            int
    MaxEntries      int // max in-memory entries (LRU eviction)
}

type CacheRow struct {
    ID        string
    UserID    string
    ModelID   string
    Prompt    string
    Response  string
    Embedding []float32
    CreatedAt int64
}

type Hit struct {
    Row        CacheRow
    Similarity float64
}

// Cache is an in-memory semantic cache with optional LanceDB persistence.
// Current implementation uses pure Go (no CGO) with LRU eviction.
type Cache struct {
    cfg   Config
    mu    sync.RWMutex
    mem   []CacheRow
    log   *zap.Logger
}

// New creates a semantic cache. Currently uses pure in-memory storage to avoid
// CGO dependency from LanceDB Go client. LanceDB persistence can be added later.
func New(ctx context.Context, cfg Config, logger *zap.Logger) (*Cache, error) {
    if !cfg.Enabled {
        return nil, nil
    }
    if cfg.Table == "" {
        cfg.Table = "semanticcache"
    }
    if cfg.TopK <= 0 {
        cfg.TopK = 3
    }
    if cfg.SimilarityLimit <= 0 {
        cfg.SimilarityLimit = 0.85
    }
    if cfg.MaxEntries <= 0 {
        cfg.MaxEntries = 10000
    }

    c := &Cache{cfg: cfg, log: logger}
    return c, nil
}

func (c *Cache) Enabled() bool {
    return c != nil && c.cfg.Enabled
}

// Embedder computes embedding for text input (provided by handler to avoid coupling engines here).
type Embedder func(ctx context.Context, input string) ([]float32, error)

// Lookup finds the most similar cached item for the same user + model.
func (c *Cache) Lookup(ctx context.Context, userID, modelID, prompt string, embed Embedder) (*Hit, error) {
    if c == nil || !c.cfg.Enabled {
        return nil, nil
    }
    vec, err := embed(ctx, prompt)
    if err != nil {
        return nil, err
    }
    return c.LookupWithVec(userID, modelID, vec)
}

// LookupWithVec reuses a pre-computed embedding to avoid double work in handlers.
func (c *Cache) LookupWithVec(userID, modelID string, vec []float32) (*Hit, error) {
    if c == nil || !c.cfg.Enabled {
        return nil, nil
    }
    c.mu.RLock()
    defer c.mu.RUnlock()

    var best CacheRow
    bestSim := -1.0
    for _, r := range c.mem {
        if r.UserID != userID || r.ModelID != modelID {
            continue
        }
        sim := cosine(vec, r.Embedding)
        if sim > bestSim {
            bestSim = sim
            best = r
        }
    }

    if bestSim >= c.cfg.SimilarityLimit && bestSim <= 1.0 {
        return &Hit{Row: best, Similarity: bestSim}, nil
    }
    return nil, nil
}

// Insert writes a new cache row with LRU eviction.
func (c *Cache) Insert(ctx context.Context, row CacheRow) error {
    if c == nil || !c.cfg.Enabled {
        return nil
    }
    if row.CreatedAt == 0 {
        row.CreatedAt = time.Now().Unix()
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    // Evict oldest if at capacity
    if len(c.mem) >= c.cfg.MaxEntries {
        c.mem = c.mem[1:]
    }
    c.mem = append(c.mem, row)
    return nil
}

// Count returns current cache size (for debugging/monitoring).
func (c *Cache) Count() int {
    if c == nil {
        return 0
    }
    c.mu.RLock()
    defer c.mu.RUnlock()
    return len(c.mem)
}

func cosine(a, b []float32) float64 {
    if len(a) == 0 || len(a) != len(b) {
        return -1
    }
    var dot float64
    var na float64
    var nb float64
    for i := 0; i < len(a); i++ {
        av := float64(a[i])
        bv := float64(b[i])
        dot += av * bv
        na += av * av
        nb += bv * bv
    }
    denom := math.Sqrt(na) * math.Sqrt(nb)
    if denom == 0 {
        return -1
    }
    return dot / denom
}
