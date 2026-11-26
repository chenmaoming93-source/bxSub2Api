# MVP-005: Manage versioned exchange products, inventory, and limits

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: Product pricing history, three catalog types, finite/unlimited inventory, optimistic reservation metadata, and Token issuance configuration form a complete operational slice.
- Dependencies: `MVP-002`

## Outcome

An operator can create, version, publish, pause/offline, and stock growth products while existing orders retain the product version and benefit snapshot they purchased.

## Context

Product versions are append-only after order reference. The prototype displays Token quota, subscription, and physical products, with stock, per-user limits, fulfillment mode, and sold-out/offline states.

## In Scope

- Add `GET|POST /admin/growth/products`, `PUT /admin/growth/products/:id`, `POST /admin/growth/products/:id/publish`, `POST /admin/growth/products/:id/stock`, and `POST /admin/growth/products/:id/offline`.
- Support `TOKEN_QUOTA`, `SUBSCRIPTION`, and `PHYSICAL`, active/draft/offline status, point price, benefit JSON, per-user limit, stock, and token quota per issuance.
- Write version rows, optimistic inventory data, inventory ledger entries, RBAC `growth.read`/`growth.manage`, and audit records.
- Provide user-readable eligibility fields needed by later U/W reads.
- Add service and route tests for version immutability and stock arithmetic.

## Out of Scope

- No exchange order or wallet debit.
- No actual Token/subscription/physical fulfillment and no product deletion.

## Implementation Notes

- Use a draft version for changed price or benefits; published versions are immutable and referenced by order snapshots.
- Treat stock `-1` as unlimited and `0` as sold out but still listable.
- Validate Token products include a positive token amount and a fulfillment adapter before activation.
- Inventory adjustments must write `growth_inventory_ledger` with operator, reason, and before/after values.

## Acceptance Criteria

- [x] Creating a product produces an initial draft version, and publishing makes exactly that version current.
- [x] Editing price/benefits after publication creates a new draft and leaves old versions unchanged.
- [x] Stock adjustment rejects negative finite stock, writes one inventory ledger row, and cannot underflow under concurrency.
- [x] Token products without a positive issuance amount or fulfillment mode cannot be published.
- [x] List responses expose version, price, stock/limit, benefit summary, fulfillment mode, and active status required by admin and user cards.

## Verification Plan

- `cd backend && go test -tags=unit ./internal/service/growth ./internal/server/routes -run Product`
- `go test -tags=integration ./internal/service/growth -run 'ProductVersion|Inventory'`
- `go test ./internal/server/...`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Unit tests | `go test -tags=unit ./internal/service/growth ./internal/server/routes -run Product -count=1` | PASS: product validation/view model and growth product route/RBAC declarations. |
| MySQL integration | `GROWTH_MYSQL_TEST_DSN='root@tcp(127.0.0.1:3307)/?parseTime=true' go test -tags=integration ./internal/service/growth -run 'ProductVersion|Inventory' -count=1 -v` | PASS on MySQL 8.4.9: draft/current versions, immutable history, Token publish guard, stock ledger, underflow rejection, concurrency and list fields. |
| Generation/regression | `make generate && go test ./internal/server/... ./internal/handler/growth ./internal/rbac -count=1` | PASS. |
| Service | `backend/internal/service/growth/product_service.go` | Transactional product version, inventory, stock ledger, audit and catalog reads. |
| Handler/routes | `backend/internal/handler/growth/product_handler.go`, `backend/internal/server/routes/admin.go` | Eight M-face product endpoints; no DELETE route. |
| Inventory | `RBAC_ENDPOINT_INVENTORY.md` | Product endpoints registered. |

## Execution Notes

Product codes are immutable through the API. Product versions remain append-only after publication; stock mode `UNLIMITED` is exposed as `-1` while finite inventory tracks available/reserved/sold.
