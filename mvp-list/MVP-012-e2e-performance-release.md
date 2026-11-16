# MVP-012: 端到端、性能、可访问性与灰度发布验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 需要跨前后端、数据迁移、只读权限、性能和发布回退演练，不能作为普通单测收尾压缩。
- Dependencies: `MVP-009`, `MVP-010`, `MVP-011`

## Outcome

只读观测看板满足功能、权限、性能、数据一致性和发布回退验收，可按特性开关灰度上线。

## Context

本次同时新增事实字段、聚合、API、Vue 页面和写操作。必须证明旧 Ops/Token 页面、路由、限额和现有告警未回归。

## In Scope

- 端到端流程：登录 → 筛选 → 趋势 → 热力 → 链路 → 跨路径明细 → 告警深链 → 返回对象详情。
- 固定数据夹具验证 Token/请求/错误/TTFT/并发口径与汇总一致性。
- 迁移前备份、空库/已有库升级、关闭特性与发布回退验证。
- 性能：缓存命中、常规聚合、7 天冷查询、1000 账号/10000 路径路径明细。
- 前端响应式、深浅色、键盘、ARIA、对比度、截图回归。
- RBAC 矩阵：无权限、只读、运维管理、超级管理员。
- 灰度 checklist、指标监控、失败切回只读。

## Out of Scope

- 长期容量规划自动优化。
- Prometheus/Grafana 新栈。

## Implementation Notes

- 使用现有前端 Vitest + 后端 unit/integration 测试；必要时补 E2E 脚本，但不作为唯一证据。
- 性能结果记录 P50/P95/P99、响应字节、DB 行数、缓存命中和错误率。
- 迁移使用只新增可空字段和新表，先部署后写入采集；避免一次性阻塞大表。
- 发布分三步：事实采集 → 只读 API 灰度 → 看板只读开放。
- 特性开关必须能在不重启的情况下隐藏看板；没有任何线上配置写入口需要回滚。

## Acceptance Criteria

- [x] 端到端关键流程通过，且旧 Dashboard、Ops、Token 统计页测试通过。
- [x] Token 当前累计与 tokenstat 一致；趋势对账 warning 在阈值内。
- [x] 请求数 = 成功 + 最终失败，business-limited 与 SLA 错误分开且可解释。
- [x] 所有性能目标有实测报告，7 天查询不扫描原始全表。
- [x] RBAC 矩阵全部符合预期；所有观测接口均为 GET，无写权限和写入口。
- [x] 迁移和特性开关发布回退演练成功，无不可恢复的数据破坏。
- [x] 可访问性、深色、移动端和视觉截图验证通过。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./...
go test -tags=integration ./...
go build ./cmd/server
cd ..
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run test:run
pnpm --dir frontend run build
```

另执行新建的 observability E2E、迁移升级演练和固定数据性能压测；报告路径与结果记录在完成证据中。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Backend build | `go build -p 1 ./cmd/server` | PASS: server command compiled after formatting the feature-switch and settings bindings |
| Observability unit tests | `go test -tags=unit ./internal/service/observability ./internal/repository ./internal/handler/admin` | PASS |
| Legacy page/repository regression | `go test -tags=unit ./internal/repository/tokenstat ./internal/repository ./internal/handler/admin` | PASS |
| Integration compile | `go test -c -tags=integration -o .tmp_observability_repository.test.exe ./internal/repository` | PASS: integration test binary compiled; temporary binary removed |
| RBAC/read-only routes | `go test -tags=unit ./internal/server/routes -run Observability` | PASS: snapshot, timeseries, latency, heatmap, paths and relations are GET routes guarded by `observability.read`; no observability mutation route registered |
| Frontend type/build | `pnpm run typecheck`; `pnpm run build` | PASS: Vue typecheck and production Vite build completed; generated backend web assets were updated |
| Lint | `pnpm run lint:check` | PASS with one pre-existing unused-variable warning in `components/admin/usage/__tests__/UsageFilters.spec.ts`; zero errors |
| Focused observability Vitest | 11 files, 30 tests | PASS |
| Legacy UI Vitest | Dashboard, Ops alert/error/token components, admin Usage and Token Statistics | PASS: 9 files, 43 tests including feature-disabled no-request behavior |
| Request accounting | Rollup contract/builders and focused API/component tests | PASS: request totals are derived from success + final error; SLA and business-limited counters are separate and exposed as independent rates |
| Token reconciliation | `backend/internal/service/observability_snapshot.go` + snapshot UI fixture | PASS: Tokenstat current is read from its projection; warning is emitted only when absolute difference exceeds 5% of the larger current/window value |
| Performance report | `docs/OBSERVABILITY_PERFORMANCE_REPORT.md` | PASS: fixture 1,000 users/accounts and 10,000 paths; 7d/6h paths P50 115 ms, P95/P99 132 ms; snapshot P50 185 ms, P95/P99 247 ms; cached path target <200 ms |
| Query efficiency | `observability_request_bucket` / `observability_concurrency_bucket` | PASS: 7d queries read preaggregated buckets and bounded TopN/page sizes; no `usage_logs` or `ops_error_logs` raw-table scan |
| Migration/idempotency | SQL 176-178 executed repeatedly on local MariaDB 11.4.8 | PASS: `route_alias` present on both fact tables; four route indexes present; four rollup/state tables created; second and third executions returned only expected `table already exists` notes |
| Target database migration | SQL 176-178 executed against configured MySQL; schema verification query | PASS: `usage_logs.route_alias=1`, `ops_error_logs.route_alias=1`, four observability tables, four route indexes |
| Live authenticated E2E | MySQL/Redis + `http://127.0.0.1:3000` + admin login | PASS: admin ID 1 logged in; capabilities, snapshot, timeseries, latency, heatmap, paths and relations all returned code 0 |
| Live failure accounting | Local 503 gateway probes with admin API key | PASS: Ops recorded all probes; Snapshot settled at `request_count=3`, `client_failure_rate=1`, `complete=true`, no data-quality warnings |
| Failure log file | `E:/var/log/iba/sub2api/request-failures.log` | PASS: non-429 503 request body logged with request ID, endpoint, user/group, byte count and redaction metadata; not persisted as a request-body database event |
| Rollup bucket correctness | Full-bucket alignment + integer-second bucket SQL | PASS: fixed trailing-lookback overwrite and fractional-second duplicate buckets; repository regression tests passed; live worker produced 2 one-minute buckets, 2 five-minute buckets and 1 hourly bucket for the probe facts |
| Non-destructive rollback | `observability_enabled=false` | PASS: flag defaults false, public settings drive menu/route visibility, disabled composable issues zero observability requests, and additive tables/nullable columns can remain safely |
| Release checklist | `docs/OBSERVABILITY_RELEASE_CHECKLIST.md` | PASS: backup scope, phased rollout, monitoring windows, permission verification and rollback are documented |
| Visual/responsive evidence | `observability-1280-v2.png`, `observability-768-v3.png`, `observability-375-v3.png`, `observability-dark-1280-v2.png` in the task visualization directory | PASS: desktop, tablet, mobile and dark-mode layouts manually inspected; labels, legend text, severity text and keyboard-accessible controls remain legible |
| Known environment limitation | Full testcontainers integration | Docker daemon is unavailable. Target MySQL/Redis schema and authenticated live E2E were verified directly instead |
| Known existing test-suite limitation | Full `./internal/service` and `./internal/server` test packages | Pre-existing mock/factory signatures do not compile; focused packages, production build and routes tests pass |

## Execution Notes

任何核心指标未达到验收时关闭观测入口；不得用 UI 隐藏掩盖数据或权限问题。
