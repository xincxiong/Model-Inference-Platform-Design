package handler

import (
    "context"
    "errors"
    "strings"

    "github.com/xincxiong/model-inference-platform/backend/internal/middleware"
    "github.com/xincxiong/model-inference-platform/backend/internal/model"
    "github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
    "github.com/xincxiong/model-inference-platform/backend/internal/semcache"
)

// buildEmbedder returns a function that produces embeddings via the configured embedding model.
func buildEmbedder(mr *modelrouter.ModelRouter, embedModel string, auth middleware.AuthInfo) semcache.Embedder {
    return func(ctx context.Context, input string) ([]float32, error) {
        if embedModel == "" {
            return nil, errors.New("embedding model not configured")
        }
        resolved, err := mr.Resolve(embedModel, auth.UserID)
        if err != nil {
            return nil, err
        }
        if err := mr.ValidateType(resolved, []string{"embedding"}); err != nil {
            return nil, err
        }
        eng := mr.GetEngine(resolved)
        resp, err := eng.Embedding(ctx, model.EmbeddingRequest{Model: embedModel, Input: input})
        if err != nil {
            return nil, err
        }
        if len(resp.Data) == 0 {
            return nil, errors.New("empty embedding response")
        }
        // Convert []float64 to []float32
        emb64 := resp.Data[0].Embedding
        emb32 := make([]float32, len(emb64))
        for i, v := range emb64 {
            emb32[i] = float32(v)
        }
        return emb32, nil
    }
}

func joinChatMessages(msgs []model.ChatMessage) string {
    var b strings.Builder
    for _, m := range msgs {
        b.WriteString(m.Role)
        b.WriteString(":")
        b.WriteString(m.ContentString())
        b.WriteString("\n")
    }
    return b.String()
}
