//go:build unit || integration

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	growthcrypto "github.com/Wei-Shaw/sub2api/internal/service/growth/crypto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tjfoc/gmsm/sm2"
)

type growthUserStub struct {
	user  *service.User
	users map[string]*service.User
	err   error
}

func (s growthUserStub) GetByEmail(_ context.Context, email string) (*service.User, error) {
	if s.users != nil {
		return s.users[email], s.err
	}
	return s.user, s.err
}

type growthNonceStub struct {
	mu   sync.Mutex
	seen map[string]bool
	err  error
}

func (s *growthNonceStub) Take(_ context.Context, nonce string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen == nil {
		s.seen = make(map[string]bool)
	}
	if s.seen[nonce] {
		return false, nil
	}
	s.seen[nonce] = true
	return true, nil
}

func TestGrowthPayloadValidEnvelopeAndAuthSubject(t *testing.T) {
	now := time.Unix(1763389980, 0)
	privateKey := mustGrowthTestPrivateKey(t)
	nonces := &growthNonceStub{}
	decryptor := newGrowthPayloadDecryptor(
		growthUserStub{user: &service.User{ID: 31, Role: service.RoleUser, Concurrency: 7}},
		nonces,
		5*time.Minute,
		privateKey,
		func() time.Time { return now },
	)
	body := sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "zhangsan@corp.com", now))

	engine := gin.New()
	engine.POST("/api/v1/growth/checkins", decryptor.Middleware(), func(c *gin.Context) {
		subject, ok := GetAuthSubjectFromContext(c)
		require.True(t, ok)
		require.Equal(t, int64(31), subject.UserID)
		require.Equal(t, 7, subject.Concurrency)
		payload, ok := GetGrowthPayloadFromContext(c)
		require.True(t, ok)
		require.Equal(t, "trace-growth", payload.RequestID)
		c.Status(http.StatusNoContent)
	})
	recorder := performGrowthRequest(engine, body)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestGrowthPayloadFailureMatrix(t *testing.T) {
	now := time.Unix(1763389980, 0)
	privateKey := mustGrowthTestPrivateKey(t)
	user := growthUserStub{user: &service.User{ID: 31, Role: service.RoleUser, Concurrency: 2}}

	tests := []struct {
		name       string
		body       func() []byte
		user       growthUserStub
		cache      func() *growthNonceStub
		now        func() time.Time
		wantStatus int
		wantCode   string
	}{
		{
			name: "unknown kid",
			body: func() []byte {
				return mutateGrowthEnvelope(t, sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "zhangsan@corp.com", now)), func(value map[string]any) {
					value["kid"] = "unknown"
				})
			},
			user: user, cache: func() *growthNonceStub { return &growthNonceStub{} }, now: func() time.Time { return now },
			wantStatus: http.StatusBadRequest, wantCode: growthcrypto.CodeKeyUnknown,
		},
		{
			name: "tampered mac",
			body: func() []byte {
				return mutateGrowthEnvelope(t, sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "zhangsan@corp.com", now)), func(value map[string]any) {
					value["mac"] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
				})
			},
			user: user, cache: func() *growthNonceStub { return &growthNonceStub{} }, now: func() time.Time { return now },
			wantStatus: http.StatusUnauthorized, wantCode: growthcrypto.CodeMACInvalid,
		},
		{
			name: "wrong private key",
			body: func() []byte {
				otherKey, err := growthcrypto.ParsePrivateKeyHex(strings.Repeat("0", 63) + "2")
				require.NoError(t, err)
				return sealGrowthTestRequest(t, otherKey, growthTestPayload("/growth/checkins", "zhangsan@corp.com", now))
			},
			user: user, cache: func() *growthNonceStub { return &growthNonceStub{} }, now: func() time.Time { return now },
			wantStatus: http.StatusUnauthorized, wantCode: growthcrypto.CodeKeyUnwrapFailed,
		},
		{
			name: "expired",
			body: func() []byte {
				return sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "zhangsan@corp.com", now.Add(-6*time.Minute)))
			},
			user: user, cache: func() *growthNonceStub { return &growthNonceStub{} }, now: func() time.Time { return now },
			wantStatus: http.StatusUnauthorized, wantCode: growthcrypto.CodeReplayRejected,
		},
		{
			name: "path mismatch",
			body: func() []byte {
				return sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/me", "zhangsan@corp.com", now))
			},
			user: user, cache: func() *growthNonceStub { return &growthNonceStub{} }, now: func() time.Time { return now },
			wantStatus: http.StatusBadRequest, wantCode: growthcrypto.CodePathMismatch,
		},
		{
			name: "payload invalid",
			body: func() []byte {
				payload := growthTestPayload("/growth/checkins", "zhangsan@corp.com", now)
				payload.Username = ""
				return sealGrowthTestRequest(t, privateKey, payload)
			},
			user: user, cache: func() *growthNonceStub { return &growthNonceStub{} }, now: func() time.Time { return now },
			wantStatus: http.StatusBadRequest, wantCode: growthcrypto.CodePayloadInvalid,
		},
		{
			name: "missing user",
			body: func() []byte {
				return sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "missing@corp.com", now))
			},
			user: growthUserStub{user: nil}, cache: func() *growthNonceStub { return &growthNonceStub{} }, now: func() time.Time { return now },
			wantStatus: http.StatusNotFound, wantCode: growthcrypto.CodeUserNotFound,
		},
		{
			name: "nonce unavailable",
			body: func() []byte {
				return sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "zhangsan@corp.com", now))
			},
			user: user, cache: func() *growthNonceStub { return &growthNonceStub{err: errors.New("redis down")} }, now: func() time.Time { return now },
			wantStatus: http.StatusServiceUnavailable, wantCode: "GROWTH_UNAVAILABLE",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decryptor := newGrowthPayloadDecryptor(test.user, test.cache(), 5*time.Minute, privateKey, test.now)
			engine := gin.New()
			engine.POST("/api/v1/growth/checkins", decryptor.Middleware(), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			recorder := performGrowthRequest(engine, test.body())
			require.Equal(t, test.wantStatus, recorder.Code)
			var response ErrorResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, test.wantCode, response.Code)
		})
	}
}

func TestGrowthPayloadReplayAndUserIsolation(t *testing.T) {
	now := time.Unix(1763389980, 0)
	privateKey := mustGrowthTestPrivateKey(t)
	nonces := &growthNonceStub{}
	users := growthUserStub{users: map[string]*service.User{
		"zhangsan@corp.com": {ID: 31, Role: service.RoleUser, Concurrency: 2},
		"lisi@corp.com":     {ID: 32, Role: service.RoleUser, Concurrency: 3},
	}}
	decryptor := newGrowthPayloadDecryptor(users, nonces, 5*time.Minute, privateKey, func() time.Time { return now })
	engine := gin.New()
	engine.POST("/api/v1/growth/checkins", decryptor.Middleware(), func(c *gin.Context) {
		subject, ok := GetAuthSubjectFromContext(c)
		require.True(t, ok)
		c.Header("X-Test-User", strconv.FormatInt(subject.UserID, 10))
		c.Status(http.StatusNoContent)
	})

	body := sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "zhangsan@corp.com", now))
	first := performGrowthRequest(engine, body)
	require.Equal(t, http.StatusNoContent, first.Code)
	require.Equal(t, "31", first.Header().Get("X-Test-User"))
	replay := performGrowthRequest(engine, body)
	require.Equal(t, http.StatusUnauthorized, replay.Code)

	secondBody := sealGrowthTestRequest(t, privateKey, growthTestPayload("/growth/checkins", "lisi@corp.com", now))
	second := performGrowthRequest(engine, secondBody)
	require.Equal(t, http.StatusNoContent, second.Code)
	require.Equal(t, "32", second.Header().Get("X-Test-User"))
}

func TestGrowthPayloadMalformedJSON(t *testing.T) {
	engine := gin.New()
	decryptor := newGrowthPayloadDecryptor(growthUserStub{}, &growthNonceStub{}, 5*time.Minute, mustGrowthTestPrivateKey(t), time.Now)
	engine.POST("/api/v1/growth/checkins", decryptor.Middleware(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := performGrowthRequest(engine, []byte(`{"kid":`))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestProductionGrowthKeyParsesAndDerivesPublicKey(t *testing.T) {
	require.NotNil(t, growthSM2PrivateKey)
	publicKey, err := growthcrypto.PublicKeyHex(growthSM2PrivateKey)
	require.NoError(t, err)
	require.Equal(t,
		"20b71146efd12e7e14ba9ce133743e0c9ac1cea1b689c3061637ce0238fbc933e4e69686a273866a1a023a985f7652865a776c05a7116015223e73b5a8a61b34",
		publicKey,
	)
	t.Logf("growth-sm2-v1 public key: %s", publicKey)
}

func growthTestPayload(path, username string, now time.Time) growthcrypto.Payload {
	return growthcrypto.Payload{
		Path:           path,
		Username:       username,
		Params:         json.RawMessage(`{"request_id":"trace-growth"}`),
		RequestID:      "trace-growth",
		IssuedAt:       now.Unix(),
		Nonce:          "nonce-" + username,
		IdempotencyKey: "idempotency-" + username,
	}
}

func mustGrowthTestPrivateKey(t *testing.T) *sm2.PrivateKey {
	t.Helper()
	key, err := growthcrypto.ParsePrivateKeyHex(strings.Repeat("0", 63) + "1")
	require.NoError(t, err)
	return key
}

func sealGrowthTestRequest(t *testing.T, privateKey *sm2.PrivateKey, payload growthcrypto.Payload) []byte {
	t.Helper()
	random := make([]byte, 64)
	for index := range random {
		random[index] = byte(index + 1)
	}
	envelope, err := growthcrypto.SealEnvelope(
		&privateKey.PublicKey,
		payload,
		[]byte("0123456789abcdef"),
		[]byte("fedcba9876543210"),
		bytes.NewReader(random),
	)
	require.NoError(t, err)
	raw, err := json.Marshal(envelope)
	require.NoError(t, err)
	return raw
}

func mutateGrowthEnvelope(t *testing.T, raw []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var value map[string]any
	require.NoError(t, json.Unmarshal(raw, &value))
	mutate(value)
	result, err := json.Marshal(value)
	require.NoError(t, err)
	return result
}

func performGrowthRequest(engine *gin.Engine, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/growth/checkins", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}
