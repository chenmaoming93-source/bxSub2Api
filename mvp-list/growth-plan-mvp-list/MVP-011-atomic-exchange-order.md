# MVP-011: Create exchange orders atomically across wallet and inventory

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: One exchange must validate a current product version, limits, balance and stock, then commit reservation, wallet debit, snapshot, and outbox creation together under concurrency.
- Dependencies: `MVP-005`, `MVP-006`, `MVP-008`

## Outcome

AIWork and the Web user can request a product exchange by idempotency key and receive one `FULFILLING` order with exact points deducted, without overselling or double spending.

## Context

The U contract requires `expected_version` only as a stale-catalog hint; server pricing always wins. Failures for version change, offline, sold out, insufficient balance, or limit must not create a debit order.

## In Scope

- Add `POST /growth/exchanges`, `POST /growth/exchanges/:order_id`, `POST /user/growth/exchanges`, `GET /user/growth/exchanges`, and `GET /user/growth/exchanges/:id`.
- Shared order service: current version lookup, expected-version check, limit, stock reservation, wallet debit, order snapshot, and fulfillment job insertion in one transaction.
- Apply L1 `growth.exchange` and `(user_id, idempotency_key)` order uniqueness.
- Add `growth.self.exchange` for W creation and tests for all prescribed error codes.
- Return order id/number, status, product version, cost, transfer id, balance, and contract/W fields.

## Out of Scope

- No external fulfillment call inside the DB transaction.
- No admin refund/retry implementation, physical completion, or unlimited per-user bypass.

## Implementation Notes

- Conditional stock and wallet updates must use affected-row count; never trust client price/token amount.
- Insert the outbox job in the same transaction, then allow a separate worker to process it.
- U and W share service, order uniqueness, and idempotency scope semantics.

## Acceptance Criteria

- [x] A valid exchange creates one reserved inventory delta, wallet debit, paired ledger, product snapshot, order, and pending fulfillment job.
- [x] Repeated create with the same user/key returns the existing order and deducts points once.
- [x] Version change, offline, sold-out, insufficient balance, and purchase limit errors create no order, debit, reservation, or outbox row.
- [x] Concurrent exchanges cannot oversell finite stock or overspend one wallet.
- [x] U and W order detail/list expose authoritative statuses and reject access to another user's order.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth ./internal/handler/growth -run Exchange`
- `go test -tags=unit ./internal/handler/growth -run 'Contract|Exchange'`
- `go test -tags=integration ./internal/service/growth -run 'Exchange|Oversell|DoubleSpend|OrderIsolation'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit/contract tests | `go test -tags=unit ./internal/service/growth ./internal/handler/growth -run Exchange -count=1` | PASS: U/W exchange contract fields and route declarations. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'Exchange|Oversell|DoubleSpend|OrderIsolation' -count=1 -v` | PASS: atomic reservation/debit/order/job, replay, version/offline/sold-out/limit failures, stock and wallet races, and cross-user order isolation. |
| Generation/server regression | `go generate ./cmd/server && go test ./internal/server/... ./internal/handler/growth -count=1` | PASS. |
| Service | `backend/internal/service/growth/order_service.go` | Shared U/W order transaction using the transfer kernel without nested transactions. |
| U/W handlers | `backend/internal/handler/growth/order_handler.go`, `read_handler.go` | U create/status and W create/list/status endpoints. |
| Routes/inventory | `backend/internal/server/routes/growth.go`, `user_growth.go`, `RBAC_ENDPOINT_INVENTORY.md` | `growth.self.exchange` protects W creation; W reads use `growth.self.read`. |

## Execution Notes

The inventory reservation, wallet debit, order snapshot and PENDING fulfillment job all commit in one DB transaction. External fulfillment is deliberately absent until `MVP-012`.
