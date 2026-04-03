package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type CompletionsHandler struct {
	store  *store.Store
	router *modelrouter.ModelRouter
}

func NewCompletionsHandler(s *store.Store, mr *modelrouter.ModelRouter) *CompletionsHandler {
	return &CompletionsHandler{store: s, router: mr}
}

var completionsAllowedTypes = []string{"text-to-text"}

func (h *CompletionsHandler) Create(c *gin.Context) {
	var req model.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	resolved, err := h.router.Resolve(req.Model)
	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, modelrouter.ErrModelNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": gin.H{"message": err.Error(), "type": "not_found_error"}})
		return
	}

	if err := h.router.ValidateType(resolved, completionsAllowedTypes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	eng := h.router.GetEngine(resolved)
	resp, err := eng.Completion(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	auth := middleware.GetAuthInfo(c)
	go h.recordUsage(auth, resolved, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	c.JSON(http.StatusOK, resp)
}

func (h *CompletionsHandler) recordUsage(auth middleware.AuthInfo, resolved *modelrouter.ResolvedModel, inputTokens, outputTokens int) {
	ctx := context.Background()
	cost := float64(inputTokens)/1_000_000*resolved.InputPrice + float64(outputTokens)/1_000_000*resolved.OutputPrice
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, resolved.ID, inputTokens, outputTokens, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
