# MVP-017: Operate exchange orders and user point accounts from admin

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: Two coupled operational tabs include failed-order triage, retry/refund, user drilldown, high-risk adjustment, filters, CSV, and local state updates.
- Dependencies: `MVP-012`, `MVP-013`, `MVP-014`, `MVP-016`

## Outcome

An operator can find and resolve failed exchanges and an authorized administrator can inspect a user's points, ledger, and exchanges and perform an audited adjustment from the admin module.

## Context

The prototype has an overdue-failure banner, order filters/detail timeline/retry/refund, and a user table with a three-part drawer and adjustment dialog. The page design requires current-filter CSV export and hides adjust actions without `growth.adjust`.

## In Scope

- Implement `OrdersTab.vue`, `ExchangeOrderDrawer.vue`, `UserPointsTab.vue`, `UserPointsDrawer.vue`, and `PointsAdjustDialog.vue`.
- Add order search/status/product/date filters, failure-age warning, detail timeline, retry/refund reason and second confirmation, and current-filter CSV.
- Add user search/sort/point-range filters, row drawer with wallet/ledger/exchange views, and before/after adjustment confirmation.
- Enforce page `growth.read`, order `growth.manage`, and adjustment `growth.adjust` at the UI and API boundary.
- Add component tests and local regression of tab query behavior.

## Out of Scope

- No admin placing exchanges for users, direct ledger edits, batch refunds, or CSV export of unfiltered server-wide data.
- No automatic order recovery from the browser.

## Implementation Notes

- Retry is visible only for eligible failed orders and must state that points will not be charged twice.
- Refund and adjustment are destructive confirmations; keep the API as the final authority even when UI prevalidation passes.
- Product click from an order may open the user drawer, but must preserve the current filtered order list.

## Acceptance Criteria

- [x] Orders expose the required filters, overdue-failure banner, timeline, retry, refund, and current-filter CSV.
- [x] Retry never duplicates point deduction; refund requires reason and displays the returned refunded status/identifier.
- [x] User table search/filter/sort and three-part drawer use authoritative growth APIs.
- [x] `growth.adjust` absence hides adjustment actions; an allowed adjustment shows before/after balance and refreshes the user row after success.
- [x] Both tabs preserve URL query state, handle empty/error/permission states, and remain usable at 375px without page overflow.

## Verification Plan

- `cd frontend && npm run test:run -- src/components/growth src/views/growth/admin`
- `npm run typecheck`
- `npm run lint:check -- src/components/growth src/views/growth`
- `npm run build`
- Manually exercise failed order retry/refund, overdue banner, user filters, and adjustment with and without `growth.adjust`.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Browser order E2E | Exchange `token_1m`, wait for fulfillment worker, inspect order | PASS: order moved through CREATED/FULFILLING to REFUNDED; detail showed both transfer IDs, failure reason and refund step; UI refreshed from authoritative detail. |
| Browser user E2E | Search user, open details, adjust +100 | PASS: before/after preview showed `0 → 100`, confirmation warning appeared, row and ledger refreshed locally, and toast reported the exact balance transition. |
| Filters/export | Orders and users tabs with status/product/date/search/range/sort controls | PASS: URL query state is preserved; CSV builder consumes the current filtered result set. |
| Responsive visual | [orders dark](../evidence/screenshots/admin-orders-1440-dark.png), [orders light](../evidence/screenshots/admin-orders-1440-light.png), 375 browser checks | PASS: no page-level horizontal overflow; table regions scroll locally while controls stack. |
| Backend financial tests | Full growth MySQL integration suite | PASS: outbox retry/refund/manual operations, no duplicate debit, idempotent refund, failure recovery, user list includes zero-wallet users, reconciliation and audit. |
| Frontend gates | `npm run typecheck`, focused tests, focused `eslint`, `npm run build` | PASS. |

## Execution Notes

Both operational tabs are complete with URL-backed filters, formal drawers/dialogs, current-result CSV export and permission-aware actions. The user list now uses `LEFT JOIN growth_wallets`, so a user without an existing wallet can receive the first audited adjustment.
