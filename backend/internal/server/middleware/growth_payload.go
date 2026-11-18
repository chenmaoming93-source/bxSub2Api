package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	growthservice "github.com/Wei-Shaw/sub2api/internal/service/growth"
	growthcrypto "github.com/Wei-Shaw/sub2api/internal/service/growth/crypto"

	"github.com/gin-gonic/gin"
	"github.com/tjfoc/gmsm/sm2"
)

const (
	// growthSM2PrivateKeyHex is the fixed server-only key assigned with
	// growth-sm2-v1. It is intentionally not loaded from config, environment,
	// KMS, or an API key store. A key change requires a coordinated code release
	// with AIWork Main.
	growthSM2PrivateKeyHex = "be0a68d8fd7fcf76f2ee116216ecb3034c0d16916fca0d9d4b5601ac3ea81e9b"
)

var growthSM2PrivateKey = mustGrowthSM2PrivateKey()

type growthUserLookup interface {
	GetByEmail(ctx context.Context, email string) (*service.User, error)
}

type growthNonceCache interface {
	Take(ctx context.Context, nonce string) (bool, error)
}

type GrowthPayloadDecryptor struct {
	handler gin.HandlerFunc
}

func (d GrowthPayloadDecryptor) Middleware() gin.HandlerFunc {
	return d.handler
}

func NewGrowthPayloadDecryptor(users service.UserRepository, nonces *growthservice.RedisNonceCache, cfg *config.Config) GrowthPayloadDecryptor {
	window := 5 * time.Minute
	if cfg != nil {
		window = cfg.Growth.Crypto.WindowDuration()
	}
	return newGrowthPayloadDecryptor(users, nonces, window, growthSM2PrivateKey, time.Now)
}

func newGrowthPayloadDecryptor(
	users growthUserLookup,
	nonces growthNonceCache,
	window time.Duration,
	privateKey *sm2.PrivateKey,
	now func() time.Time,
) GrowthPayloadDecryptor {
	return GrowthPayloadDecryptor{handler: func(c *gin.Context) {
		var envelope growthcrypto.Envelope
		if err := c.ShouldBindJSON(&envelope); err != nil {
			abortGrowthCrypto(c, growthcrypto.ErrEnvelopeInvalid)
			return
		}
		var nonceErr error
		payload, err := growthcrypto.DecodeEnvelope(envelope, growthcrypto.DecodeOptions{
			PrivateKey: privateKey,
			Now:        now(),
			Window:     window,
			WantPath:   growthRouteTemplate(c.FullPath()),
			TakeNonce: func(nonce string) bool {
				accepted, cacheErr := nonces.Take(c.Request.Context(), nonce)
				if cacheErr != nil {
					nonceErr = cacheErr
					return false
				}
				return accepted
			},
		})
		if nonceErr != nil {
			AbortWithError(c, http.StatusServiceUnavailable, "GROWTH_UNAVAILABLE", "growth nonce service unavailable")
			return
		}
		if err != nil {
			abortGrowthCrypto(c, err)
			return
		}

		user, err := users.GetByEmail(c.Request.Context(), strings.ToLower(strings.TrimSpace(payload.Username)))
		if err != nil || user == nil {
			AbortWithError(c, http.StatusNotFound, growthcrypto.CodeUserNotFound, "growth account is not provisioned")
			return
		}
		c.Set(string(ContextKeyGrowthPayload), payload)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      user.ID,
			Concurrency: user.Concurrency,
			SuperAdmin:  user.IsSuperAdmin(),
		})
		c.Set(string(ContextKeyUserRole), user.Role)
		c.Next()
	}}
}

func GetGrowthPayloadFromContext(c *gin.Context) (*growthcrypto.Payload, bool) {
	value, exists := c.Get(string(ContextKeyGrowthPayload))
	if !exists {
		return nil, false
	}
	payload, ok := value.(*growthcrypto.Payload)
	return payload, ok
}

func abortGrowthCrypto(c *gin.Context, err error) {
	status := http.StatusBadRequest
	code := growthcrypto.CodeEnvelopeInvalid
	var contractErr *growthcrypto.ContractError
	if errors.As(err, &contractErr) && contractErr.Code != "" {
		code = contractErr.Code
	}
	switch code {
	case growthcrypto.CodeKeyUnwrapFailed, growthcrypto.CodeMACInvalid,
		growthcrypto.CodeDecryptFailed, growthcrypto.CodeReplayRejected:
		status = http.StatusUnauthorized
	}
	AbortWithError(c, status, code, growthCryptoMessage(code))
	slog.Warn("growth envelope rejected",
		"code", code,
		"request_id_header", strings.TrimSpace(c.GetHeader("X-Request-Id")),
		"client_ip", c.ClientIP(),
		"path", c.FullPath(),
	)
}

func growthCryptoMessage(code string) string {
	switch code {
	case growthcrypto.CodeKeyUnknown:
		return "growth envelope key is unknown"
	case growthcrypto.CodeKeyUnwrapFailed:
		return "growth envelope key unwrap failed"
	case growthcrypto.CodeMACInvalid:
		return "growth envelope integrity check failed"
	case growthcrypto.CodeDecryptFailed:
		return "growth envelope decryption failed"
	case growthcrypto.CodeReplayRejected:
		return "growth envelope replay rejected"
	case growthcrypto.CodePayloadInvalid:
		return "growth payload is invalid"
	case growthcrypto.CodePathMismatch:
		return "growth payload path mismatch"
	default:
		return "growth envelope is invalid"
	}
}

func growthRouteTemplate(path string) string {
	return strings.TrimPrefix(path, "/api/v1")
}

func mustGrowthSM2PrivateKey() *sm2.PrivateKey {
	key, err := growthcrypto.ParsePrivateKeyHex(growthSM2PrivateKeyHex)
	if err != nil {
		panic(err)
	}
	return key
}
