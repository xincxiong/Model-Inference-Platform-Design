package handler

import (
	"context"
	"errors"
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

type ComputePoolsHandler struct {
	store *store.Store
}

func NewComputePoolsHandler(s *store.Store) *ComputePoolsHandler {
	return &ComputePoolsHandler{store: s}
}

// Create POST /v0/pools/compute — instantiate a pool from an active subscription.
func (h *ComputePoolsHandler) Create(c *gin.Context) {
	var req model.CreateComputePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	// Load the subscription (must be active, not expired, owned by user)
	var sub struct {
		ID         string
		UserID     string
		SKUID      string
		GPUType    string
		GPUCount   int
		Region     string
		Sharing    string
		SLAClass   string
		StartAt    time.Time
		EndAt      time.Time
		Status     string
		TermMonths int
	}
	err := h.store.DB.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.sku_id, sk.gpu_type, sk.gpu_count, sk.region, sk.sharing_mode,
			sk.sla_class, s.start_at, s.end_at, s.status, sk.term_months
		FROM pool_subscriptions s
		JOIN pool_skus sk ON sk.id = s.sku_id
		WHERE s.id = $1 AND s.user_id = $2`, req.SubscriptionID, auth.UserID).
		Scan(&sub.ID, &sub.UserID, &sub.SKUID, &sub.GPUType, &sub.GPUCount, &sub.Region, &sub.Sharing,
			&sub.SLAClass, &sub.StartAt, &sub.EndAt, &sub.Status, &sub.TermMonths)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "subscription not found"}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if sub.Status != model.SubStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "subscription is " + sub.Status + ", cannot create pool"}})
		return
	}
	if sub.EndAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "subscription has expired"}})
		return
	}

	// Defaults
	schedulerPolicy := req.SchedulerPolicy
	if schedulerPolicy == "" {
		schedulerPolicy = "binpack"
	}
	if schedulerPolicy != "binpack" && schedulerPolicy != "spread" && schedulerPolicy != "topology-aware" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "scheduler_policy must be binpack | spread | topology-aware"}})
		return
	}

	// Generate pool ID
	queueName := "pool-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	poolID := "pool-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]

	var newID string
	err = h.store.DB.QueryRow(ctx, `
		INSERT INTO compute_pools
			(id, name, description, gpu_type, gpu_count, region, sharing_mode,
			 subscription_id, user_id, volcano_queue, scheduler_policy, hard_isolation,
			 status, service_start_at, service_end_at, sla_class)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'active',$13,$14,$15)
		RETURNING id`,
		poolID, req.Name, req.Description, sub.GPUType, sub.GPUCount, sub.Region, sub.Sharing,
		sub.ID, auth.UserID, queueName, schedulerPolicy, req.HardIsolation,
		sub.StartAt, sub.EndAt, sub.SLAClass,
	).Scan(&newID)
	if err != nil {
		// partial unique index will reject duplicate active pool per subscription
		if strings.Contains(err.Error(), "idx_compute_pools_sub_active") {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "an active pool already exists for this subscription"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	pool, err := h.fetchPool(ctx, newID, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, pool)
}

// List GET /v0/pools/compute — list user's pools with live usage.
func (h *ComputePoolsHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()
	rows, err := h.store.DB.Query(ctx, `
		SELECT p.id, p.name, p.description, p.gpu_type, p.gpu_count, p.region, p.sharing_mode,
			p.subscription_id, p.user_id, p.volcano_queue, p.scheduler_policy, p.hard_isolation,
			p.status, p.used_gpu, p.service_start_at, p.service_end_at, p.sla_class,
			p.created_at, p.updated_at,
			sk.id, sk.term, s.end_at, s.status
		FROM compute_pools p
		JOIN pool_subscriptions s ON s.id = p.subscription_id
		JOIN pool_skus sk ON sk.id = s.sku_id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var data []model.ComputePoolWithUsage
	for rows.Next() {
		pu, err := h.scanPoolWithUsage(ctx, rows)
		if err != nil {
			continue
		}
		data = append(data, pu)
	}
	c.JSON(http.StatusOK, model.ListComputePoolsResponse{Data: data})
}

// Get GET /v0/pools/compute/:id
func (h *ComputePoolsHandler) Get(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	pool, err := h.fetchPool(c.Request.Context(), id, auth.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "pool not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, model.GetComputePoolResponse{Data: pool})
}

// Patch PATCH /v0/pools/compute/:id
func (h *ComputePoolsHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	var req model.PatchComputePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	sets := []string{}
	args := []any{}
	i := 1
	if req.Name != nil {
		sets = append(sets, "name = $"+itoa(i))
		args = append(args, *req.Name)
		i++
	}
	if req.Description != nil {
		sets = append(sets, "description = $"+itoa(i))
		args = append(args, *req.Description)
		i++
	}
	if req.SchedulerPolicy != nil {
		p := *req.SchedulerPolicy
		if p != "binpack" && p != "spread" && p != "topology-aware" {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "invalid scheduler_policy"}})
			return
		}
		sets = append(sets, "scheduler_policy = $"+itoa(i))
		args = append(args, p)
		i++
	}
	if req.HardIsolation != nil {
		sets = append(sets, "hard_isolation = $"+itoa(i))
		args = append(args, *req.HardIsolation)
		i++
	}
	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "no fields to update"}})
		return
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id, auth.UserID)

	sql := "UPDATE compute_pools SET " + strings.Join(sets, ", ") +
		" WHERE id = $" + itoa(i) + " AND user_id = $" + itoa(i+1)
	tag, err := h.store.DB.Exec(ctx, sql, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "pool not found"}})
		return
	}

	pool, err := h.fetchPool(ctx, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, model.GetComputePoolResponse{Data: pool})
}

// Archive POST /v0/pools/compute/:id/archive — soft delete (status=archived).
func (h *ComputePoolsHandler) Archive(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	// Refuse if there are running jobs on this pool
	var activeJobs int
	_ = h.store.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM fine_tuning_jobs WHERE pool_id = $1 AND status IN ('queued','running','validating')`, id).
		Scan(&activeJobs)
	if activeJobs > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"message": "cannot archive pool with " + itoa(activeJobs) + " running job(s); cancel them first"}})
		return
	}

	tag, err := h.store.DB.Exec(ctx,
		`UPDATE compute_pools SET status = 'archived', updated_at = NOW() WHERE id = $1 AND user_id = $2 AND status <> 'archived'`,
		id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "pool not found or already archived"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "archived"})
}

// Usage GET /v0/pools/compute/:id/usage — current allocation snapshot.
func (h *ComputePoolsHandler) Usage(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()

	var pool struct {
		ID, Name, GPUType, SharingMode, Status string
		GPUCount int
	}
	err := h.store.DB.QueryRow(ctx, `
		SELECT id, name, gpu_type, gpu_count, sharing_mode, status
		FROM compute_pools WHERE id = $1 AND user_id = $2`, id, auth.UserID).
		Scan(&pool.ID, &pool.Name, &pool.GPUType, &pool.GPUCount, &pool.SharingMode, &pool.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "pool not found"}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	// Live aggregate: only count in-flight jobs toward used_gpu.
	var usedGPU, activeJobs, queueDepth int
	_ = h.store.DB.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(gpu_request) FILTER (WHERE status IN ('queued','running','validating')), 0),
			COUNT(*) FILTER (WHERE status = 'running'),
			COUNT(*) FILTER (WHERE status IN ('queued','validating'))
		FROM fine_tuning_jobs WHERE pool_id = $1`, id).Scan(&usedGPU, &activeJobs, &queueDepth)

	freeGPU := pool.GPUCount - usedGPU
	if freeGPU < 0 {
		freeGPU = 0
	}
	utilPct := 0.0
	if pool.GPUCount > 0 {
		utilPct = float64(usedGPU) / float64(pool.GPUCount) * 100
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             pool.ID,
		"name":           pool.Name,
		"gpu_type":       pool.GPUType,
		"gpu_count":      pool.GPUCount,
		"used_gpu":       usedGPU,
		"free_gpu":       freeGPU,
		"utilization_pct": utilPct,
		"active_jobs":    activeJobs,
		"queue_depth":    queueDepth,
		"sharing_mode":   pool.SharingMode,
		"status":         pool.Status,
	})
}

// ─── helpers ────────────────────────────────────────────────────────────────

// fetchPool loads a pool with its joined subscription info and live usage.
func (h *ComputePoolsHandler) fetchPool(ctx context.Context, id, userID string) (model.ComputePoolWithUsage, error) {
	var pu model.ComputePoolWithUsage
	var endAt *time.Time
	var skuID, skuTerm, subStatus string
	var subEndAtT time.Time
	err := h.store.DB.QueryRow(ctx, `
		SELECT p.id, p.name, p.description, p.gpu_type, p.gpu_count, p.region, p.sharing_mode,
			p.subscription_id, p.user_id, p.volcano_queue, p.scheduler_policy, p.hard_isolation,
			p.status, p.used_gpu, p.service_start_at, p.service_end_at, p.sla_class,
			p.created_at, p.updated_at,
			sk.id, sk.term, s.end_at, s.status
		FROM compute_pools p
		JOIN pool_subscriptions s ON s.id = p.subscription_id
		JOIN pool_skus sk ON sk.id = s.sku_id
		WHERE p.id = $1 AND p.user_id = $2`, id, userID).
		Scan(&pu.ID, &pu.Name, &pu.Description, &pu.GPUType, &pu.GPUCount, &pu.Region, &pu.SharingMode,
			&pu.SubscriptionID, &pu.UserID, &pu.VolcanoQueue, &pu.SchedulerPolicy, &pu.HardIsolation,
			&pu.Status, &pu.UsedGPU, &pu.ServiceStartAt, &endAt, &pu.SLAClass,
			&pu.CreatedAt, &pu.UpdatedAt,
			&skuID, &skuTerm, &subEndAtT, &subStatus)
	if err != nil {
		return pu, err
	}
	pu.ServiceEndAt = endAt
	pu.SKUID = skuID
	pu.SKUTerm = skuTerm
	pu.SubEndAt = subEndAtT.Format(time.RFC3339)
	pu.SubStatus = subStatus

	// Live usage (running + queued jobs)
	_ = h.store.DB.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(gpu_request) FILTER (WHERE status IN ('queued','running','validating')), 0),
			COUNT(*) FILTER (WHERE status = 'running'),
			COUNT(*) FILTER (WHERE status IN ('queued','validating'))
		FROM fine_tuning_jobs WHERE pool_id = $1`, id).
		Scan(&pu.UsedGPU, &pu.ActiveJobs, &pu.QueueDepth)
	pu.FreeGPU = pu.GPUCount - pu.UsedGPU
	if pu.FreeGPU < 0 {
		pu.FreeGPU = 0
	}
	if pu.GPUCount > 0 {
		pu.UtilizationPct = float64(pu.UsedGPU) / float64(pu.GPUCount) * 100
	}
	return pu, nil
}

// scanPoolWithUsage is the row-scanner variant for List.
func (h *ComputePoolsHandler) scanPoolWithUsage(ctx context.Context, rows pgx.Row) (model.ComputePoolWithUsage, error) {
	var pu model.ComputePoolWithUsage
	var endAt *time.Time
	var skuID, skuTerm, subStatus string
	var subEndAtT time.Time
	err := rows.Scan(
		&pu.ID, &pu.Name, &pu.Description, &pu.GPUType, &pu.GPUCount, &pu.Region, &pu.SharingMode,
		&pu.SubscriptionID, &pu.UserID, &pu.VolcanoQueue, &pu.SchedulerPolicy, &pu.HardIsolation,
		&pu.Status, &pu.UsedGPU, &pu.ServiceStartAt, &endAt, &pu.SLAClass,
		&pu.CreatedAt, &pu.UpdatedAt,
		&skuID, &skuTerm, &subEndAtT, &subStatus,
	)
	if err != nil {
		return pu, err
	}
	pu.ServiceEndAt = endAt
	pu.SKUID = skuID
	pu.SKUTerm = skuTerm
	pu.SubEndAt = subEndAtT.Format(time.RFC3339)
	pu.SubStatus = subStatus

	_ = h.store.DB.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(gpu_request) FILTER (WHERE status IN ('queued','running','validating')), 0),
			COUNT(*) FILTER (WHERE status = 'running'),
			COUNT(*) FILTER (WHERE status IN ('queued','validating'))
		FROM fine_tuning_jobs WHERE pool_id = $1`, pu.ID).
		Scan(&pu.UsedGPU, &pu.ActiveJobs, &pu.QueueDepth)
	pu.FreeGPU = pu.GPUCount - pu.UsedGPU
	if pu.FreeGPU < 0 {
		pu.FreeGPU = 0
	}
	if pu.GPUCount > 0 {
		pu.UtilizationPct = float64(pu.UsedGPU) / float64(pu.GPUCount) * 100
	}
	return pu, nil
}
