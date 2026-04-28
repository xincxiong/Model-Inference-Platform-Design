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
	"github.com/xincxiong/model-inference-platform/backend/internal/hami"
	"go.uber.org/zap"
)

const (
	workerKeyPrefix = "wp:worker:"
	indexKeyPrefix  = "wp:index:"
	defaultTTL      = 60 * time.Second
)

type WorkerStatus string

const (
	WorkerStatusReady    WorkerStatus = "ready"
	WorkerStatusBusy     WorkerStatus = "busy"
	WorkerStatusDraining WorkerStatus = "draining"
)

type Worker struct {
	ID        string     `json:"id"`
	Addr      string     `json:"addr"`
	Models    []string   `json:"models"`
	Load      float64    `json:"load"`
	Status    WorkerStatus `json:"status"`
	UpdatedAt time.Time  `json:"updated_at"`
	NodeName  string     `json:"node_name"`
	PodName   string     `json:"pod_name"`
	GPUAllocated int     `json:"gpu_allocated"`
}

type Pool struct {
	rdb         *redis.Client
	logger      *zap.Logger
	ttl         time.Duration
	mu          sync.Mutex
	localCB     map[string]int
	hamiClient  *hami.Scheduler
	k8sWatcher *hami.K8sWatcher
}

func New(rdb *redis.Client, logger *zap.Logger, hamiClient *hami.Scheduler) *Pool {
	p := &Pool{
		rdb:        rdb,
		logger:       logger,
		ttl:          defaultTTL,
		localCB:      make(map[string]int),
		hamiClient:   hamiClient,
	}
	
	if hamiClient != nil {
		p.k8sWatcher = hami.NewK8sWatcher(logger, p)
	}
	
	return p
}

func (p *Pool) Register(ctx context.Context, w *Worker) error {
	if err := p.upsert(ctx, w); err != nil {
		return err
	}
	
	go p.heartbeat(ctx, w)
	
	return nil
}

func (p *Pool) Deregister(ctx context.Context, id string) error {
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
			continue
		}
		
		if p.hamiClient != nil && p.hamiClient.IsEnabled() {
			if !p.validateHAMiAllocation(ctx, w) {
				p.logger.Warn("worker hami allocation validation failed",
					zap.String("worker_id", w.ID),
					zap.String("pod_name", w.PodName))
				continue
			}
		}
		
		candidates = append(candidates, w)
	}

	if len(candidates) == 0 {
		return nil, ErrNoWorker
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Load < candidates[j].Load
	})

	return candidates[0], nil
}

func (p *Pool) validateHAMiAllocation(ctx context.Context, w *Worker) bool {
	if w.NodeName == "" || w.PodName == "" {
		return true
	}
	
	return p.k8sWatcher.ValidatePodGPUAllocation(w.PodName, w.NodeName)
}

func (p *Pool) UpdateHAMiAllocation(ctx context.Context, workerID string, nodeName string, gpuMemMiB int) error {
	w, err := p.get(ctx, workerID)
	if err != nil {
		return err
	}
	
	w.NodeName = nodeName
	w.GPUAllocated = gpuMemMiB
	w.UpdatedAt = time.Now()
	
	return p.upsert(ctx, w)
}

func (p *Pool) UpdateLoad(ctx context.Context, id string, load float64) error {
	key := workerKeyPrefix + id
	return p.rdb.HSet(ctx, key, "load", load, "updated_at", time.Now().Unix()).Err()
}

func (p *Pool) SetStatus(ctx context.Context, id string, status WorkerStatus) error {
	key := workerKeyPrefix + id
	return p.rdb.HSet(ctx, key, "status", string(status), "updated_at", time.Now().Unix()).Err()
}

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
		"node_name", w.NodeName,
		"pod_name", w.PodName,
		"gpu_allocated", w.GPUAllocated,
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
		NodeName: vals["node_name"],
		PodName:  vals["pod_name"],
	}

	if ts, ok := vals["updated_at"]; ok {
		var unix int64
		fmt.Sscan(ts, &unix)
		w.UpdatedAt = time.Unix(unix, 0)
	}
	fmt.Sscanf(vals["load"], "%f", &w.Load)
	fmt.Sscanf(vals["gpu_allocated"], "%d", &w.GPUAllocated)

	if m, ok := vals["models"]; ok {
		json.Unmarshal([]byte(m), &w.Models)
	}

	return w, nil
}

var ErrNoWorker = fmt.Errorf("no healthy worker available")
