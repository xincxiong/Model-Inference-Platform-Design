package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xincxiong/model-inference-platform/backend/internal/hami"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
	"go.uber.org/zap"
)

type DedicatedEndpointsHandler struct {
	store     *store.Store
	scheduler *hami.Scheduler
	logger    *zap.Logger
}

func NewDedicatedEndpointsHandler(s *store.Store, scheduler *hami.Scheduler, logger *zap.Logger) *DedicatedEndpointsHandler {
	return &DedicatedEndpointsHandler{store: s, scheduler: scheduler, logger: logger}
}

// ListTemplates GET /v0/dedicated_endpoints/templates
func (h *DedicatedEndpointsHandler) ListTemplates(c *gin.Context) {
	rows, err := h.store.DB.Query(c.Request.Context(),
		`SELECT id, name, model_type, provider FROM models
		 WHERE status = 'active' AND model_type IN ('text-to-text', 'vision')
		 ORDER BY model_type, id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var templates []model.DedicatedEndpointTemplate
	for rows.Next() {
		var t model.DedicatedEndpointTemplate
		if err := rows.Scan(&t.ModelName, &t.Name, &t.ModelType, &t.Provider); err != nil {
			continue
		}
		templates = append(templates, t)
	}
	c.JSON(http.StatusOK, model.DedicatedEndpointTemplatesResponse{Templates: templates})
}

// List GET /v0/dedicated_endpoints
func (h *DedicatedEndpointsHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(c.Request.Context(),
		`SELECT id, name, description, model_name, flavor_name, gpu_type, gpu_count, region,
			min_replicas, max_replicas, scaling_policy, routing_prefix, status, current_replicas,
			gpu_memory_mib, gpu_cores, scheduler_policy, topology_aware, hard_isolation,
			scheduled_node, physical_gpu_id, created_at, updated_at
		 FROM dedicated_endpoints WHERE user_id = $1 ORDER BY created_at DESC`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var data []model.DedicatedEndpoint
	for rows.Next() {
		ep, err := h.scanEndpoint(rows)
		if err != nil {
			continue
		}
		data = append(data, ep)
	}
	c.JSON(http.StatusOK, model.DedicatedEndpointsListResponse{Data: data})
}

func (h *DedicatedEndpointsHandler) scanEndpoint(rows interface {
	Scan(dest ...any) error
}) (model.DedicatedEndpoint, error) {
	var ep model.DedicatedEndpoint
	var policyJSON []byte
	err := rows.Scan(
		&ep.ID, &ep.Name, &ep.Description, &ep.ModelName, &ep.FlavorName,
		&ep.GPUType, &ep.GPUCount, &ep.Region, &ep.MinReplicas, &ep.MaxReplicas,
		&policyJSON, &ep.RoutingPrefix, &ep.Status, &ep.CurrentReplicas,
		&ep.GPUMemoryMiB, &ep.GPUCores, &ep.SchedulerPolicy,
		&ep.TopologyAware, &ep.HardIsolation,
		&ep.ScheduledNode, &ep.PhysicalGPUID,
		&ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		return ep, err
	}
	_ = json.Unmarshal(policyJSON, &ep.ScalingPolicy)
	if ep.ScalingPolicy == nil {
		ep.ScalingPolicy = map[string]interface{}{}
	}
	ep.RoutingKey = ep.RoutingPrefix + ":" + ep.ModelName
	return ep, nil
}

// Create POST /v0/dedicated_endpoints
func (h *DedicatedEndpointsHandler) Create(c *gin.Context) {
	var req model.CreateDedicatedEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	if req.GPUCount < 1 {
		req.GPUCount = 1
	}
	if req.Region == "" {
		req.Region = "cn-east-1"
	}
	if req.FlavorName == "" {
		req.FlavorName = "base"
	}
	if req.MinReplicas < 0 {
		req.MinReplicas = 0
	}
	if req.MaxReplicas < 1 {
		req.MaxReplicas = 4
	}
	if req.MaxReplicas < req.MinReplicas {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "max_replicas must be >= min_replicas"}})
		return
	}
	// HAMi defaults
	if req.GPUMemoryMiB <= 0 {
		req.GPUMemoryMiB = 8192 // default 8GB
	}
	if req.SchedulerPolicy == "" {
		req.SchedulerPolicy = "binpack"
	}

	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	var modelType string
	err := h.store.DB.QueryRow(ctx,
		`SELECT model_type FROM models WHERE id = $1 AND status = 'active'`, req.ModelName).
		Scan(&modelType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "model_name is not an active catalog model"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if modelType != "text-to-text" && modelType != "vision" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "dedicated endpoints only support text-to-text or vision base models"}})
		return
	}

	policy := req.ScalingPolicy
	if policy == nil {
		policy = map[string]interface{}{}
	}
	policyBytes, _ := json.Marshal(policy)

	routingPrefix, err := h.allocRoutingPrefix(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// ── HAMi scheduling decision ──────────────────────────────────────────
	scheduledNode := ""
	physicalGPUID := 0
	if h.scheduler != nil {
		schedCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()

		schedResult, schedErr := h.scheduler.ScheduleEndpoint(schedCtx, hami.EndpointScheduleRequest{
			EndpointID: routingPrefix,
			ModelName:  req.ModelName,
			GPUSpec: hami.GPUResourceSpec{
				Count:     req.GPUCount,
				MemoryMiB: req.GPUMemoryMiB,
				Cores:     req.GPUCores,
				Vendor:    hami.VendorNVIDIA,
			},
			SchedulerPolicy: hami.SchedulerPolicy(req.SchedulerPolicy),
			TopologyAware:   req.TopologyAware,
			MinReplicas:     req.MinReplicas,
			MaxReplicas:     req.MaxReplicas,
			IsDedicated:     true,
		})
		if schedErr != nil {
			// Log warning but do not block endpoint creation in stub mode
			h.logger.Warn("hami scheduling failed, proceeding with stub placement",
				zap.String("routing_prefix", routingPrefix),
				zap.Error(schedErr))
		} else {
			scheduledNode = schedResult.NodeName
			physicalGPUID = schedResult.PhysicalGPUID
		}
	}

	var id string
	err = h.store.DB.QueryRow(ctx,
		`INSERT INTO dedicated_endpoints (
			user_id, name, description, model_name, flavor_name, gpu_type, gpu_count, region,
			min_replicas, max_replicas, scaling_policy, routing_prefix, status, current_replicas,
			gpu_memory_mib, gpu_cores, scheduler_policy, topology_aware, hard_isolation,
			scheduled_node, physical_gpu_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,'running',$13,
			$14,$15,$16,$17,$18,$19,$20) RETURNING id`,
		auth.UserID, req.Name, req.Description, req.ModelName, req.FlavorName, req.GPUType, req.GPUCount, req.Region,
		req.MinReplicas, req.MaxReplicas, policyBytes, routingPrefix, req.MinReplicas,
		req.GPUMemoryMiB, req.GPUCores, req.SchedulerPolicy, req.TopologyAware, req.HardIsolation,
		scheduledNode, physicalGPUID,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	row := h.store.DB.QueryRow(ctx,
		`SELECT id, name, description, model_name, flavor_name, gpu_type, gpu_count, region,
			min_replicas, max_replicas, scaling_policy, routing_prefix, status, current_replicas,
			gpu_memory_mib, gpu_cores, scheduler_policy, topology_aware, hard_isolation,
			scheduled_node, physical_gpu_id, created_at, updated_at
		 FROM dedicated_endpoints WHERE id = $1`, id)
	ep, err := h.scanEndpoint(row)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"id": id, "routing_key": routingPrefix + ":" + req.ModelName})
		return
	}
	c.JSON(http.StatusCreated, ep)
}

func (h *DedicatedEndpointsHandler) allocRoutingPrefix(ctx context.Context) (string, error) {
	for i := 0; i < 8; i++ {
		u := strings.ReplaceAll(uuid.New().String(), "-", "")
		prefix := "ep_" + u[:8]
		var exists bool
		_ = h.store.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM dedicated_endpoints WHERE routing_prefix = $1)`, prefix).Scan(&exists)
		if !exists {
			return prefix, nil
		}
	}
	return "", fmt.Errorf("could not allocate unique routing prefix")
}

// Patch PATCH /v0/dedicated_endpoints/:id
func (h *DedicatedEndpointsHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	var req model.PatchDedicatedEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	row := h.store.DB.QueryRow(ctx,
		`SELECT id, name, description, model_name, flavor_name, gpu_type, gpu_count, region,
			min_replicas, max_replicas, scaling_policy, routing_prefix, status, current_replicas,
			gpu_memory_mib, gpu_cores, scheduler_policy, topology_aware, hard_isolation,
			scheduled_node, physical_gpu_id, created_at, updated_at
		 FROM dedicated_endpoints WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	ep, err := h.scanEndpoint(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "endpoint not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	if req.Name != nil {
		ep.Name = *req.Name
	}
	if req.Description != nil {
		ep.Description = *req.Description
	}
	if req.MinReplicas != nil {
		ep.MinReplicas = *req.MinReplicas
	}
	if req.MaxReplicas != nil {
		ep.MaxReplicas = *req.MaxReplicas
	}
	if req.CurrentReplicas != nil {
		ep.CurrentReplicas = *req.CurrentReplicas
	}
	if req.Status != nil {
		s := *req.Status
		switch s {
		case "running", "stopped", "provisioning", "error", "deleting":
			ep.Status = s
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid status"}})
			return
		}
	}
	// HAMi fields
	if req.GPUMemoryMiB != nil {
		ep.GPUMemoryMiB = *req.GPUMemoryMiB
	}
	if req.GPUCores != nil {
		ep.GPUCores = *req.GPUCores
	}
	if req.SchedulerPolicy != nil {
		ep.SchedulerPolicy = *req.SchedulerPolicy
	}

	if ep.MaxReplicas < ep.MinReplicas {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "max_replicas must be >= min_replicas"}})
		return
	}

	policyBytes, _ := json.Marshal(ep.ScalingPolicy)
	_, err = h.store.DB.Exec(ctx,
		`UPDATE dedicated_endpoints SET
			name = $2, description = $3, min_replicas = $4, max_replicas = $5,
			status = $6, current_replicas = $7, scaling_policy = $8::jsonb,
			gpu_memory_mib = $9, gpu_cores = $10, scheduler_policy = $11,
			updated_at = $12
		 WHERE id = $1 AND user_id = $13`,
		id, ep.Name, ep.Description, ep.MinReplicas, ep.MaxReplicas, ep.Status, ep.CurrentReplicas,
		policyBytes, ep.GPUMemoryMiB, ep.GPUCores, ep.SchedulerPolicy,
		time.Now().UTC(), auth.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	row2 := h.store.DB.QueryRow(ctx,
		`SELECT id, name, description, model_name, flavor_name, gpu_type, gpu_count, region,
			min_replicas, max_replicas, scaling_policy, routing_prefix, status, current_replicas,
			gpu_memory_mib, gpu_cores, scheduler_policy, topology_aware, hard_isolation,
			scheduled_node, physical_gpu_id, created_at, updated_at
		 FROM dedicated_endpoints WHERE id = $1`, id)
	ep2, err := h.scanEndpoint(row2)
	if err != nil {
		c.JSON(http.StatusOK, ep)
		return
	}
	c.JSON(http.StatusOK, ep2)
}

// Delete DELETE /v0/dedicated_endpoints/:id
func (h *DedicatedEndpointsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	cmd, err := h.store.DB.Exec(c.Request.Context(),
		`DELETE FROM dedicated_endpoints WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "endpoint not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
