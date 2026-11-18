# MVP-009: Award daily check-ins exactly once per business date

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: Check-in combines server-side cycle calculation, active rule resolution, request/accounting idempotency, quota exhaustion, and the shared transfer transaction.
- Dependencies: `MVP-006`, `MVP-008`

## Outcome

AIWork can submit a completed check-in and receive a synchronous award result, while duplicate submissions, cross-request retries, concurrent calls, and quota exhaustion cannot create a second or partial award.

## Context

AIWork determines that the user checked in; sub2api only applies `daily_checkin` rules. The client cannot send a business date, cycle key, points, or source account. Web has no check-in endpoint.

## In Scope

- Add `POST /api/v1/growth/checkins` under the SM2 gateway only.
- Resolve the active `CHECKIN` task rule and compute `business_date`/`cycle_key` in the configured timezone.
- Apply L1 `growth.checkin` idempotency and L2 `(user_id, task_id, cycle_key)`/claim uniqueness.
- Execute task-quota-to-wallet transfer and return `AWARDED`, `ALREADY_CLAIMED`, `NOT_ELIGIBLE`, or `UNAVAILABLE` contract fields.
- Update W overview status and task progress/streak display from persisted results.
- Add unit, contract, and concurrent integration tests.

## Out of Scope

- No evidence lookup in `usage_logs`, AIWork task judgment, Web check-in route, custom date input, or manual streak editing.
- No reward bypass for super administrators.

## Implementation Notes

- Reuse the decrypted payload idempotency key; never read it from a client header.
- Use a conditional task quota update and roll back every write when quota is insufficient.
- A duplicate same-day request returns the existing claim result with `awarded_points=0` semantics from the contract.

## Acceptance Criteria

- [x] A valid check-in creates one claim, transfer, wallet increase, and exactly two ledger rows, and returns `AWARDED` with current balance.
- [x] Repeating the request with the same idempotency key returns the first result with replay metadata and performs no additional transfer.
- [x] Repeating with another idempotency key but the same user/task/cycle returns `ALREADY_CLAIMED` and no new transfer.
- [x] Two concurrent check-ins create at most one award.
- [x] Exhausted task quota returns `UNAVAILABLE`, no wallet/ledger change, and leaves quota non-negative.
- [x] The cycle changes only at the server's configured business-day boundary.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth ./internal/handler/growth -run Checkin`
- `go test -tags=unit ./internal/handler/growth -run 'Contract|Checkin'`
- `go test -tags=integration ./internal/service/growth -run 'Checkin|QuotaExhausted|Concurrent'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit/contract tests | `go test -tags=unit ./internal/service/growth ./internal/handler/growth -run Checkin -count=1` | PASS: contract fields and check-in response shape. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'Checkin|QuotaExhausted|Concurrent' -count=1 -v` | PASS: award/claim/ledger atomicity, same-key replay, same-cycle duplicate, concurrent single winner, day rollover and exhausted quota. |
| Generation/server regression | `go generate ./cmd/server && go test ./internal/server/... ./internal/handler/growth -count=1` | PASS. |
| Service | `backend/internal/service/growth/checkin_service.go` | Server-time cycle resolution, active rule lookup, L2 claim dedupe, transfer callback and overview continuity. |
| Handler/route | `backend/internal/handler/growth/checkin_handler.go`, `backend/internal/handler/growth/idempotency_helper.go`, `backend/internal/server/routes/growth.go` | U-only SM2 check-in using payload idempotency key. |

## Execution Notes

The L1 coordinator replays the first AWARDED response when the same payload key is reused. Independent keys racing for the last quota can resolve to `UNAVAILABLE` rather than `ALREADY_CLAIMED`, but still produce at most one award.
