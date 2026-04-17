package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

// ModelVersionsHandler handles CRUD for model versions.
type ModelVersionsHandler struct {
	store *store.Store
}

func NewModelVersionsHandler(s *store.Store) *ModelVersionsHandler {
	return &ModelVersionsHandler{store: s}
}

// ── Response types ────────────────────────────────────────────────────────

type ModelVersion struct {
	ID           string    `json:"id"`
	ModelID      string    `json:"model_id"`
	Version      string    `json:"version"`
	Description  string    `json:"description"`
	Changelog    string    `json:"changelog"`
	BackendAddr  string    `json:"backend_addr"`
	TrafficPct   int       `json:"traffic_pct"`   // 0-100, used for A/B
	Status       string    `json:"status"`        // active | shadow | inactive
	IsDefault    bool      `json:"is_default"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	ActivatedAt  *time.Time `json:"activated_at,omitempty"`
}

type CreateVersionRequest struct {
	Version     string `json:"version" binding:"required"`
	Description string `json:"description"`
	Changelog   string `json:"changelog"`
	BackendAddr string `json:"backend_addr"`
	TrafficPct  int    `json:"traffic_pct"` // default 0
}

type PatchVersionRequest struct {
	Description *string `json:"description"`
	Changelog   *string `json:"changelog"`
	BackendAddr *string `json:"backend_addr"`
	TrafficPct  *int    `json:"traffic_pct"`
	Status      *string `json:"status"`
	IsDefault   *bool   `json:"is_default"`
}

type ABTestConfig struct {
	Versions []ABSlot `json:"versions"`
}

type ABSlot struct {
	VersionID  string `json:"version_id"`
	TrafficPct int    `json:"traffic_pct"`
}

// ── Handlers ──────────────────────────────────────────────────────────────

// List returns all versions for a model.
func (h *ModelVersionsHandler) List(c *gin.Context) {
	modelID := c.Param("model_id")
	rows, err := h.store.DB.Query(context.Background(),
		`SELECT id, model_id, version, description, changelog, backend_addr,
		        traffic_pct, status, is_default, created_by, created_at, activated_at
		 FROM model_versions WHERE model_id = $1 ORDER BY created_at DESC`, modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var versions []ModelVersion
	for rows.Next() {
		var v ModelVersion
		if err := rows.Scan(&v.ID, &v.ModelID, &v.Version, &v.Description, &v.Changelog,
			&v.BackendAddr, &v.TrafficPct, &v.Status, &v.IsDefault, &v.CreatedBy,
			&v.CreatedAt, &v.ActivatedAt); err != nil {
			continue
		}
		versions = append(versions, v)
	}
	if versions == nil {
		versions = []ModelVersion{}
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": versions})
}

// Create adds a new version record for a model.
func (h *ModelVersionsHandler) Create(c *gin.Context) {
	modelID := c.Param("model_id")
	auth := middleware.GetAuthInfo(c)

	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	if req.TrafficPct < 0 || req.TrafficPct > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "traffic_pct must be 0-100"}})
		return
	}

	var id string
	err := h.store.DB.QueryRow(context.Background(),
		`INSERT INTO model_versions
		   (model_id, version, description, changelog, backend_addr, traffic_pct, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		modelID, req.Version, req.Description, req.Changelog,
		req.BackendAddr, req.TrafficPct, auth.UserID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	var v ModelVersion
	_ = h.store.DB.QueryRow(context.Background(),
		`SELECT id, model_id, version, description, changelog, backend_addr,
		        traffic_pct, status, is_default, created_by, created_at, activated_at
		 FROM model_versions WHERE id = $1`, id).
		Scan(&v.ID, &v.ModelID, &v.Version, &v.Description, &v.Changelog,
			&v.BackendAddr, &v.TrafficPct, &v.Status, &v.IsDefault, &v.CreatedBy,
			&v.CreatedAt, &v.ActivatedAt)

	c.JSON(http.StatusCreated, v)
}

// Patch updates a version record (traffic_pct, status, backend_addr, is_default …).
func (h *ModelVersionsHandler) Patch(c *gin.Context) {
	id := c.Param("version_id")

	var req PatchVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	if req.TrafficPct != nil && (*req.TrafficPct < 0 || *req.TrafficPct > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "traffic_pct must be 0-100"}})
		return
	}

	// Build dynamic UPDATE
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	i := 1
	addField := func(col string, val interface{}) {
		setClauses = append(setClauses, col+" = $"+itoa(i))
		args = append(args, val)
		i++
	}

	if req.Description != nil { addField("description", *req.Description) }
	if req.Changelog != nil   { addField("changelog", *req.Changelog) }
	if req.BackendAddr != nil { addField("backend_addr", *req.BackendAddr) }
	if req.TrafficPct != nil  { addField("traffic_pct", *req.TrafficPct) }
	if req.Status != nil      { addField("status", *req.Status) }
	if req.IsDefault != nil   { addField("is_default", *req.IsDefault) }

	args = append(args, id)
	query := "UPDATE model_versions SET " + joinComma(setClauses) + " WHERE id = $" + itoa(i)

	if _, err := h.store.DB.Exec(context.Background(), query, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	var v ModelVersion
	_ = h.store.DB.QueryRow(context.Background(),
		`SELECT id, model_id, version, description, changelog, backend_addr,
		        traffic_pct, status, is_default, created_by, created_at, activated_at
		 FROM model_versions WHERE id = $1`, id).
		Scan(&v.ID, &v.ModelID, &v.Version, &v.Description, &v.Changelog,
			&v.BackendAddr, &v.TrafficPct, &v.Status, &v.IsDefault, &v.CreatedBy,
			&v.CreatedAt, &v.ActivatedAt)

	c.JSON(http.StatusOK, v)
}

// Activate marks a version as active and optionally sets it as default.
// It also records activated_at for audit.
func (h *ModelVersionsHandler) Activate(c *gin.Context) {
	id := c.Param("version_id")

	_, err := h.store.DB.Exec(context.Background(),
		`UPDATE model_versions SET status='active', activated_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "version activated"})
}

// Deactivate gracefully drains a version by setting traffic_pct to 0 and status to inactive.
func (h *ModelVersionsHandler) Deactivate(c *gin.Context) {
	id := c.Param("version_id")

	_, err := h.store.DB.Exec(context.Background(),
		`UPDATE model_versions SET status='inactive', traffic_pct=0, updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "version deactivated"})
}

// SetABTest replaces the traffic distribution for a model's versions.
// The sum of traffic_pct across all versions must equal 100.
func (h *ModelVersionsHandler) SetABTest(c *gin.Context) {
	modelID := c.Param("model_id")

	var cfg ABTestConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	total := 0
	for _, s := range cfg.Versions {
		total += s.TrafficPct
	}
	if total != 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "traffic_pct must sum to 100"}})
		return
	}

	tx, err := h.store.DB.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer tx.Rollback(context.Background())

	// Reset all versions of this model to 0
	if _, err := tx.Exec(context.Background(),
		`UPDATE model_versions SET traffic_pct=0 WHERE model_id=$1`, modelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	for _, s := range cfg.Versions {
		if _, err := tx.Exec(context.Background(),
			`UPDATE model_versions SET traffic_pct=$1 WHERE id=$2 AND model_id=$3`,
			s.TrafficPct, s.VersionID, modelID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "A/B test config applied"})
}

// ── small helpers (to avoid importing fmt/strings just for these) ─────────

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n >= 10 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	pos--
	buf[pos] = byte('0' + n)
	return string(buf[pos:])
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}
