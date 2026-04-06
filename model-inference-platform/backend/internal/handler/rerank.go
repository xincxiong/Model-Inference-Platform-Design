package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type RerankHandler struct {
	store  *store.Store
	router *modelrouter.ModelRouter
}

func NewRerankHandler(s *store.Store, mr *modelrouter.ModelRouter) *RerankHandler {
	return &RerankHandler{store: s, router: mr}
}

var rerankAllowedTypes = []string{"rerank"}

func (h *RerankHandler) Create(c *gin.Context) {
	var req model.RerankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	auth := middleware.GetAuthInfo(c)
	resolved, err := h.router.Resolve(req.Model, auth.UserID)
	if err != nil {
		status, errType := modelrouter.HTTPStatusForResolveError(err)
		c.JSON(status, gin.H{"error": gin.H{"message": err.Error(), "type": errType}})
		return
	}

	if err := h.router.ValidateType(resolved, rerankAllowedTypes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	eng := h.router.GetEngine(resolved)
	resp, err := eng.Rerank(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	go h.recordUsage(auth, resolved, req)

	c.JSON(http.StatusOK, resp)
}

func (h *RerankHandler) recordUsage(auth middleware.AuthInfo, resolved *modelrouter.ResolvedModel, req model.RerankRequest) {
	ctx := context.Background()

	totalTokens := len(strings.Fields(req.Query)) + 4
	for _, doc := range req.Documents {
		totalTokens += len(strings.Fields(doc)) + 2
	}

	cost := float64(totalTokens) / 1_000_000 * resolved.InputPrice
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,$5,0,$6)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, resolved.ID, totalTokens, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
