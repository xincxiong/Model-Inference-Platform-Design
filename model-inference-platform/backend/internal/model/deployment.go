package model

import "time"

// BillingMode constants for deployments
const (
	BillingModeToken = "token" // 按 Token 调用
	BillingModeTPU   = "tpu"   // 按置备吐单元
	BillingModeUnit  = "unit"  // 按模型单元（独占 GPU）
)

// DeploymentStatus constants
const (
	DeploymentStatusProvisioning = "provisioning"
	DeploymentStatusRunning      = "running"
	DeploymentStatusStopped      = "stopped"
	DeploymentStatusFailed       = "failed"
	DeploymentStatusDeleting     = "deleting"
)

// ─── Request / Response ────────────────────────────────────────────────────

type CreateDeploymentRequest struct {
	Name        string `json:"name" binding:"required"`
	ModelName   string `json:"model_name" binding:"required"`
	BillingMode string `json:"billing_mode" binding:"required"` // token | tpu | unit
	MinReplicas int    `json:"min_replicas"`
	MaxReplicas int    `json:"max_replicas"`
	Description string `json:"description"`
}

type PatchDeploymentRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	MinReplicas *int    `json:"min_replicas"`
	MaxReplicas *int    `json:"max_replicas"`
	Status      *string `json:"status"`
}

// ─── API Shape ──────────────────────────────────────────────────────────────

type Deployment struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ModelName   string    `json:"model_name"`
	BillingMode string    `json:"billing_mode"`
	MinReplicas int       `json:"min_replicas"`
	MaxReplicas int       `json:"max_replicas"`
	Status      string    `json:"status"`
	Endpoint    string    `json:"endpoint"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DeploymentListResponse struct {
	Object string       `json:"object"`
	Data   []Deployment `json:"data"`
}
