# MVP-006: Implement double-entry wallet transfers for every growth award

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: This is the accounting kernel shared by check-in, completion, exchange, refund, and adjustment; it needs transaction and concurrency correctness before any command endpoint uses it.
- Dependencies: `MVP-002`, `MVP-003`, `MVP-004`

## Outcome

Growth services can atomically transfer task quota into a user wallet, debit a wallet into the redeemed account, and reverse a redeem with exactly two balanced ledger entries per successful transfer and deterministic idempotent replay.

## Context

Every award must conditionally consume task quota and update the wallet, budget distributed total, transfer, claim/progress records, and two ledger entries in one database transaction. Redemption does not consume budget; it moves owned wallet points to `REDEEMED`.

## In Scope

- Add repository transaction primitives and a `GrowthTransferService` with task-to-wallet, wallet-to-redeemed, and reverse-redeem operations.
- Enforce idempotency by transfer business key and return existing results on duplicate keys.
- Add wallet creation/upsert, conditional balance updates, current-balance calculation, and shared business-time/cycle-key helpers.
- Add integration tests for rollback, concurrent spend/award, transfer-entry pairing, and ledger sum invariants.
- Expose no public HTTP route in this MVP.

## Out of Scope

- No rule matching, completion_ref handling, HTTP handlers, product order orchestration, outbox worker, or manual adjustment UI.
- No update/delete of ledger rows.

## Implementation Notes

- Use `SELECT ... FOR UPDATE` where wallet mutation requires it and conditional SQL for task quota/balance checks.
- A failed conditional update must roll back the entire transaction and never emit a one-sided ledger.
- Keep operation-specific business tables in callers or transaction adapters so this MVP retains one bounded accounting outcome.

## Acceptance Criteria

- [x] A successful award changes task distributed points, budget distributed amount, wallet balance, one transfer, and exactly two ledger entries atomically.
- [x] Repeating the same transfer business key returns the original transfer and adds no rows or balance.
- [x] Concurrent awards/spends cannot drive task quota or wallet balance negative.
- [x] Any injected failure after the first write leaves no budget, wallet, transfer, or ledger delta.
- [x] `wallet balance = SUM(user ledger)` and both sides of every transfer net to zero in integration tests.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth -run 'Transfer|Ledger|CycleKey'`
- `go test -tags=integration ./internal/service/growth -run 'Transfer|Concurrent|Rollback'`
- Run the repository's MySQL 8/GoldenDB integration suite with transactions enabled.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit tests | `go test -tags=unit ./internal/service/growth -run 'Transfer|Ledger|CycleKey' -count=1` | PASS: business timezone cycle keys, weekly/once keys, and wallet/account ledger sign calculations. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'Transfer|Concurrent|Rollback' -count=1 -v` | PASS on MySQL 8.4.9: award/debit/reverse transactions, replay, quota and balance concurrency, injected post-ledger rollback, two rows per transfer, net-zero entries, and wallet=ledger sum. |
| Service | `backend/internal/service/growth/transfer_service.go` | Shared transactional accounting primitives for task award, wallet debit and redeem reversal. |

## Execution Notes

The transfer kernel exposes same-transaction hooks for later check-in, completion and exchange services while this MVP itself registers no HTTP route. `spent_total` is cumulative net consumption and a refund reduces it without changing `earned_total`.
