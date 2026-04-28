package engine

import (
	"context"

	"github.com/xincxiong/model-inference-platform/backend/internal/model"
)

type Engine interface {
	ChatCompletion(ctx context.Context, req model.ChatCompletionRequest) (<-chan model.ChatCompletionChunk, error)
	Completion(ctx context.Context, req model.CompletionRequest) (*model.CompletionResponse, error)
	Embedding(ctx context.Context, req model.EmbeddingRequest) (*model.EmbeddingResponse, error)
	Rerank(ctx context.Context, req model.RerankRequest) (*model.RerankResponse, error)
	ImageGeneration(ctx context.Context, req model.ImageGenerationRequest) (*model.ImageGenerationResponse, error)
	VideoGeneration(ctx context.Context, req model.VideoGenerationRequest) (*model.VideoGenerationResponse, error)
	Transcription(ctx context.Context, req model.TranscriptionRequest) (*model.TranscriptionResponse, error)
	Speech(ctx context.Context, req model.SpeechRequest) ([]byte, error)
}
