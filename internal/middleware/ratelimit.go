package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// redisCmdable is the subset of redis commands used by the rate limiter.
// This allows mocking in tests.
type redisCmdable interface {
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// RateLimiter is a Redis-backed fixed-window rate limiter.
// It tracks request counts per IP address using a sliding window approach.
type RateLimiter struct {
	rdb    redisCmdable
	limit  int
	window time.Duration
	logger *slog.Logger
}

// NewRateLimiter creates a new RateLimiter.
// limit: maximum number of requests allowed per window.
// window: duration of the rate limit window.
func NewRateLimiter(rdb redisCmdable, limit int, window time.Duration, logger *slog.Logger) *RateLimiter {
	return &RateLimiter{
		rdb:    rdb,
		limit:  limit,
		window: window,
		logger: logger,
	}
}

// Limit returns a Gin middleware that enforces the rate limit.
// Rate limit is keyed by client IP address.
// On limit exceeded, returns 429 Too Many Requests with Retry-After header.
// If Redis is unavailable, the middleware fails open (allows the request).
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ip := c.ClientIP()

		// Calculate the current window start time
		now := time.Now()
		windowStart := now.Truncate(rl.window)
		windowKey := fmt.Sprintf("ratelimit:%s:%d", ip, windowStart.Unix())

		// Increment the counter for this window
		count, err := rl.rdb.Incr(ctx, windowKey).Result()
		if err != nil {
			// Fail open: if Redis is down, allow the request
			rl.logger.Error("rate limiter: redis error, failing open",
				slog.String("ip", ip),
				slog.Any("error", err),
			)
			c.Next()
			return
		}

		// Set expiration on first request in window
		if count == 1 {
			if err := rl.rdb.Expire(ctx, windowKey, rl.window).Err(); err != nil {
				rl.logger.Error("rate limiter: failed to set expiry",
					slog.String("key", windowKey),
					slog.Any("error", err),
				)
			}
		}

		// Check if limit exceeded
		if count > int64(rl.limit) {
			// Calculate seconds until window resets
			windowEnd := windowStart.Add(rl.window)
			retryAfter := int(math.Ceil(windowEnd.Sub(now).Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}

			rl.logger.Warn("rate limit exceeded",
				slog.String("ip", ip),
				slog.Int64("count", count),
				slog.Int("limit", rl.limit),
				slog.Int("retry_after", retryAfter),
			)

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
