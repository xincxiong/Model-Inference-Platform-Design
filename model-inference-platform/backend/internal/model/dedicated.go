package model

import "time"

// DedicatedEndpointTemplate is a catalog row suitable for creating a dedicated endpoint (Phase 2).
type DedicatedEndpointTemplate struct {
	ModelName string `json:"model_name"`
	Name      string `json:"name"`
	ModelType string `json:"model_type"`
	Provider  string `json:"provider"`
}

// CreateDedicatedEndpointRequest is the body for POST /v0/dedicated_endpoints.
type CreateDedicatedEndpointRequest struct {
	Name          string                 `json:"name" binding:"required"`
	Description   string                 `json:"description"`
	ModelName     string                 `json:"model_name" binding:"required"`
	FlavorName    string                 `json:"flavor_name"`
	GPUType       string                 `json:"gpu_type" binding:"required"`
	GPUCount      int                    `json:"gpu_count"`
	Region        string                 `json:"region"`
	MinReplicas   int                    `json:"min_replicas"`
	MaxReplicas   int                    `json:"max_replicas"`
	ScalingPolicy map[string]interface{} `json:"scaling_policy"`

	// ── HAMi GPU virtualization fields ────────────────────────────────────
	// GPUMemoryMiB is GPU memory to reserve per worker pod in MiB (hard isolation).
	// HAMi enforces this limit at hardware level. 0 = use platform default.
	// Example: 20480 = 20GB
	GPUMemoryMiB int `json:"gpu_memory_mib"`

	// GPUCores is the GPU compute core quota per worker pod (0–100%).
	// 0 = no limit (unrestricted). Example: 60 = 60% of physical GPU cores.
	GPUCores int `json:"gpu_cores"`

	// SchedulerPolicy controls HAMi placement strategy.
	// Values: "binpack" (default, max utilization) | "spread" (HA) | "topology-aware" (NVLink MoE)
	SchedulerPolicy string `json:"scheduler_policy"`

	// TopologyAware enables NVLink/NVSwitch topology optimization for multi-GPU models.
	TopologyAware bool `json:"topology_aware"`

	// HardIsolation enforces strict GPU memory isolation via HAMi device plugin.
	// When true, workers cannot exceed GPUMemoryMiB regardless of physical availability.
	HardIsolation bool `json:"hard_isolation"`
}

// PatchDedicatedEndpointRequest is the body for PATCH /v0/dedicated_endpoints/:id.
type PatchDedicatedEndpointRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	MinReplicas     *int    `json:"min_replicas"`
	MaxReplicas     *int    `json:"max_replicas"`
	Status          *string `json:"status"` // running, stopped, provisioning, error
	CurrentReplicas *int    `json:"current_replicas"`

	// ── HAMi fields (patchable after creation) ────────────────────────────
	GPUMemoryMiB    *int    `json:"gpu_memory_mib"`
	GPUCores        *int    `json:"gpu_cores"`
	SchedulerPolicy *string `json:"scheduler_policy"`
}

// DedicatedEndpoint is the API shape for a dedicated inference endpoint.
type DedicatedEndpoint struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	ModelName       string                 `json:"model_name"`
	FlavorName      string                 `json:"flavor_name"`
	GPUType         string                 `json:"gpu_type"`
	GPUCount        int                    `json:"gpu_count"`
	Region          string                 `json:"region"`
	MinReplicas     int                    `json:"min_replicas"`
	MaxReplicas     int                    `json:"max_replicas"`
	ScalingPolicy   map[string]interface{} `json:"scaling_policy"`
	RoutingPrefix   string                 `json:"routing_prefix"`
	RoutingKey      string                 `json:"routing_key"` // e.g. ep_ab12cd34:deepseek-ai/DeepSeek-V4
	Status          string                 `json:"status"`
	CurrentReplicas int                    `json:"current_replicas"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`

	// ── HAMi GPU virtualization fields ────────────────────────────────────
	GPUMemoryMiB    int    `json:"gpu_memory_mib"`
	GPUCores        int    `json:"gpu_cores"`
	SchedulerPolicy string `json:"scheduler_policy"`
	TopologyAware   bool   `json:"topology_aware"`
	HardIsolation   bool   `json:"hard_isolation"`

	// ── HAMi placement result (read-only, set by scheduler) ───────────────
	ScheduledNode string `json:"scheduled_node,omitempty"`
	PhysicalGPUID int    `json:"physical_gpu_id,omitempty"`
}

type DedicatedEndpointsListResponse struct {
	Data []DedicatedEndpoint `json:"data"`
}

type DedicatedEndpointTemplatesResponse struct {
	Templates []DedicatedEndpointTemplate `json:"templates"`
}
