package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type ModelsHandler struct {
	store *store.Store
}

func NewModelsHandler(s *store.Store) *ModelsHandler {
	return &ModelsHandler{store: s}
}

// List returns models in OpenAI-compatible format.
func (h *ModelsHandler) List(c *gin.Context) {
	rows, err := h.store.DB.Query(context.Background(),
		`SELECT id, name, model_type, provider, description, input_price, output_price, max_context, speed, quality_score, features, status
		 FROM models WHERE status = 'active' ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var entries []model.ModelInfoEntry
	for rows.Next() {
		var m model.ModelInfo
		var featuresStr string
		if err := rows.Scan(&m.ID, &m.Name, &m.ModelType, &m.Provider, &m.Description, &m.InputPrice, &m.OutputPrice,
			&m.MaxContext, &m.Speed, &m.QualityScore, &featuresStr, &m.Status); err != nil {
			continue
		}
		entries = append(entries, model.ModelInfoEntry{
			ID:      m.ID,
			Object:  "model",
			Created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
			OwnedBy: m.Provider,
		})
	}

	c.JSON(http.StatusOK, model.ModelsListResponse{Object: "list", Data: entries})
}

// ListDetailed returns full model info for the console.
func (h *ModelsHandler) ListDetailed(c *gin.Context) {
	rows, err := h.store.DB.Query(context.Background(),
		`SELECT id, name, model_type, provider, description, input_price, output_price, max_context, speed, quality_score, features, status
		 FROM models WHERE status = 'active' ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var models []model.ModelInfo
	for rows.Next() {
		var m model.ModelInfo
		var featuresStr string
		if err := rows.Scan(&m.ID, &m.Name, &m.ModelType, &m.Provider, &m.Description, &m.InputPrice, &m.OutputPrice,
			&m.MaxContext, &m.Speed, &m.QualityScore, &featuresStr, &m.Status); err != nil {
			continue
		}
		if featuresStr != "" {
			m.Features = strings.Split(featuresStr, ",")
		}
		models = append(models, m)
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}
