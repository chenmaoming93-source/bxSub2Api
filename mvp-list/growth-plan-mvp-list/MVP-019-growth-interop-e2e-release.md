# MVP-019: Prove cross-plane interop, financial invariants, and release readiness

- Protocol: `mvp-list/v1`
- State: `BLOCKED`
- Estimate: `1.5 days`
- Estimate rationale: This final slice runs the complete sub2api U/M/W journey against real infrastructure, validates the AIWork-facing crypto contract, exercises concurrency/failure recovery, and captures release evidence.
- Dependencies: `MVP-010`, `MVP-017`, `MVP-018`

## Outcome

The sub2api growth plan is demonstrably release-ready: its AIWork-facing envelopes pass the committed contract fixture, all three planes share correct accounting, concurrency cannot duplicate or overspend, admin and user workflows match the prototype, and all required test commands and evidence pass.

## Context

Unit tests are insufficient for crypto fixture compatibility, real database locks, route coverage, frontend/backend field alignment, and visual behavior. The frozen contract requires Mock-pass then real-sub2api-pass with a zero field-level diff. The AIWork Main/Renderer implementation belongs to another project team and is outside this repository's delivery.

## In Scope

- Add a sub2api end-to-end suite covering a user mapped through SM2, budget/task setup, check-in, task completion, product exchange, fulfillment, user/admin views, adjustment, failure/refund, and reconciliation.
- Run the shared crypto vector as a Go server decryptor test against the serialized fixture intended for the AIWork integration contract.
- Exercise duplicate idempotency keys, duplicated completion_ref, task quota exhaustion, wallet overdraw, concurrent exchange, failing fulfillment, and retry/recovery.
- Run backend unit/integration/route coverage tests and frontend test/typecheck/lint/build.
- Capture light/dark 1440px, tablet, and mobile screenshots for admin budget/tasks/orders and user overview/store/ledger; record commands/results.
- Review security logs for absence of keys, session keys, and plaintext payloads; verify open blockers have owners and release disposition.

## Out of Scope

- No load testing beyond the concurrency assertions needed for financial invariants.
- No new feature, provider integration, contract field, or automatic accounting repair.
- No AIWork project implementation or modification; actual cross-project joint testing is coordinated separately after sub2api delivery.

## Implementation Notes

- Prefer a deterministic seeded integration environment and clean database per test group.
- Verify both HTTP envelopes for all failure classes consumed by AIWork.
- Treat any field/status/error divergence from v4.0 as release-blocking rather than a frontend adapter workaround.

## Acceptance Criteria

- [ ] The complete sub2api U/M/W journey passes against MySQL 8/GoldenDB and Redis-backed nonce/idempotency storage, with the AIWork contract fixture encrypting and decrypting successfully.
- [x] No concurrency, replay, quota exhaustion, refund, or retry scenario creates a duplicate award, overdraw, oversell, one-sided ledger, or negative quota.
- [ ] Every specified v4.0 endpoint/status/error is covered by a field-level contract assertion; no removed I-face or polling path exists.
- [x] The task lifecycle proves no delete, immutable code, immutable published rules, quota-preserving publish, safe offline/revoke, and funded re-online.
- [x] Admin and Web visual checks match the supplied prototype’s information hierarchy and required states in light/dark and responsive layouts.
- [x] Delivery materials clearly identify U API, test vectors, and integration notes as sub2api outputs; no AIWork source change is included in the release.
- [ ] All listed backend/frontend commands pass and evidence is recorded in this document and the progress row.

## Verification Plan

- `cd backend && go generate ./ent && go test -tags=unit ./...`
- `go test -tags=integration ./internal/repository/... ./internal/service/growth/... ./internal/handler/growth/...`
- `go test -tags=unit ./internal/handler/growth/... -run Contract`
- `go test -tags=e2e -v -timeout=300s ./internal/integration/...`
- `go test ./internal/server/... && golangci-lint run ./...`
- `cd frontend && npm run test:run`
- `npm run typecheck && npm run lint:check && npm run build`
- Manually execute the seeded full journey and store screenshots/log summaries as completion evidence.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Runtime | Native MySQL 8.4 on `127.0.0.1:3307`; latest backend on `:8080`; frontend on `:5173` | PASS: `/health` HTTP 200; archived growth schema and RBAC seed applied to the active database. Docker Desktop engine was unavailable, so native MySQL was used. |
| Backend unit | `go test -p 1 -tags=unit ./... -count=1` | PASS after serializing package builds for Windows memory limits. |
| Growth integration | `GROWTH_MYSQL_TEST_DSN=root@tcp(127.0.0.1:3307)/?parseTime=true go test -tags=integration ./internal/service/growth -count=1` | PASS: concurrency, idempotency, quota exhaustion, overdraw, stock reservation, failed fulfillment/refund, retry, task lifecycle and reconciliation. Refund regression now restores limited stock. |
| Route/contract | `go test ./internal/server/... ./internal/handler/growth ./internal/service/growth`, handler contract and SM2 crypto contract | PASS for declared routes and committed fixture; a full encrypted HTTP U-to-M-to-W release journey remains uncovered. |
| Frontend | `npm run typecheck`, `npm run test:run`, focused growth tests, focused ESLint and `npm run build` | PASS: 152 files / 885 tests and 13 focused tests; production build succeeded with existing bundle warnings. |
| Browser journey | Admin create/publish/adjust; Web exchange/failure/refund/ledger | PASS for M/W journeys and visual states. See [growth-release-checklist.md](../evidence/growth-release-checklist.md). |
| Visual evidence | 12 desktop screenshots and 16 responsive screenshots under [`../evidence/screenshots/`](../evidence/screenshots/) | PASS: 375/768/1440 light/dark; automated checks found `scrollWidth === innerWidth`. |
| Changed-line lint | `golangci-lint v2.9 run --build-tags=unit --new-from-rev=HEAD ./...` | PASS: 0 issues for this delivery. |
| Full repository lint | `golangci-lint v2.9 run --build-tags=unit ./...` | BLOCKED: 98 pre-existing findings outside this delivery, primarily unused compatibility symbols and unchecked close operations across unrelated packages. |
| E2E externals | `E2E_MOCK=true go test -tags=e2e -v -timeout=300s ./internal/integration/...` | PASS as a package, but provider cases skipped because Claude/Gemini keys were absent; these tests do not cover the growth encrypted journey. |

## Execution Notes

Implementation and automated checks are complete for the delivered growth scope, but the release-readiness gate remains blocked. Full repository lint fails on unrelated pre-existing code, and no real encrypted U-to-M-to-W HTTP journey or production quota/subscription adapter exists in the local environment. Progress must remain conservative until those blockers are resolved or explicitly re-scoped.
