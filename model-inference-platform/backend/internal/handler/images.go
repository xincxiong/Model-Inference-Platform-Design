package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/engine"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type ImagesHandler struct {
	store  *store.Store
	engine engine.Engine
}

func NewImagesHandler(s *store.Store, eng engine.Engine) *ImagesHandler {
	return &ImagesHandler{store: s, engine: eng}
}

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

	resp, err := h.engine.ImageGeneration(c.Request.Context(), req)
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
	go h.recordUsage(auth, req.Model, n)

	c.JSON(http.StatusOK, resp)
}

func (h *ImagesHandler) recordUsage(auth middleware.AuthInfo, modelName string, n int) {
	ctx := context.Background()
	var outputPrice float64
	_ = h.store.DB.QueryRow(ctx,
		`SELECT output_price FROM models WHERE id = $1`, modelName).Scan(&outputPrice)

	cost := outputPrice * float64(n)
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,0,$5,$6)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, modelName, n, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
