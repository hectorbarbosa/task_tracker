package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// CtxRequestID is the key for request ID in gin.Context
	CtxRequestID contextKey = "request_id"
)

// RequestID generates a unique request ID and adds it to the context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from header or generate new one
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in context
		c.Set(string(CtxRequestID), requestID)

		// Set response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// RequestLogger returns a middleware that logs HTTP requests with structured logging.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get user ID if authenticated
		userID, _ := c.Get(string(CtxUserID))

		// Build structured log entry with essential fields only
		fields := []any{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", latency),
		}

		// Add user ID if authenticated
		if userID != nil {
			fields = append(fields, slog.Any("user_id", userID))
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields = append(fields, slog.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		// Log based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.Error("server error", fields...)
		case status >= 400:
			logger.Warn("client error", fields...)
		case status >= 300:
			logger.Info("redirect", fields...)
		default:
			logger.Info("request", fields...)
		}
	}
}
