package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AuthInfo struct {
	UserID      string
	APIKeyID    string
	ServiceTier string
}

const (
	authInfoKey   = "auth_info"
	redisCacheTTL = 60 * time.Second
)

func AuthMiddleware(db *pgxpool.Pool, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "Missing Authorization header", "type": "authentication_error"},
			})
			return
		}

		key := strings.TrimPrefix(auth, "Bearer ")
		if key == auth {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "Invalid Authorization format, expected 'Bearer <key>'", "type": "authentication_error"},
			})
			return
		}

		hash := hashKey(key)

		cached, err := rdb.HGetAll(context.Background(), "apikey:"+hash).Result()
		if err == nil && len(cached) > 0 {
			c.Set(authInfoKey, AuthInfo{
				UserID:      cached["user_id"],
				APIKeyID:    cached["key_id"],
				ServiceTier: cached["tier"],
			})
			c.Next()
			return
		}

		var keyID, userID, tier string
		var expiresAt *time.Time
		err = db.QueryRow(context.Background(),
			`SELECT k.id, k.user_id, k.service_tier, k.expires_at
			 FROM api_keys k WHERE k.key_hash = $1`, hash).
			Scan(&keyID, &userID, &tier, &expiresAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "Invalid API key", "type": "authentication_error"},
			})
			return
		}

		if expiresAt != nil && expiresAt.Before(time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"message": "API key expired", "type": "authentication_error"},
			})
			return
		}

		rdb.HSet(context.Background(), "apikey:"+hash, map[string]interface{}{
			"user_id": userID, "key_id": keyID, "tier": tier,
		})
		rdb.Expire(context.Background(), "apikey:"+hash, redisCacheTTL)

		c.Set(authInfoKey, AuthInfo{UserID: userID, APIKeyID: keyID, ServiceTier: tier})
		c.Next()
	}
}

func GetAuthInfo(c *gin.Context) AuthInfo {
	v, _ := c.Get(authInfoKey)
	info, _ := v.(AuthInfo)
	return info
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
