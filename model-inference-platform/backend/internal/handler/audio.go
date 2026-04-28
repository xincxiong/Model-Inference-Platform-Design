package handler

import (
    "context"
    "io"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/xincxiong/model-inference-platform/backend/internal/middleware"
    "github.com/xincxiong/model-inference-platform/backend/internal/model"
    "github.com/xincxiong/model-inference-platform/backend/internal/modelrouter"
    "github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type AudioHandler struct {
    store  *store.Store
    router *modelrouter.ModelRouter
}

func NewAudioHandler(s *store.Store, mr *modelrouter.ModelRouter) *AudioHandler {
    return &AudioHandler{store: s, router: mr}
}

var speechAllowedTypes = []string{"speech"}

// CreateTranscription handles POST /v1/audio/transcriptions
func (h *AudioHandler) CreateTranscription(c *gin.Context) {
    auth := middleware.GetAuthInfo(c)

    // Parse multipart form
    modelID := c.PostForm("model")
    if modelID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "model is required", "type": "invalid_request_error"}})
        return
    }

    file, _, err := c.Request.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "file is required", "type": "invalid_request_error"}})
        return
    }
    defer file.Close()

    resolved, err := h.router.Resolve(modelID, auth.UserID)
    if err != nil {
        status, errType := modelrouter.HTTPStatusForResolveError(err)
        c.JSON(status, gin.H{"error": gin.H{"message": err.Error(), "type": errType}})
        return
    }

    if err := h.router.ValidateType(resolved, speechAllowedTypes); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
        return
    }

    // Read audio content
    content, err := io.ReadAll(io.LimitReader(file, 25*1024*1024)) // 25MB limit
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to read file", "type": "api_error"}})
        return
    }

    eng := h.router.GetEngine(resolved)
    resp, err := eng.Transcription(c.Request.Context(), model.TranscriptionRequest{
        Model:    modelID,
        File:     content,
        Language: c.PostForm("language"),
        Prompt:   c.PostForm("prompt"),
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "server_error"}})
        return
    }

    go h.recordUsage(auth, resolved, len(content), len(resp.Text))
    c.JSON(http.StatusOK, resp)
}

// CreateSpeech handles POST /v1/audio/speech
func (h *AudioHandler) CreateSpeech(c *gin.Context) {
    var req model.SpeechRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
        return
    }

    auth := middleware.GetAuthInfo(c)
    resolved, err := h.router.Resolve(req.Model, auth.UserID)
    if err != nil {
        status, errType := modelrouter.HTTPStatusForResolveError(err)
        c.JSON(status, gin.H{"error": gin.H{"message": err.Error(), "type": errType}})
        return
    }

    if err := h.router.ValidateType(resolved, speechAllowedTypes); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error(), "type": "invalid_request_error"}})
        return
    }

    eng := h.router.GetEngine(resolved)
    audioData, err := eng.Speech(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "server_error"}})
        return
    }

    go h.recordUsage(auth, resolved, len(req.Input), len(audioData))

    contentType := "audio/mpeg"
    if req.ResponseFormat == "wav" {
        contentType = "audio/wav"
    } else if req.ResponseFormat == "opus" {
        contentType = "audio/opus"
    }

    c.Data(http.StatusOK, contentType, audioData)
}

func (h *AudioHandler) recordUsage(auth middleware.AuthInfo, resolved *modelrouter.ResolvedModel, inputLen, outputLen int) {
    ctx := context.Background()
    // Estimate tokens: ~4 chars per token for audio/text
    inputTokens := inputLen / 4
    outputTokens := outputLen / 4
    cost := float64(inputTokens)/1_000_000*resolved.InputPrice + float64(outputTokens)/1_000_000*resolved.OutputPrice

    _, _ = h.store.DB.Exec(ctx,
        `INSERT INTO usage_records (id, user_id, api_key_id, model, input_tokens, output_tokens, cost)
         VALUES ($1,$2,$3,$4,$5,$6,$7)`,
        uuid.New().String(), auth.UserID, auth.APIKeyID, resolved.ID, inputTokens, outputTokens, cost)
    _, _ = h.store.DB.Exec(ctx,
        `UPDATE billing_accounts SET balance = balance - $1 WHERE user_id = $2`, cost, auth.UserID)
}
