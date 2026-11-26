package crypto

import (
	"crypto/cipher"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm3"
	"github.com/tjfoc/gmsm/sm4"
)

const (
	GrowthKID         = "growth-sm2-v1"
	SM4KeySize        = 16
	SM4IVSize         = 16
	SMMACSize         = 32
	DefaultTimeWindow = 5 * time.Minute
)

const (
	CodeEnvelopeInvalid = "CRYPTO_ENVELOPE_INVALID"
	CodeKeyUnknown      = "CRYPTO_KEY_UNKNOWN"
	CodePayloadInvalid  = "CRYPTO_PAYLOAD_INVALID"
	CodePathMismatch    = "CRYPTO_PATH_MISMATCH"
	CodeKeyUnwrapFailed = "CRYPTO_KEY_UNWRAP_FAILED"
	CodeMACInvalid      = "CRYPTO_MAC_INVALID"
	CodeDecryptFailed   = "CRYPTO_DECRYPT_FAILED"
	CodeReplayRejected  = "CRYPTO_REPLAY_REJECTED"
	CodeUserNotFound    = "GROWTH_USER_NOT_FOUND"
)

var (
	ErrEnvelopeInvalid = &ContractError{Code: CodeEnvelopeInvalid}
	ErrKeyUnknown      = &ContractError{Code: CodeKeyUnknown}
	ErrPayloadInvalid  = &ContractError{Code: CodePayloadInvalid}
	ErrPathMismatch    = &ContractError{Code: CodePathMismatch}
	ErrKeyUnwrapFailed = &ContractError{Code: CodeKeyUnwrapFailed}
	ErrMACInvalid      = &ContractError{Code: CodeMACInvalid}
	ErrDecryptFailed   = &ContractError{Code: CodeDecryptFailed}
	ErrReplayRejected  = &ContractError{Code: CodeReplayRejected}
)

// ContractError is the stable error surface consumed by the Gin middleware.
type ContractError struct {
	Code string
	Err  error
}

func (e *ContractError) Error() string {
	if e.Err == nil {
		return e.Code
	}
	return e.Code + ": " + e.Err.Error()
}

func (e *ContractError) Unwrap() error { return e.Err }

func (e *ContractError) Is(target error) bool {
	other, ok := target.(*ContractError)
	return ok && other.Code == e.Code
}

// Envelope is the plaintext HTTP body received from AIWork Main.
type Envelope struct {
	KID          string `json:"kid"`
	EncryptedKey string `json:"encrypted_key"`
	IV           string `json:"iv"`
	Ciphertext   string `json:"ciphertext"`
	MAC          string `json:"mac"`
}

// Payload is the decrypted, contract-defined AIWork request payload.
type Payload struct {
	Path           string          `json:"path"`
	Username       string          `json:"username"`
	Params         json.RawMessage `json:"params"`
	RequestID      string          `json:"request_id,omitempty"`
	IssuedAt       int64           `json:"issued_at"`
	Nonce          string          `json:"nonce"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

type DecodeOptions struct {
	PrivateKey *sm2.PrivateKey
	Now        time.Time
	Window     time.Duration
	WantPath   string
	TakeNonce  func(nonce string) bool
}

// DecodeEnvelope validates and decrypts the frozen v4.0 envelope.
func DecodeEnvelope(env Envelope, opts DecodeOptions) (*Payload, error) {
	if opts.PrivateKey == nil || opts.WantPath == "" || opts.TakeNonce == nil {
		return nil, fmt.Errorf("growth crypto: incomplete decode options")
	}
	if env.KID != GrowthKID {
		return nil, ErrKeyUnknown
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(env.EncryptedKey)
	if err != nil {
		return nil, &ContractError{Code: CodeEnvelopeInvalid, Err: err}
	}
	iv, err := base64.StdEncoding.DecodeString(env.IV)
	if err != nil {
		return nil, &ContractError{Code: CodeEnvelopeInvalid, Err: err}
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, &ContractError{Code: CodeEnvelopeInvalid, Err: err}
	}
	mac, err := base64.StdEncoding.DecodeString(env.MAC)
	if err != nil {
		return nil, &ContractError{Code: CodeEnvelopeInvalid, Err: err}
	}
	if len(encryptedKey) == 0 || len(iv) != SM4IVSize || len(ciphertext) == 0 ||
		len(ciphertext)%sm4.BlockSize != 0 || len(mac) != SMMACSize {
		return nil, ErrEnvelopeInvalid
	}

	sessionKey, err := sm2.Decrypt(opts.PrivateKey, encryptedKey, sm2.C1C3C2)
	if err != nil || len(sessionKey) != SM4KeySize {
		return nil, &ContractError{Code: CodeKeyUnwrapFailed, Err: err}
	}
	expectedMAC := SM3HMAC(sessionKey, []byte(env.KID+env.EncryptedKey+env.IV+env.Ciphertext))
	if !hmac.Equal(expectedMAC, mac) {
		return nil, ErrMACInvalid
	}

	plaintext, err := decryptSM4CBC(sessionKey, iv, ciphertext)
	if err != nil {
		return nil, &ContractError{Code: CodeDecryptFailed, Err: err}
	}
	if !utf8.Valid(plaintext) {
		return nil, &ContractError{Code: CodeDecryptFailed, Err: fmt.Errorf("plaintext is not valid UTF-8")}
	}

	var payload Payload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, &ContractError{Code: CodePayloadInvalid, Err: err}
	}
	if err := validatePayload(payload); err != nil {
		return nil, err
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	window := opts.Window
	if window <= 0 {
		window = DefaultTimeWindow
	}
	delta := now.Unix() - payload.IssuedAt
	if delta < 0 {
		delta = -delta
	}
	if delta > int64(window/time.Second) || !opts.TakeNonce(payload.Nonce) {
		return nil, ErrReplayRejected
	}
	if payload.Path != opts.WantPath {
		return nil, ErrPathMismatch
	}

	return &payload, nil
}

// SealEnvelope builds a contract-compatible test or integration fixture.
// Production request encryption remains in AIWork Main.
func SealEnvelope(
	publicKey *sm2.PublicKey,
	payload Payload,
	sessionKey []byte,
	iv []byte,
	random io.Reader,
) (Envelope, error) {
	if publicKey == nil || len(sessionKey) != SM4KeySize || len(iv) != SM4IVSize {
		return Envelope{}, fmt.Errorf("growth crypto: invalid seal inputs")
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	ciphertext, err := encryptSM4CBC(sessionKey, iv, plaintext)
	if err != nil {
		return Envelope{}, err
	}
	encryptedKey, err := sm2.Encrypt(publicKey, sessionKey, random, sm2.C1C3C2)
	if err != nil {
		return Envelope{}, err
	}

	env := Envelope{
		KID:          GrowthKID,
		EncryptedKey: base64.StdEncoding.EncodeToString(encryptedKey),
		IV:           base64.StdEncoding.EncodeToString(iv),
		Ciphertext:   base64.StdEncoding.EncodeToString(ciphertext),
	}
	env.MAC = base64.StdEncoding.EncodeToString(
		SM3HMAC(sessionKey, []byte(env.KID+env.EncryptedKey+env.IV+env.Ciphertext)),
	)
	return env, nil
}

func SM3HMAC(key, message []byte) []byte {
	mac := hmac.New(sm3.New, key)
	_, _ = mac.Write(message)
	return mac.Sum(nil)
}

// ParsePrivateKeyHex parses the server-side SM2 private scalar and derives the
// public point. GoldenDB and entropy are irrelevant here; this is pure key data.
func ParsePrivateKeyHex(value string) (*sm2.PrivateKey, error) {
	d, ok := new(big.Int).SetString(strings.TrimSpace(value), 16)
	if !ok {
		return nil, fmt.Errorf("growth crypto: invalid SM2 private key hex")
	}
	curve := sm2.P256Sm2()
	if d.Sign() <= 0 || d.Cmp(curve.Params().N) >= 0 {
		return nil, fmt.Errorf("growth crypto: SM2 private scalar out of range")
	}
	x, y := curve.ScalarBaseMult(d.Bytes())
	return &sm2.PrivateKey{
		PublicKey: sm2.PublicKey{Curve: curve, X: x, Y: y},
		D:         d,
	}, nil
}

func PublicKeyHex(privateKey *sm2.PrivateKey) (string, error) {
	if privateKey == nil || privateKey.X == nil || privateKey.Y == nil {
		return "", fmt.Errorf("growth crypto: private key has no public point")
	}
	return fmt.Sprintf("%064x%064x", privateKey.X, privateKey.Y), nil
}

func validatePayload(payload Payload) error {
	if payload.Path == "" || payload.Username == "" || payload.IssuedAt <= 0 || payload.Nonce == "" {
		return ErrPayloadInvalid
	}
	if len(payload.Params) == 0 || string(payload.Params) == "null" {
		return ErrPayloadInvalid
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(payload.Params, &params); err != nil {
		return &ContractError{Code: CodePayloadInvalid, Err: err}
	}
	for _, forbidden := range forbiddenParamFields {
		if _, exists := params[forbidden]; exists {
			return &ContractError{
				Code: CodePayloadInvalid,
				Err:  fmt.Errorf("forbidden client field %q", forbidden),
			}
		}
	}
	return nil
}

var forbiddenParamFields = []string{
	"user_id",
	"points",
	"awarded_points",
	"balance",
	"current_balance",
	"token_amount",
	"token_quota",
	"pool_code",
	"pool_balance",
	"rule_id",
	"rule_version",
	"reward",
	"business_date",
	"cycle_key",
	"success",
	"price",
	"points_cost",
}

func encryptSM4CBC(key, iv, plaintext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plaintext, block.BlockSize())
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
	return out, nil
}

func decryptSM4CBC(key, iv, ciphertext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != block.BlockSize() || len(ciphertext)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("invalid SM4 block or IV length")
	}
	out := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ciphertext)
	return pkcs7Unpad(out, block.BlockSize())
}

func pkcs7Pad(value []byte, blockSize int) []byte {
	padding := blockSize - len(value)%blockSize
	return append(value, bytesOf(byte(padding), padding)...)
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, fmt.Errorf("invalid PKCS#7 value length")
	}
	padding := int(value[len(value)-1])
	if padding == 0 || padding > blockSize || padding > len(value) {
		return nil, fmt.Errorf("invalid PKCS#7 padding")
	}
	for _, b := range value[len(value)-padding:] {
		if int(b) != padding {
			return nil, fmt.Errorf("invalid PKCS#7 padding")
		}
	}
	return value[:len(value)-padding], nil
}

func bytesOf(value byte, count int) []byte {
	result := make([]byte, count)
	for index := range result {
		result[index] = value
	}
	return result
}
