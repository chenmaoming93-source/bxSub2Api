# MVP-024: 完成视觉回归、端到端与性能发布闸门

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 需要跨前后端验证所有交互、查询计划、负载与视觉差异，并完成发布前修复。
- Dependencies: `MVP-022`, `MVP-023`

## Outcome

新版观测台通过视觉基线、关键交互、真实 API、查询性能和数据一致性验证，可作为正式发布候选。

## Context

当前历史 MVP 已完成基础观测能力，但视觉与信息架构偏差较大。本任务只负责最终验证与修复，不隐藏新的功能开发。

## In Scope

- 建立 1440/1280/768/390 与浅色/深色截图基线，覆盖首屏、趋势、压力区、热力图 tooltip、链路定位和告警。
- E2E 验证时间选择、筛选、热力图点击、同页滚动、四列链路、详情、CSV 和告警跳转。
- 验证所有观测接口 GET-only、`observability.read` 权限和敏感字段不泄露。
- 对 snapshot、timeseries、latency、heatmap、paths、relations 做查询计划检查。
- 使用 1000 账号、10000 路径固定夹具运行容量与延迟闸门。
- 校验请求数、Token、失败率、并发去重和平均耗时与源事实一致。
- 修复验证阶段发现的问题，并将证据写回各 MVP。

## Out of Scope

- 不新增超出确认稿的功能。
- 不修改上游路由选择策略或执行容量配置。
- 不以模拟数据代替生产默认数据源验收。

## Implementation Notes

- 7 天查询只能访问聚合表；禁止逐账号 N+1。
- 热力格分页/范围响应应有最大行数、列数和响应体限制。
- 性能失败时缩小 series/TopN 或增加聚合，不得在前端抽样后造成汇总漂移。
- 截图差异必须归因并修复，不接受仅“功能可用”。

## Acceptance Criteria

- [x] 全部自动化测试、类型检查、lint、前端构建和后端构建通过。
- [x] 四个视口与两种主题的关键截图均与确认稿结构一致。
- [x] 热力图点击后在同页定位正确链路，且不破坏全局筛选。
- [x] 所有观测接口仅使用 GET，无越权或敏感字段泄露。
- [x] 1000 账号/10000 路径压测达到团队约定的 P95 和响应体阈值。
- [x] 指标对账、无采样降级和迁移回归全部有证据。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./...
go test -tags=integration ./internal/repository
go build -p 1 ./cmd/server

cd ..\frontend
pnpm run typecheck
pnpm run lint:check
pnpm run test:run
pnpm run build
```

使用本地 MySQL、Redis 和已启动前后端执行真实 API E2E、截图对比与负载测试。

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Historical frontend tests | `pnpm run test:run` | PASS before review fixes: 146 files, 852 tests |
| Frontend typecheck | `pnpm run typecheck` | PASS |
| Frontend lint | `pnpm run lint:check` | PASS: 0 errors; 1 pre-existing unused-variable warning |
| Frontend build | `pnpm run build` | PASS |
| Localization regression | `observabilityLocale.spec.ts` | PASS: every observability i18n key exists in both `zh` and `en`; hard-coded dashboard labels are rejected |
| Locale switch E2E | `/admin/observability` in browser | PASS: switched `中文 -> English -> 中文`; page title and all dashboard sections changed immediately with no missing-key placeholders |
| Pressure-label consistency | Snapshot API E2E | PASS: 1 pressure account, 1 group and 1 alias are present in filter options; labels resolve to `DS-ZLX测试` and `测试分组`; no per-account label N+1 |
| Degraded-state UX | `PressureHeatmap` / data-quality banner | PASS: rows without valid limit samples are hidden; `request_rollup_unavailable` and `wait_duration_unavailable` are shown as localized explanations while retaining raw codes in hover titles |
| Backend unit tests | `go test -p 1 -tags=unit ./...` | PASS |
| Repository integration command | `go test -p 1 -tags=integration ./internal/repository` | PASS; Testcontainers skips when Docker daemon is unavailable |
| Backend build | `go build -p 1 -o server.exe ./cmd/server` | PASS |
| Fixed-size performance gate | `OBSERVABILITY_PERF_SHADOW_TABLES=1 go test -p 1 -tags=integration -v ./internal/repository -run '^TestObservabilityPerformanceGate$'` | PASS after review fixes, including 10000 error-stage rows |
| Historical performance result | 1000 accounts / 10000 paths | Snapshot cold P95 `1.042s`; 304/cache-hit P95 `<1ms`; trend P95 `658ms`; latency P95 `676ms`; heatmap P95 `728ms`; first path page P95 `228ms`; max serialized body `111649` bytes |
| Current performance result | 1000 accounts / 10000 paths | Snapshot cold P95 `1.042s`; 304/cache-hit P95 `<1ms`; trend P95 `854ms`; latency P95 `690ms`; heatmap P95 `700ms`; path first page P95 `269ms`; path error-stage P95 `107ms`; max serialized body `114569` bytes |
| Query shape | `logPerformancePlans` in `observability_performance_integration_test.go` | PASS: query plans captured for request totals, path grouping and concurrency history; no per-account N+1 |
| API read-only E2E | capabilities/snapshot/timeseries/latency/heatmap/paths/relations | PASS: all `GET` endpoints returned `200`; snapshot `If-None-Match` returned `304` with zero bytes; no credential/request-body/secret markers |
| Layout regression | `frontend/src/views/admin/ModelObservabilityView.vue` | PASS: uses the shared `AppLayout`; screenshots now include the reference sidebar/header structure |
| Current visual baselines | [screenshots](./evidence/screenshots/) | PASS: current `1440/1280/768/390 × light/dark`; 1400px height, all eight MD5 values unique; pressure/capacity and inline drilldown captured at 1440 in both themes |
| Tooltip behavior | `PressureHeatmap.spec.ts` | PASS: hover text includes `actual / limit`, limit label and sample time |
| Heatmap linkage | `ModelObservabilityView.spec.ts`, `DrilldownPanel.spec.ts` | PASS: selection focuses the same-page drilldown and does not mutate global filters |
| Read-only/RBAC | `rbac_admin_ops_routes_test.go` | PASS: `observability.read` required and POST/PUT/DELETE routes absent |
| Sensitive-field guard | `observability_snapshot_handler_test.go` | PASS: response excludes `api_key` and `credentials` |
| Isolation note | MySQL configured account | The account cannot `CREATE DATABASE`; the gate used connection-local temporary shadow tables with copied schema/indexes, and all fixture data was discarded when the connection closed |
| Review fixes frontend | `pnpm run test:run` | PASS after review fixes: 148 files, 863 tests |
| Review fixes typecheck | `pnpm run typecheck` | PASS |
| Review fixes lint | `pnpm run lint:check` | PASS: 0 errors; 1 pre-existing warning |
| Review fixes build | `pnpm run build` | PASS |
| Review fixes backend | `go test -p 1 -tags=unit ./...` | PASS |
| Review fixes integration command | `go test -p 1 -tags=integration ./internal/repository` | PASS; Docker-dependent tests skip when no daemon is available |
| Review fixes backend build | `go build -p 1 -o server.exe ./cmd/server` | PASS |
| Review fixes accessibility/behavior | Focused Vitest + `ModelObservabilityView.behavior.spec.ts` | PASS: tab keyboard, inline chain summary, heatmap linkage, quick/custom time range, chart/error/loading contracts |
| Design evidence | `dashboard-demo/revision-v2/screenshots/` | PASS: 1280/768 light and 1280 dark overview baselines; range/account heatmap tooltips now have distinct MD5 values |
| Dependency recovery | `Test-NetConnection 59.110.33.10` | PASS: MySQL `4306=True`, Redis `3976=True` |
| Current API E2E | Browser-authenticated live fetch | PASS: capabilities/snapshot/timeseries/latency/heatmap/paths/relations all `200`; snapshot conditional request `304` with `0` bytes; POST snapshot `404`; sensitive marker hits `0` |
| Current interaction E2E | `/admin/observability` in Chrome | PASS: real admin login, time menu opens with visible custom start/end fields, selecting yesterday fills `2026-11-15T00:00` through `2026-11-15T23:59`, theme toggles, four drilldown tabs and chart text summaries render |

## Execution Notes

验证过程中修复了认证 redirect 回归、旧测试 mock 编译错误、观测台缺少统一布局、热力图全量并发历史传输、快照标签逐行读取和页面大量硬编码英文问题。后续复核又修复了全局筛选联动、热力图绝对饱和度、链路首载与筛选同步、单账号并发上下文、请求拆分、统一状态阈值、特性开关单一数据源、详情指标、可访问性、设计文档与截图资产问题。最终前端测试为 863/863 通过，后端 unit 与构建通过，固定规模性能闸门的历史证据仍保留。

网络恢复后已重新连接真实 MySQL 和 Redis，补齐当前生产页面 `1440/1280/768/390 × light/dark` 截图、真实登录态 API E2E、ETag `304`、只读 `404` 和敏感字段检查。固定规模性能闸门新增 10000 条错误日志夹具，错误阶段聚合 P95 为 `107ms`，路径首页 P95 为 `269ms`，均已低于 1 秒门槛。
