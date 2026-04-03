package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type TierLimits struct {
	RPM int
	TPM int
}

var tierConfig = map[string]TierLimits{
	"default": {RPM: 60, TPM: 100000},
	"flex":    {RPM: 0, TPM: 0},
	"auto":    {RPM: 120, TPM: 200000},
}

func RateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		info := GetAuthInfo(c)
		if info.ServiceTier == "flex" {
			c.Next()
			return
		}

		tier := info.ServiceTier
		if tier == "" {
			tier = "default"
		}
		limits, ok := tierConfig[tier]
		if !ok {
			limits = tierConfig["default"]
		}

		if limits.RPM > 0 {
			rpmKey := fmt.Sprintf("rl:rpm:%s:%d", info.APIKeyID, time.Now().Unix()/60)
			count, _ := rdb.Incr(context.Background(), rpmKey).Result()
			if count == 1 {
				rdb.Expire(context.Background(), rpmKey, 61*time.Second)
			}
			if count > int64(limits.RPM) {
				retryAfter := 60 - (time.Now().Second())
				c.Header("Retry-After", strconv.Itoa(retryAfter))
				c.Header("X-RateLimit-Limit-Requests", strconv.Itoa(limits.RPM))
				c.Header("X-RateLimit-Remaining-Requests", "0")
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Rate limit exceeded: %d RPM for tier '%s'", limits.RPM, tier),
						"type":    "rate_limit_error",
					},
				})
				return
			}
			c.Header("X-RateLimit-Limit-Requests", strconv.Itoa(limits.RPM))
			c.Header("X-RateLimit-Remaining-Requests", strconv.FormatInt(int64(limits.RPM)-count, 10))
		}

		c.Next()
	}
}

func RecordTokenUsage(rdb *redis.Client, keyID, tier string, tokens int) {
	if tier == "flex" {
		return
	}
	limits, ok := tierConfig[tier]
	if !ok || limits.TPM <= 0 {
		return
	}
	tpmKey := fmt.Sprintf("rl:tpm:%s:%d", keyID, time.Now().Unix()/60)
	rdb.IncrBy(context.Background(), tpmKey, int64(tokens))
	rdb.Expire(context.Background(), tpmKey, 61*time.Second)
}
