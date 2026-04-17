// Package workerpool manages a pool of inference worker instances.
//
// Workers register themselves via Register, sending heartbeats every TTL/2.
// The pool uses Redis to persist worker state so multiple gateway replicas
// share the same view of the fleet.
//
// Layout of Redis keys
//
//	wp:worker:<id>      – HASH  (fields: id, addr, models, load, status, updated_at)
//	wp:index:<model>    – SET   (member = worker id)
package workerpool

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	workerKeyPrefix = "wp:worker:"
	indexKeyPrefix  = "wp:index:"
	defaultTTL      = 60 * time.Second
)

// WorkerStatus represents the availability state of a worker.
type WorkerStatus string

const (
	WorkerStatusReady    WorkerStatus = "ready"
	WorkerStatusBusy     WorkerStatus = "busy"
	WorkerStatusDraining WorkerStatus = "draining"
)

// Worker holds metadata about one inference worker instance.
type Worker struct {
	ID        string       `json:"id"`
	Addr      string       `json:"addr"`        // base URL, e.g. http://10.0.1.5:8000
	Models    []string     `json:"models"`      // model IDs this worker can serve
	Load      float64      `json:"load"`        // 0.0–1.0 normalised load
	Status    WorkerStatus `json:"status"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// Pool is a Redis-backed worker pool.
type Pool struct {
	rdb    *redis.Client
	logger *zap.Logger
	ttl    time.Duration

	mu      sync.Mutex
	localCB map[string]int // simple local failure counter (circuit breaker hint)
}

// New creates a Pool with the given Redis client.
func New(rdb *redis.Client, logger *zap.Logger) *Pool {
	return &Pool{
		rdb:     rdb,
		logger:  logger,
		ttl:     defaultTTL,
		localCB: make(map[string]int),
	}
}

// Register writes a worker record to Redis and keeps it alive.
// Call this inside the worker process; it blocks until ctx is done.
func (p *Pool) Register(ctx context.Context, w *Worker) error {
	if err := p.upsert(ctx, w); err != nil {
		return err
	}
	go p.heartbeat(ctx, w)
	return nil
}

// Deregister removes a worker from the pool immediately.
func (p *Pool) Deregister(ctx context.Context, id string) error {
	// Remove from all model indexes
	workers, _ := p.ListAll(ctx)
	for _, w := range workers {
		if w.ID == id {
			for _, m := range w.Models {
				p.rdb.SRem(ctx, indexKeyPrefix+m, id)
			}
			break
		}
	}
	return p.rdb.Del(ctx, workerKeyPrefix+id).Err()
}

// Pick selects the least-loaded healthy worker that can serve the given model.
// Returns ErrNoWorker if no suitable worker is available.
func (p *Pool) Pick(ctx context.Context, modelID string) (*Worker, error) {
	ids, err := p.rdb.SMembers(ctx, indexKeyPrefix+modelID).Result()
	if err != nil || len(ids) == 0 {
		return nil, ErrNoWorker
	}

	var candidates []*Worker
	for _, id := range ids {
		w, err := p.get(ctx, id)
		if err != nil {
			continue
		}
		if w.Status != WorkerStatusReady {
			continue
		}
		if time.Since(w.UpdatedAt) > p.ttl {
			continue // stale, skip
		}
		candidates = append(candidates, w)
	}

	if len(candidates) == 0 {
		return nil, ErrNoWorker
	}

	// Sort by load ascending, with a small random tiebreak to spread evenly.
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Load < candidates[j].Load
	})

	return candidates[0], nil
}

// UpdateLoad updates the load value for a worker (called by the worker after each request).
func (p *Pool) UpdateLoad(ctx context.Context, id string, load float64) error {
	key := workerKeyPrefix + id
	return p.rdb.HSet(ctx, key, "load", load, "updated_at", time.Now().Unix()).Err()
}

// SetStatus updates the status of a worker.
func (p *Pool) SetStatus(ctx context.Context, id string, status WorkerStatus) error {
	key := workerKeyPrefix + id
	return p.rdb.HSet(ctx, key, "status", string(status), "updated_at", time.Now().Unix()).Err()
}

// ListAll returns all known workers (including stale ones).
func (p *Pool) ListAll(ctx context.Context) ([]*Worker, error) {
	var cursor uint64
	var keys []string
	for {
		batch, next, err := p.rdb.Scan(ctx, cursor, workerKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	workers := make([]*Worker, 0, len(keys))
	for _, k := range keys {
		id := strings.TrimPrefix(k, workerKeyPrefix)
		w, err := p.get(ctx, id)
		if err != nil {
			continue
		}
		workers = append(workers, w)
	}
	return workers, nil
}

// HealthySummary returns counts of workers by status.
func (p *Pool) HealthySummary(ctx context.Context) map[WorkerStatus]int {
	all, _ := p.ListAll(ctx)
	counts := map[WorkerStatus]int{}
	for _, w := range all {
		if time.Since(w.UpdatedAt) <= p.ttl {
			counts[w.Status]++
		}
	}
	return counts
}

// ── internal ──────────────────────────────────────────────────────────────

func (p *Pool) upsert(ctx context.Context, w *Worker) error {
	modelsJSON, _ := json.Marshal(w.Models)
	key := workerKeyPrefix + w.ID

	pipe := p.rdb.Pipeline()
	pipe.HSet(ctx, key,
		"id", w.ID,
		"addr", w.Addr,
		"models", string(modelsJSON),
		"load", w.Load,
		"status", string(w.Status),
		"updated_at", time.Now().Unix(),
	)
	pipe.Expire(ctx, key, p.ttl*2)

	for _, m := range w.Models {
		pipe.SAdd(ctx, indexKeyPrefix+m, w.ID)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (p *Pool) heartbeat(ctx context.Context, w *Worker) {
	ticker := time.NewTicker(p.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.upsert(ctx, w); err != nil {
				p.logger.Warn("worker heartbeat failed", zap.String("worker_id", w.ID), zap.Error(err))
			}
		}
	}
}

func (p *Pool) get(ctx context.Context, id string) (*Worker, error) {
	vals, err := p.rdb.HGetAll(ctx, workerKeyPrefix+id).Result()
	if err != nil || len(vals) == 0 {
		return nil, fmt.Errorf("worker %q not found", id)
	}

	w := &Worker{
		ID:     vals["id"],
		Addr:   vals["addr"],
		Status: WorkerStatus(vals["status"]),
	}

	if ts, ok := vals["updated_at"]; ok {
		var unix int64
		fmt.Sscan(ts, &unix)
		w.UpdatedAt = time.Unix(unix, 0)
	}
	fmt.Sscanf(vals["load"], "%f", &w.Load)

	if m, ok := vals["models"]; ok {
		json.Unmarshal([]byte(m), &w.Models)
	}

	return w, nil
}

// ErrNoWorker is returned when no healthy worker is available for a model.
var ErrNoWorker = fmt.Errorf("no healthy worker available")
