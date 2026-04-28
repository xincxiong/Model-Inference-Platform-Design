// Package engine provides inference engine implementations.
// SGLang (https://github.com/sgl-project/sglang) is a high-performance
// inference engine with RadixAttention for efficient prefix caching.
//
// SGLang provides an OpenAI-compatible HTTP API at port 30000:
//   - POST /v1/chat/completions (streaming + non-streaming)
//   - POST /v1/completions
//   - POST /v1/embeddings
//   - GET /health
//
// Key features:
//   - RadixAttention: automatic prefix caching across requests
//   - Faster decoding than vLLM for multi-turn conversations
//   - Native support for structured output (JSON mode)
package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"go.uber.org/zap"
)

// SGLangClient implements Engine interface for SGLang inference server.
type SGLangClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// SGLangConfig holds configuration for SGLang client.
type SGLangConfig struct {
	// BaseURL is the SGLang server endpoint (e.g., "http://localhost:30000")
	BaseURL string
	// Timeout for HTTP requests.
	Timeout time.Duration
}

// DefaultSGLangConfig returns sensible defaults.
func DefaultSGLangConfig() SGLangConfig {
	return SGLangConfig{
		BaseURL: "http://localhost:30000",
		Timeout: 120 * time.Second,
	}
}

// NewSGLangClient creates a new SGLang engine client.
func NewSGLangClient(cfg SGLangConfig, logger *zap.Logger) *SGLangClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &SGLangClient{
		baseURL: strings.TrimSuffix(cfg.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				IdleConnTimeout:     60 * time.Second,
				DisableCompression:  false,
				MaxConnsPerHost:     10,
			},
		},
		logger: logger,
	}
}

// ChatCompletion streams token-by-token responses from SGLang.
func (s *SGLangClient) ChatCompletion(ctx context.Context, req model.ChatCompletionRequest) (<-chan model.ChatCompletionChunk, error) {
	sglangReq := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"stream":      true,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}
	if req.TopP != nil {
		sglangReq["top_p"] = *req.TopP
	}
	if req.FrequencyPenalty != nil {
		sglangReq["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		sglangReq["presence_penalty"] = *req.PresencePenalty
	}
	if req.Stop != nil {
		sglangReq["stop"] = req.Stop
	}
	// SGLang-specific: JSON mode for structured output
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		sglangReq["response_format"] = map[string]string{"type": "json_object"}
	}

	body, err := json.Marshal(sglangReq)
	if err != nil {
		return nil, fmt.Errorf("marshal sglang request: %w", err)
	}

	url := s.baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.logger.Warn("sglang chat completion request failed", zap.Error(err))
		return nil, fmt.Errorf("sglang request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("sglang returned %d: %s", resp.StatusCode, string(errBody))
	}

	ch := make(chan model.ChatCompletionChunk)
	go s.parseSSEStream(resp.Body, ch, req.Model)

	return ch, nil
}

// Completion returns a text completion from SGLang.
func (s *SGLangClient) Completion(ctx context.Context, req model.CompletionRequest) (*model.CompletionResponse, error) {
	sglangReq := map[string]interface{}{
		"model":      req.Model,
		"prompt":     req.Prompt,
		"max_tokens": req.MaxTokens,
		"stream":     false,
	}
	if req.Temperature != nil {
		sglangReq["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		sglangReq["top_p"] = *req.TopP
	}
	if req.Stop != nil {
		sglangReq["stop"] = req.Stop
	}

	body, _ := json.Marshal(sglangReq)
	url := s.baseURL + "/v1/completions"

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sglang completion: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("sglang returned %d: %s", resp.StatusCode, string(errBody))
	}

	var result model.CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode sglang response: %w", err)
	}
	return &result, nil
}

// Embedding returns embeddings from SGLang.
func (s *SGLangClient) Embedding(ctx context.Context, req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	sglangReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Input,
	}

	body, _ := json.Marshal(sglangReq)
	url := s.baseURL + "/v1/embeddings"

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sglang embedding: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("sglang returned %d: %s", resp.StatusCode, string(errBody))
	}

	var result model.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode sglang embedding: %w", err)
	}
	return &result, nil
}

// Rerank is not supported by SGLang natively.
func (s *SGLangClient) Rerank(ctx context.Context, req model.RerankRequest) (*model.RerankResponse, error) {
	return nil, fmt.Errorf("SGLang does not support native rerank endpoint")
}

// ImageGeneration is not supported by SGLang.
func (s *SGLangClient) ImageGeneration(ctx context.Context, req model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	return nil, fmt.Errorf("SGLang does not support image generation")
}

func (s *SGLangClient) VideoGeneration(ctx context.Context, req model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	return nil, fmt.Errorf("SGLang does not support video generation")
}

func (s *SGLangClient) Transcription(ctx context.Context, req model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	return nil, fmt.Errorf("SGLang does not support audio transcription")
}

func (s *SGLangClient) Speech(ctx context.Context, req model.SpeechRequest) ([]byte, error) {
	return nil, fmt.Errorf("SGLang does not support speech synthesis")
}

// Health checks SGLang server connectivity.
func (s *SGLangClient) Health(ctx context.Context) error {
	url := s.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sglang health: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sglang health: status %d", resp.StatusCode)
	}
	return nil
}

// parseSSEStream parses Server-Sent Events from SGLang response.
func (s *SGLangClient) parseSSEStream(body io.ReadCloser, ch chan model.ChatCompletionChunk, modelID string) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "[DONE]" {
			return
		}

		var sseData struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			Model   string `json:"model"`
			Choices []struct {
				Index        int                    `json:"index"`
				Delta        map[string]interface{} `json:"delta"`
				FinishReason *string                `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(dataStr), &sseData); err != nil {
			s.logger.Warn("sglang sse parse error", zap.Error(err), zap.String("data", dataStr))
			continue
		}

		chunk := model.ChatCompletionChunk{
			ID:      sseData.ID,
			Object:  sseData.Object,
			Created: sseData.Created,
			Model:   sseData.Model,
		}

		for _, c := range sseData.Choices {
			choice := model.ChatCompletionChunkChoice{Index: c.Index}
			if role, ok := c.Delta["role"].(string); ok {
				choice.Delta.Role = role
			}
			if content, ok := c.Delta["content"].(string); ok {
				choice.Delta.Content = content
			}
			if c.FinishReason != nil {
				choice.FinishReason = c.FinishReason
			}
			chunk.Choices = append(chunk.Choices, choice)
		}

		ch <- chunk
	}

	if err := scanner.Err(); err != nil {
		s.logger.Warn("sglang sse scanner error", zap.Error(err))
	}
}