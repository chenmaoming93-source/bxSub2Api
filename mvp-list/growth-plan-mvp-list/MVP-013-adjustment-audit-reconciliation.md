# MVP-013: Make manual adjustments auditable and reconciliation verifiable

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: High-risk wallet adjustments, user drilldown APIs, audit search, and invariant reconciliation need both financial correctness and operational diagnostics.
- Dependencies: `MVP-003`, `MVP-006`, `MVP-012`

## Outcome

An authorized administrator can inspect user point accounts and perform a reasoned audited increase or decrease, while the system can report and alert on budget, task-quota, wallet, and ledger invariant mismatches without auto-correcting them.

## Context

Manual increase follows compensated budget allocation validation and adds wallet points; decrease moves wallet points to the redeemed account. No double approval exists in v1. The prototype also requires user ledger/exchange views and failed-order triage.

## In Scope

- Add `GET /admin/growth/users`, `GET /admin/growth/users/:id/ledger`, and `GET /admin/growth/users/:id/exchanges`.
- Add `POST /admin/growth/users/:id/adjustments` guarded by `growth.adjust`, with required reason and before/after balance.
- Implement adjustment as formal transfer plus paired ledger entries; increased points must respect available budget rules, and decreases cannot overdraw.
- Add audit-log query and overview/reconciliation endpoints or service commands for total, allocated, distributed, task remainder, wallet, and ledger equations.
- Add a repeatable reconciliation report/worker or test entry point and alert signal without automatic data repair.
- Add unit/integration tests for permission, atomicity, and every reconciliation equation.

## Out of Scope

- No second-person approval, direct ledger edit, arbitrary SQL from UI, or automatic correction.
- No charts or historical trend analytics in the first release.

## Implementation Notes

- Treat `growth_adjust` as high-risk and audit both amount and resulting balance.
- Detect anomalies from persisted facts; avoid computing one side from the other and then declaring equality.
- Include operator, request id, reason, resource, and before/after summary in audit data.

## Acceptance Criteria

- [x] Increase/decrease adjustment changes wallet and paired ledger atomically and writes exactly one formal transfer plus audit record.
- [x] A user without `growth.adjust` receives 403; missing/short reason and overdraw are rejected without partial writes.
- [x] User list exposes current, total earned, total spent, latest change; drawer APIs paginate ledger and exchanges.
- [x] Reconciliation detects total, allocated, distributed, task quota, wallet, and transfer-entry mismatches independently.
- [x] An anomaly produces a stable report/alert result and does not modify account or ledger values.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth ./internal/server/routes -run 'Adjust|Reconciliation|Audit'`
- `go test -tags=integration ./internal/service/growth -run 'Adjustment|Reconciliation'`
- `go test ./internal/server/...`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit/route tests | `go test -tags=unit ./internal/service/growth ./internal/server/routes -run 'Adjust|Reconciliation|Audit' -count=1` | PASS: adjustment validation, report contract and `growth.adjust` route declaration. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'Adjustment|Reconciliation' -count=1 -v` | PASS: increase/decrease transfers and audits, overdraw/short-reason rejection, user list, ledger sums, stable reconciliation issues and no auto-repair. |
| Server regression | `go generate ./cmd/server && go test ./internal/server/... ./internal/handler/growth -count=1` | PASS. |
| Service | `backend/internal/service/growth/admin_service.go` | Audited adjustments, user point query and six independent reconciliation checks. |
| Handler/routes | `backend/internal/handler/growth/admin_points_handler.go`, `backend/internal/server/routes/admin.go` | User point list/drilldown, `growth.adjust`, audit and reconciliation APIs. |

## Execution Notes

Increase adjustments consume unallocated budget and increment budget distributed total; reconciliation includes positive ADJUST transfers in the distributed equation. Detected anomalies are returned only and never corrected.
