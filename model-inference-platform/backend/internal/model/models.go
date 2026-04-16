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
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"key_prefix"`
	ServiceTier string     `json:"service_tier"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ModelType    string   `json:"type"`
	Provider     string   `json:"provider"`
	Description  string   `json:"description"`
	InputPrice   float64  `json:"input_price_per_million"`
	OutputPrice  float64  `json:"output_price_per_million"`
	MaxContext   int      `json:"max_context"`
	Speed        float64  `json:"speed"`
	QualityScore float64  `json:"quality_score"`
	Features     []string `json:"features"`
	Status       string   `json:"status"`
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

// =============================================
// Chat Completions API
// =============================================

type ChatCompletionRequest struct {
	Model            string          `json:"model" binding:"required"`
	Messages         []ChatMessage   `json:"messages" binding:"required"`
	Stream           bool            `json:"stream"`
	Temperature      *float64        `json:"temperature,omitempty"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	ServiceTier      string          `json:"service_tier,omitempty"`
	ResponseFormat   *ResponseFormat `json:"response_format,omitempty"`
	Tools            []ToolDef       `json:"tools,omitempty"`
	ToolChoice       interface{}     `json:"tool_choice,omitempty"`
	Stop             interface{}     `json:"stop,omitempty"`
	N                *int            `json:"n,omitempty"`
}

type ChatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

func (m ChatMessage) ContentString() string {
	switch v := m.Content.(type) {
	case string:
		return v
	default:
		return ""
	}
}

type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema interface{} `json:"json_schema,omitempty"`
}

type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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

// =============================================
// Completions API (FIM / text completion)
// =============================================

type CompletionRequest struct {
	Model       string      `json:"model" binding:"required"`
	Prompt      string      `json:"prompt" binding:"required"`
	Suffix      string      `json:"suffix,omitempty"`
	MaxTokens   *int        `json:"max_tokens,omitempty"`
	Temperature *float64    `json:"temperature,omitempty"`
	TopP        *float64    `json:"top_p,omitempty"`
	Stream      bool        `json:"stream"`
	Stop        interface{} `json:"stop,omitempty"`
}

type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

type CompletionChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

// =============================================
// Responses API
// =============================================

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
	Tools              []ToolDef   `json:"tools,omitempty"`
	ToolChoice         interface{} `json:"tool_choice,omitempty"`
	Text               *TextFormat `json:"text,omitempty"`
}

type TextFormat struct {
	Format ResponseFormat `json:"format"`
}

type ResponseObject struct {
	ID         string         `json:"id"`
	Object     string         `json:"object"`
	Model      string         `json:"model"`
	CreatedAt  int64          `json:"created_at"`
	Status     string         `json:"status"`
	Output     []ResponseItem `json:"output"`
	OutputText string         `json:"output_text"`
	Usage      ResponseUsage  `json:"usage"`
}

type ResponseItem struct {
	Type    string                `json:"type"`
	Role    string                `json:"role,omitempty"`
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

// =============================================
// Embeddings API
// =============================================

type EmbeddingRequest struct {
	Model          string      `json:"model" binding:"required"`
	Input          interface{} `json:"input" binding:"required"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingUsage  `json:"usage"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// =============================================
// Rerank API
// =============================================

type RerankRequest struct {
	Model     string   `json:"model" binding:"required"`
	Query     string   `json:"query" binding:"required"`
	Documents []string `json:"documents" binding:"required"`
	TopN      *int     `json:"top_n,omitempty"`
}

type RerankResponse struct {
	Object  string         `json:"object"`
	Results []RerankResult `json:"results"`
	Model   string         `json:"model"`
	Usage   RerankUsage    `json:"usage"`
}

type RerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document"`
}

type RerankUsage struct {
	TotalTokens int `json:"total_tokens"`
}

// =============================================
// Images API
// =============================================

type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt" binding:"required"`
	N              *int   `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// =============================================
// OpenAI Models API
// =============================================

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

// =============================================
// API Key API
// =============================================

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

// =============================================
// Billing API
// =============================================

type RedeemPromoRequest struct {
	Code string `json:"code" binding:"required"`
}

type UsageSummary struct {
	Balance        float64      `json:"balance"`
	TotalSpent     float64      `json:"total_spent"`
	TotalTokens    int64        `json:"total_tokens"`
	DailyBreakdown []DailyUsage `json:"daily_breakdown,omitempty"`
}

type DailyUsage struct {
	Date         string  `json:"date"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}
