# MVP-018: Deliver the read-focused user growth page and exchange center

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: The page combines overview, read-only check-in guidance, tasks, product state machine, exchange flow with bounded polling, and cursor ledger in responsive light/dark layouts.
- Dependencies: `MVP-008`, `MVP-009`, `MVP-011`, `MVP-012`, `MVP-014`

## Outcome

A logged-in Web user can see authoritative points and check-in status, browse tasks and products, redeem an eligible product, follow fulfillment briefly, and review cursor-paginated point history at `/growth`.

## Context

Web is JWT-authenticated and never signs in or completes tasks. The prototype has overview/store/ledger tabs. Sign-in belongs to AIWork; Web displays guidance only. Exchange success updates the balance from the server response.

## In Scope

- Implement the lazy `/growth` route, `UserGrowthView.vue`, `OverviewTab.vue`, `StoreTab.vue`, `LedgerTab.vue`, `PointsHero.vue`, `CheckinCard.vue`, and `ProductCard.vue`.
- Add URL-backed tabs, the user sidebar item, `growth.self.read`, overview one-call loading, task/AIWork guidance, product filters/sorting, detail drawer, eligibility states, exchange confirmation, and local balance refresh.
- Poll order status at most five times at two-second intervals only while visible and order is pending; stop on terminal state or page hide.
- Add cursor "load more", type filter/time range, inline ledger business reference, empty/error/loading states, i18n, and responsive tests.

## Out of Scope

- No Web check-in button or `/user/growth/checkins` request.
- No task completion button or evidence submission in Web.
- No local balance mutation, unbounded polling, or chart/trend feature.

## Implementation Notes

- Check-in text comes from server-owned `claimed_today`, `streak_days`, and `next_eligible_at`; if absent, hide the card.
- Product button states are `可兑换`, `积分不足 +差額`, `已售罄`, `已达限购`, and omit offline products.
- Use `PointsAmount.vue`: positive is teal, negative is rose; do not use stock-market red/green semantics.

## Acceptance Criteria

- [x] `/growth` loads overview with current/earned/spent/monthly points and a read-only check-in card containing no action button.
- [x] Page load and interaction send no Web check-in or completion request; all task actions guide to AIWork without optimistic balance changes.
- [x] Product cards show correct eligibility reasons and successful exchange updates balance from the response and reveals pending order state.
- [x] Pending order polling stops after success/failure/refund, page visibility loss, or five attempts.
- [x] Ledger filters and cursor load-more append without duplicate or skipped rows and show business reference details.
- [x] 375px, 768px, and 1440px layouts work in light/dark modes with no page-level horizontal overflow.

## Verification Plan

- `cd frontend && npm run test:run -- src/components/growth src/views/growth/user`
- `npm run typecheck`
- `npm run lint:check -- src/components/growth src/views/growth`
- `npm run build`
- Manually inspect `/growth?tab=overview|store|ledger` and verify the Network panel contains no check-in request.

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Browser E2E | `/growth?tab=overview/store/ledger` with authoritative MySQL data | PASS: hero metrics, read-only check-in, AIWork-only task guidance, eligible product, exchange confirmation, refunded order and three ledger entries matched the server. |
| Network audit | `performance.getEntriesByType('resource')` after load and interactions | PASS: only W-plane overview/tasks/products/ledger/exchanges requests appeared; no `/user/growth/checkins` or completion request was sent. |
| Exchange behavior | Confirm 50-point exchange and observe post-order worker result | PASS: balance changed from server-returned `current_balance`; pending status was shown; after worker failure the refund entry and restored balance appeared. |
| Polling tests | `src/views/growth/__tests__/pollExchange.spec.ts` | PASS: terminal state stops immediately, hidden page performs zero polls, and total attempts never exceed five with four 2-second waits. |
| Ledger tests | Cursor/type browser check and `appendUniqueByEntryId` unit test | PASS: REFUND filter returned only the refund row; cursor append removes duplicate entry IDs. |
| Responsive visual | `user-{overview,store,ledger}-{375,768,1440}-{light,dark}.png` under `mvp-list/evidence/screenshots` | PASS: 18 screenshots captured; automated width checks returned `scrollWidth === innerWidth` for all mobile/tablet pages. |
| Frontend gates | `npm run typecheck`, focused tests, focused `eslint`, `npm run build` | PASS. |

## Execution Notes

The full read-focused Web experience is complete. Exchange local balance refresh now also refreshes the ledger page immediately, preventing a stale ledger after a successful or refunded exchange.
