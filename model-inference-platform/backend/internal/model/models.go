package model

import "time"

// --- Database models ---

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type APIKey struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	KeyHash     string    `json:"-"`
	KeyPrefix   string    `json:"key_prefix"`
	ServiceTier string    `json:"service_tier"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ModelInfo struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ModelType     string  `json:"type"`
	Provider      string  `json:"provider"`
	InputPrice    float64 `json:"input_price_per_million"`
	OutputPrice   float64 `json:"output_price_per_million"`
	MaxContext     int     `json:"max_context"`
	Status        string  `json:"status"`
}

type BillingAccount struct {
	ID      string  `json:"id"`
	UserID  string  `json:"user_id"`
	Balance float64 `json:"balance"`
}

type UsageRecord struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	APIKeyID     string    `json:"api_key_id"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	CreatedAt    time.Time `json:"created_at"`
}

type PromoCode struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Amount    float64   `json:"amount"`
	Used      bool      `json:"used"`
	UsedByID  *string   `json:"used_by_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// --- API request/response models ---

type ChatCompletionRequest struct {
	Model       string          `json:"model" binding:"required"`
	Messages    []ChatMessage   `json:"messages" binding:"required"`
	Stream      bool            `json:"stream"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	ServiceTier string          `json:"service_tier,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   Usage                  `json:"usage"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatCompletionChunk struct {
	ID      string                      `json:"id"`
	Object  string                      `json:"object"`
	Created int64                       `json:"created"`
	Model   string                      `json:"model"`
	Choices []ChatCompletionChunkChoice `json:"choices"`
	Usage   *Usage                      `json:"usage,omitempty"`
}

type ChatCompletionChunkChoice struct {
	Index        int              `json:"index"`
	Delta        ChatMessageDelta `json:"delta"`
	FinishReason *string          `json:"finish_reason"`
}

type ChatMessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// --- Responses API models ---

type ResponsesCreateRequest struct {
	Model              string      `json:"model" binding:"required"`
	Input              interface{} `json:"input" binding:"required"`
	Instructions       string      `json:"instructions,omitempty"`
	Stream             bool        `json:"stream"`
	Store              *bool       `json:"store,omitempty"`
	PreviousResponseID string      `json:"previous_response_id,omitempty"`
	Temperature        *float64    `json:"temperature,omitempty"`
	MaxOutputTokens    *int        `json:"max_output_tokens,omitempty"`
	TopP               *float64    `json:"top_p,omitempty"`
}

type ResponseObject struct {
	ID         string          `json:"id"`
	Object     string          `json:"object"`
	Model      string          `json:"model"`
	CreatedAt  int64           `json:"created_at"`
	Status     string          `json:"status"`
	Output     []ResponseItem  `json:"output"`
	OutputText string          `json:"output_text"`
	Usage      ResponseUsage   `json:"usage"`
}

type ResponseItem struct {
	Type    string               `json:"type"`
	Role    string               `json:"role,omitempty"`
	Content []ResponseItemContent `json:"content,omitempty"`
}

type ResponseItemContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// --- OpenAI Models API ---

type ModelsListResponse struct {
	Object string           `json:"object"`
	Data   []ModelInfoEntry `json:"data"`
}

type ModelInfoEntry struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// --- API Key API ---

type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required"`
	ServiceTier string `json:"service_tier,omitempty"`
}

type CreateAPIKeyResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	KeyPrefix   string `json:"key_prefix"`
	ServiceTier string `json:"service_tier"`
	CreatedAt   string `json:"created_at"`
}

// --- Billing API ---

type RedeemPromoRequest struct {
	Code string `json:"code" binding:"required"`
}

type UsageSummary struct {
	Balance       float64        `json:"balance"`
	TotalSpent    float64        `json:"total_spent"`
	TotalTokens   int64          `json:"total_tokens"`
	DailyBreakdown []DailyUsage  `json:"daily_breakdown,omitempty"`
}

type DailyUsage struct {
	Date         string  `json:"date"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}
