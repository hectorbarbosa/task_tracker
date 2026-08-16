package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ─── Mock Redis ──────────────────────────────────────────────────────────────

// mockRedisCmd implements redisCmdable for unit tests.
type mockRedisCmd struct {
	counts    map[string]int64
	incrErr   error
	expireErr error
}

func newMockRedisCmd() *mockRedisCmd {
	return &mockRedisCmd{counts: make(map[string]int64)}
}

func (m *mockRedisCmd) Incr(_ context.Context, key string) *redis.IntCmd {
	if m.incrErr != nil {
		cmd := redis.NewIntCmd(context.Background())
		cmd.SetErr(m.incrErr)
		return cmd
	}
	m.counts[key]++
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(m.counts[key])
	return cmd
}

func (m *mockRedisCmd) Expire(_ context.Context, _ string, _ time.Duration) *redis.BoolCmd {
	if m.expireErr != nil {
		cmd := redis.NewBoolCmd(context.Background())
		cmd.SetErr(m.expireErr)
		return cmd
	}
	cmd := redis.NewBoolCmd(context.Background())
	cmd.SetVal(true)
	return cmd
}

// ─── Test helpers ────────────────────────────────────────────────────────────

func init() {
	gin.SetMode(gin.TestMode)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, nil))
}

type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// setupRouter creates a test router with the rate limiter middleware.
func setupRouter(rl *RateLimiter) *gin.Engine {
	router := gin.New()
	router.Use(rl.Limit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return router
}

// doRequest performs a GET request and returns the response.
func doRequest(router *gin.Engine, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestRateLimiter_UnderLimit(t *testing.T) {
	mock := newMockRedisCmd()
	rl := NewRateLimiter(mock, 5, time.Minute, testLogger())
	router := setupRouter(rl)

	for i := 0; i < 5; i++ {
		w := doRequest(router, "1.2.3.4")
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_OverLimit(t *testing.T) {
	mock := newMockRedisCmd()
	rl := NewRateLimiter(mock, 3, time.Minute, testLogger())
	router := setupRouter(rl)

	// First 3 should pass
	for i := 0; i < 3; i++ {
		w := doRequest(router, "1.2.3.4")
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 4th should be rate limited
	w := doRequest(router, "1.2.3.4")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	// Verify response body
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if body["error"] != "rate limit exceeded" {
		t.Errorf("expected error 'rate limit exceeded', got %q", body["error"])
	}

	// Verify Retry-After header
	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	mock := newMockRedisCmd()
	rl := NewRateLimiter(mock, 2, time.Minute, testLogger())
	router := setupRouter(rl)

	// IP 1: 2 requests (at limit)
	for i := 0; i < 2; i++ {
		w := doRequest(router, "1.1.1.1")
		if w.Code != http.StatusOK {
			t.Fatalf("IP1 request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// IP 1: 3rd request should be limited
	w := doRequest(router, "1.1.1.1")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("IP1 3rd request: expected 429, got %d", w.Code)
	}

	// IP 2: should still be allowed (different counter)
	for i := 0; i < 2; i++ {
		w = doRequest(router, "2.2.2.2")
		if w.Code != http.StatusOK {
			t.Fatalf("IP2 request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	mock := newMockRedisCmd()
	// Use a 1-second window; sleep past it to trigger a new window key.
	rl := NewRateLimiter(mock, 2, 1*time.Second, testLogger())
	router := setupRouter(rl)

	// First 2 requests pass
	for i := 0; i < 2; i++ {
		w := doRequest(router, "1.2.3.4")
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request within same window should be limited
	w := doRequest(router, "1.2.3.4")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("within window: expected 429, got %d", w.Code)
	}

	// Wait for window to expire (sleep past 1 second)
	time.Sleep(1100 * time.Millisecond)

	// After window reset, requests should pass again (new window key)
	w = doRequest(router, "1.2.3.4")
	if w.Code != http.StatusOK {
		t.Fatalf("after window reset: expected 200, got %d", w.Code)
	}
}

func TestRateLimiter_RedisDown_FailsOpen(t *testing.T) {
	mock := newMockRedisCmd()
	mock.incrErr = errors.New("redis connection refused")
	rl := NewRateLimiter(mock, 1, time.Minute, testLogger())
	router := setupRouter(rl)

	// Even with Redis down, requests should pass (fail-open)
	for i := 0; i < 10; i++ {
		w := doRequest(router, "1.2.3.4")
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200 (fail-open), got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_ExpireError_StillWorks(t *testing.T) {
	mock := newMockRedisCmd()
	mock.expireErr = errors.New("expire failed")
	rl := NewRateLimiter(mock, 3, time.Minute, testLogger())
	router := setupRouter(rl)

	// Expire error should be logged but not affect rate limiting
	for i := 0; i < 3; i++ {
		w := doRequest(router, "1.2.3.4")
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	w := doRequest(router, "1.2.3.4")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestRateLimiter_KeyFormat(t *testing.T) {
	mock := newMockRedisCmd()
	rl := NewRateLimiter(mock, 10, time.Minute, testLogger())
	router := setupRouter(rl)

	doRequest(router, "192.168.1.100")

	// Verify the key was created with the expected format
	if len(mock.counts) != 1 {
		t.Fatalf("expected 1 key, got %d", len(mock.counts))
	}

	for key := range mock.counts {
		// Key should have format: ratelimit:{ip}:{timestamp}
		// Just verify it starts with the expected prefix
		expected := "ratelimit:192.168.1.100:"
		if len(key) <= len(expected) {
			t.Errorf("key too short: %q", key)
		}
		if key[:len(expected)] != expected {
			t.Errorf("expected key to start with %q, got %q", expected, key)
		}
	}
}
