// Package engine provides inference engine implementations.
// vLLM (https://github.com/vllm-project/vllm) is the primary engine for
// high-performance LLM inference with PagedAttention and continuous batching.
//
// vLLM provides an OpenAI-compatible HTTP API at port 8000:
//   - POST /v1/chat/completions (streaming + non-streaming)
//   - POST /v1/completions
//   - POST /v1/embeddings
//   - GET /health
//
// This client supports both streaming and non-streaming modes.
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

// VLLMClient implements Engine interface for vLLM inference server.
type VLLMClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// VLLMConfig holds configuration for vLLM client.
type VLLMConfig struct {
	// BaseURL is the vLLM server endpoint (e.g., "http://localhost:8000")
	BaseURL string
	// Timeout for HTTP requests.
	Timeout time.Duration
}

// DefaultVLLMConfig returns sensible defaults.
func DefaultVLLMConfig() VLLMConfig {
	return VLLMConfig{
		BaseURL: "http://localhost:8000",
		Timeout: 120 * time.Second,
	}
}

// NewVLLMClient creates a new vLLM engine client.
func NewVLLMClient(cfg VLLMConfig, logger *zap.Logger) *VLLMClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &VLLMClient{
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

// ChatCompletion streams token-by-token responses from vLLM.
func (v *VLLMClient) ChatCompletion(ctx context.Context, req model.ChatCompletionRequest) (<-chan model.ChatCompletionChunk, error) {
	// vLLM expects OpenAI-compatible request format
	vllmReq := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"stream":      true, // always stream for vLLM
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}
	if req.TopP != nil {
		vllmReq["top_p"] = *req.TopP
	}
	if req.FrequencyPenalty != nil {
		vllmReq["frequency_penalty"] = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		vllmReq["presence_penalty"] = *req.PresencePenalty
	}
	if req.Stop != nil {
		vllmReq["stop"] = req.Stop
	}

	body, err := json.Marshal(vllmReq)
	if err != nil {
		return nil, fmt.Errorf("marshal vllm request: %w", err)
	}

	url := v.baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := v.httpClient.Do(httpReq)
	if err != nil {
		v.logger.Warn("vllm chat completion request failed", zap.Error(err))
		return nil, fmt.Errorf("vllm request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("vllm returned %d: %s", resp.StatusCode, string(errBody))
	}

	// Parse SSE stream
	ch := make(chan model.ChatCompletionChunk)
	go v.parseSSEStream(resp.Body, ch, req.Model)

	return ch, nil
}

// Completion returns a text completion from vLLM (FIM mode).
func (v *VLLMClient) Completion(ctx context.Context, req model.CompletionRequest) (*model.CompletionResponse, error) {
	vllmReq := map[string]interface{}{
		"model":      req.Model,
		"prompt":     req.Prompt,
		"max_tokens": req.MaxTokens,
		"stream":     false,
	}
	if req.Temperature != nil {
		vllmReq["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		vllmReq["top_p"] = *req.TopP
	}
	if req.Stop != nil {
		vllmReq["stop"] = req.Stop
	}

	body, _ := json.Marshal(vllmReq)
	url := v.baseURL + "/v1/completions"

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vllm completion: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("vllm returned %d: %s", resp.StatusCode, string(errBody))
	}

	var result model.CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode vllm response: %w", err)
	}
	return &result, nil
}

// Embedding returns embeddings from vLLM embedding endpoint.
func (v *VLLMClient) Embedding(ctx context.Context, req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	vllmReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Input,
	}

	body, _ := json.Marshal(vllmReq)
	url := v.baseURL + "/v1/embeddings"

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vllm embedding: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("vllm returned %d: %s", resp.StatusCode, string(errBody))
	}

	var result model.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode vllm embedding: %w", err)
	}
	return &result, nil
}

// Rerank is not supported by vLLM natively; returns error.
// Users should deploy a dedicated rerank model (e.g., BGE Reranker) via vLLM
// and call it through ChatCompletion with a custom prompt template.
func (v *VLLMClient) Rerank(ctx context.Context, req model.RerankRequest) (*model.RerankResponse, error) {
	return nil, fmt.Errorf("vLLM does not support native rerank endpoint; deploy a rerank model separately")
}

// ImageGeneration is not supported by vLLM; returns error.
// Users should deploy a text-to-image model (e.g., FLUX) via a separate service.
func (v *VLLMClient) ImageGeneration(ctx context.Context, req model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	return nil, fmt.Errorf("vLLM does not support image generation; deploy FLUX/SD model separately")
}

func (v *VLLMClient) VideoGeneration(ctx context.Context, req model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	return nil, fmt.Errorf("vLLM does not support video generation; deploy CogVideoX/SVD model separately")
}

func (v *VLLMClient) Transcription(ctx context.Context, req model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	return nil, fmt.Errorf("vLLM does not support audio transcription; deploy Whisper model separately")
}

func (v *VLLMClient) Speech(ctx context.Context, req model.SpeechRequest) ([]byte, error) {
	return nil, fmt.Errorf("vLLM does not support speech synthesis; deploy CosyVoice/TTS model separately")
}

// Health checks vLLM server connectivity.
func (v *VLLMClient) Health(ctx context.Context) error {
	url := v.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vllm health: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vllm health: status %d", resp.StatusCode)
	}
	return nil
}

// parseSSEStream parses Server-Sent Events from vLLM response.
func (v *VLLMClient) parseSSEStream(body io.ReadCloser, ch chan model.ChatCompletionChunk, modelID string) {
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
			v.logger.Warn("sse parse error", zap.Error(err), zap.String("data", dataStr))
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
		v.logger.Warn("sse scanner error", zap.Error(err))
	}
}