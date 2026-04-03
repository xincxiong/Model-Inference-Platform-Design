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

type RerankHandler struct {
	store  *store.Store
	engine engine.Engine
}

func NewRerankHandler(s *store.Store, eng engine.Engine) *RerankHandler {
	return &RerankHandler{store: s, engine: eng}
}

func (h *RerankHandler) Create(c *gin.Context) {
	var req model.RerankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	resp, err := h.engine.Rerank(c.Request.Context(), req)
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

func (h *RerankHandler) recordUsage(auth middleware.AuthInfo, req model.RerankRequest) {
	ctx := context.Background()
	var inputPrice float64
	_ = h.store.DB.QueryRow(ctx,
		`SELECT input_price FROM models WHERE id = $1`, req.Model).Scan(&inputPrice)

	totalTokens := len(strings.Fields(req.Query)) + 4
	for _, doc := range req.Documents {
		totalTokens += len(strings.Fields(doc)) + 2
	}

	cost := float64(totalTokens) / 1_000_000 * inputPrice
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,$5,0,$6)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, req.Model, totalTokens, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
