package engine

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
)

type MockEngine struct{}

func NewMockEngine() *MockEngine {
	return &MockEngine{}
}

// ChatCompletion streams token-by-token mock responses.
func (m *MockEngine) ChatCompletion(ctx context.Context, req model.ChatCompletionRequest) (<-chan model.ChatCompletionChunk, error) {
	ch := make(chan model.ChatCompletionChunk)
	completionID := fmt.Sprintf("chatcmpl-%s", uuid.New().String()[:12])
	created := time.Now().Unix()

	userMsg := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userMsg = msg.ContentString()
		}
	}

	response := generateMockResponse(userMsg, req.Model)
	tokens := strings.Fields(response)

	go func() {
		defer close(ch)

		ch <- model.ChatCompletionChunk{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: []model.ChatCompletionChunkChoice{
				{Index: 0, Delta: model.ChatMessageDelta{Role: "assistant"}},
			},
		}

		for i, token := range tokens {
			select {
			case <-ctx.Done():
				return
			default:
			}

			content := token
			if i < len(tokens)-1 {
				content += " "
			}

			time.Sleep(30 * time.Millisecond)

			ch <- model.ChatCompletionChunk{
				ID:      completionID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   req.Model,
				Choices: []model.ChatCompletionChunkChoice{
					{Index: 0, Delta: model.ChatMessageDelta{Content: content}},
				},
			}
		}

		done := "stop"
		ch <- model.ChatCompletionChunk{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: []model.ChatCompletionChunkChoice{
				{Index: 0, Delta: model.ChatMessageDelta{}, FinishReason: &done},
			},
		}
	}()

	return ch, nil
}

// Completion returns a mock text completion (FIM).
func (m *MockEngine) Completion(_ context.Context, req model.CompletionRequest) (*model.CompletionResponse, error) {
	generated := fmt.Sprintf("/* mock completion for model %s */\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}", req.Model)
	promptTokens := len(strings.Fields(req.Prompt)) + 4
	completionTokens := len(strings.Fields(generated))

	return &model.CompletionResponse{
		ID:      fmt.Sprintf("cmpl-%s", uuid.New().String()[:12]),
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.CompletionChoice{
			{Index: 0, Text: generated, FinishReason: "stop"},
		},
		Usage: model.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

// Embedding returns mock 1536-dimensional embeddings.
func (m *MockEngine) Embedding(_ context.Context, req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	var inputs []string
	switch v := req.Input.(type) {
	case string:
		inputs = []string{v}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				inputs = append(inputs, s)
			}
		}
	}

	dim := 1536
	data := make([]model.EmbeddingData, len(inputs))
	totalTokens := 0

	for i, input := range inputs {
		tokens := len(strings.Fields(input)) + 2
		totalTokens += tokens

		vec := make([]float64, dim)
		r := rand.New(rand.NewSource(int64(hashStr(input))))
		norm := 0.0
		for j := range vec {
			vec[j] = r.NormFloat64()
			norm += vec[j] * vec[j]
		}
		norm = math.Sqrt(norm)
		for j := range vec {
			vec[j] /= norm
		}

		data[i] = model.EmbeddingData{
			Object:    "embedding",
			Embedding: vec,
			Index:     i,
		}
	}

	return &model.EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage: model.EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

// Rerank returns mock relevance scores.
func (m *MockEngine) Rerank(_ context.Context, req model.RerankRequest) (*model.RerankResponse, error) {
	results := make([]model.RerankResult, len(req.Documents))
	totalTokens := len(strings.Fields(req.Query)) + 4

	for i, doc := range req.Documents {
		totalTokens += len(strings.Fields(doc)) + 2
		score := 1.0 - float64(i)*0.15
		if score < 0.05 {
			score = 0.05
		}
		results[i] = model.RerankResult{
			Index:          i,
			RelevanceScore: score,
			Document:       doc,
		}
	}

	topN := len(results)
	if req.TopN != nil && *req.TopN < topN {
		topN = *req.TopN
	}
	results = results[:topN]

	return &model.RerankResponse{
		Object:  "list",
		Results: results,
		Model:   req.Model,
		Usage:   model.RerankUsage{TotalTokens: totalTokens},
	}, nil
}

// ImageGeneration returns a mock placeholder image URL.
func (m *MockEngine) ImageGeneration(_ context.Context, req model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	n := 1
	if req.N != nil && *req.N > 0 {
		n = *req.N
	}
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}

	data := make([]model.ImageData, n)
	for i := range data {
		data[i] = model.ImageData{
			URL:           fmt.Sprintf("https://placehold.co/%s/1a73e8/ffffff?text=Mock+Image+%d", strings.ReplaceAll(size, "x", "x"), i+1),
			RevisedPrompt: req.Prompt,
		}
	}

	return &model.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data:    data,
	}, nil
}

func generateMockResponse(userMsg, modelName string) string {
	if strings.Contains(strings.ToLower(userMsg), "hello") || strings.Contains(strings.ToLower(userMsg), "你好") {
		return fmt.Sprintf("你好！我是 %s 模型。很高兴为你服务。有什么我可以帮助你的吗？", modelName)
	}
	if strings.Contains(strings.ToLower(userMsg), "transformer") {
		return "Transformer 是一种基于自注意力机制的神经网络架构，由 Google 在 2017 年的论文 \"Attention is All You Need\" 中提出。它摒弃了传统的 RNN 和 CNN 结构，完全依赖注意力机制来处理序列数据。Transformer 的核心组件包括多头自注意力层、前馈网络层和残差连接。它已成为现代大语言模型（如 GPT、BERT）的基础架构。"
	}
	return fmt.Sprintf("这是来自 %s 模型的模拟回复。在生产环境中，这里将由 vLLM 推理引擎生成真实的模型输出。您的输入是：\"%s\"。Mock 引擎会模拟真实的流式响应行为，包括逐 token 输出和延迟模拟。", modelName, userMsg)
}

func hashStr(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
