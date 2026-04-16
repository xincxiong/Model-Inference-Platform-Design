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
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type BatchHandler struct {
	store *store.Store
}

func NewBatchHandler(s *store.Store) *BatchHandler {
	return &BatchHandler{store: s}
}

type batchCreateRequest struct {
	InputFileID        string `json:"input_file_id" binding:"required"`
	Endpoint           string `json:"endpoint" binding:"required"`
	CompletionWindow   string `json:"completion_window"`
	Metadata           map[string]string `json:"metadata"`
}

type batchRow struct {
	ID               string
	UserID           string
	InputFileID      string
	Endpoint         string
	CompletionWindow string
	Status           string
	OutputFileID     *string
	ErrorFileID      *string
	RequestCounts    map[string]int
	Metadata         map[string]string
	CancelledAt      *time.Time
	CancellingAt     *time.Time
	CompletedAt      *time.Time
	ExpiredAt        *time.Time
	FailedAt         *time.Time
	CreatedAt        time.Time
	InProgressAt     *time.Time
	ExpiresAt        *time.Time
}

func (h *BatchHandler) Create(c *gin.Context) {
	var req batchCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
		return
	}

	// Validate endpoint — must be /v1/chat/completions or /v1/embeddings
	validEndpoints := map[string]bool{
		"/v1/chat/completions": true,
		"/v1/completions":      true,
		"/v1/embeddings":       true,
	}
	if !validEndpoints[req.Endpoint] {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": fmt.Sprintf("endpoint must be one of: %v", getKeys(validEndpoints)),
			"type":    "invalid_request_error",
		}})
		return
	}

	if req.CompletionWindow == "" {
		req.CompletionWindow = "24h"
	}

	auth := middleware.GetAuthInfo(c)
	batchID := "batch_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]

	metaBytes, _ := json.Marshal(req.Metadata)
	countsBytes, _ := json.Marshal(map[string]int{"total": 0, "completed": 0, "failed": 0})

	ctx := c.Request.Context()
	_, err := h.store.DB.Exec(ctx,
		`INSERT INTO batches (id, user_id, input_file_id, endpoint, completion_window, status, request_counts, metadata)
		 VALUES ($1,$2,$3,$4,$5,'validating',$6::jsonb,$7::jsonb)`,
		batchID, auth.UserID, req.InputFileID, req.Endpoint, req.CompletionWindow, countsBytes, metaBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "api_error"}})
		return
	}

	go h.processBatch(batchID, auth.UserID, req.InputFileID, req.Endpoint)

	b, err := h.fetchBatch(ctx, batchID, auth.UserID)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"id": batchID, "object": "batch", "status": "validating"})
		return
	}
	c.JSON(http.StatusCreated, batchToAPI(b))
}

func (h *BatchHandler) Get(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	b, err := h.fetchBatch(c.Request.Context(), id, auth.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "batch not found", "type": "invalid_request_error"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, batchToAPI(b))
}

func (h *BatchHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	limit := 20
	rows, err := h.store.DB.Query(c.Request.Context(),
		`SELECT id, user_id, input_file_id, endpoint, completion_window, status,
		        output_file_id, error_file_id, request_counts, metadata,
		        cancelled_at, cancelling_at, completed_at, expired_at, failed_at,
		        created_at, in_progress_at, expires_at
		 FROM batches WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		auth.UserID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var batches []map[string]interface{}
	for rows.Next() {
		b, err := scanBatchRow(rows)
		if err != nil {
			continue
		}
		batches = append(batches, batchToAPI(b))
	}
	if batches == nil {
		batches = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": batches, "has_more": false})
}

func (h *BatchHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)
	ctx := c.Request.Context()
	now := time.Now().UTC()
	res, err := h.store.DB.Exec(ctx,
		`UPDATE batches SET status = 'cancelling', cancelling_at = $3, updated_at = $3
		 WHERE id = $1 AND user_id = $2 AND status IN ('validating','in_progress')`,
		id, auth.UserID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "batch cannot be cancelled in current status"}})
		return
	}

	// Immediately finalize cancellation
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancelTime := time.Now().UTC()
		_, _ = h.store.DB.Exec(context.Background(),
			`UPDATE batches SET status = 'cancelled', cancelled_at = $2, updated_at = $2
			 WHERE id = $1 AND status = 'cancelling'`, id, cancelTime)
	}()

	b, _ := h.fetchBatch(ctx, id, auth.UserID)
	c.JSON(http.StatusOK, batchToAPI(b))
}

// processBatch simulates batch validation → in_progress → completed pipeline (MVP).
func (h *BatchHandler) processBatch(batchID, userID, inputFileID, endpoint string) {
	ctx := context.Background()
	db := h.store.DB

	// Count lines in input file to estimate request count
	var contentBytes []byte
	_ = db.QueryRow(ctx, `SELECT content FROM files WHERE id = $1 AND user_id = $2`,
		inputFileID, userID).Scan(&contentBytes)

	totalRequests := 0
	if len(contentBytes) > 0 {
		lines := strings.Split(strings.TrimSpace(string(contentBytes)), "\n")
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				totalRequests++
			}
		}
	}
	if totalRequests == 0 {
		totalRequests = 10 // default for demo
	}

	// validating → in_progress
	time.Sleep(800 * time.Millisecond)
	now := time.Now().UTC()
	tag, _ := db.Exec(ctx,
		`UPDATE batches SET status = 'in_progress', in_progress_at = $2, updated_at = $2,
		        request_counts = jsonb_build_object('total', $3::int, 'completed', 0, 'failed', 0)
		 WHERE id = $1 AND status = 'validating'`,
		batchID, now, totalRequests)
	if tag.RowsAffected() == 0 {
		return
	}

	// Simulate processing: update completed count over time
	completed := 0
	batchSize := max(1, totalRequests/5)
	for completed < totalRequests {
		time.Sleep(600 * time.Millisecond)
		// Check if cancelled
		var status string
		_ = db.QueryRow(ctx, `SELECT status FROM batches WHERE id = $1`, batchID).Scan(&status)
		if status == "cancelling" || status == "cancelled" {
			return
		}
		completed += batchSize
		if completed > totalRequests {
			completed = totalRequests
		}
		_, _ = db.Exec(ctx,
			`UPDATE batches SET request_counts = jsonb_build_object('total', $2::int, 'completed', $3::int, 'failed', 0),
			        updated_at = $4
			 WHERE id = $1`,
			batchID, totalRequests, completed, time.Now().UTC())
	}

	// Create output file with mock results
	outputContent := buildMockOutputContent(totalRequests, endpoint)
	outputFileID := "file-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]
	_, _ = db.Exec(ctx,
		`INSERT INTO files (id, user_id, filename, purpose, bytes, checksum, content)
		 SELECT $1, user_id, 'batch_output_' || $1 || '.jsonl', 'batch_output', $2, $3, $4
		 FROM batches WHERE id = $5`,
		outputFileID, len(outputContent), "mock", outputContent, batchID)

	completedAt := time.Now().UTC()
	_, _ = db.Exec(ctx,
		`UPDATE batches SET status = 'completed', completed_at = $2, output_file_id = $3,
		        updated_at = $2,
		        request_counts = jsonb_build_object('total', $4::int, 'completed', $4::int, 'failed', 0)
		 WHERE id = $1 AND status = 'in_progress'`,
		batchID, completedAt, outputFileID, totalRequests)
}

func buildMockOutputContent(n int, endpoint string) []byte {
	lines := make([]string, 0, n)
	for i := range n {
		result := map[string]interface{}{
			"id":          fmt.Sprintf("batch-req-%04d", i),
			"custom_id":   fmt.Sprintf("request-%d", i),
			"status_code": 200,
			"response": map[string]interface{}{
				"object": "chat.completion",
				"model":  "mock",
				"choices": []map[string]interface{}{
					{"message": map[string]interface{}{"role": "assistant", "content": "Mock response"}, "index": 0, "finish_reason": "stop"},
				},
			},
		}
		if endpoint == "/v1/embeddings" {
			result["response"] = map[string]interface{}{
				"object": "list",
				"data": []map[string]interface{}{
					{"object": "embedding", "embedding": []float64{0.1, 0.2, 0.3}, "index": 0},
				},
			}
		}
		b, _ := json.Marshal(result)
		lines = append(lines, string(b))
	}
	return []byte(strings.Join(lines, "\n"))
}

func (h *BatchHandler) fetchBatch(ctx context.Context, id, userID string) (batchRow, error) {
	row := h.store.DB.QueryRow(ctx,
		`SELECT id, user_id, input_file_id, endpoint, completion_window, status,
		        output_file_id, error_file_id, request_counts, metadata,
		        cancelled_at, cancelling_at, completed_at, expired_at, failed_at,
		        created_at, in_progress_at, expires_at
		 FROM batches WHERE id = $1 AND user_id = $2`, id, userID)
	return scanBatchRow(row)
}

func scanBatchRow(s interface{ Scan(...any) error }) (batchRow, error) {
	var b batchRow
	var rcJSON, metaJSON []byte
	err := s.Scan(
		&b.ID, &b.UserID, &b.InputFileID, &b.Endpoint, &b.CompletionWindow, &b.Status,
		&b.OutputFileID, &b.ErrorFileID, &rcJSON, &metaJSON,
		&b.CancelledAt, &b.CancellingAt, &b.CompletedAt, &b.ExpiredAt, &b.FailedAt,
		&b.CreatedAt, &b.InProgressAt, &b.ExpiresAt,
	)
	if err != nil {
		return b, err
	}
	_ = json.Unmarshal(rcJSON, &b.RequestCounts)
	_ = json.Unmarshal(metaJSON, &b.Metadata)
	if b.RequestCounts == nil {
		b.RequestCounts = map[string]int{"total": 0, "completed": 0, "failed": 0}
	}
	return b, nil
}

func batchToAPI(b batchRow) map[string]interface{} {
	out := map[string]interface{}{
		"id":                b.ID,
		"object":            "batch",
		"endpoint":          b.Endpoint,
		"status":            b.Status,
		"input_file_id":     b.InputFileID,
		"completion_window": b.CompletionWindow,
		"created_at":        b.CreatedAt.Unix(),
		"request_counts":    b.RequestCounts,
		"metadata":          b.Metadata,
	}
	if b.OutputFileID != nil {
		out["output_file_id"] = *b.OutputFileID
	}
	if b.ErrorFileID != nil {
		out["error_file_id"] = *b.ErrorFileID
	}
	if b.CancelledAt != nil {
		out["cancelled_at"] = b.CancelledAt.Unix()
	}
	if b.CancellingAt != nil {
		out["cancelling_at"] = b.CancellingAt.Unix()
	}
	if b.CompletedAt != nil {
		out["completed_at"] = b.CompletedAt.Unix()
	}
	if b.FailedAt != nil {
		out["failed_at"] = b.FailedAt.Unix()
	}
	if b.InProgressAt != nil {
		out["in_progress_at"] = b.InProgressAt.Unix()
	}
	if b.ExpiresAt != nil {
		out["expires_at"] = b.ExpiresAt.Unix()
	}
	return out
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
