package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
	"github.com/xincxiong/model-inference-platform/backend/internal/semcache"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type CompletionsHandler struct {
	store      *store.Store
	router     *modelrouter.ModelRouter
	cache      *semcache.Cache
	embedModel string
}

func NewCompletionsHandler(s *store.Store, mr *modelrouter.ModelRouter, cache *semcache.Cache, embedModel string) *CompletionsHandler {
	return &CompletionsHandler{store: s, router: mr, cache: cache, embedModel: embedModel}
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

	auth := middleware.GetAuthInfo(c)
	resolved, err := h.router.Resolve(req.Model, auth.UserID)
	if err != nil {
		status, errType := modelrouter.HTTPStatusForResolveError(err)
		c.JSON(status, gin.H{"error": gin.H{"message": err.Error(), "type": errType}})
		return
	}

	if err := h.router.ValidateType(resolved, completionsAllowedTypes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	ctx := c.Request.Context()
	prompt := req.Prompt
	promptTokens := len(strings.Fields(prompt)) + 2
	var promptVec []float32

	if h.cache != nil && h.cache.Enabled() && h.embedModel != "" {
		embedder := buildEmbedder(h.router, h.embedModel, auth)
		if vec, err := embedder(ctx, prompt); err == nil {
			promptVec = vec
			if hit, err := h.cache.LookupWithVec(auth.UserID, resolved.ID, vec); err == nil && hit != nil {
				content := hit.Row.Response
				created := hit.Row.CreatedAt
				if created == 0 {
					created = time.Now().Unix()
				}
				resp := model.CompletionResponse{
					ID:      hit.Row.ID,
					Object:  "text_completion",
					Created: created,
					Model:   req.Model,
					Choices: []model.CompletionChoice{{
						Index:        0,
						Text:         content,
						FinishReason: "stop",
					}},
					Usage: model.Usage{
						PromptTokens:     promptTokens,
						CompletionTokens: len(strings.Fields(content)),
						TotalTokens:      promptTokens + len(strings.Fields(content)),
					},
				}
				c.JSON(http.StatusOK, resp)
				go h.recordUsage(auth, resolved, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
				return
			}
		}
	}

	eng := h.router.GetEngine(resolved)
	resp, err := eng.Completion(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	go h.recordUsage(auth, resolved, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	c.JSON(http.StatusOK, resp)

	if h.cache != nil && h.cache.Enabled() && len(promptVec) > 0 && len(resp.Choices) > 0 {
		_ = h.cache.Insert(ctx, semcache.CacheRow{
			ID:        resp.ID,
			UserID:    auth.UserID,
			ModelID:   resolved.ID,
			Prompt:    prompt,
			Response:  resp.Choices[0].Text,
			Embedding: promptVec,
			CreatedAt: resp.Created,
		})
	}
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
