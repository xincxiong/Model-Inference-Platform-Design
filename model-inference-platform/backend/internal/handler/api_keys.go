package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/model"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type APIKeysHandler struct {
	store *store.Store
}

func NewAPIKeysHandler(s *store.Store) *APIKeysHandler {
	return &APIKeysHandler{store: s}
}

func (h *APIKeysHandler) Create(c *gin.Context) {
	var req model.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	auth := middleware.GetAuthInfo(c)
	rawKey := generateAPIKey()
	prefix := rawKey[:8]
	hash := sha256Hash(rawKey)

	tier := req.ServiceTier
	if tier == "" {
		tier = "default"
	}

	var id string
	err := h.store.DB.QueryRow(context.Background(),
		`INSERT INTO api_keys (user_id, name, key_hash, key_prefix, service_tier)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		auth.UserID, req.Name, hash, prefix, tier).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, model.CreateAPIKeyResponse{
		ID:          id,
		Name:        req.Name,
		Key:         rawKey,
		KeyPrefix:   prefix + "...",
		ServiceTier: tier,
	})
}

func (h *APIKeysHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	rows, err := h.store.DB.Query(context.Background(),
		`SELECT id, name, key_prefix, service_tier, expires_at, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.ServiceTier, &k.ExpiresAt, &k.CreatedAt); err != nil {
			continue
		}
		keys = append(keys, k)
	}

	if keys == nil {
		keys = []model.APIKey{}
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

func (h *APIKeysHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)

	result, err := h.store.DB.Exec(context.Background(),
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "API key not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("sk-%s", hex.EncodeToString(b))
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
