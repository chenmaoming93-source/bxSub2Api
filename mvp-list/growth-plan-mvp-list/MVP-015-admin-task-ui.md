# MVP-015: Manage tasks and rule versions from the admin workbench

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: This tab spans filters, quota-aware creation, rule drafting, publish/offline/re-online confirmation, version history, and several state-dependent actions.
- Dependencies: `MVP-004`, `MVP-014`

## Outcome

An operator can create, edit, publish, offline, re-online, allocate quota to, and inspect historical versions of growth tasks entirely from `/admin/growth?tab=tasks`, with no delete affordance.

## Context

The prototype shows task type, reward/version, period, limits, quota, status, pending version, and actions. The page design forbids task deletion and requires initial quota in the same create transaction.

## In Scope

- Implement `TasksTab.vue`, `TaskFormDrawer.vue`, `TaskRuleHistoryDrawer.vue`, and task-specific quota dialog reuse.
- Add search/type/status filters, DRAFT/ACTIVE/OFFLINE action sets, pending/new version badges, quota usage, pagination, and task-code immediate availability check.
- Implement publish with immediate/scheduled effective time, publish-new-version, offline with reason and optional revoke, re-online with required fresh quota, and adjacent-version difference highlighting.
- Add local row updates, conflict refresh after `TASK_CODE_DUPLICATED`/state errors, and tests for all state/action permutations.

## Out of Scope

- No delete button, `deleted_at`, task code editing after create, or completion/evidence configuration owned by AIWork.
- No direct points award or Web check-in control.

## Implementation Notes

- Disable create if unallocated budget is insufficient and offer a route to the budget tab.
- Display `已发布 vN` and `待发布 vN+1` independently so users know that saved edits are not live.
- Keep all labels in i18n and reuse the status badge/component vocabulary.

## Acceptance Criteria

- [x] Task table lists every required field and shows only actions valid for the current state.
- [x] Create requires positive code/title/rule and initial quota; duplicate code is detected before or at submit without leaving a row.
- [x] Editing an active task produces a pending version and does not change displayed effective reward until publish succeeds.
- [x] Offline and re-online complete all required confirmations/reasons/allocations and update status and quota locally after success.
- [x] Rule history displays immutable versions, effective ranges, operator, and adjacent differences.
- [x] Browser source and tests contain no delete action or DELETE call for tasks.

## Verification Plan

- `cd frontend && npm run test:run -- src/components/growth src/views/growth/admin`
- `npm run typecheck`
- `npm run lint:check -- src/components/growth src/views/growth`
- `npm run build`
- Manually exercise DRAFT, ACTIVE with pending rule, ACTIVE with exhausted quota, and OFFLINE task states at `/admin/growth?tab=tasks`.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Browser E2E | Create `daily_checkin`, allocate 20,000, publish v1, inspect history | PASS: DRAFT showed pending v1; publish changed the row locally to ACTIVE with 10 points, `1/day · 1/user`; history exposed v1, publisher and effective range. |
| State matrix | `frontend/src/components/growth/growthRules.ts` and focused tests | PASS: DRAFT/ACTIVE/ACTIVE+pending/OFFLINE actions are derived independently; no delete action exists. |
| Visual | [1440 light](../evidence/screenshots/admin-tasks-1440-light.png), [1440 dark](../evidence/screenshots/admin-tasks-1440-dark.png) | PASS: filters, quota usage, status, pending/published badges and state-dependent actions match the prototype hierarchy. |
| Frontend gates | `npm run typecheck`, focused tests, focused `eslint`, `npm run build` | PASS. |
| Backend lifecycle regression | `GROWTH_MYSQL_TEST_DSN=... go test -tags=integration ./internal/service/growth -count=1` | PASS: creation, immutable code, rule publishing/history, quota-preserving edit, offline/revoke and funded re-online paths. |

## Execution Notes

Create/edit drawers, quota allocation, publish/offline/re-online confirmations and rule-history comparison are complete. The task API intentionally has no DELETE route or UI affordance.
