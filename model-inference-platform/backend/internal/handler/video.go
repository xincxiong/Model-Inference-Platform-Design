package handler

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/xincxiong/model-inference-platform/backend/internal/middleware"
    "github.com/xincxiong/model-inference-platform/backend/internal/model"
    "github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
    "github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type VideoHandler struct {
    store  *store.Store
    router *modelrouter.ModelRouter
}

func NewVideoHandler(s *store.Store, mr *modelrouter.ModelRouter) *VideoHandler {
    return &VideoHandler{store: s, router: mr}
}

var videoAllowedTypes = []string{"text-to-video"}

// Generate handles POST /v1/videos/generations
func (h *VideoHandler) Generate(c *gin.Context) {
    var req model.VideoGenerationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
        return
    }

    auth := middleware.GetAuthInfo(c)
    resolved, err := h.router.Resolve(req.Model, auth.UserID)
    if err != nil {
        status, errType := modelrouter.HTTPStatusForResolveError(err)
        c.JSON(status, gin.H{"error": gin.H{"message": err.Error(), "type": errType}})
        return
    }

    if err := h.router.ValidateType(resolved, videoAllowedTypes); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
        return
    }

    eng := h.router.GetEngine(resolved)
    resp, err := eng.VideoGeneration(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "server_error"}})
        return
    }

    // Estimate tokens: ~4 chars per token for prompt, ~1000 per video frame
    inputTokens := len(req.Prompt) / 4
    outputTokens := 1000 * len(resp.Data)
    go h.recordUsage(auth, resolved, inputTokens, outputTokens)

    c.JSON(http.StatusOK, resp)
}

func (h *VideoHandler) recordUsage(auth middleware.AuthInfo, resolved *modelrouter.ResolvedModel, inputTokens, outputTokens int) {
    ctx := context.Background()
    cost := float64(inputTokens)/1_000_000*resolved.InputPrice + float64(outputTokens)/1_000_000*resolved.OutputPrice

    _, _ = h.store.DB.Exec(ctx,
        `INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
         VALUES ($1,$2,$3,$4,$5,$6,$7)`,
        uuid.New().String(), auth.UserID, auth.APIKeyID, resolved.ID, inputTokens, outputTokens, cost)
    _, _ = h.store.DB.Exec(ctx,
        `UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
