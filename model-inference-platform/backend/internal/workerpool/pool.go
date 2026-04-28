package workerpool

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
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
	healthKeyPrefix = "wp:health:"
	defaultTTL      = 60 * time.Second
	healthCheckInterval = 10 * time.Second
)

// Load balancing strategy
type LoadBalanceStrategy string

const (
	StrategyLeastLoad    LoadBalanceStrategy = "least_load"     // 最小负载优先
	StrategyRoundRobin   LoadBalanceStrategy = "round_robin"    // 轮询
	StrategyConsistentHash LoadBalanceStrategy = "consistent_hash" // 一致性哈希
	StrategyRandom       LoadBalanceStrategy = "random"         // 随机
)

type WorkerStatus string

const (
	WorkerStatusReady    WorkerStatus = "ready"
	WorkerStatusBusy     WorkerStatus = "busy"
	WorkerStatusDraining WorkerStatus = "draining"
	WorkerStatusUnhealthy WorkerStatus = "unhealthy"
)

type Worker struct {
	ID           string       `json:"id"`
	Addr         string       `json:"addr"`
	Models       []string     `json:"models"`
	Load         float64      `json:"load"`
	Status       WorkerStatus `json:"status"`
	UpdatedAt    time.Time    `json:"updated_at"`
	NodeName     string       `json:"node_name"`
	PodName      string       `json:"pod_name"`
	GPUAllocated int          `json:"gpu_allocated"`
	Region       string       `json:"region"`        // 多区域支持
	Zone         string       `json:"zone"`          // 可用区
	HealthScore  float64      `json:"health_score"`  // 健康分数 0-100
	ConsecutiveFailures int   `json:"consecutive_failures"`
	LastHealthCheck time.Time `json:"last_health_check"`
}

type Pool struct {
	rdb         *redis.Client
	logger      *zap.Logger
	ttl         time.Duration
	mu          sync.Mutex
	localCB     map[string]int
	hamiClient  *hami.Scheduler
	k8sWatcher  *hami.K8sWatcher
	strategy    LoadBalanceStrategy
	rrIndex     map[string]int // round-robin index per model
	healthCheckCtx context.Context
	healthCheckCancel context.CancelFunc
}

func New(rdb *redis.Client, logger *zap.Logger, hamiClient *hami.Scheduler) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		rdb:        rdb,
		logger:     logger,
		ttl:        defaultTTL,
		localCB:    make(map[string]int),
		hamiClient: hamiClient,
		strategy:   StrategyLeastLoad,
		rrIndex:    make(map[string]int),
		healthCheckCtx: ctx,
		healthCheckCancel: cancel,
	}
	
	if hamiClient != nil {
		p.k8sWatcher = hami.NewK8sWatcher(logger, p)
	}
	
	// 启动健康检查协程
	go p.startHealthCheckLoop()
	
	return p
}

func (p *Pool) SetStrategy(strategy LoadBalanceStrategy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.strategy = strategy
}

func (p *Pool) Register(ctx context.Context, w *Worker) error {
	w.HealthScore = 100
	w.LastHealthCheck = time.Now()
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

// Pick selects a worker based on the configured load balancing strategy
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
		// 健康分数低于 50 的 worker 不参与调度
		if w.HealthScore < 50 {
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

	p.mu.Lock()
	strategy := p.strategy
	p.mu.Unlock()

	switch strategy {
	case StrategyLeastLoad:
		return p.pickLeastLoad(candidates), nil
	case StrategyRoundRobin:
		return p.pickRoundRobin(modelID, candidates), nil
	case StrategyConsistentHash:
		return p.pickConsistentHash(modelID, candidates), nil
	case StrategyRandom:
		return candidates[rand.Intn(len(candidates))], nil
	default:
		return p.pickLeastLoad(candidates), nil
	}
}

func (p *Pool) pickLeastLoad(candidates []*Worker) *Worker {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Load < candidates[j].Load
	})
	return candidates[0]
}

func (p *Pool) pickRoundRobin(modelID string, candidates []*Worker) *Worker {
	p.mu.Lock()
	idx := p.rrIndex[modelID]
	p.rrIndex[modelID] = (idx + 1) % len(candidates)
	p.mu.Unlock()
	return candidates[idx]
}

func (p *Pool) pickConsistentHash(modelID string, candidates []*Worker) *Worker {
	h := fnv.New32a()
	h.Write([]byte(modelID))
	hash := h.Sum32()
	
	idx := int(hash) % len(candidates)
	return candidates[idx]
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

// RecordHealthCheck records a health check result
func (p *Pool) RecordHealthCheck(ctx context.Context, id string, healthy bool) error {
	w, err := p.get(ctx, id)
	if err != nil {
		return err
	}
	
	w.LastHealthCheck = time.Now()
	
	if healthy {
		w.HealthScore = math.Min(100, w.HealthScore+10)
		w.ConsecutiveFailures = 0
		if w.Status == WorkerStatusUnhealthy {
			w.Status = WorkerStatusReady
		}
	} else {
		w.HealthScore = math.Max(0, w.HealthScore-20)
		w.ConsecutiveFailures++
		if w.ConsecutiveFailures >= 3 {
			w.Status = WorkerStatusUnhealthy
		}
	}
	
	return p.upsert(ctx, w)
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

// HealthSummaryWithDetails returns detailed health summary
func (p *Pool) HealthSummaryWithDetails(ctx context.Context) map[string]interface{} {
	all, _ := p.ListAll(ctx)
	counts := map[WorkerStatus]int{}
	totalLoad := 0.0
	healthyCount := 0
	
	for _, w := range all {
		if time.Since(w.UpdatedAt) <= p.ttl {
			counts[w.Status]++
			totalLoad += w.Load
			if w.HealthScore >= 80 {
				healthyCount++
			}
		}
	}
	
	avgLoad := 0.0
	if len(all) > 0 {
		avgLoad = totalLoad / float64(len(all))
	}
	
	return map[string]interface{}{
		"total_workers":   len(all),
		"healthy_workers": healthyCount,
		"status_counts":   counts,
		"average_load":    avgLoad,
	}
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
		"region", w.Region,
		"zone", w.Zone,
		"health_score", w.HealthScore,
		"consecutive_failures", w.ConsecutiveFailures,
		"last_health_check", w.LastHealthCheck.Unix(),
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
		Region:   vals["region"],
		Zone:     vals["zone"],
	}

	if ts, ok := vals["updated_at"]; ok {
		var unix int64
		fmt.Sscan(ts, &unix)
		w.UpdatedAt = time.Unix(unix, 0)
	}
	if ts, ok := vals["last_health_check"]; ok {
		var unix int64
		fmt.Sscan(ts, &unix)
		w.LastHealthCheck = time.Unix(unix, 0)
	}
	fmt.Sscanf(vals["load"], "%f", &w.Load)
	fmt.Sscanf(vals["health_score"], "%f", &w.HealthScore)
	fmt.Sscanf(vals["gpu_allocated"], "%d", &w.GPUAllocated)
	fmt.Sscanf(vals["consecutive_failures"], "%d", &w.ConsecutiveFailures)

	if m, ok := vals["models"]; ok {
		json.Unmarshal([]byte(m), &w.Models)
	}

	return w, nil
}

// startHealthCheckLoop periodically checks worker health
func (p *Pool) startHealthCheckLoop() {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.healthCheckCtx.Done():
			return
		case <-ticker.C:
			p.checkAllWorkersHealth(p.healthCheckCtx)
		}
	}
}

func (p *Pool) checkAllWorkersHealth(ctx context.Context) {
	workers, err := p.ListAll(ctx)
	if err != nil {
		p.logger.Error("failed to list workers for health check", zap.Error(err))
		return
	}
	
	for _, w := range workers {
		// 检查 worker 是否超时未更新
		if time.Since(w.UpdatedAt) > p.ttl {
			p.RecordHealthCheck(ctx, w.ID, false)
			p.logger.Warn("worker timeout", zap.String("worker_id", w.ID))
		}
	}
}

// Close stops the health check loop
func (p *Pool) Close() {
	if p.healthCheckCancel != nil {
		p.healthCheckCancel()
	}
}

var ErrNoWorker = fmt.Errorf("no healthy worker available")
