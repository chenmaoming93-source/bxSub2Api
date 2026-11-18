# MVP-007: Protect the AIWork U-face with a verified SM2 envelope gateway

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: Cryptographic parsing, anti-replay state, user mapping, route registration, config propagation, and security error tests cross several runtime boundaries.
- Dependencies: `MVP-001`, `MVP-002`

## Outcome

All `/api/v1/growth/*` requests are decrypted and integrity-checked before business handling, mapped to a non-zero `AuthSubject`, protected from replay and path confusion, and visible in RBAC route coverage as an explicit trust boundary.

## Context

The U face is POST-only, uses a fixed `growth-sm2-v1` envelope, and has no JWT or identity header. The middleware must validate MAC before decryption, use constant-time comparison, and never log secrets or plaintext payloads.

## In Scope

- Add `backend/internal/server/middleware/growth_payload.go` with SM2 C1C3C2 unwrap, SM3-HMAC, SM4-CBC/PKCS#7 decrypt, time/nonce replay checks, route-template binding, and email-to-user mapping.
- Add a Redis or process-safe TTL nonce cache and wire it through middleware providers.
- Add the fixed U route tree under `backend/internal/server/routes/growth.go`, register it in `router.go`, and classify `/api/v1/growth/` as a known exclusion with a precise reason.
- Add `growth.crypto.time_window` to config structure, defaults/example, and `backend/config/config.yaml`, reusing top-level timezone.
- Add unit/integration tests for every crypto failure, user-not-found, path mismatch, and AuthSubject write.

## Out of Scope

- No business response implementation beyond route wiring.
- No I-face route, key rotation, KMS, config-encrypted private key, or account-active check.

## Implementation Notes

- Do not log or return private keys, decrypted SM4 keys, nonces, or plaintext payloads.
- The production private key is a server-only code constant; tests inject the fixed test key without weakening production parsing.
- Preserve two response envelopes: string-code middleware aborts and numeric business responses.

## Acceptance Criteria

- [x] Valid contract v4.0 envelopes decrypt and populate the payload plus `AuthSubject{UserID: mappedID}`.
- [x] Unknown `kid`, unwrap failure, altered MAC, bad ciphertext, expired timestamp, reused nonce, incorrect path, malformed payload, and missing user return the specified `CRYPTO_*` or `GROWTH_USER_NOT_FOUND` response.
- [x] MAC is checked before plaintext decryption and comparison is constant-time.
- [x] Replays are rejected across requests and different users do not share a nonce/idempotency scope.
- [x] `ValidateRouteCoverage` passes and identifies the U prefix as internal encrypted-payload trust boundary.
- [x] Config loading tests cover the time window and development YAML contains it.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/server/middleware -run GrowthPayload`
- `go test -tags=unit ./internal/config ./internal/service/growth -run 'Growth|Crypto|Nonce'`
- `go test ./internal/server -run 'RouteCoverage|Growth'`
- `go test -tags=integration ./internal/server -run GrowthCrypto`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Middleware unit tests | `go test -tags=unit ./internal/server/middleware -run GrowthPayload -count=1` | PASS: valid request/AuthSubject, failure matrix, nonce replay, two-user mapping, malformed JSON, and production key derivation. |
| Config/nonce tests | `go test -tags=unit ./internal/config ./internal/service/growth -run 'Growth|Crypto|Nonce' -count=1` | PASS: 5-minute default/validation and Redis TTL nonce behavior. |
| Integration-tag middleware run | `go test -tags=integration ./internal/server/... -run 'GrowthPayload|GrowthCrypto' -count=1` | PASS. |
| Server/RBAC regression | `go test ./internal/rbac ./internal/server/... -count=1` | PASS, including `/api/v1/growth/` route-coverage exclusion. |
| Generated code regression | `go test -tags=unit ./ent/... -count=1` | PASS after clean Ent regeneration. |
| Middleware | `backend/internal/server/middleware/growth_payload.go` | SM2 gateway, nonce cache dispatch, user mapping, safe audit logging and string-code errors. |
| Nonce cache | `backend/internal/service/growth/nonce_cache.go` | Redis `SET NX` TTL anti-replay. |
| Routes/config | `backend/internal/server/routes/growth.go`, `backend/internal/server/router.go`, `backend/internal/config/config.go`, `backend/config/config.yaml` | U route tree, RBAC exclusion and `growth.crypto.time_window: 5m`. |
| AIWork public key | `growth-sm2-v1` | `82faea4b9093687dd34416141fd05ac825944de64a55e60be17873e4539524931a90619fe65ca78657dabe658ff7278c5a0a5161981c9e3a55ed75115fe6b72d` |

## Execution Notes

Business handlers remain intentionally pending for `MVP-008` and later; this gateway test scope stops at decryption, mapping and route coverage. The production key pair is fixed in code/handoff with no runtime rotation.
