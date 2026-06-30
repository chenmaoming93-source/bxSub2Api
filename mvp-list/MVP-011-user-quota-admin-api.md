# MVP-011: User Model Daily Quota Admin API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: Reuse the existing admin user route and DTO patterns for two focused endpoints.
- Dependencies: `MVP-005`, `MVP-007`, `MVP-009`

## Outcome

Admins can read and update per-user, per-model daily token limits without modifying other users.

## Context

Reference `backend/internal/handler/admin/user_handler.go` and `/admin/users/:id/platform-quotas`.

## In Scope

- Add GET/PUT user model quota routes, handler, and service methods.
- Implement explicit upsert semantics for the specified user/model quota rows.
- Invalidate target user/model quota cache entries and add permission, validation, and isolation tests.

## Out of Scope

- End-user self-service configuration or frontend modals.

## Implementation Notes

- Reused existing admin route and handler patterns.
- The implemented paths are `GET /admin/users/:id/model-token-quotas` and `PUT /admin/users/:id/model-token-quotas`.
- Invalid users are checked through the admin user service before quota list/update operations.

## Acceptance Criteria

- [x] GET returns only target user records.
- [x] PUT does not affect unspecified users.
- [x] Invalid user, model, or limit receives a stable 4xx response.

## Verification Plan

- `cd backend; go test ./internal/handler/admin ./internal/server -run 'UserModelTokenQuota'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `backend/internal/handler/admin/user_model_token_quota_handler.go`; `backend/internal/service/user_model_token_quota_admin_service.go`; `backend/internal/repository/daily_token_quota_repo.go`; `backend/internal/server/routes/admin.go` | Added admin GET/PUT `/admin/users/:id/model-token-quotas`, user existence checks, user/model/limit validation, repository list/upsert, and cache invalidation. |
| Tests | `cd backend; go test -c -o .\.gotmp\handler_admin_usermodelquota_test.exe .\internal\handler\admin`; `& .\.gotmp\handler_admin_usermodelquota_test.exe --% -test.run UserModelTokenQuota -test.v` | PASS: user-scoped list, update isolation, invalid user ID, missing user, and negative limit coverage. Direct `go test` execution was blocked by Windows Application Control, so the fixed binary workaround was used. |
| Tests | `cd backend; go test .\internal\server -run UserModelTokenQuota -v` | PASS: admin route requires admin auth. |
| Tests | `cd backend; go test .\internal\repository -run "UpsertUserModelQuotas|DailyTokenQuota" -v` | PASS: repository user isolation, existing quota behavior, cache tests, increment, rollover, and rollback tests. |
| Compile | `cd backend; go test -c -o .\.gotmp\cmd_server_usermodelquota_test.exe .\cmd\server` | PASS: manual Wire graph update compiles. |

## Execution Notes

- `go test .\internal\handler\admin -run UserModelTokenQuota -v` was attempted with repo-local `GOCACHE`, `GOMODCACHE`, and `GOTMPDIR`, but Windows Application Control blocked the generated `admin.test.exe`; the fixed `.gotmp` binary invocation above is the recorded verification.
