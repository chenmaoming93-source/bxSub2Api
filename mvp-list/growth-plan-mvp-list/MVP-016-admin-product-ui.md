# MVP-016: Manage exchange products, versions, and stock

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: This is a focused table/card workbench over already implemented product APIs, with one reusable versioned product drawer and stock adjustment flow.
- Dependencies: `MVP-005`, `MVP-014`

## Outcome

An operator can browse growth products in table or card mode, create/edit a draft version, publish, pause/offline, and adjust stock from the product tab.

## Context

The prototype requires product type, points/version, stock warning states, per-user limit, fulfillment mode, status, and edit/stock actions. Version changes preserve historical order snapshots.

## In Scope

- Implement `ProductsTab.vue`, `ProductFormDrawer.vue`, and reusable version badge/form controls.
- Add search/type/status filters, table/card segmented view, version history display, publish and offline confirmation, stock adjustment reason, and sold-out/offline visual states.
- Validate Token issuance amount and fulfillment mode before publish; provide clear invalid-version feedback from the API.
- Add local row/card refresh after mutation and focused component tests.

## Out of Scope

- No product delete, order creation, fulfillment worker, or new rich-text/image upload system.
- No new UI library or global styling changes.

## Implementation Notes

- Use the existing product icon/benefit schema without embedding arbitrary scripts from benefit JSON.
- The stock `-1` display is `不限`, `0` is `已售罄`, and finite values `<=10` receive warning treatment.

## Acceptance Criteria

- [x] Table and card modes render the same authoritative product data and switch without page reload.
- [x] Create/edit/publish produces the expected draft/current version feedback and retains older versions in history.
- [x] Offline and stock actions require the designed confirmation/reason and update the visible row.
- [x] Publishing an invalid Token product is blocked with an actionable error.
- [x] Empty, error, permission, sold-out, and offline states are covered; 375px layout has no page-level horizontal overflow.

## Verification Plan

- `cd frontend && npm run test:run -- src/components/growth src/views/growth/products`
- `npm run typecheck`
- `npm run lint:check -- src/components/growth src/views/growth`
- `npm run build`
- Manually inspect `/admin/growth?tab=products` in table/card modes and mobile/light/dark layouts.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Browser E2E | Create and publish `token_1m` | PASS: DRAFT showed pending v1; publishing produced current v1 with 50 points, stock 100, limit 3 and automatic fulfillment; table/card views showed identical authoritative values. |
| Validation | Set Token amount to `0` in product editor | PASS: actionable i18n warning appeared and Save was disabled; backend `PRODUCT_PUBLISH_INVALID` is mapped to the same message. |
| Visual | [card/table dark](../evidence/screenshots/admin-products-1440-dark.png), [1440 light](../evidence/screenshots/admin-products-1440-light.png) | PASS: same-data table/card views, pending/current badges, stock warning, status and permission-aware actions. Responsive checks found no page-level overflow. |
| Frontend gates | `npm run typecheck`, focused tests, focused `eslint`, `npm run build` | PASS. |
| Backend version/inventory tests | `GROWTH_MYSQL_TEST_DSN=... go test -tags=integration ./internal/service/growth -count=1` | PASS: immutable historical versions, publication validation, stock ledger and inventory invariants. |

## Execution Notes

Formal create/edit, version history, stock, publish and offline flows are complete. Failed refunds now restore limited inventory (`available + 1`, `reserved - 1`) and the integration regression asserts the restored values.
