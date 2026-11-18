# Growth MVP Release Checklist

- Date: `2026-11-18` (`Asia/Shanghai`)
- Scope: sub2api M plane, W plane, U plane, shared growth accounting, admin UI and Web user UI.
- AIWork application source: not modified.

## Runtime Environment

| Component | Result |
|---|---|
| MySQL | Native MySQL `8.4.9` on `127.0.0.1:3307`; integration tests created and removed isolated temporary databases. |
| Backend | Latest local server build started on `0.0.0.0:8080`; `/health` returned HTTP 200. |
| Frontend | Vite `5.4.21` on `127.0.0.1:5173` for visual and browser workflow verification. |
| Docker | Docker Desktop engine was unavailable; native MySQL and configured Redis were used. No MVP acceptance criterion required the Docker engine itself. |

## Automated Verification

| Command | Result |
|---|---|
| `go test -p 1 -tags=unit ./... -count=1` | PASS after serializing package builds to fit Windows memory limits. |
| `GROWTH_MYSQL_TEST_DSN=root@tcp(127.0.0.1:3307)/?parseTime=true go test -tags=integration ./internal/service/growth -count=1` | PASS: budget, tasks, products, transfer, check-in, completion, orders, fulfillment, recovery, admin adjustment and reconciliation. |
| `go test -tags=integration ./internal/repository/growthmysqltest -count=1` | PASS: archived growth schema applies to MySQL 8.4. |
| `go test ./internal/server/... ./internal/handler/growth ./internal/service/growth -count=1` | PASS. |
| `go test -tags=unit ./internal/handler/growth/... -run Contract -count=1` | PASS. |
| `go test -tags=unit ./internal/service/growth/crypto -run Contract -count=1` | PASS: committed SM2 v4 envelope fixture decrypts through the server contract. |
| `E2E_MOCK=true go test -tags=e2e -v -timeout=300s ./internal/integration/... -count=1` | PASS: integration package completed; external provider cases skipped because no Claude/Gemini API keys were configured. |
| `npm run typecheck` | PASS. |
| `npm run test:run` | PASS: 152 files / 885 tests. |
| `npm run test:run -- src/components/growth src/views/growth` | PASS: 3 files / 13 tests. |
| Focused growth `eslint` | PASS: no errors. |
| `npm run build` | PASS; only existing dynamic-import and bundle-size warnings. |
| `golangci-lint v2.9 run --build-tags=unit --new-from-rev=HEAD ./...` | PASS: 0 issues on this change set. |

## Browser Journey

1. Initialized a `100,000`-point total budget.
2. Created `daily_checkin` with `20,000` allocated quota; confirmed pending v1, published v1 and inspected immutable rule history.
3. Created `token_1m`, confirmed pending v1, rejected zero token issuance, then published the valid v1 with stock, price, purchase limit and automatic fulfillment.
4. Performed an audited `+100` adjustment from `0` to `100`; verified local row refresh, ledger update and audit-visible success.
5. Redeemed the Token product from the Web user page. The server returned a balance of `50`; the browser displayed the pending order and stopped after the bounded five polling attempts.
6. The fulfillment worker exhausted retries because no production Token adapter is configured, then automatically refunded. The order showed status `REFUNDED`, debit transfer, refund transfer and failure reason; the wallet returned to `100`.
7. Verified user ledger rows for adjustment, exchange debit and refund, including cursor/reference fields and type filtering.
8. Audited browser resource requests: no Web check-in or task-completion requests were made.
9. Verified invalid admin tab normalization and URL-backed tab state.
10. Checked 375, 768 and 1440 widths in light/dark themes; all pages had `scrollWidth === innerWidth`.

## Evidence

- Admin desktop: `admin-budget-1440-{light,dark}.png`, `admin-tasks-1440-{light,dark}.png`, `admin-products-1440-{light,dark}.png`, `admin-orders-1440-{light,dark}.png`.
- Admin responsive: `admin-budget-{375,768}-{light,dark}.png`.
- User desktop: `user-{overview,store,ledger}-1440-{light,dark}.png`.
- User responsive: `user-{overview,store,ledger}-{375,768}-{light,dark}.png`.
- Screenshot directory: `mvp-list/evidence/screenshots/`.

## Known Release Disposition

- Full-repository `golangci-lint run ./...` still reports pre-existing findings outside this delivery, mostly unused compatibility symbols and unchecked close operations in unrelated repositories, services, handlers and tests. The changed-line gate for this delivery passes with 0 issues.
- The local automatic Token adapter is intentionally absent; exchanges therefore fail safely and refund instead of silently granting quota. Production release requires the approved quota/subscription adapter to be wired by the owning service integration.
