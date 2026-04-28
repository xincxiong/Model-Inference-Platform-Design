// Package loadbalancer provides intelligent load balancing for inference workers.
package loadbalancer

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// BalancerStrategy defines the load balancing algorithm.
type BalancerStrategy string

const (
	BalancerLeastConnections BalancerStrategy = "least_connections"
	BalancerRoundRobin       BalancerStrategy = "round_robin"
	BalancerConsistentHash   BalancerStrategy = "consistent_hash"
	BalancerWeighted         BalancerStrategy = "weighted"
	BalancerRegional         BalancerStrategy = "regional" // Multi-region aware
)

// Worker represents an inference worker instance.
type Worker struct {
	ID            string
	Addr          string
	Region        string
	Zone          string
	Models        []string
	ActiveConns   int
	MaxConns      int
	CPUUsage      float64
	MemoryUsage   float64
	GPUUsage      float64
	LastSeen      time.Time
	Healthy       bool
	Weight        int // for weighted balancing
}

// Request represents an incoming inference request.
type Request struct {
	ID        string
	ModelID   string
	Priority  int           // 0-10, higher = more important
	Region    string        // client's region
	Timeout   time.Duration
	Enqueued  time.Time
}

// LoadBalancer manages traffic distribution across workers.
type LoadBalancer struct {
	rdb      *redis.Client
	logger   *zap.Logger
	strategy BalancerStrategy
	workers  map[string]*Worker // worker ID -> Worker
	mu       sync.RWMutex
	rrIndex  map[string]int // model -> round-robin index
	hashRing *ConsistentHashRing
}

// New creates a new LoadBalancer.
func New(rdb *redis.Client, logger *zap.Logger, strategy BalancerStrategy) *LoadBalancer {
	lb := &LoadBalancer{
		rdb:      rdb,
		logger:   logger,
		strategy: strategy,
		workers:  make(map[string]*Worker),
		rrIndex:  make(map[string]int),
	}
	
	if strategy == BalancerConsistentHash {
		lb.hashRing = NewConsistentHashRing(100) // 100 virtual nodes per worker
	}
	
	return lb
}

// SetStrategy updates the load balancing strategy.
func (lb *LoadBalancer) SetStrategy(strategy BalancerStrategy) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.strategy = strategy
	
	if strategy == BalancerConsistentHash && lb.hashRing == nil {
		lb.hashRing = NewConsistentHashRing(100)
	}
}

// RegisterWorker adds a new worker to the pool.
func (lb *LoadBalancer) RegisterWorker(ctx context.Context, w *Worker) error {
	lb.mu.Lock()
	lb.workers[w.ID] = w
	if lb.hashRing != nil {
		lb.hashRing.Add(w.ID, w.Addr)
	}
	lb.mu.Unlock()
	
	// Persist to Redis
	key := fmt.Sprintf("lb:worker:%s", w.ID)
	lb.rdb.HSet(ctx, key, map[string]interface{}{
		"id":          w.ID,
		"addr":        w.Addr,
		"region":      w.Region,
		"zone":        w.Zone,
		"models":      fmt.Sprintf("%v", w.Models),
		"max_conns":   w.MaxConns,
		"weight":      w.Weight,
		"healthy":     w.Healthy,
		"last_seen":   time.Now().Unix(),
	})
	lb.rdb.Expire(ctx, key, 120*time.Second)
	
	// Index by model
	for _, model := range w.Models {
		lb.rdb.SAdd(ctx, fmt.Sprintf("lb:model:%s", model), w.ID)
	}
	
	lb.logger.Info("registered worker",
		zap.String("worker_id", w.ID),
		zap.String("region", w.Region),
		zap.Strings("models", w.Models))
	
	return nil
}

// DeregisterWorker removes a worker from the pool.
func (lb *LoadBalancer) DeregisterWorker(ctx context.Context, workerID string) {
	lb.mu.Lock()
	if w, exists := lb.workers[workerID]; exists {
		for _, model := range w.Models {
			lb.rdb.SRem(ctx, fmt.Sprintf("lb:model:%s", model), workerID)
		}
		delete(lb.workers, workerID)
	}
	if lb.hashRing != nil {
		lb.hashRing.Remove(workerID)
	}
	lb.mu.Unlock()
	
	lb.rdb.Del(ctx, fmt.Sprintf("lb:worker:%s", workerID))
}

// SelectWorker picks the best worker for a request.
func (lb *LoadBalancer) SelectWorker(ctx context.Context, req *Request) (*Worker, error) {
	lb.mu.RLock()
	strategy := lb.strategy
	lb.mu.RUnlock()
	
	// Get eligible workers for the model
	eligibleWorkers := lb.getEligibleWorkers(ctx, req.ModelID)
	
	if len(eligibleWorkers) == 0 {
		return nil, fmt.Errorf("no healthy workers for model %s", req.ModelID)
	}
	
	switch strategy {
	case BalancerLeastConnections:
		return lb.selectLeastConnections(eligibleWorkers), nil
	case BalancerRoundRobin:
		return lb.selectRoundRobin(req.ModelID, eligibleWorkers), nil
	case BalancerConsistentHash:
		return lb.selectConsistentHash(req.ModelID, eligibleWorkers), nil
	case BalancerWeighted:
		return lb.selectWeighted(eligibleWorkers), nil
	case BalancerRegional:
		return lb.selectRegional(req, eligibleWorkers), nil
	default:
		return lb.selectLeastConnections(eligibleWorkers), nil
	}
}

func (lb *LoadBalancer) getEligibleWorkers(ctx context.Context, modelID string) []*Worker {
	// Get worker IDs for this model from Redis
	workerIDs, _ := lb.rdb.SMembers(ctx, fmt.Sprintf("lb:model:%s", modelID)).Result()
	
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	var eligible []*Worker
	for _, id := range workerIDs {
		if w, exists := lb.workers[id]; exists {
			if w.Healthy && time.Since(w.LastSeen) < 60*time.Second {
				eligible = append(eligible, w)
			}
		}
	}
	
	return eligible
}

func (lb *LoadBalancer) selectLeastConnections(workers []*Worker) *Worker {
	best := workers[0]
	bestScore := math.MaxFloat64
	
	for _, w := range workers {
		// Score = active_connections / max_connections
		score := float64(w.ActiveConns) / float64(w.MaxConns)
		if score < bestScore {
			bestScore = score
			best = w
		}
	}
	
	return best
}

func (lb *LoadBalancer) selectRoundRobin(modelID string, workers []*Worker) *Worker {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	idx := lb.rrIndex[modelID]
	worker := workers[idx%len(workers)]
	lb.rrIndex[modelID] = (idx + 1) % len(workers)
	
	return worker
}

func (lb *LoadBalancer) selectConsistentHash(modelID string, workers []*Worker) *Worker {
	if lb.hashRing == nil {
		return workers[0]
	}
	
	workerID := lb.hashRing.Get(modelID)
	for _, w := range workers {
		if w.ID == workerID {
			return w
		}
	}
	
	return workers[0]
}

func (lb *LoadBalancer) selectWeighted(workers []*Worker) *Worker {
	totalWeight := 0
	for _, w := range workers {
		totalWeight += w.Weight
	}
	
	if totalWeight == 0 {
		return workers[0]
	}
	
	// Simple weighted random selection
	target := int(hashString(fmt.Sprintf("%d", time.Now().UnixNano())) % uint32(totalWeight))
	cumulative := 0
	
	for _, w := range workers {
		cumulative += w.Weight
		if target < cumulative {
			return w
		}
	}
	
	return workers[len(workers)-1]
}

func (lb *LoadBalancer) selectRegional(req *Request, workers []*Worker) *Worker {
	// First try same region
	var sameRegion []*Worker
	var sameZone []*Worker
	
	for _, w := range workers {
		if w.Region == req.Region {
			sameRegion = append(sameRegion, w)
			if w.Zone == req.Region { // zone matches region
				sameZone = append(sameZone, w)
			}
		}
	}
	
	// Prefer same zone > same region > any region
	if len(sameZone) > 0 {
		return lb.selectLeastConnections(sameZone)
	}
	if len(sameRegion) > 0 {
		return lb.selectLeastConnections(sameRegion)
	}
	
	// Fallback to least connections across all regions
	return lb.selectLeastConnections(workers)
}

// UpdateWorkerStats updates a worker's statistics.
func (lb *LoadBalancer) UpdateWorkerStats(ctx context.Context, workerID string, activeConns int, gpuUsage float64) {
	lb.mu.Lock()
	if w, exists := lb.workers[workerID]; exists {
		w.ActiveConns = activeConns
		w.GPUUsage = gpuUsage
		w.LastSeen = time.Now()
	}
	lb.mu.Unlock()
	
	// Update Redis
	lb.rdb.HSet(ctx, fmt.Sprintf("lb:worker:%s", workerID),
		"active_conns", activeConns,
		"gpu_usage", gpuUsage,
		"last_seen", time.Now().Unix(),
	)
}

// GetWorkerStats returns statistics for all workers.
func (lb *LoadBalancer) GetWorkerStats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	stats := make(map[string]interface{})
	for id, w := range lb.workers {
		stats[id] = map[string]interface{}{
			"active_conns": w.ActiveConns,
			"max_conns":    w.MaxConns,
			"gpu_usage":    w.GPUUsage,
			"healthy":      w.Healthy,
			"region":       w.Region,
		}
	}
	
	return stats
}

// ConsistentHashRing implements consistent hashing.
type ConsistentHashRing struct {
	mu          sync.RWMutex
	replicas    int
	ring        map[uint32]string // hash -> worker ID
	sortedKeys  []uint32
	workerAddrs map[string]string // worker ID -> address
}

func NewConsistentHashRing(replicas int) *ConsistentHashRing {
	return &ConsistentHashRing{
		replicas:    replicas,
		ring:        make(map[uint32]string),
		workerAddrs: make(map[string]string),
	}
}

func (chr *ConsistentHashRing) Add(workerID, addr string) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	
	chr.workerAddrs[workerID] = addr
	
	for i := 0; i < chr.replicas; i++ {
		hash := hashString(fmt.Sprintf("%s-%d", workerID, i))
		chr.ring[hash] = workerID
		chr.sortedKeys = append(chr.sortedKeys, hash)
	}
	
	// Sort keys for binary search
	sortUint32(chr.sortedKeys)
}

func (chr *ConsistentHashRing) Remove(workerID string) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	
	delete(chr.workerAddrs, workerID)
	
	for i := 0; i < chr.replicas; i++ {
		hash := hashString(fmt.Sprintf("%s-%d", workerID, i))
		delete(chr.ring, hash)
	}
	
	// Rebuild sorted keys
	chr.sortedKeys = chr.sortedKeys[:0]
	for hash := range chr.ring {
		chr.sortedKeys = append(chr.sortedKeys, hash)
	}
	sortUint32(chr.sortedKeys)
}

func (chr *ConsistentHashRing) Get(key string) string {
	if len(chr.ring) == 0 {
		return ""
	}
	
	chr.mu.RLock()
	defer chr.mu.RUnlock()
	
	hash := hashString(key)
	
	// Binary search for the first hash >= key hash
	idx := binarySearchUint32(chr.sortedKeys, hash)
	if idx >= len(chr.sortedKeys) {
		idx = 0 // Wrap around
	}
	
	return chr.ring[chr.sortedKeys[idx]]
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func sortUint32(slice []uint32) {
	for i := range slice {
		for j := i + 1; j < len(slice); j++ {
			if slice[i] > slice[j] {
				slice[i], slice[j] = slice[j], slice[i]
			}
		}
	}
}

func binarySearchUint32(slice []uint32, target uint32) int {
	left, right := 0, len(slice)
	for left < right {
		mid := left + (right-left)/2
		if slice[mid] < target {
			left = mid + 1
		} else {
			right = mid
		}
	}
	return left
}
