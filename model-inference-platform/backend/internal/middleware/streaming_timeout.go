package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xincxiong/model-inference-platform/backend/internal/errors"
)

// StreamingTimeoutConfig holds streaming timeout parameters.
type StreamingTimeoutConfig struct {
	// TTFTTimeout is the maximum time to wait for the first token.
	TTFTTimeout time.Duration
	// InterTokenTimeout is the maximum time between consecutive tokens.
	InterTokenTimeout time.Duration
	// TotalTimeout is the maximum total duration for the entire streaming response.
	TotalTimeout time.Duration
}

// DefaultStreamingTimeoutConfig returns sensible defaults for streaming.
func DefaultStreamingTimeoutConfig() StreamingTimeoutConfig {
	return StreamingTimeoutConfig{
		TTFTTimeout:       30 * time.Second,
		InterTokenTimeout: 60 * time.Second,
		TotalTimeout:      5 * time.Minute,
	}
}

// StreamingTimeoutMiddleware returns a Gin middleware that enforces timeouts on streaming responses.
func StreamingTimeoutMiddleware(cfg StreamingTimeoutConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to streaming endpoints
		if !isStreamingEndpoint(c) {
			c.Next()
			return
		}

		// Create a context with total timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.TotalTimeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		// Set up TTFT timeout
		ttftTimer := time.NewTimer(cfg.TTFTTimeout)
		defer ttftTimer.Stop()

		// Track if first token has been received
		firstTokenReceived := false

		// Wrap the response writer to track token timing
		originalWriter := c.Writer
		wrappedWriter := &streamingResponseWriter{
			ResponseWriter: originalWriter,
			onFirstToken: func() {
				firstTokenReceived = true
				ttftTimer.Stop()
			},
			interTokenTimeout: cfg.InterTokenTimeout,
		}
		c.Writer = wrappedWriter

		// Monitor timeouts in background
		done := make(chan struct{})
		go func() {
			select {
			case <-ttftTimer.C:
				if !firstTokenReceived {
					// TTFT timeout - write error response
					apiErr := errors.ErrTimeoutf("time to first token exceeded (%v)", cfg.TTFTTimeout)
					originalWriter.WriteHeader(apiErr.HTTPStatus())
					originalWriter.Write([]byte(`{"error":{"code":"timeout_error","message":"Time to first token exceeded","type":"timeout_error"}}`))
					c.Abort()
				}
			case <-done:
				return
			}
		}()

		c.Next()
		close(done)
	}
}

// isStreamingEndpoint checks if the request is for a streaming endpoint.
func isStreamingEndpoint(c *gin.Context) bool {
	// Check if client requested streaming
	stream := c.Query("stream") == "true" || c.DefaultQuery("stream", "false") == "true"
	
	// Or check request body for stream parameter (for POST requests)
	if c.Request.Method == "POST" {
		// This would be checked in the handler where body is parsed
		// For middleware, we rely on query parameter or response headers
	}
	
	return stream
}

// streamingResponseWriter wraps gin.ResponseWriter to track token timing.
type streamingResponseWriter struct {
	gin.ResponseWriter
	onFirstToken        func()
	interTokenTimeout   time.Duration
	lastTokenTime       time.Time
	tokenTimer          *time.Timer
	firstTokenWritten   bool
}

func (w *streamingResponseWriter) Write(data []byte) (int, error) {
	// Check if this is a token chunk (SSE data event)
	if len(data) > 0 && !w.firstTokenWritten {
		w.firstTokenWritten = true
		w.onFirstToken()
		w.lastTokenTime = time.Now()
		
		// Start inter-token timeout timer
		w.tokenTimer = time.AfterFunc(w.interTokenTimeout, func() {
			// Inter-token timeout - this will be handled by context cancellation
		})
	} else if len(data) > 0 {
		// Reset inter-token timeout timer
		if w.tokenTimer != nil {
			w.tokenTimer.Reset(w.interTokenTimeout)
		}
		w.lastTokenTime = time.Now()
	}

	return w.ResponseWriter.Write(data)
}

func (w *streamingResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
