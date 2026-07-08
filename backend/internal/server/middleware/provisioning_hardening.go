package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// --- rate limiter ---

// ProvisioningRateLimiter is a simple in-memory rate limiter for the external
// provisioning endpoint.
type ProvisioningRateLimiter struct {
	mu          sync.Mutex
	authWindow  time.Duration
	bizWindow   time.Duration
	authLimit   int
	bizLimit    int
	authBuckets map[string]*tokenBucket
	bizBuckets  map[string]*tokenBucket
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

// NewProvisioningRateLimiter creates a limiter.
func NewProvisioningRateLimiter(authWindow, bizWindow time.Duration, authLimit, bizLimit int) *ProvisioningRateLimiter {
	return &ProvisioningRateLimiter{
		authWindow: authWindow, bizWindow: bizWindow,
		authLimit: authLimit, bizLimit: bizLimit,
		authBuckets: make(map[string]*tokenBucket),
		bizBuckets:  make(map[string]*tokenBucket),
	}
}

// DefaultProvisioningRateLimiter returns a limiter with defaults:
// auth failures: 10/min per IP; business calls: 60/min per IP.
func DefaultProvisioningRateLimiter() *ProvisioningRateLimiter {
	l := NewProvisioningRateLimiter(time.Minute, time.Minute, 10, 60)
	l.StartCleanup(5 * time.Minute)
	return l
}

// AllowAuth checks whether an auth-failure is allowed for this IP.
func (l *ProvisioningRateLimiter) AllowAuth(ip string) bool {
	return l.allow(&l.authBuckets, ip, l.authWindow, l.authLimit)
}

// AllowBiz checks whether a business call is allowed for this IP.
func (l *ProvisioningRateLimiter) AllowBiz(ip string) bool {
	return l.allow(&l.bizBuckets, ip, l.bizWindow, l.bizLimit)
}

func (l *ProvisioningRateLimiter) allow(buckets *map[string]*tokenBucket, ip string, window time.Duration, limit int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := (*buckets)[ip]
	if !ok {
		b = &tokenBucket{tokens: float64(limit) - 1, lastTime: now}
		(*buckets)[ip] = b
		return true
	}

	elapsed := now.Sub(b.lastTime)
	b.tokens += elapsed.Seconds() * (float64(limit) / window.Seconds())
	if b.tokens > float64(limit) {
		b.tokens = float64(limit)
	}
	b.lastTime = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// StartCleanup begins a background goroutine that periodically removes stale buckets.
func (l *ProvisioningRateLimiter) StartCleanup(interval time.Duration) {
	go func() {
		for range time.NewTicker(interval).C {
			l.mu.Lock()
			cutoff := time.Now().Add(-l.authWindow * 2)
			for ip, b := range l.authBuckets {
				if b.lastTime.Before(cutoff) {
					delete(l.authBuckets, ip)
				}
			}
			cutoff = time.Now().Add(-l.bizWindow * 2)
			for ip, b := range l.bizBuckets {
				if b.lastTime.Before(cutoff) {
					delete(l.bizBuckets, ip)
				}
			}
			l.mu.Unlock()
		}
	}()
}

// --- audit logger ---

// ProvisioningAuditLogger records audit events (no secrets).
type ProvisioningAuditLogger struct{}

// NewProvisioningAuditLogger creates an audit logger.
func NewProvisioningAuditLogger() *ProvisioningAuditLogger {
	return &ProvisioningAuditLogger{}
}

// LogSuccess logs a successful provisioning call.
func (a *ProvisioningAuditLogger) LogSuccess(userID int64, platform, sourceIP string, userCreated, keyCreated bool) {
	slog.Info("provisioning_api_key_ensure",
		slog.String("event", "provisioning.api_key_ensure"),
		slog.Int64("user_id", userID),
		slog.String("platform", platform),
		slog.String("source_ip", sourceIP),
		slog.Bool("user_created", userCreated),
		slog.Bool("key_created", keyCreated),
		slog.String("result", "success"),
	)
}

// LogFailure logs a failed provisioning call.
func (a *ProvisioningAuditLogger) LogFailure(platform, sourceIP, reason string) {
	slog.Warn("provisioning_api_key_ensure",
		slog.String("event", "provisioning.api_key_ensure"),
		slog.String("platform", platform),
		slog.String("source_ip", sourceIP),
		slog.String("result", "failure"),
		slog.String("reason", reason),
	)
}

// --- hardening middleware ---

const MaxProvisioningBodySize = 4 * 1024 // 4KB

// ProvisioningHardening bundles body size, Content-Type, rate limiting, and audit logging.
type ProvisioningHardening struct {
	limiter     *ProvisioningRateLimiter
	auditLogger *ProvisioningAuditLogger
}

// NewProvisioningHardening creates the hardening wrapper.
func NewProvisioningHardening(limiter *ProvisioningRateLimiter, auditLogger *ProvisioningAuditLogger) *ProvisioningHardening {
	// nil limiter = no rate limiting
	if auditLogger == nil {
		auditLogger = NewProvisioningAuditLogger()
	}
	return &ProvisioningHardening{limiter: limiter, auditLogger: auditLogger}
}

// Middleware returns a gin middleware enforcing Content-Type, body size, and rate limit.
func (h *ProvisioningHardening) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content-Type enforcement.
		ct := c.GetHeader("Content-Type")
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(ct)), "application/json") {
			AbortWithError(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
			return
		}

		// Body size limit.
		c.Request.Body = io.NopCloser(io.LimitReader(c.Request.Body, MaxProvisioningBodySize))

		// Business-call rate limit (skipped when limiter is nil).
		if h.limiter != nil {
			ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}
		if !h.limiter.AllowBiz(ip) {
			slog.Warn("provisioning rate limit exceeded",
				slog.String("event", "provisioning.rate_limit"),
				slog.String("source_ip", ip),
			)
			AbortWithError(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests, please try again later")
				return
			}
		}

		c.Next()
	}
}

// AuditSuccess logs success.
func (h *ProvisioningHardening) AuditSuccess(c *gin.Context, userID int64, platform string, userCreated, keyCreated bool) {
	h.auditLogger.LogSuccess(userID, platform, clientIP(c), userCreated, keyCreated)
}

// AuditFailure logs failure.
func (h *ProvisioningHardening) AuditFailure(c *gin.Context, platform, reason string) {
	h.auditLogger.LogFailure(platform, clientIP(c), reason)
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		return "unknown"
	}
	return ip
}

// generateID creates a random hex ID.
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
