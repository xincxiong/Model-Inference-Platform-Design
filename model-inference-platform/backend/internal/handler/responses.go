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

type ResponsesHandler struct {
	store  *store.Store
	engine engine.Engine
}

func NewResponsesHandler(s *store.Store, eng engine.Engine) *ResponsesHandler {
	return &ResponsesHandler{store: s, engine: eng}
}

func (h *ResponsesHandler) Create(c *gin.Context) {
	var req model.ResponsesCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"message": err.Error(), "type": "invalid_request_error"},
		})
		return
	}

	auth := middleware.GetAuthInfo(c)
	messages := h.buildMessages(c.Request.Context(), req, auth.UserID)

	chatReq := model.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxOutputTokens,
		TopP:        req.TopP,
	}

	if req.Stream {
		h.handleStream(c, chatReq, req, auth)
	} else {
		h.handleNonStream(c, chatReq, req, auth)
	}
}

func (h *ResponsesHandler) buildMessages(ctx context.Context, req model.ResponsesCreateRequest, userID string) []model.ChatMessage {
	var messages []model.ChatMessage

	if req.PreviousResponseID != "" {
		var prevInput, prevOutput json.RawMessage
		_ = h.store.DB.QueryRow(ctx,
			`SELECT input, output FROM responses WHERE id = $1 AND user_id = $2`,
			req.PreviousResponseID, userID).Scan(&prevInput, &prevOutput)

		var prevMsgs []model.ChatMessage
		_ = json.Unmarshal(prevInput, &prevMsgs)
		messages = append(messages, prevMsgs...)

		var prevItems []model.ResponseItem
		_ = json.Unmarshal(prevOutput, &prevItems)
		for _, item := range prevItems {
			if item.Type == "message" && item.Role == "assistant" {
				for _, c := range item.Content {
					if c.Type == "output_text" {
						messages = append(messages, model.ChatMessage{Role: "assistant", Content: c.Text})
					}
				}
			}
		}
	}

	if req.Instructions != "" {
		messages = append([]model.ChatMessage{{Role: "system", Content: req.Instructions}}, messages...)
	}

	switch v := req.Input.(type) {
	case string:
		messages = append(messages, model.ChatMessage{Role: "user", Content: v})
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				role, _ := m["role"].(string)
				content, _ := m["content"].(string)
				if role != "" && content != "" {
					messages = append(messages, model.ChatMessage{Role: role, Content: content})
				}
			}
		}
	}

	return messages
}

func (h *ResponsesHandler) handleStream(c *gin.Context, chatReq model.ChatCompletionRequest, origReq model.ResponsesCreateRequest, auth middleware.AuthInfo) {
	ch, err := h.engine.ChatCompletion(c.Request.Context(), chatReq)
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
	responseID := fmt.Sprintf("resp_%s", uuid.New().String()[:12])
	var fullContent strings.Builder
	inputTokens := estimateTokens(chatReq.Messages)

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
	go h.storeAndRecord(auth, origReq, chatReq.Messages, responseID, fullContent.String(), inputTokens, outputTokens)
}

func (h *ResponsesHandler) handleNonStream(c *gin.Context, chatReq model.ChatCompletionRequest, origReq model.ResponsesCreateRequest, auth middleware.AuthInfo) {
	ch, err := h.engine.ChatCompletion(c.Request.Context(), chatReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": err.Error(), "type": "server_error"},
		})
		return
	}

	var fullContent strings.Builder
	for chunk := range ch {
		if len(chunk.Choices) > 0 {
			fullContent.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	responseID := fmt.Sprintf("resp_%s", uuid.New().String()[:12])
	inputTokens := estimateTokens(chatReq.Messages)
	outputTokens := len(strings.Fields(fullContent.String()))

	resp := model.ResponseObject{
		ID:        responseID,
		Object:    "response",
		Model:     chatReq.Model,
		CreatedAt: time.Now().Unix(),
		Status:    "completed",
		Output: []model.ResponseItem{
			{
				Type: "message",
				Role: "assistant",
				Content: []model.ResponseItemContent{
					{Type: "output_text", Text: fullContent.String()},
				},
			},
		},
		OutputText: fullContent.String(),
		Usage: model.ResponseUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
		},
	}

	go h.storeAndRecord(auth, origReq, chatReq.Messages, responseID, fullContent.String(), inputTokens, outputTokens)
	c.JSON(http.StatusOK, resp)
}

func (h *ResponsesHandler) storeAndRecord(auth middleware.AuthInfo, req model.ResponsesCreateRequest, messages []model.ChatMessage, respID, text string, inTok, outTok int) {
	ctx := context.Background()
	shouldStore := req.Store == nil || *req.Store

	if shouldStore {
		inputJSON, _ := json.Marshal(messages)
		output := []model.ResponseItem{{
			Type: "message", Role: "assistant",
			Content: []model.ResponseItemContent{{Type: "output_text", Text: text}},
		}}
		outputJSON, _ := json.Marshal(output)

		_, _ = h.store.DB.Exec(ctx,
			`INSERT INTO responses (id, user_id, model, input, output, output_text, input_tokens, output_tokens)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			respID, auth.UserID, req.Model, inputJSON, outputJSON, text, inTok, outTok)

		h.store.Redis.Set(ctx, "resp:"+respID, string(outputJSON), 30*time.Minute)
	}

	var inputPrice, outputPrice float64
	_ = h.store.DB.QueryRow(ctx, `SELECT input_price, output_price FROM models WHERE id = $1`, req.Model).
		Scan(&inputPrice, &outputPrice)

	cost := float64(inTok)/1_000_000*inputPrice + float64(outTok)/1_000_000*outputPrice
	_, _ = h.store.DB.Exec(ctx,
		`INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.New().String(), auth.UserID, auth.APIKeyID, req.Model, inTok, outTok, cost)
	_, _ = h.store.DB.Exec(ctx,
		`UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}

func (h *ResponsesHandler) Get(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)

	var resp model.ResponseObject
	var inputJSON, outputJSON []byte
	err := h.store.DB.QueryRow(context.Background(),
		`SELECT id, model, input, output, output_text, input_tokens, output_tokens, 
		        EXTRACT(EPOCH FROM created_at)::bigint
		 FROM responses WHERE id = $1 AND user_id = $2`, id, auth.UserID).
		Scan(&resp.ID, &resp.Model, &inputJSON, &outputJSON, &resp.OutputText,
			&resp.Usage.InputTokens, &resp.Usage.OutputTokens, &resp.CreatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"message": "Response not found", "type": "not_found_error"},
		})
		return
	}

	resp.Object = "response"
	resp.Status = "completed"
	resp.Usage.TotalTokens = resp.Usage.InputTokens + resp.Usage.OutputTokens
	_ = json.Unmarshal(outputJSON, &resp.Output)

	c.JSON(http.StatusOK, resp)
}

// GetInputItems returns the input items of a stored response.
func (h *ResponsesHandler) GetInputItems(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)

	var inputJSON []byte
	err := h.store.DB.QueryRow(context.Background(),
		`SELECT input FROM responses WHERE id = $1 AND user_id = $2`, id, auth.UserID).
		Scan(&inputJSON)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"message": "Response not found", "type": "not_found_error"},
		})
		return
	}

	var items []model.ChatMessage
	_ = json.Unmarshal(inputJSON, &items)

	responseItems := make([]model.ResponseItem, 0, len(items))
	for _, msg := range items {
		responseItems = append(responseItems, model.ResponseItem{
			Type: "message",
			Role: msg.Role,
			Content: []model.ResponseItemContent{
				{Type: "input_text", Text: msg.ContentString()},
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   responseItems,
	})
}
