package model

import "time"

// ComputePool lifecycle.
const (
	PoolStatusActive    = "active"
	PoolStatusSuspended = "suspended"
	PoolStatusArchived  = "archived"
)

// Sharing modes (mirrored from pool_skus).
const (
	PoolSharingExclusive  = "exclusive"
	PoolSharingSharedFIFO = "shared-fifo"
)

// ComputePool is the operational resource bound to a pool_subscription.
// One subscription = one active ComputePool (1:1 enforced by partial unique index).
type ComputePool struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	GPUType         string     `json:"gpu_type"`
	GPUCount        int        `json:"gpu_count"`
	Region          string     `json:"region"`
	SharingMode     string     `json:"sharing_mode"`
	SubscriptionID  string     `json:"subscription_id"`
	UserID          string     `json:"user_id"`
	VolcanoQueue    string     `json:"volcano_queue"`
	SchedulerPolicy string     `json:"scheduler_policy"`
	HardIsolation   bool       `json:"hard_isolation"`
	Status          string     `json:"status"`
	UsedGPU         int        `json:"used_gpu"`
	ServiceStartAt  time.Time  `json:"service_start_at"`
	ServiceEndAt    *time.Time `json:"service_end_at,omitempty"`
	SLAClass        string     `json:"sla_class"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CreateComputePoolRequest is the body for POST /v0/pools/compute.
// The pool is instantiated from an active pool_subscription.
type CreateComputePoolRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	SubscriptionID  string `json:"subscription_id" binding:"required"`
	SchedulerPolicy string `json:"scheduler_policy"` // binpack | spread | topology-aware
	HardIsolation   bool   `json:"hard_isolation"`
}

// PatchComputePoolRequest is the body for PATCH /v0/pools/compute/:id.
type PatchComputePoolRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	SchedulerPolicy *string `json:"scheduler_policy"`
	HardIsolation   *bool   `json:"hard_isolation"`
}

// ComputePoolWithUsage enriches a pool with live runtime data.
type ComputePoolWithUsage struct {
	ComputePool
	FreeGPU        int     `json:"free_gpu"`
	UtilizationPct float64 `json:"utilization_pct"`
	ActiveJobs     int     `json:"active_jobs"`
	QueueDepth     int     `json:"queue_depth"`
	// Subscription snapshot (joined) for display.
	SKUID      string `json:"sku_id,omitempty"`
	SKUTerm    string `json:"sku_term,omitempty"`
	SubEndAt   string `json:"subscription_end_at,omitempty"`
	SubStatus  string `json:"subscription_status,omitempty"`
}

// ListComputePoolsResponse is the response for GET /v0/pools/compute.
type ListComputePoolsResponse struct {
	Data []ComputePoolWithUsage `json:"data"`
}

// GetComputePoolResponse is the response for GET /v0/pools/compute/:id.
type GetComputePoolResponse struct {
	Data ComputePoolWithUsage `json:"data"`
}
