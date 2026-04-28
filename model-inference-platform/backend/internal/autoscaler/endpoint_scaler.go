// Package autoscaler provides automatic scaling for dedicated endpoints.
package autoscaler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ScalingStrategy defines how the endpoint should scale.
type ScalingStrategy string

const (
	StrategyQPS       ScalingStrategy = "qps"        // Scale based on queries per second
	StrategyQueueDepth ScalingStrategy = "queue_depth" // Scale based on pending request queue
	StrategyGPUUtil    ScalingStrategy = "gpu_util"    // Scale based on GPU utilization
	StrategyComposite  ScalingStrategy = "composite"   // Composite of multiple metrics
)

// EndpointConfig holds scaling configuration for an endpoint.
type EndpointConfig struct {
	EndpointID      string
	MinReplicas     int
	MaxReplicas     int
	TargetQPS       float64       // target queries per second per replica
	TargetQueueDepth int          // target pending requests per replica
	TargetGPUUtil   float64       // target GPU utilization (0-100)
	ScaleUpStep     int           // max replicas to add in one scaling event
	ScaleDownStep   int           // max replicas to remove in one scaling event
	ScaleUpCooldown time.Duration // cooldown after scaling up
	ScaleDownCooldown time.Duration // cooldown after scaling down
	Strategy        ScalingStrategy
	EvaluationInterval time.Duration // how often to evaluate scaling
}

// DefaultEndpointConfig returns sensible defaults.
func DefaultEndpointConfig(endpointID string) EndpointConfig {
	return EndpointConfig{
		EndpointID:        endpointID,
		MinReplicas:       1,
		MaxReplicas:       10,
		TargetQPS:         50.0,
		TargetQueueDepth:  5,
		TargetGPUUtil:     75.0,
		ScaleUpStep:       2,
		ScaleDownStep:     1,
		ScaleUpCooldown:   60 * time.Second,
		ScaleDownCooldown: 300 * time.Second,
		Strategy:          StrategyComposite,
		EvaluationInterval: 15 * time.Second,
	}
}

// Metrics holds current metrics for an endpoint.
type Metrics struct {
	CurrentReplicas int
	CurrentQPS      float64
	QueueDepth      int
	GPUUtilization  float64
}

// Scaler manages automatic scaling of dedicated endpoints.
type Scaler struct {
	rdb      *redis.Client
	logger   *zap.Logger
	configs  map[string]*EndpointConfig
	mu       sync.RWMutex
	lastScaleUp   map[string]time.Time
	lastScaleDown map[string]time.Time
	scaleFn    func(endpointID string, desiredReplicas int) error // callback to actually scale
}

// New creates a new Scaler.
func New(rdb *redis.Client, logger *zap.Logger, scaleFn func(endpointID string, desiredReplicas int) error) *Scaler {
	return &Scaler{
		rdb:         rdb,
		logger:      logger,
		configs:     make(map[string]*EndpointConfig),
		lastScaleUp:   make(map[string]time.Time),
		lastScaleDown: make(map[string]time.Time),
		scaleFn:     scaleFn,
	}
}

// Register registers an endpoint for automatic scaling.
func (s *Scaler) Register(ctx context.Context, cfg EndpointConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[cfg.EndpointID] = &cfg
	
	// Start evaluation loop
	go s.evaluateLoop(ctx, cfg)
	
	s.logger.Info("registered endpoint for autoscaling",
		zap.String("endpoint_id", cfg.EndpointID),
		zap.Int("min_replicas", cfg.MinReplicas),
		zap.Int("max_replicas", cfg.MaxReplicas),
		zap.String("strategy", string(cfg.Strategy)))
}

// Unregister removes an endpoint from automatic scaling.
func (s *Scaler) Unregister(endpointID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.configs, endpointID)
}

// evaluateLoop periodically evaluates whether to scale the endpoint.
func (s *Scaler) evaluateLoop(ctx context.Context, cfg EndpointConfig) {
	ticker := time.NewTicker(cfg.EvaluationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.evaluate(ctx, cfg)
		}
	}
}

// evaluate checks metrics and decides whether to scale.
func (s *Scaler) evaluate(ctx context.Context, cfg EndpointConfig) {
	// Collect current metrics
	metrics := s.collectMetrics(ctx, cfg)
	
	// Calculate desired replicas based on strategy
	desiredReplicas := s.calculateDesiredReplicas(cfg, metrics)
	
	// Enforce min/max bounds
	desiredReplicas = int(math.Max(float64(cfg.MinReplicas), float64(desiredReplicas)))
	desiredReplicas = int(math.Min(float64(cfg.MaxReplicas), float64(desiredReplicas)))
	
	// Check if scaling is needed
	currentReplicas := metrics.CurrentReplicas
	if desiredReplicas == currentReplicas {
		return
	}
	
	// Check cooldown
	if desiredReplicas > currentReplicas {
		// Scale up
		if time.Since(s.lastScaleUp[cfg.EndpointID]) < cfg.ScaleUpCooldown {
			return // still in cooldown
		}
		
		// Limit scale up step
		actualScaleUp := desiredReplicas - currentReplicas
		if actualScaleUp > cfg.ScaleUpStep {
			actualScaleUp = cfg.ScaleUpStep
		}
		desiredReplicas = currentReplicas + actualScaleUp
		
		s.logger.Info("scaling up endpoint",
			zap.String("endpoint_id", cfg.EndpointID),
			zap.Int("current_replicas", currentReplicas),
			zap.Int("desired_replicas", desiredReplicas),
			zap.Float64("current_qps", metrics.CurrentQPS),
			zap.Int("queue_depth", metrics.QueueDepth),
			zap.Float64("gpu_util", metrics.GPUUtilization))
		
		if err := s.scaleFn(cfg.EndpointID, desiredReplicas); err != nil {
			s.logger.Error("failed to scale up", zap.String("endpoint_id", cfg.EndpointID), zap.Error(err))
			return
		}
		
		s.lastScaleUp[cfg.EndpointID] = time.Now()
		
	} else if desiredReplicas < currentReplicas {
		// Scale down
		if time.Since(s.lastScaleDown[cfg.EndpointID]) < cfg.ScaleDownCooldown {
			return // still in cooldown
		}
		
		// Limit scale down step
		actualScaleDown := currentReplicas - desiredReplicas
		if actualScaleDown > cfg.ScaleDownStep {
			actualScaleDown = cfg.ScaleDownStep
		}
		desiredReplicas = currentReplicas - actualScaleDown
		
		s.logger.Info("scaling down endpoint",
			zap.String("endpoint_id", cfg.EndpointID),
			zap.Int("current_replicas", currentReplicas),
			zap.Int("desired_replicas", desiredReplicas),
			zap.Float64("current_qps", metrics.CurrentQPS),
			zap.Int("queue_depth", metrics.QueueDepth),
			zap.Float64("gpu_util", metrics.GPUUtilization))
		
		if err := s.scaleFn(cfg.EndpointID, desiredReplicas); err != nil {
			s.logger.Error("failed to scale down", zap.String("endpoint_id", cfg.EndpointID), zap.Error(err))
			return
		}
		
		s.lastScaleDown[cfg.EndpointID] = time.Now()
	}
}

// collectMetrics gathers current metrics for the endpoint.
func (s *Scaler) collectMetrics(ctx context.Context, cfg EndpointConfig) Metrics {
	// Get current replica count from Redis
	replicaKey := fmt.Sprintf("endpoint:%s:replicas", cfg.EndpointID)
	currentReplicas, _ := s.rdb.Get(ctx, replicaKey).Int()
	
	// Get QPS from Redis (requests in last 60 seconds)
	qpsKey := fmt.Sprintf("endpoint:%s:qps", cfg.EndpointID)
	currentQPS, _ := s.rdb.Get(ctx, qpsKey).Float64()
	
	// Get queue depth
	queueKey := fmt.Sprintf("endpoint:%s:queue_depth", cfg.EndpointID)
	queueDepth, _ := s.rdb.Get(ctx, queueKey).Int()
	
	// Get GPU utilization
	gpuKey := fmt.Sprintf("endpoint:%s:gpu_util", cfg.EndpointID)
	gpuUtil, _ := s.rdb.Get(ctx, gpuKey).Float64()
	
	return Metrics{
		CurrentReplicas: currentReplicas,
		CurrentQPS:      currentQPS,
		QueueDepth:      queueDepth,
		GPUUtilization:  gpuUtil,
	}
}

// calculateDesiredReplicas determines the optimal number of replicas.
func (s *Scaler) calculateDesiredReplicas(cfg EndpointConfig, metrics Metrics) int {
	switch cfg.Strategy {
	case StrategyQPS:
		return s.calculateByQPS(cfg, metrics)
	case StrategyQueueDepth:
		return s.calculateByQueueDepth(cfg, metrics)
	case StrategyGPUUtil:
		return s.calculateByGPUUtil(cfg, metrics)
	case StrategyComposite:
		return s.calculateComposite(cfg, metrics)
	default:
		return metrics.CurrentReplicas
	}
}

func (s *Scaler) calculateByQPS(cfg EndpointConfig, metrics Metrics) int {
	if metrics.CurrentQPS == 0 || cfg.TargetQPS == 0 {
		return cfg.MinReplicas
	}
	
	desired := int(math.Ceil(metrics.CurrentQPS / cfg.TargetQPS))
	return desired
}

func (s *Scaler) calculateByQueueDepth(cfg EndpointConfig, metrics Metrics) int {
	if metrics.QueueDepth == 0 || cfg.TargetQueueDepth == 0 {
		return metrics.CurrentReplicas
	}
	
	desired := int(math.Ceil(float64(metrics.QueueDepth) / float64(cfg.TargetQueueDepth)))
	return desired
}

func (s *Scaler) calculateByGPUUtil(cfg EndpointConfig, metrics Metrics) int {
	if metrics.GPUUtilization == 0 || cfg.TargetGPUUtil == 0 {
		return metrics.CurrentReplicas
	}
	
	ratio := metrics.GPUUtilization / cfg.TargetGPUUtil
	desired := int(math.Ceil(float64(metrics.CurrentReplicas) * ratio))
	return desired
}

func (s *Scaler) calculateComposite(cfg EndpointConfig, metrics Metrics) int {
	// Calculate based on each metric and take the maximum
	qpsReplicas := s.calculateByQPS(cfg, metrics)
	queueReplicas := s.calculateByQueueDepth(cfg, metrics)
	gpuReplicas := s.calculateByGPUUtil(cfg, metrics)
	
	// Take the maximum to ensure all constraints are met
	desired := qpsReplicas
	if queueReplicas > desired {
		desired = queueReplicas
	}
	if gpuReplicas > desired {
		desired = gpuReplicas
	}
	
	return desired
}

// UpdateMetrics updates metrics for an endpoint (called by handlers).
func (s *Scaler) UpdateMetrics(ctx context.Context, endpointID string, metrics Metrics) error {
	pipe := s.rdb.Pipeline()
	
	pipe.Set(ctx, fmt.Sprintf("endpoint:%s:replicas", endpointID), metrics.CurrentReplicas, 60*time.Second)
	pipe.Set(ctx, fmt.Sprintf("endpoint:%s:qps", endpointID), metrics.CurrentQPS, 60*time.Second)
	pipe.Set(ctx, fmt.Sprintf("endpoint:%s:queue_depth", endpointID), metrics.QueueDepth, 60*time.Second)
	pipe.Set(ctx, fmt.Sprintf("endpoint:%s:gpu_util", endpointID), metrics.GPUUtilization, 60*time.Second)
	
	_, err := pipe.Exec(ctx)
	return err
}
