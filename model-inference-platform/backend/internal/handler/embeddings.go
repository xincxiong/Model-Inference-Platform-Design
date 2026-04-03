package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/engine"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type EmbeddingsHandler struct {
	store  *store.Store
	engine engine.Engine
}

func NewEmbeddingsHandler(s *store.Store, eng engine.Engine) *EmbeddingsHandler {
	return &EmbeddingsHandler{store: s, engine: eng}
}

func (h *EmbeddingsHandler) Create(c *gin.Context) {
	var req model.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	resp, err := h.engine.Embedding(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	auth := middleware.GetAuthInfo(c)
	go h.recordUsage(auth, req)

	c.JSON(http.StatusOK, resp)
}

func (h *EmbeddingsHandler) recordUsage(auth middleware.AuthInfo, req model.EmbeddingRequest) {
	ctx := context.Background()
	var inputPrice float64
	_ = h.store.DB.QueryRow(ctx,
		`SELECT input_price FROM models WHERE id = $1`, req.Model).Scan(&inputPrice)

	totalTokens := 0
	switch v := req.Input.(type) {
	case string:
		totalTokens = len(strings.Fields(v)) + 2
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				totalTokens += len(strings.Fields(s)) + 2
			}
		}
	}

	cost := float64(totalTokens) / 1_000_000 * inputPrice
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,$5,0,$6)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, req.Model, totalTokens, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
