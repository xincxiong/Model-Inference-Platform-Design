package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
)

type MockEngine struct{}

func NewMockEngine() *MockEngine {
	return &MockEngine{}
}

func (m *MockEngine) ChatCompletion(ctx context.Context, req model.ChatCompletionRequest) (<-chan model.ChatCompletionChunk, error) {
	ch := make(chan model.ChatCompletionChunk)
	completionID := fmt.Sprintf("chatcmpl-%s", uuid.New().String()[:12])
	created := time.Now().Unix()

	userMsg := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userMsg = msg.Content
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

func generateMockResponse(userMsg, modelName string) string {
	if strings.Contains(strings.ToLower(userMsg), "hello") || strings.Contains(strings.ToLower(userMsg), "你好") {
		return fmt.Sprintf("你好！我是 %s 模型。很高兴为你服务。有什么我可以帮助你的吗？", modelName)
	}
	if strings.Contains(strings.ToLower(userMsg), "transformer") {
		return "Transformer 是一种基于自注意力机制的神经网络架构，由 Google 在 2017 年的论文 \"Attention is All You Need\" 中提出。它摒弃了传统的 RNN 和 CNN 结构，完全依赖注意力机制来处理序列数据。Transformer 的核心组件包括多头自注意力层、前馈网络层和残差连接。它已成为现代大语言模型（如 GPT、BERT）的基础架构。"
	}
	return fmt.Sprintf("这是来自 %s 模型的模拟回复。在生产环境中，这里将由 vLLM 推理引擎生成真实的模型输出。您的输入是：\"%s\"。Mock 引擎会模拟真实的流式响应行为，包括逐 token 输出和延迟模拟。", modelName, userMsg)
}
