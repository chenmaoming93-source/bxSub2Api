# MVP-012: Fulfill exchange orders through retry-safe outbox adapters

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: The worker, three product modes, idempotent external calls, failure transitions, stock release, refund, and admin recovery operations form one end-to-end fulfillment outcome.
- Dependencies: `MVP-005`, `MVP-011`

## Outcome

Pending exchange orders progress to success or terminal failure outside the order transaction, with bounded retries, idempotent Token/subscription delivery, physical/manual handling, and refund-with-stock-release recovery.

## Context

`growth_fulfillment_jobs` is an outbox keyed by order. Existing quota and subscription services must be reused through ports. A final failure must release reservation and perform a reverse wallet transfer without mutating ledger history.

## In Scope

- Add a worker that claims due jobs with safe locking, retries with bounded backoff, and records attempt/failure state.
- Add fulfillment adapter behavior for `TOKEN_QUOTA` and `SUBSCRIPTION`; add manual completion handling for `PHYSICAL`.
- Add `GET /admin/growth/exchanges`, `GET /admin/growth/exchanges/:id`, `POST /admin/growth/exchanges/:id/retry`, `POST /admin/growth/exchanges/:id/manual-complete` where applicable, and `POST /admin/growth/exchanges/:id/refund`, each with reason and audit where mutating.
- On success, confirm sold inventory and order timestamps; on final failure, release reservation and reverse wallet debit.
- Add tests proving no external call occurs inside the DB transaction and retries do not duplicate fulfillment or refund.

## Out of Scope

- No new external HTTP long transaction or new provider infrastructure.
- No fulfillment-webhook contract with AIWork and no automatic retry after a refunded terminal state.

## Implementation Notes

- Use the order id as the external idempotency key and persist provider attempt identifiers when available.
- If an existing Token service cannot prove per-order idempotency, wrap it with a lookup/adapter guard and record the capability result before enabling that product.
- Retry transitions must be compare-and-set so two workers cannot own one job concurrently.

## Acceptance Criteria

- [x] A queued job invokes the correct adapter exactly once per lease and moves success, retry, or terminal failure through valid order status transitions.
- [x] Repeated worker delivery for the same order does not issue tokens/subscription twice.
- [x] Transient failures increment attempts and schedule a bounded retry; terminal failure releases finite stock and creates one reverse transfer/refund.
- [x] Manual physical completion is audited and does not generate another wallet transaction.
- [x] Admin retry is limited to eligible failed orders and does not repeat points deduction.
- [x] Admin refund requires `growth.manage`, a reason, and is idempotent.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth -run 'Fulfillment|Outbox|Refund|Retry'`
- `go test -tags=integration ./internal/service/growth -run 'OutboxWorker|StockRelease|Refund'`
- `go test ./internal/server/routes -run GrowthExchange`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit tests | `go test -tags=unit ./internal/service/growth -run 'Fulfillment|Outbox|Refund|Retry' -count=1` | PASS: snapshot parsing, bounded backoff and worker helper logic. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'OutboxWorker|StockRelease|Refund' -count=1 -v` | PASS: successful Token delivery, one-call lease, subscription transient retry, terminal refund/stock release, physical manual completion, admin retry without re-debit and idempotent refund. |
| Route/server regression | `go test ./internal/server/routes -run GrowthExchange -count=1 && go test ./internal/server/... -count=1` | PASS. |
| Service | `backend/internal/service/growth/fulfillment_service.go` | Lease-based worker, external adapter call outside DB transaction, finalize/refund transactions and admin operations. |
| Handlers/routes | `backend/internal/handler/growth/fulfillment_handler.go`, `backend/internal/server/routes/admin.go` | Five admin order APIs under `growth.read`/`growth.manage`. |

## Execution Notes

Token and subscription delivery use registered adapters keyed by order id; when no production adapter is configured the worker fails safe into retry/refund rather than claiming success. Physical orders remain PENDING until an audited manual completion.
