package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestProvisioningRateLimiter_AllowBiz(t *testing.T) {
	limiter := NewProvisioningRateLimiter(time.Second, time.Second, 5, 10)

	// All 10 should be allowed.
	for i := 0; i < 10; i++ {
		if !limiter.AllowBiz("10.0.0.1") {
			t.Fatalf("biz call %d should have been allowed", i)
		}
	}
	// 11th should be denied.
	if limiter.AllowBiz("10.0.0.1") {
		t.Fatal("biz call 11 should have been denied")
	}

	// Different IP still allowed.
	if !limiter.AllowBiz("10.0.0.2") {
		t.Fatal("different IP should be allowed")
	}
}

func TestProvisioningRateLimiter_AllowAuth(t *testing.T) {
	limiter := NewProvisioningRateLimiter(time.Second, time.Second, 3, 10)

	for i := 0; i < 3; i++ {
		if !limiter.AllowAuth("10.0.0.1") {
			t.Fatalf("auth failure %d should have been allowed", i)
		}
	}
	if limiter.AllowAuth("10.0.0.1") {
		t.Fatal("auth failure 4 should have been denied")
	}
}

func TestProvisioningRateLimiter_Concurrent(t *testing.T) {
	limiter := NewProvisioningRateLimiter(time.Second, time.Second, 10, 100)
	var wg sync.WaitGroup
	allowed := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- limiter.AllowBiz("10.0.0.1")
		}()
	}
	wg.Wait()
	close(allowed)

	count := 0
	for a := range allowed {
		if a {
			count++
		}
	}
	if count != 100 {
		t.Fatalf("expected 100 allowed, got %d", count)
	}
}

func TestProvisioningRateLimiter_Refill(t *testing.T) {
	limiter := NewProvisioningRateLimiter(time.Second, 100*time.Millisecond, 100, 5)

	// Use all tokens.
	for i := 0; i < 5; i++ {
		if !limiter.AllowBiz("10.0.0.1") {
			t.Fatalf("biz call %d should have been allowed", i)
		}
	}
	if limiter.AllowBiz("10.0.0.1") {
		t.Fatal("should be denied")
	}

	// Wait for refill.
	time.Sleep(250 * time.Millisecond)

	// Should have at least 1 token.
	if !limiter.AllowBiz("10.0.0.1") {
		t.Fatal("should be allowed after refill")
	}
}

func TestHardeningMiddleware_ContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	hardening := NewProvisioningHardening(nil, nil)
	r.POST("/ensure", hardening.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Missing Content-Type.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ensure", strings.NewReader("{}"))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", w.Code)
	}

	// Correct Content-Type.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/ensure", strings.NewReader(`{"user_email":"test@test.com","platform":"openai"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHardeningMiddleware_BodySizeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	hardening := NewProvisioningHardening(nil, nil)
	r.POST("/ensure", hardening.Middleware(), func(c *gin.Context) {
		// ShouldBindJSON should fail with truncated body.
		var v map[string]any
		if err := c.ShouldBindJSON(&v); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})

	// Body larger than 4KB.
	largeBody := `{"user_email":"test@test.com","platform":"openai","padding":"` + strings.Repeat("x", MaxProvisioningBodySize+100) + `"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ensure", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", w.Code)
	}
}

func TestHardeningMiddleware_RateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	limiter := NewProvisioningRateLimiter(time.Minute, 100*time.Millisecond, 100, 3)
	hardening := NewProvisioningHardening(limiter, nil)
	r.POST("/ensure", hardening.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/ensure", strings.NewReader(`{"user_email":"t@t.com","platform":"o"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200, got %d", i, w.Code)
		}
	}

	// 4th should be rate limited.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ensure", strings.NewReader(`{"user_email":"t@t.com","platform":"o"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestAuditLogger(t *testing.T) {
	logger := NewProvisioningAuditLogger()

	// Just verify no panics. slog output goes to default logger.
	logger.LogSuccess(1, "openai", "10.0.0.1", true, true)
	logger.LogFailure("openai", "10.0.0.1", "user_not_found")
}
