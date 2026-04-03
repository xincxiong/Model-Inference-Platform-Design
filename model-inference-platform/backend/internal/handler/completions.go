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

type CompletionsHandler struct {
	store  *store.Store
	engine engine.Engine
}

func NewCompletionsHandler(s *store.Store, eng engine.Engine) *CompletionsHandler {
	return &CompletionsHandler{store: s, engine: eng}
}

func (h *CompletionsHandler) Create(c *gin.Context) {
	var req model.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	resp, err := h.engine.Completion(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	auth := middleware.GetAuthInfo(c)
	go h.recordUsage(auth, req.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	c.JSON(http.StatusOK, resp)
}

func (h *CompletionsHandler) recordUsage(auth middleware.AuthInfo, modelName string, inputTokens, outputTokens int) {
	ctx := context.Background()
	var inputPrice, outputPrice float64
	_ = h.store.DB.QueryRow(ctx,
		`SELECT input_price, output_price FROM models WHERE id = $1`, modelName).
		Scan(&inputPrice, &outputPrice)

	cost := float64(inputTokens)/1_000_000*inputPrice + float64(outputTokens)/1_000_000*outputPrice
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, modelName, inputTokens, outputTokens, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}

func estimatePromptTokens(text string) int {
	return len(strings.Fields(text)) + 4
}
