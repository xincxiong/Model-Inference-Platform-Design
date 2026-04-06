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
	Name           string                 `json:"name" binding:"required"`
	Description    string                 `json:"description"`
	ModelName      string                 `json:"model_name" binding:"required"`
	FlavorName     string                 `json:"flavor_name"`
	GPUType        string                 `json:"gpu_type" binding:"required"`
	GPUCount       int                    `json:"gpu_count"`
	Region         string                 `json:"region"`
	MinReplicas    int                    `json:"min_replicas"`
	MaxReplicas    int                    `json:"max_replicas"`
	ScalingPolicy  map[string]interface{} `json:"scaling_policy"`
}

// PatchDedicatedEndpointRequest is the body for PATCH /v0/dedicated_endpoints/:id.
type PatchDedicatedEndpointRequest struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	MinReplicas   *int    `json:"min_replicas"`
	MaxReplicas   *int    `json:"max_replicas"`
	Status          *string `json:"status"` // running, stopped, provisioning, error
	CurrentReplicas *int    `json:"current_replicas"`
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
}

type DedicatedEndpointsListResponse struct {
	Data []DedicatedEndpoint `json:"data"`
}

type DedicatedEndpointTemplatesResponse struct {
	Templates []DedicatedEndpointTemplate `json:"templates"`
}
