# MVP-010: Award declared task completions with business and limit deduplication

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: The completion command must synchronously combine completion_ref uniqueness, current rule limits, quota exhaustion, transfer accounting, and exact contract responses.
- Dependencies: `MVP-006`, `MVP-008`

## Outcome

AIWork can declare a completed task with `completion_ref` and receive a final synchronous award outcome; retries and alternate idempotency keys cannot duplicate the same completion, and violated limits or exhausted quota award zero.

## Context

Completion judgment belongs to AIWork. sub2api trusts the declaration, does not query `usage_logs`, and only enforces rules, limits, task quota, and accounting idempotency. There is no async status endpoint.

## In Scope

- Add `POST /api/v1/growth/tasks/:task_id/completion-requests` under SM2.
- Validate route/body task id agreement and required `completion_ref`; ignore or reject forbidden accounting fields.
- Resolve active task/rule, enforce cycle/daily/per-user totals, and conditionally consume quota.
- Write completion request, user-task progress, claim, transfer, and paired ledger entries atomically.
- Return `AWARDED`, `ALREADY_COMPLETED`, `NOT_ELIGIBLE`, or `POOL_EXHAUSTED` as defined.
- Add unit, contract, and concurrency tests including completion_ref reuse with a new idempotency key.

## Out of Scope

- No external evidence verification, async polling, rejection state, I-face events, user-supplied points/rule/version, or W completion endpoint.

## Implementation Notes

- Unique `(user_id, task_id, completion_ref)` is the final business dedupe guard; read the existing result after a unique conflict.
- Quota exhaustion maps to 503 `POOL_EXHAUSTED`; rule/limit rejection maps to `NOT_ELIGIBLE`.
- Include `request_id` only as trace metadata, never as an award key.

## Acceptance Criteria

- [x] A valid completion creates one completion request, transfer, claim/progress update, and exactly two ledger entries and returns `AWARDED`.
- [x] Reusing the same completion_ref, even with a new idempotency key, returns the first result without another award.
- [x] Daily/per-user/period limits return `NOT_ELIGIBLE` with no accounting rows.
- [x] Exhausted task quota returns `POOL_EXHAUSTED` with no accounting rows and does not make quota negative.
- [x] Concurrent requests for the same completion create at most one award, and different completions cannot exceed quota.
- [x] Code and tests contain no `usage_logs` lookup or asynchronous completion polling path.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth ./internal/handler/growth -run Completion`
- `go test -tags=unit ./internal/handler/growth -run 'Contract|Completion|AlreadyCompleted|NotEligible|PoolExhausted'`
- `go test -tags=integration ./internal/service/growth -run 'Completion|CompletionRef|Concurrent'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit/contract tests | `go test -tags=unit ./internal/service/growth ./internal/handler/growth -run 'Completion|Checkin' -count=1` | PASS: response contract and completion integration suite compilation. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'Completion|Dedupe|Limits|Concurrency|Quota' -count=1 -v` | PASS: completion_ref dedupe with changed idempotency key, same-cycle duplicate, daily/per-user limit, concurrent single award, and exhausted quota rollback. |
| Generation/server regression | `go generate ./cmd/server && go test ./internal/server/... ./internal/handler/growth -count=1` | PASS. |
| Service | `backend/internal/service/growth/completion_service.go` | Trusted AIWork declaration path with rule limits, completion records, user progress, claim and shared transfer. |
| Handler/route | `backend/internal/handler/growth/completion_handler.go`, `backend/internal/server/routes/growth.go` | U-only SM2 completion command with payload idempotency. |

## Execution Notes

No evidence lookup, `usage_logs` query, asynchronous verification, polling endpoint or I-face event is implemented. A repeated completion_ref with a new idempotency key returns `ALREADY_COMPLETED` and retains the first award amount without another transfer.
