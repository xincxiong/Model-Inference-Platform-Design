package middleware

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xincxiong/model-inference-platform/backend/internal/errors"
)

// RetryConfig holds retry parameters.
type RetryConfig struct {
	MaxRetries   int           // maximum number of retries
	InitialDelay time.Duration // initial backoff delay
	MaxDelay     time.Duration // maximum backoff delay
	RetryableStatus map[int]bool // HTTP status codes that trigger retry
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		RetryableStatus: map[int]bool{
			http.StatusServiceUnavailable: true,
			http.StatusGatewayTimeout:     true,
			http.StatusTooManyRequests:    true,
			http.StatusInternalServerError: true,
		},
	}
}

// RetryMiddleware returns a Gin middleware that retries requests on transient failures.
func RetryMiddleware(cfg RetryConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip retry for non-idempotent methods or if already retried
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" {
			// For POST/PUT/DELETE, only retry if the request hasn't been processed
			if c.GetHeader("X-Retry-Count") != "" {
				c.Next()
				return
			}
		}

		var lastErr error
		for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
			if attempt > 0 {
				// Calculate backoff with jitter
				delay := calculateBackoff(attempt, cfg.InitialDelay, cfg.MaxDelay)
				
				// Check if client disconnected
				if c.Request.Context().Err() != nil {
					return
				}
				
				time.Sleep(delay)
				
				// Set retry header
				c.Request.Header.Set("X-Retry-Count", string(rune('0'+attempt)))
			}

			// Create a response recorder to check status
			c.Next()
			
			// If no error or status is not retryable, return
			if len(c.Errors) == 0 && !cfg.RetryableStatus[c.Writer.Status()] {
				return
			}

			// Capture error for next attempt
			if len(c.Errors) > 0 {
				lastErr = c.Errors.Last()
			}
			
			// Reset response for retry
			c.Writer = &responseWriter{ResponseWriter: c.Writer}
		}

		// All retries exhausted
		if lastErr != nil {
			apiErr := errors.ErrInternalf("request failed after %d retries: %v", cfg.MaxRetries, lastErr)
			c.JSON(apiErr.HTTPStatus(), apiErr.ToResponse())
			c.Abort()
		}
	}
}

// calculateBackoff calculates exponential backoff with jitter.
func calculateBackoff(attempt int, initialDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff: initialDelay * 2^attempt
	backoff := float64(initialDelay) * math.Pow(2, float64(attempt))
	
	// Add jitter: random value between 0 and backoff/2
	jitter := rand.Float64() * backoff / 2
	delay := backoff + jitter
	
	// Cap at maxDelay
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}
	
	return time.Duration(delay)
}

// RetryRequest is a helper function to retry HTTP requests with context support.
func RetryRequest(ctx context.Context, fn func() (*http.Response, error), cfg RetryConfig) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateBackoff(attempt, cfg.InitialDelay, cfg.MaxDelay)
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := fn()
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = errors.ErrInternalf("server error: %d", resp.StatusCode)
			resp.Body.Close()
		}
	}

	return nil, lastErr
}

// responseWriter wraps gin.ResponseWriter to allow resetting.
type responseWriter struct {
	gin.ResponseWriter
}
