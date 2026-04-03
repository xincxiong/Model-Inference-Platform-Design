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

type ImagesHandler struct {
	store  *store.Store
	router *modelrouter.ModelRouter
}

func NewImagesHandler(s *store.Store, mr *modelrouter.ModelRouter) *ImagesHandler {
	return &ImagesHandler{store: s, router: mr}
}

var imageAllowedTypes = []string{"text-to-image"}

func (h *ImagesHandler) Generate(c *gin.Context) {
	var req model.ImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	if req.Model == "" {
		req.Model = "black-forest-labs/FLUX.1-dev"
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

	if err := h.router.ValidateType(resolved, imageAllowedTypes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	eng := h.router.GetEngine(resolved)
	resp, err := eng.ImageGeneration(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	auth := middleware.GetAuthInfo(c)
	n := 1
	if req.N != nil {
		n = *req.N
	}
	go h.recordUsage(auth, resolved, n)

	c.JSON(http.StatusOK, resp)
}

func (h *ImagesHandler) recordUsage(auth middleware.AuthInfo, resolved *modelrouter.ResolvedModel, n int) {
	ctx := context.Background()
	cost := resolved.OutputPrice * float64(n)
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,0,$5,$6)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, resolved.ID, n, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
