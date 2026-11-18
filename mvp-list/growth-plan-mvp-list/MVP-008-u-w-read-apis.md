# MVP-008: Deliver authoritative U-face and Web read models

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: Seven AIWork reads plus the Web overview, task, product, order, and ledger reads must share services while preserving distinct authentication and response contracts.
- Dependencies: `MVP-004`, `MVP-005`, `MVP-006`, `MVP-007`

## Outcome

AIWork can read the mapped user's points, tasks, ledger, products, and exchanges through encrypted POST requests, while a browser user can load the same authoritative growth data through JWT-protected W endpoints.

## Context

The frozen contract defines U response fields and status enums. The Web page needs one-call overview, cursor ledger, product eligibility, and task progress. W is same-origin JWT and must not use SM2.

## In Scope

- Implement U reads: `POST /growth/me`, `POST /growth/tasks`, `POST /growth/tasks/:task_id`, `POST /growth/ledger`, `POST /growth/products`, `POST /growth/products/:product_id`, and `POST /growth/exchanges`.
- Implement W reads: `GET /user/growth/overview`, `GET /user/growth/tasks`, `GET /user/growth/products`, `GET /user/growth/products/:id`, `GET /user/growth/exchanges`, and `GET /user/growth/ledger`.
- Add shared read services/view models for wallet, check-in status, task display/action, product eligibility, and exchange state.
- Add `growth.self.read` RBAC, server time/timezone fields, cursor pagination for ledger, and page pagination for lists.
- Add contract tests for exact U field names/types/statuses and authorization isolation for W.

## Out of Scope

- No check-in, completion, exchange creation, status polling mutation, admin, or Vue rendering.
- No client-supplied `user_id`, business date, cycle key, balances, or points.

## Implementation Notes

- U user identity comes only from the decrypted payload's mapped `AuthSubject`; W identity comes only from the authenticated JWT subject.
- U task `action` is server-authored; unavailable/offline tasks must not leak into user-visible lists.
- W overview must derive check-in state from persisted server data and never guess from browser date.

## Acceptance Criteria

- [x] All seven U read endpoints return the frozen contract field names, types, status enums, `server_time`, and timezone conventions.
- [x] `/growth/me` and W overview cannot return another mapped user's wallet or progress.
- [x] Ledger uses `before_entry_id`/`limit` with `next_cursor` and `has_more`, with no offset drift.
- [x] Tasks and products exclude offline/draft data and expose server-computed progress, eligibility, reason, and current version.
- [x] W endpoints require `growth.self.read`; U endpoints remain under the SM2 exclusion and do not require JWT.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/handler/growth ./internal/service/growth -run 'Read|Overview|Ledger|Product|Task'`
- `go test -tags=unit ./internal/handler/growth -run 'Contract|Read'`
- `go test -tags=integration ./internal/handler/growth -run 'URead|WRead|Isolation'`
- `go test ./internal/server -run 'RouteCoverage|Growth'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit contract tests | `go test -tags=unit ./internal/handler/growth ./internal/service/growth ./internal/server/routes -run 'Read|Overview|Ledger|Product|Task|Contract' -count=1` | PASS: required U field names, view calculations, and W route permission declarations. |
| MySQL read-model integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'ReadModels|Isolation|Cursor' -count=1 -v` | PASS: two-user isolation, active-only task/product filters, overview/checkin, product eligibility, exchanges, and cursor pagination. |
| Handler contract run | `go test -tags=integration ./internal/handler/growth -run 'URead|WRead|Contract' -count=1 -v` | PASS. |
| Server/RBAC regression | `go test ./internal/rbac ./internal/handler/growth ./internal/server/... -count=1` | PASS, including user-role self permission seed. |
| Read service | `backend/internal/service/growth/read_service.go` | Shared authoritative U/W read models and cursor ledger. |
| U handlers/routes | `backend/internal/handler/growth/read_handler.go`, `backend/internal/server/routes/growth.go` | Seven U reads behind the SM2 gateway. |
| W handlers/routes | `backend/internal/handler/growth/read_handler.go`, `backend/internal/server/routes/user_growth.go` | Six JWT/RBAC reads; no Web check-in or completion endpoint. |
| Inventory/RBAC | `RBAC_ENDPOINT_INVENTORY.md`, `163_seed_rbac_compatibility.sql`, `183_add_growth_rbac_permissions.sql` | W routes recorded and `growth.self.*` assigned to the built-in user role. |

## Execution Notes

Exchange POST remains list-only in this MVP; creating an exchange is added by `MVP-011`. Product/task U lists are page-based while ledger uses append-only cursor pagination.
