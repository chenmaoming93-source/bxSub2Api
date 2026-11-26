# MVP-004: Manage versioned growth tasks without deletion

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: Creation combines quota allocation and rules, while publish/offline/re-online have a precise state machine and historical immutability.
- Dependencies: `MVP-002`, `MVP-003`

## Outcome

An operator can create a task with initial quota, create and publish immutable rule versions, inspect rule history, take a task offline, and re-online it with fresh quota; task history remains permanently queryable.

## Context

The task table is the catalog authority. `task_code` is immutable and reusable never; published rules cannot be overwritten; delete is intentionally absent. The prototype requires task status, version, limits, quota usage, and action visibility by state.

## In Scope

- Add `GET|POST /admin/growth/tasks`, `GET /admin/growth/tasks?code=...`, `PUT /admin/growth/tasks/:id`, `POST /admin/growth/tasks/:id/publish`, `POST /admin/growth/tasks/:id/offline`, `POST /admin/growth/tasks/:id/online`, and `GET /admin/growth/tasks/:id/rule-versions`.
- Make create and initial ALLOCATE one transaction; edit never changes quota.
- End the current rule and activate the next published version atomically.
- Implement required offline reason and optional unused-quota revoke; re-online requires `initial_allocation > 0` and an applicable rule.
- Add `growth.read`/`growth.manage` RBAC metadata, audit entries, and state-machine tests.

## Out of Scope

- No `DELETE` route, `deleted_at`, task-code edits, rule mutation after publication, or user-facing reward execution.
- No task UI beyond the backend view model.

## Implementation Notes

- Support `CHECKIN`, `DAILY`, and `ACHIEVEMENT` categories and `DAILY`, `ONCE`, and `WEEKLY` periods.
- For duplicate `task_code`, map unique-key violation to the prescribed internal error and do not create an orphan task.
- Task code availability may use `GET /admin/growth/tasks?code=...` to satisfy the page's pre-submit check without inventing a second source of truth.

## Acceptance Criteria

- [x] Creating a task with zero quota, insufficient unallocated budget, or duplicate code fails without any budget or task row.
- [x] Editing published reward fields produces a draft `version+1`; only publish makes it effective and only one rule is effective at a time.
- [x] Published rule versions and task codes cannot be modified through any registered route.
- [x] Offline requires a reason, removes the task from U/W availability, and optionally revokes exactly the remaining quota.
- [x] Re-online requires positive allocation, reserves it before activation, and preserves historical distributed points.
- [x] No DELETE method is registered or documented as available.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth -run 'Task|Rule|Publish|Offline|Online'`
- `go test -tags=integration ./internal/service/growth -run 'TaskLifecycle'`
- `go test ./internal/server/routes -run GrowthTask`
- Inspect the generated route list and RBAC coverage output.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit tests | `go test -tags=unit ./internal/service/growth ./internal/server/routes -run 'Task|Budget' -count=1` | PASS: task validation, rule/version view model, RBAC route declarations, and no task DELETE route. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run TaskLifecycle -count=1 -v` | PASS on MySQL 8.4.9: atomic creation/allocation, duplicate/zero/insufficient rejection, two immutable published versions, offline revoke, re-online allocation, code/task history preservation. |
| Generation and regression | `make generate && go test ./internal/rbac ./internal/server/... ./internal/handler/growth -count=1` | PASS: Ent/Wire regeneration, RBAC seed parity, route coverage, middleware and server tests. |
| Service | `backend/internal/service/growth/task_service.go` | Transactional task/rule lifecycle and quota composition. |
| Handler/routes | `backend/internal/handler/growth/task_handler.go`, `backend/internal/server/routes/admin.go` | Eight task APIs using `growth.read`/`growth.manage`; no DELETE registration. |
| Inventory | `RBAC_ENDPOINT_INVENTORY.md` | Task routes registered under growth module. |

## Execution Notes

Publishing a replacement rule for an OFFLINE task keeps it OFFLINE until `POST /tasks/:id/online` reserves fresh quota. This resolves the lifecycle/document wording conflict while preserving the required explicit re-online allocation step.
