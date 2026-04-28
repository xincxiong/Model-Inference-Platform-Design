package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xincxiong/model-inference-platform/backend/internal/middleware"
	"github.com/xincxiong/model-inference-platform/backend/internal/store"
)

type FilesHandler struct {
	store *store.Store
}

func NewFilesHandler(s *store.Store) *FilesHandler {
	return &FilesHandler{store: s}
}

// Upload handles multipart file upload (POST /v1/files)
func (h *FilesHandler) Upload(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	purpose := c.PostForm("purpose")
	if purpose == "" {
		purpose = "batch"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "file field is required", "type": "invalid_request_error"}})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, 512*1024*1024)) // 512MB limit
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to read file", "type": "api_error"}})
		return
	}

	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])
	fileID := "file-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]

	ctx := c.Request.Context()

	// Upload to S3 when可用，失败回退到 DB
	if h.store.S3 != nil {
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if err := h.store.S3.Upload(ctx, fileID, contentType, content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "failed to upload to s3: " + err.Error(), "type": "api_error"}})
			return
		}
	}

	_, err = h.store.DB.Exec(ctx,
		`INSERT INTO files (id, user_id, filename, purpose, bytes, checksum, content)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		fileID, auth.UserID, header.Filename, purpose, len(content), checksum, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error(), "type": "api_error"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         fileID,
		"object":     "file",
		"filename":   header.Filename,
		"purpose":    purpose,
		"bytes":      len(content),
		"created_at": time.Now().Unix(),
		"status":     "processed",
	})
}

// List handles GET /v1/files
func (h *FilesHandler) List(c *gin.Context) {
	auth := middleware.GetAuthInfo(c)
	purpose := c.Query("purpose")

	query := `SELECT id, filename, purpose, bytes, created_at FROM files WHERE user_id = $1`
	args := []interface{}{auth.UserID}
	if purpose != "" {
		query += fmt.Sprintf(" AND purpose = $%d", len(args)+1)
		args = append(args, purpose)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := h.store.DB.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	defer rows.Close()

	type FileObject struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Filename  string `json:"filename"`
		Purpose   string `json:"purpose"`
		Bytes     int64  `json:"bytes"`
		CreatedAt int64  `json:"created_at"`
		Status    string `json:"status"`
	}

	var files []FileObject
	for rows.Next() {
		var f FileObject
		var createdAt time.Time
		if err := rows.Scan(&f.ID, &f.Filename, &f.Purpose, &f.Bytes, &createdAt); err != nil {
			continue
		}
		f.Object = "file"
		f.Status = "processed"
		f.CreatedAt = createdAt.Unix()
		files = append(files, f)
	}
	if files == nil {
		files = []FileObject{}
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": files})
}

// Get handles GET /v1/files/:id
func (h *FilesHandler) Get(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)

	var filename, purpose string
	var bytes int64
	var createdAt time.Time
	err := h.store.DB.QueryRow(c.Request.Context(),
		`SELECT filename, purpose, bytes, created_at FROM files WHERE id = $1 AND user_id = $2`,
		id, auth.UserID).Scan(&filename, &purpose, &bytes, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "file not found", "type": "invalid_request_error"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"object":     "file",
		"filename":   filename,
		"purpose":    purpose,
		"bytes":      bytes,
		"created_at": createdAt.Unix(),
		"status":     "processed",
	})
}

// Delete handles DELETE /v1/files/:id
func (h *FilesHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)

	ctx := c.Request.Context()
	res, err := h.store.DB.Exec(ctx,
		`DELETE FROM files WHERE id = $1 AND user_id = $2`, id, auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "file not found"}})
		return
	}

	if h.store.S3 != nil {
		_ = h.store.S3.Delete(ctx, id)
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "object": "file", "deleted": true})
}

// GetContent handles GET /v1/files/:id/content — returns raw file bytes
func (h *FilesHandler) GetContent(c *gin.Context) {
	id := c.Param("id")
	auth := middleware.GetAuthInfo(c)

	var content []byte
	var filename string
	ctx := c.Request.Context()

	if h.store.S3 != nil {
		data, contentType, err := h.store.S3.Download(ctx, id)
		if err == nil {
			// Fetch filename from DB for attachment header
			err = h.store.DB.QueryRow(ctx,
				`SELECT filename FROM files WHERE id = $1 AND user_id = $2`,
				id, auth.UserID).Scan(&filename)
			if err != nil {
				filename = id
			}
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			c.Data(http.StatusOK, contentType, data)
			return
		}
	}

	err := h.store.DB.QueryRow(ctx,
		`SELECT content, filename FROM files WHERE id = $1 AND user_id = $2`,
		id, auth.UserID).Scan(&content, &filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "file not found"}})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/octet-stream", content)
}
