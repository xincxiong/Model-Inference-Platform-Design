package hami

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// Scheduler wraps the HAMi Client with higher-level scheduling logic
// tailored to the inference platform's endpoint lifecycle.
type Scheduler struct {
	client *Client
	logger *zap.Logger
}

// NewScheduler creates a Scheduler from environment variables.
//
// Environment variables:
//
//	HAMI_ENABLED               - "true" to enable real HAMi scheduling (default: false)
//	HAMI_SCHEDULER_ENDPOINT    - HAMi extender URL (default: cluster DNS)
//	HAMI_SCHEDULER_TIMEOUT_SEC - HTTP timeout in seconds (default: 5)
func NewScheduler(logger *zap.Logger) *Scheduler {
	enabled := os.Getenv("HAMI_ENABLED") == "true"
	endpoint := os.Getenv("HAMI_SCHEDULER_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://hami-scheduler.hami-system.svc.cluster.local:9090"
	}

	timeoutSec := 5
	if s := os.Getenv("HAMI_SCHEDULER_TIMEOUT_SEC"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			timeoutSec = n
		}
	}

	cfg := ClientConfig{
		Endpoint: endpoint,
		Enabled:  enabled,
		Timeout:  time.Duration(timeoutSec) * time.Second,
	}

	return &Scheduler{
		client: NewClient(cfg, logger),
		logger: logger,
	}
}

// ScheduleEndpoint selects an optimal GPU placement for a new dedicated endpoint.
// It applies the appropriate scheduling policy based on endpoint configuration:
//
//   - Dedicated (high-availability) endpoints → PolicySpread
//   - Shared inference pool → PolicyBinpack (default)
//   - MoE models with expert parallelism → PolicyTopologyAware
//
// Returns the placement result; callers should record node/GPU info in the DB
// and pass it through when creating the Kubernetes Deployment.
func (s *Scheduler) ScheduleEndpoint(ctx context.Context, req EndpointScheduleRequest) (*ScheduleResult, error) {
	policy := selectPolicy(req)

	hamiReq := ScheduleRequest{
		EndpointID:    req.EndpointID,
		ModelName:     req.ModelName,
		GPUSpec:       req.GPUSpec,
		Policy:        policy,
		Replicas:      req.MinReplicas,
		TopologyAware: req.TopologyAware || policy == PolicyTopologyAware,
	}

	if hamiReq.Replicas < 1 {
		hamiReq.Replicas = 1
	}

	s.logger.Info("scheduling endpoint",
		zap.String("endpoint_id", req.EndpointID),
		zap.String("model", req.ModelName),
		zap.String("policy", string(policy)),
		zap.Int("gpu_mem_mib", req.GPUSpec.MemoryMiB),
		zap.Int("gpu_cores", req.GPUSpec.Cores),
	)

	result, err := s.client.Schedule(ctx, hamiReq)
	if err != nil {
		return nil, fmt.Errorf("hami schedule endpoint %s: %w", req.EndpointID, err)
	}

	if !result.Scheduled {
		return nil, fmt.Errorf("hami could not schedule endpoint %s: %s", req.EndpointID, result.Reason)
	}

	s.logger.Info("endpoint scheduled",
		zap.String("endpoint_id", req.EndpointID),
		zap.String("node", result.NodeName),
		zap.Int("gpu_id", result.PhysicalGPUID),
		zap.Int("allocated_mem_mib", result.AllocatedMemoryMiB),
	)

	return result, nil
}

// GetNodeGPUSummary returns a summary of available GPU resources across all nodes.
func (s *Scheduler) GetNodeGPUSummary(ctx context.Context) ([]NodeGPUInfo, error) {
	return s.client.ListNodes(ctx)
}

// Ping checks HAMi scheduler connectivity for health checks.
func (s *Scheduler) Ping(ctx context.Context) error {
	return s.client.Ping(ctx)
}

// IsEnabled returns whether real HAMi scheduling is active.
func (s *Scheduler) IsEnabled() bool {
	return s.client.enabled
}

// ─── EndpointScheduleRequest ─────────────────────────────────────────────────

// EndpointScheduleRequest maps a DedicatedEndpoint creation request
// to HAMi scheduling parameters.
type EndpointScheduleRequest struct {
	EndpointID      string
	ModelName       string
	GPUSpec         GPUResourceSpec
	SchedulerPolicy SchedulerPolicy
	TopologyAware   bool
	MinReplicas     int
	MaxReplicas     int
	// IsDedicated distinguishes dedicated endpoints (spread) from shared pool (binpack)
	IsDedicated bool
}

// ─── Private helpers ──────────────────────────────────────────────────────────

// selectPolicy determines the optimal HAMi scheduling policy for an endpoint.
func selectPolicy(req EndpointScheduleRequest) SchedulerPolicy {
	// Explicit override wins
	if req.SchedulerPolicy != "" {
		return req.SchedulerPolicy
	}
	// Topology-aware for large MoE models (>= 70B activation params typical)
	// identified by gpumem >= 40GB (heuristic)
	if req.TopologyAware || req.GPUSpec.MemoryMiB >= 40960 {
		return PolicyTopologyAware
	}
	// Dedicated endpoints prefer high-availability spread
	if req.IsDedicated && req.MaxReplicas > 1 {
		return PolicySpread
	}
	// Default: binpack maximizes GPU utilization for shared inference
	return PolicyBinpack
}
