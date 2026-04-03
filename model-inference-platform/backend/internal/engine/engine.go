package engine

import (
	"context"

	"github.com/xincxiong/model-inference-platform/backend/internal/model"
)

type Engine interface {
	ChatCompletion(ctx context.Context, req model.ChatCompletionRequest) (<-chan model.ChatCompletionChunk, error)
}
