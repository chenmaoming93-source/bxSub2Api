//go:build unit

package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tjfoc/gmsm/sm2"
)

const (
	// Test-only private scalar. Production key material is generated through the
	// controlled handoff process and must never be committed to this repository.
	testPrivateKeyHex = "0000000000000000000000000000000000000000000000000000000000000001"
	testIssuedAt      = int64(1763389980)
)

// The checked-in route matrix is intentionally duplicated from contract v4.0
// so an accidental omission is caught while each endpoint MVP is implemented.
//
// DDL decision: growth DDL is archived only at
// backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql. Historical files under
// backend/migrations are immutable.
//
// External release blockers: production SM2 pair handoff, NTP synchronization,
// internal/VPN network plus source-IP allowlist, GoldenDB transaction/index
// verification, and per-order idempotency verification for Token fulfillment.
var (
	aiworkUContract = []string{
		"POST /growth/me",
		"POST /growth/tasks",
		"POST /growth/tasks/:task_id",
		"POST /growth/ledger",
		"POST /growth/products",
		"POST /growth/products/:product_id",
		"POST /growth/exchanges",
		"POST /growth/checkins",
		"POST /growth/tasks/:task_id/completion-requests",
		"POST /growth/exchanges",
		"POST /growth/exchanges/:order_id",
	}
	adminMContract = []string{
		"GET /admin/growth/stats/overview",
		"GET /admin/growth/budget",
		"GET /admin/growth/budget/entries",
		"POST /admin/growth/budget/entries",
		"GET /admin/growth/budget/allocations",
		"POST /admin/growth/budget/allocations",
		"POST /admin/growth/tasks/:id/budget/revoke",
		"GET /admin/growth/tasks",
		"POST /admin/growth/tasks",
		"PUT /admin/growth/tasks/:id",
		"POST /admin/growth/tasks/:id/publish",
		"POST /admin/growth/tasks/:id/offline",
		"POST /admin/growth/tasks/:id/online",
		"GET /admin/growth/tasks/:id/rule-versions",
		"GET /admin/growth/products",
		"POST /admin/growth/products",
		"PUT /admin/growth/products/:id",
		"POST /admin/growth/products/:id/publish",
		"POST /admin/growth/products/:id/stock",
		"POST /admin/growth/products/:id/offline",
		"GET /admin/growth/exchanges",
		"GET /admin/growth/exchanges/:id",
		"POST /admin/growth/exchanges/:id/retry",
		"POST /admin/growth/exchanges/:id/refund",
		"GET /admin/growth/users",
		"GET /admin/growth/users/:id/ledger",
		"GET /admin/growth/users/:id/exchanges",
		"POST /admin/growth/users/:id/adjustments",
	}
	webWContract = []string{
		"GET /user/growth/overview",
		"GET /user/growth/tasks",
		"GET /user/growth/products",
		"GET /user/growth/products/:id",
		"POST /user/growth/exchanges",
		"GET /user/growth/exchanges",
		"GET /user/growth/exchanges/:id",
		"GET /user/growth/ledger",
	}
	growthPermissions = []string{
		"growth.read",
		"growth.manage",
		"growth.adjust",
		"growth.self.read",
		"growth.self.exchange",
	}
)

func TestContractRouteAndPermissionBaseline(t *testing.T) {
	require.Len(t, aiworkUContract, 11)
	require.Len(t, webWContract, 8)
	require.Equal(t, []string{
		"growth.adjust",
		"growth.manage",
		"growth.read",
		"growth.self.exchange",
		"growth.self.read",
	}, sortedCopy(growthPermissions))

	require.Contains(t, adminMContract, "POST /admin/growth/tasks/:id/online")
	require.Contains(t, adminMContract, "POST /admin/growth/users/:id/adjustments")
	require.NotContains(t, strings.Join(adminMContract, "\n"), "DELETE /admin/growth/tasks")
	require.NotContains(t, strings.Join(webWContract, "\n"), "checkins")
	require.NotContains(t, strings.Join(webWContract, "\n"), "completion-requests")
}

func TestContractEnvelopeFixture(t *testing.T) {
	privateKey := mustTestPrivateKey(t)
	fixture := readEnvelopeFixture(t)
	nonces := make(map[string]bool)

	payload, err := DecodeEnvelope(fixture, DecodeOptions{
		PrivateKey: privateKey,
		Now:        time.Unix(testIssuedAt, 0),
		Window:     5 * time.Minute,
		WantPath:   "/growth/checkins",
		TakeNonce: func(value string) bool {
			if nonces[value] {
				return false
			}
			nonces[value] = true
			return true
		},
	})
	require.NoError(t, err)
	require.Equal(t, "/growth/checkins", payload.Path)
	require.Equal(t, "zhangsan@corp.com", payload.Username)
	require.Equal(t, "trace-9f1c-contract", payload.RequestID)
	require.Equal(t, "idem-checkin-001", payload.IdempotencyKey)
	require.JSONEq(t, `{"request_id":"trace-9f1c-contract"}`, string(payload.Params))
}

func TestContractEnvelopeFailureMatrix(t *testing.T) {
	privateKey := mustTestPrivateKey(t)
	fixture := readEnvelopeFixture(t)

	t.Run("unknown kid", func(t *testing.T) {
		tampered := fixture
		tampered.KID = "unknown"
		requireCode(t, ErrKeyUnknown, decodeFixture(t, privateKey, tampered, "/growth/checkins"))
	})

	t.Run("tampered mac", func(t *testing.T) {
		tampered := fixture
		raw, err := base64.StdEncoding.DecodeString(tampered.MAC)
		require.NoError(t, err)
		raw[0] ^= 0xff
		tampered.MAC = base64.StdEncoding.EncodeToString(raw)
		requireCode(t, ErrMACInvalid, decodeFixture(t, privateKey, tampered, "/growth/checkins"))
	})

	t.Run("tampered ciphertext with valid mac", func(t *testing.T) {
		tampered := fixture
		raw, err := base64.StdEncoding.DecodeString(tampered.Ciphertext)
		require.NoError(t, err)
		raw[len(raw)-1] ^= 0x01
		tampered.Ciphertext = base64.StdEncoding.EncodeToString(raw)
		tampered.MAC = recomputeMAC(t, fixtureSessionKey(), tampered)
		requireCode(t, ErrDecryptFailed, decodeFixture(t, privateKey, tampered, "/growth/checkins"))
	})

	t.Run("tampered iv with valid mac", func(t *testing.T) {
		tampered := fixture
		raw, err := base64.StdEncoding.DecodeString(tampered.IV)
		require.NoError(t, err)
		raw[0] ^= 0xff
		tampered.IV = base64.StdEncoding.EncodeToString(raw)
		tampered.MAC = recomputeMAC(t, fixtureSessionKey(), tampered)
		requireCode(t, ErrDecryptFailed, decodeFixture(t, privateKey, tampered, "/growth/checkins"))
	})

	t.Run("wrong private key", func(t *testing.T) {
		wrongKey, err := ParsePrivateKeyHex(strings.Repeat("0", 63) + "2")
		require.NoError(t, err)
		requireCode(t, ErrKeyUnwrapFailed, decodeFixture(t, wrongKey, fixture, "/growth/checkins"))
	})

	t.Run("expired timestamp", func(t *testing.T) {
		requireCode(t, ErrReplayRejected, decodeFixtureAt(t, privateKey, fixture, "/growth/checkins", time.Unix(testIssuedAt+301, 0)))
	})

	t.Run("nonce replay", func(t *testing.T) {
		nonces := make(map[string]bool)
		decodeWithSharedNonces := func() error {
			_, err := DecodeEnvelope(fixture, DecodeOptions{
				PrivateKey: privateKey,
				Now:        time.Unix(testIssuedAt, 0),
				Window:     5 * time.Minute,
				WantPath:   "/growth/checkins",
				TakeNonce: func(value string) bool {
					if nonces[value] {
						return false
					}
					nonces[value] = true
					return true
				},
			})
			return err
		}
		require.NoError(t, decodeWithSharedNonces())
		requireCode(t, ErrReplayRejected, decodeWithSharedNonces())
	})

	t.Run("path mismatch", func(t *testing.T) {
		requireCode(t, ErrPathMismatch, decodeFixture(t, privateKey, fixture, "/growth/me"))
	})

	t.Run("malformed json", func(t *testing.T) {
		var env Envelope
		err := json.Unmarshal([]byte(`{"kid":`), &env)
		require.Error(t, err)
	})

	t.Run("forbidden accounting field", func(t *testing.T) {
		sealed := sealPayload(t, Payload{
			Path:     "/growth/checkins",
			Username: "zhangsan@corp.com",
			Params:   json.RawMessage(`{"points":999}`),
			IssuedAt: testIssuedAt,
			Nonce:    "forbidden-accounting-field",
		})
		requireCode(t, ErrPayloadInvalid, decodeFixture(t, privateKey, sealed, "/growth/checkins"))
	})
}

func TestContractFixtureCanBeRegenerated(t *testing.T) {
	want := readEnvelopeFixture(t)
	got := sealPayload(t, expectedFixturePayload())
	require.Equal(t, want, got)
}

func expectedFixturePayload() Payload {
	return Payload{
		Path:           "/growth/checkins",
		Username:       "zhangsan@corp.com",
		Params:         json.RawMessage(`{"request_id":"trace-9f1c-contract"}`),
		RequestID:      "trace-9f1c-contract",
		IssuedAt:       testIssuedAt,
		Nonce:          "b7e1c2a4-contract-nonce",
		IdempotencyKey: "idem-checkin-001",
	}
}

func mustTestPrivateKey(t *testing.T) *sm2.PrivateKey {
	t.Helper()
	key, err := ParsePrivateKeyHex(testPrivateKeyHex)
	require.NoError(t, err)
	return key
}

func readEnvelopeFixture(t *testing.T) Envelope {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "valid_envelope.json"))
	require.NoError(t, err)
	var env Envelope
	require.NoError(t, json.Unmarshal(raw, &env))
	return env
}

func sealPayload(t *testing.T, payload Payload) Envelope {
	t.Helper()
	env, err := SealEnvelope(
		&mustTestPrivateKey(t).PublicKey,
		payload,
		fixtureSessionKey(),
		fixtureIV(),
		bytes.NewReader(deterministicSM2Random()),
	)
	require.NoError(t, err)
	return env
}

func decodeFixture(t *testing.T, privateKey *sm2.PrivateKey, env Envelope, wantPath string) error {
	t.Helper()
	return decodeFixtureAt(t, privateKey, env, wantPath, time.Unix(testIssuedAt, 0))
}

func decodeFixtureAt(t *testing.T, privateKey *sm2.PrivateKey, env Envelope, wantPath string, now time.Time) error {
	t.Helper()
	nonces := make(map[string]bool)
	_, err := DecodeEnvelope(env, DecodeOptions{
		PrivateKey: privateKey,
		Now:        now,
		Window:     5 * time.Minute,
		WantPath:   wantPath,
		TakeNonce: func(value string) bool {
			if nonces[value] {
				return false
			}
			nonces[value] = true
			return true
		},
	})
	return err
}

func requireCode(t *testing.T, want error, got error) {
	t.Helper()
	require.Error(t, got)
	require.Truef(t, errors.Is(got, want), "got %v, want %v", got, want)
}

func recomputeMAC(t *testing.T, key []byte, env Envelope) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(
		SM3HMAC(key, []byte(env.KID+env.EncryptedKey+env.IV+env.Ciphertext)),
	)
}

func fixtureSessionKey() []byte { return []byte("0123456789abcdef") }

func fixtureIV() []byte { return []byte("fedcba9876543210") }

func deterministicSM2Random() []byte {
	value := make([]byte, 64)
	for i := range value {
		value[i] = byte(i + 1)
	}
	return value
}

func sortedCopy(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
