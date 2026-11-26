# MVP-003: Operate a single total budget with governed task-quota allocation

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: The global budget row is the hottest accounting lock and needs service rules, admin endpoints, audit consistency, concurrency tests, and UI-ready read models.
- Dependencies: `MVP-002`

## Outcome

An operator can increase or reduce the total budget, allocate quota to a task, revoke only unused task quota, and inspect the authoritative budget and four-entry-type change history through M-face APIs.

## Context

`total_amount`, `allocated_amount`, and `distributed_amount` form the single budget account. Reductions cannot consume already allocated quota. Every mutation must write both `growth_budget_entries` and `growth_admin_audit` in the same transaction.

## In Scope

- Implement `GrowthBudgetService` methods for TOPUP, REDUCE, ALLOCATE, and REVOKE.
- Add M routes for `GET /admin/growth/stats/overview`, `GET /admin/growth/budget`, `GET|POST /admin/growth/budget/entries`, `GET|POST /admin/growth/budget/allocations`, and `POST /admin/growth/tasks/:id/budget/revoke`.
- Implement row locking or conditional updates plus version/double-entry invariants.
- Add pagination/filtering and operator/reason fields needed by the prototype budget tab.
- Include RBAC declarations and service/route unit tests.

## Out of Scope

- No task creation or rule lifecycle; `MVP-004` composes task creation with initial ALLOCATE.
- No user wallet transfer, manual adjustment, frontend, or budget approval workflow.

## Implementation Notes

- Use database predicates and affected-row checks; do not read a balance and then blindly write it.
- Revoke only `allocated_points - distributed_points`, and only for an `OFFLINE` or otherwise ended task.
- Return `BUDGET_ALLOCATED_LOCKED` for an unsafe reduction and audit every successful or rejected privileged attempt as appropriate.

## Acceptance Criteria

- [x] TOPUP and REDUCE update the single budget row and append exactly one typed budget entry plus one audit record.
- [x] REDUCE is rejected when the result would be lower than `allocated_amount`; no partial write remains.
- [x] ALLOCATE cannot exceed unallocated budget, and concurrent allocations cannot oversubscribe the account.
- [x] REVOKE returns unused task quota to unallocated budget without changing `distributed_points` or historical ledger rows.
- [x] M read endpoints expose total, allocated, distributed, unallocated, usage, operator, reason, and timestamps with stable pagination.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth ./internal/server/routes -run Budget`
- `go test -tags=integration ./internal/service/growth -run 'Budget|Allocation|Revoke'`
- `go test ./internal/server/...`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit tests | `go test -tags=unit ./internal/service/growth ./internal/server/routes -run Budget -count=1` | PASS: budget calculations, validation, handler operator/idempotency context, and RBAC declarations. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'Budget|Allocation|Revoke' -count=1 -v` | PASS on MySQL 8.4.9: TOPUP/REDUCE/ALLOCATE/REVOKE auditing, locked reduction, concurrency oversubscription protection, and read models. |
| Server/RBAC regression | `go test ./internal/rbac ./internal/server/... -count=1` | PASS, including RBAC compatibility seed and route coverage. |
| Service | `backend/internal/service/growth/budget_service.go` | Single-row lock/transaction implementation for all four budget operations. |
| Handler and routes | `backend/internal/handler/growth/budget_handler.go`, `backend/internal/server/routes/admin.go` | Seven M-face endpoints with `growth.read`/`growth.manage`. |
| Permission seed | `backend/internal/rbac/permissions.go`, `backend/sqlArchiving/163_seed_rbac_compatibility.sql`, `backend/sqlArchiving/growth/dml/183_add_growth_rbac_permissions.sql` | Growth permissions added to catalog and deployment seed. |
| Inventory | `RBAC_ENDPOINT_INVENTORY.md` | Seven growth budget endpoints registered. |

## Execution Notes

The integration test uses a disposable MySQL schema and verifies that failed mutations leave no budget entry or audit delta.
