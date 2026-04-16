package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type DeploymentsHandler struct {
	store *store.Store
}

func NewDeploymentsHandler(s *store.Store) *DeploymentsHandler {
	return &DeploymentsHandler{store: s}
}

// List GET /v1/deployments
func (h *DeploymentsHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(c.Request.Context(),
		`SELECT id, name, description, model_name, billing_mode, min_replicas, max_replicas,
		        status, endpoint, created_at, updated_at
		 FROM deployments WHERE user_id = $1 ORDER BY created_at DESC`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var data []model.Deployment
	for rows.Next() {
		d, err := scanDeployment(rows)
		if err != nil {
			continue
		}
		data = append(data, d)
	}
	if data == nil {
		data = []model.Deployment{}
	}
	c.JSON(http.StatusOK, model.DeploymentListResponse{Object: "list", Data: data})
}

// Get GET /v1/deployments/:id
func (h *DeploymentsHandler) Get(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	row := h.store.DB.QueryRow(c.Request.Context(),
		`SELECT id, name, description, model_name, billing_mode, min_replicas, max_replicas,
		        status, endpoint, created_at, updated_at
		 FROM deployments WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	d, err := scanDeployment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "deployment not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, d)
}

// Create POST /v1/deployments
func (h *DeploymentsHandler) Create(c *gin.Context) {
	var req model.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// validate billing_mode
	switch req.BillingMode {
	case model.BillingModeToken, model.BillingModeTPU, model.BillingModeUnit:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "billing_mode must be one of: token, tpu, unit"}})
		return
	}

	// defaults
	if req.MinReplicas < 0 {
		req.MinReplicas = 0
	}
	if req.MaxReplicas < 1 {
		req.MaxReplicas = 3
	}
	if req.MaxReplicas < req.MinReplicas {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "max_replicas must be >= min_replicas"}})
		return
	}

	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	// validate model exists
	var modelType string
	err := h.store.DB.QueryRow(ctx,
		`SELECT model_type FROM models WHERE id = $1 AND status = 'active'`, req.ModelName).Scan(&modelType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "model_name is not an active catalog model"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// generate unique endpoint slug
	endpointURL := h.allocEndpoint(req.ModelName)

	var id string
	err = h.store.DB.QueryRow(ctx,
		`INSERT INTO deployments (user_id, name, description, model_name, billing_mode, min_replicas, max_replicas, status, endpoint)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'provisioning',$8) RETURNING id`,
		auth.UserID, req.Name, req.Description, req.ModelName, req.BillingMode,
		req.MinReplicas, req.MaxReplicas, endpointURL,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	row := h.store.DB.QueryRow(ctx,
		`SELECT id, name, description, model_name, billing_mode, min_replicas, max_replicas,
		        status, endpoint, created_at, updated_at
		 FROM deployments WHERE id = $1`, id)
	d, err := scanDeployment(row)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"id": id})
		return
	}
	c.JSON(http.StatusCreated, d)
}

// Patch PATCH /v1/deployments/:id
func (h *DeploymentsHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	var req model.PatchDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	row := h.store.DB.QueryRow(ctx,
		`SELECT id, name, description, model_name, billing_mode, min_replicas, max_replicas,
		        status, endpoint, created_at, updated_at
		 FROM deployments WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	d, err := scanDeployment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "deployment not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	if req.Name != nil {
		d.Name = *req.Name
	}
	if req.Description != nil {
		d.Description = *req.Description
	}
	if req.MinReplicas != nil {
		d.MinReplicas = *req.MinReplicas
	}
	if req.MaxReplicas != nil {
		d.MaxReplicas = *req.MaxReplicas
	}
	if req.Status != nil {
		switch *req.Status {
		case "running", "stopped", "provisioning", "failed", "deleting":
			d.Status = *req.Status
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid status"}})
			return
		}
	}

	if d.MaxReplicas < d.MinReplicas {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "max_replicas must be >= min_replicas"}})
		return
	}

	_, err = h.store.DB.Exec(ctx,
		`UPDATE deployments SET name=$2, description=$3, min_replicas=$4, max_replicas=$5,
		        status=$6, updated_at=$7
		 WHERE id=$1 AND user_id=$8`,
		id, d.Name, d.Description, d.MinReplicas, d.MaxReplicas, d.Status, time.Now().UTC(), auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	row2 := h.store.DB.QueryRow(ctx,
		`SELECT id, name, description, model_name, billing_mode, min_replicas, max_replicas,
		        status, endpoint, created_at, updated_at
		 FROM deployments WHERE id = $1`, id)
	d2, err := scanDeployment(row2)
	if err != nil {
		c.JSON(http.StatusOK, d)
		return
	}
	c.JSON(http.StatusOK, d2)
}

// Delete DELETE /v1/deployments/:id
func (h *DeploymentsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	cmd, err := h.store.DB.Exec(c.Request.Context(),
		`DELETE FROM deployments WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "deployment not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func scanDeployment(row interface {
	Scan(dest ...any) error
}) (model.Deployment, error) {
	var d model.Deployment
	err := row.Scan(
		&d.ID, &d.Name, &d.Description, &d.ModelName, &d.BillingMode,
		&d.MinReplicas, &d.MaxReplicas, &d.Status, &d.Endpoint,
		&d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (h *DeploymentsHandler) allocEndpoint(modelName string) string {
	slug := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	// derive short model alias
	parts := strings.Split(modelName, "/")
	alias := strings.ToLower(parts[len(parts)-1])
	alias = strings.ReplaceAll(alias, ".", "-")
	if len(alias) > 20 {
		alias = alias[:20]
	}
	return fmt.Sprintf("https://api.inference.example.com/v1/deploy/%s-%s", alias, slug)
}
