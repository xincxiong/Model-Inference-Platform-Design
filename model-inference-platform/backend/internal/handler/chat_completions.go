package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/engine"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type ChatCompletionsHandler struct {
	store  *store.Store
	engine engine.Engine
}

func NewChatCompletionsHandler(s *store.Store, eng engine.Engine) *ChatCompletionsHandler {
	return &ChatCompletionsHandler{store: s, engine: eng}
}

func (h *ChatCompletionsHandler) Create(c *gin.Context) {
	var req model.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	auth := middleware.GetAuthInfo(c)

	if req.Stream {
		h.handleStream(c, req, auth)
	} else {
		h.handleNonStream(c, req, auth)
	}
}

func (h *ChatCompletionsHandler) handleStream(c *gin.Context, req model.ChatCompletionRequest, auth middleware.AuthInfo) {
	ch, err := h.engine.ChatCompletion(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, _ := c.Writer.(http.Flusher)
	var fullContent strings.Builder
	inputTokens := estimateTokens(req.Messages)

	for chunk := range ch {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
		if len(chunk.Choices) > 0 {
			fullContent.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}

	outputTokens := len(strings.Fields(fullContent.String()))
	go h.recordUsage(auth, req.Model, inputTokens, outputTokens)
}

func (h *ChatCompletionsHandler) handleNonStream(c *gin.Context, req model.ChatCompletionRequest, auth middleware.AuthInfo) {
	ch, err := h.engine.ChatCompletion(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	var fullContent strings.Builder
	var completionID string
	for chunk := range ch {
		completionID = chunk.ID
		if len(chunk.Choices) > 0 {
			fullContent.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	inputTokens := estimateTokens(req.Messages)
	outputTokens := len(strings.Fields(fullContent.String()))

	resp := model.ChatCompletionResponse{
		ID:      completionID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.ChatCompletionChoice{
			{
				Index:        0,
				Message:      model.ChatMessage{Role: "assistant", Content: fullContent.String()},
				FinishReason: "stop",
			},
		},
		Usage: model.Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
	}

	go h.recordUsage(auth, req.Model, inputTokens, outputTokens)
	c.JSON(http.StatusOK, resp)
}

func (h *ChatCompletionsHandler) recordUsage(auth middleware.AuthInfo, modelName string, inputTokens, outputTokens int) {
	ctx := context.Background()
	var inputPrice, outputPrice float64
	_ = h.store.DB.QueryRow(ctx,
		`SELECT input_price, output_price FROM models WHERE id = $1`, modelName).
		Scan(&inputPrice, &outputPrice)

	cost := float64(inputTokens)/1_000_000*inputPrice + float64(outputTokens)/1_000_000*outputPrice

	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, modelName, inputTokens, outputTokens, cost)

	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)

	middleware.RecordTokenUsage(h.store.Redis, auth.APIKeyID, auth.ServiceTier, inputTokens+outputTokens)
}

func estimateTokens(messages []model.ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += len(strings.Fields(m.Content)) + 4
	}
	return total
}
