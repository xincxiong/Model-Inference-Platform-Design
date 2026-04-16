package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type DatasetsHandler struct {
	store *store.Store
}

func NewDatasetsHandler(s *store.Store) *DatasetsHandler {
	return &DatasetsHandler{store: s}
}

// Create handles POST /v1/datasets
func (h *DatasetsHandler) Create(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	var req struct {
		Name        string            `json:"name" binding:"required"`
		Description string            `json:"description"`
		FileID      string            `json:"file_id"` // optional reference to uploaded file
		Metadata    map[string]string `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	id := "ds-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]
	metaBytes, _ := json.Marshal(req.Metadata)

	// Fetch content from referenced file if provided
	var rows int64
	var sizeBytes int64
	if req.FileID != "" {
		var content []byte
		_ = h.store.DB.QueryRow(c.Request.Context(),
			`SELECT content FROM files WHERE id = $1 AND user_id = $2`, req.FileID, auth.UserID).Scan(&content)
		if len(content) > 0 {
			for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
				if strings.TrimSpace(line) != "" {
					rows++
				}
			}
			sizeBytes = int64(len(content))
		}
	}

	_, err := h.store.DB.Exec(c.Request.Context(),
		`INSERT INTO datasets (id, user_id, name, description, file_id, num_rows, size_bytes, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb)`,
		id, auth.UserID, req.Name, req.Description, req.FileID, rows, sizeBytes, metaBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "api_error"}})
		return
	}

	c.JSON(http.StatusCreated, h.buildDatasetResponse(id, req.Name, req.Description, req.FileID, rows, sizeBytes, req.Metadata, time.Now()))
}

// List handles GET /v1/datasets
func (h *DatasetsHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(c.Request.Context(),
		`SELECT id, name, description, file_id, num_rows, size_bytes, metadata, created_at, updated_at
		 FROM datasets WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var datasets []map[string]interface{}
	for rows.Next() {
		ds, err := scanDatasetRow(rows)
		if err != nil {
			continue
		}
		datasets = append(datasets, ds)
	}
	if datasets == nil {
		datasets = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": datasets})
}

// Get handles GET /v1/datasets/:id
func (h *DatasetsHandler) Get(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	row := h.store.DB.QueryRow(c.Request.Context(),
		`SELECT id, name, description, file_id, num_rows, size_bytes, metadata, created_at, updated_at
		 FROM datasets WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	ds, err := scanDatasetRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "dataset not found", "type": "invalid_request_error"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, ds)
}

// Patch handles PATCH /v1/datasets/:id
func (h *DatasetsHandler) Patch(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	var req struct {
		Name        *string            `json:"name"`
		Description *string            `json:"description"`
		Metadata    map[string]string  `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	sets := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Metadata != nil {
		metaBytes, _ := json.Marshal(req.Metadata)
		sets = append(sets, fmt.Sprintf("metadata = $%d::jsonb", argIdx))
		args = append(args, metaBytes)
		argIdx++
	}

	args = append(args, id, auth.UserID)
	query := fmt.Sprintf(
		"UPDATE datasets SET %s WHERE id = $%d AND user_id = $%d",
		strings.Join(sets, ", "), argIdx, argIdx+1,
	)

	res, err := h.store.DB.Exec(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "dataset not found"}})
		return
	}
	h.Get(c)
}

// Delete handles DELETE /v1/datasets/:id
func (h *DatasetsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	res, err := h.store.DB.Exec(c.Request.Context(),
		`DELETE FROM datasets WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "dataset not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "dataset", "deleted": true})
}

// GetContent handles GET /v1/datasets/:id/content — paginated row access
func (h *DatasetsHandler) GetContent(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Fetch raw content from associated file
	var content []byte
	err := h.store.DB.QueryRow(c.Request.Context(),
		`SELECT f.content FROM datasets d
		 LEFT JOIN files f ON f.id = d.file_id
		 WHERE d.id = $1 AND d.user_id = $2`, id, auth.UserID).Scan(&content)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "dataset not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	allLines := []string{}
	if len(content) > 0 {
		for _, l := range strings.Split(strings.TrimSpace(string(content)), "\n") {
			if strings.TrimSpace(l) != "" {
				allLines = append(allLines, l)
			}
		}
	}

	total := len(allLines)
	start := (page - 1) * limit
	end := start + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pageLines := allLines[start:end]

	// Parse each line as JSON, fall back to string
	rows := make([]interface{}, 0, len(pageLines))
	for _, l := range pageLines {
		var obj interface{}
		if err := json.Unmarshal([]byte(l), &obj); err != nil {
			obj = l
		}
		rows = append(rows, obj)
	}

	c.JSON(http.StatusOK, gin.H{
		"object":   "list",
		"data":     rows,
		"total":    total,
		"page":     page,
		"limit":    limit,
		"has_more": end < total,
	})
}

// Export handles GET /v1/datasets/:id/export
func (h *DatasetsHandler) Export(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	format := strings.ToLower(c.DefaultQuery("format", "jsonl"))

	var content []byte
	var name string
	err := h.store.DB.QueryRow(c.Request.Context(),
		`SELECT COALESCE(f.content, ''), d.name FROM datasets d
		 LEFT JOIN files f ON f.id = d.file_id
		 WHERE d.id = $1 AND d.user_id = $2`, id, auth.UserID).Scan(&content, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "dataset not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	safeName := strings.ReplaceAll(name, " ", "_")
	switch format {
	case "csv":
		// Convert JSONL to CSV (flat key-value)
		csvContent := jsonlToCSV(content)
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, safeName))
		c.Data(http.StatusOK, "text/csv", csvContent)
	default: // jsonl
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.jsonl"`, safeName))
		c.Data(http.StatusOK, "application/x-ndjson", content)
	}
}

// Query handles GET /v1/datasets/:id/query — simple filter on JSONL content
func (h *DatasetsHandler) Query(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	filter := c.Query("filter") // e.g. field=value
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 500 {
		limit = 50
	}

	var content []byte
	err := h.store.DB.QueryRow(c.Request.Context(),
		`SELECT COALESCE(f.content, '') FROM datasets d
		 LEFT JOIN files f ON f.id = d.file_id
		 WHERE d.id = $1 AND d.user_id = $2`, id, auth.UserID).Scan(&content)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "dataset not found"}})
		return
	}

	// Simple filter: field=value
	filterKey, filterVal, hasFilter := strings.Cut(filter, "=")

	var matched []interface{}
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if hasFilter {
			val, ok := obj[filterKey]
			if !ok {
				continue
			}
			if fmt.Sprintf("%v", val) != filterVal {
				continue
			}
		}
		matched = append(matched, obj)
		if len(matched) >= limit {
			break
		}
	}
	if matched == nil {
		matched = []interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": matched, "count": len(matched)})
}

// ----- helpers -----

func scanDatasetRow(s interface{ Scan(...any) error }) (map[string]interface{}, error) {
	var id, name, description string
	var fileID *string
	var numRows, sizeBytes int64
	var metaJSON []byte
	var createdAt, updatedAt time.Time
	err := s.Scan(&id, &name, &description, &fileID, &numRows, &sizeBytes, &metaJSON, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	var metadata map[string]interface{}
	_ = json.Unmarshal(metaJSON, &metadata)
	out := map[string]interface{}{
		"id":          id,
		"object":      "dataset",
		"name":        name,
		"description": description,
		"num_rows":    numRows,
		"size_bytes":  sizeBytes,
		"metadata":    metadata,
		"created_at":  createdAt.Unix(),
		"updated_at":  updatedAt.Unix(),
	}
	if fileID != nil {
		out["file_id"] = *fileID
	}
	return out, nil
}

func (h *DatasetsHandler) buildDatasetResponse(id, name, description, fileID string, numRows, sizeBytes int64, metadata map[string]string, t time.Time) map[string]interface{} {
	return map[string]interface{}{
		"id":          id,
		"object":      "dataset",
		"name":        name,
		"description": description,
		"file_id":     fileID,
		"num_rows":    numRows,
		"size_bytes":  sizeBytes,
		"metadata":    metadata,
		"created_at":  t.Unix(),
		"updated_at":  t.Unix(),
	}
}

func jsonlToCSV(content []byte) []byte {
	if len(content) == 0 {
		return []byte{}
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var headers []string
	var allRows []map[string]interface{}

	headerSet := map[string]bool{}
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(l), &obj); err != nil {
			continue
		}
		for k := range obj {
			if !headerSet[k] {
				headerSet[k] = true
				headers = append(headers, k)
			}
		}
		allRows = append(allRows, obj)
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(headers, ",") + "\n")
	for _, row := range allRows {
		vals := make([]string, len(headers))
		for i, h := range headers {
			v := row[h]
			if v == nil {
				vals[i] = ""
			} else {
				vals[i] = fmt.Sprintf("%v", v)
			}
		}
		sb.WriteString(strings.Join(vals, ",") + "\n")
	}
	return []byte(sb.String())
}
