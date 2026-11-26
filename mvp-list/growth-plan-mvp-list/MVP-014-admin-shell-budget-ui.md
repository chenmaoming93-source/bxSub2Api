# MVP-014: Open the growth admin module with the budget and quota workbench

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: The first vertical frontend slice includes two routes, navigation/permission/i18n registration, shared API types, page shell, and the data-dense budget tab.
- Dependencies: `MVP-003`

## Outcome

An authorized administrator can navigate to `/admin/growth`, switch between five URL-backed tabs, and manage the total budget and task quota from a working budget tab that matches the prototype.

## Context

The page design requires two routes overall and four low-intrusion registration points. This slice owns the admin route; the user route is added by `MVP-018`. The budget tab needs four KPI cards, task allocation table, increase/reduce dialogs, add/revoke quota dialogs, budget history drawer, safe reason validation, and empty/error/permission states.

## In Scope

- Add the lazy `/admin/growth` route with a complete admin shell and placeholder-free tab routing; later tabs may display their bounded empty state until implemented.
- Add admin sidebar item, both permission-matrix entries, `growth` i18n namespaces, and `frontend/src/api/growth/{admin,user,types,index}.ts`.
- Implement `AdminGrowthView.vue`, `GrowthTabs.vue`, `PointsAmount.vue`, `BudgetUsageBar.vue`, budget dialogs/drawer, and `BudgetTab.vue`.
- Add query synchronization, default invalid-tab fallback, `growth.read` page guard, `growth.manage` action visibility, local update after mutations, and dark/responsive behavior.
- Add frontend unit tests and a light/dark 1440px visual check.

## Out of Scope

- No task, product, order, or user tab business implementation in this MVP.
- No new UI dependency, global CSS redesign, or hardcoded Chinese in Vue components.

## Implementation Notes

- Reuse existing `TablePageLayout`, cards, `BaseDialog`, confirmation/toast, pagination, skeletons, and empty/error components.
- The page header contains only refresh; tab-specific primary actions appear once in their tab toolbar.
- Show color bands at `<60%`, `60-85%`, and `>85%`; exhausted quota has a distinct rose state.

## Acceptance Criteria

- [x] The admin route is lazy-loaded, the admin menu item appears only with the matching permission, and unauthorized navigation shows the page-level 403 state.
- [x] `?tab=budget|tasks|products|orders|users` survives refresh/share, invalid values fall back to `budget`, and only the active tab loads its API.
- [x] Budget KPI cards equal server totals; unallocated equals `total - allocated`; all monetary values use tabular formatting.
- [x] Increase/reduce, add/revoke, and history interactions implement reason validation, safe confirmation, loading-disabled submit, and local refresh.
- [x] At 375px and 768px the layout has no horizontal page overflow; at 1440px light/dark views match the supplied prototype hierarchy.
- [x] Typecheck, focused frontend tests, lint, and production build pass with no new `any` in growth API modules.

## Verification Plan

- `cd frontend && npm run test:run -- src/components/growth src/views/growth src/api/growth`
- `npm run typecheck`
- `npm run lint:check -- src/components/growth src/views/growth src/api/growth`
- `npm run build`
- Manually inspect `/admin/growth?tab=budget` at 375px, 768px, and 1440px in light and dark modes.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Browser E2E | `/admin/growth?tab=budget` with initialized 100,000-point budget | PASS: budget KPI totals and unallocated value refreshed locally after mutation; add-budget dialog enforced positive amount, five-character reason and loading-disabled submit. |
| URL behavior | `/admin/growth?tab=invalid` | PASS: URL automatically normalized to `?tab=budget`; valid tab navigation remained shareable/refreshable. |
| Responsive visual | [375 light](../evidence/screenshots/admin-budget-375-light.png), [375 dark](../evidence/screenshots/admin-budget-375-dark.png), [768 light](../evidence/screenshots/admin-budget-768-light.png), [768 dark](../evidence/screenshots/admin-budget-768-dark.png), [1440 light](../evidence/screenshots/admin-budget-1440-light.png), [1440 dark](../evidence/screenshots/admin-budget-1440-dark.png) | PASS: no page-level horizontal overflow (`scrollWidth === innerWidth` at 375 and 768); hierarchy and color bands hold in both themes. |
| Frontend gates | `npm run typecheck`, `npm run test:run -- src/components/growth src/views/growth`, focused `eslint`, `npm run build` | PASS: typecheck/build succeeded, 13 focused tests passed, focused lint had no errors. |
| Implementation | `frontend/src/views/growth/AdminGrowthView.vue`, `frontend/src/views/growth/admin/BudgetTab.vue`, `frontend/src/components/growth/*` | PASS: lazy route, permission guard fallback, URL-backed tabs, formal dialogs, KPI cards, quota table and budget history drawer. |

## Execution Notes

Implemented with real MySQL data and browser verification. Unauthorized growth routes intentionally remain mounted long enough for the page-level 403 component to render; API calls are skipped when the page permission is absent.
