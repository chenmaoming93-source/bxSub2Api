# MVP-017: 补齐并发水位、排队等待与今日限流指标

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 当前只有瞬时排队深度，平均等待需要新增槽位等待采样，今日限流需要日累计事实与新快照契约。
- Dependencies: `MVP-004`, `MVP-005`

## Outcome

并发水位卡可以返回全局、分组用户级、模型账号级和路由级四层水位，同时给出当前排队请求、近 5 分钟平均等待和本地自然日限流次数。

## Context

`observability_concurrency_bucket` 当前保存 `waiting_peak`，但没有等待时长样本。`observability_request_bucket` 当前有业务限制数，但没有独立的限流次数字段。快照响应也没有容量摘要结构。

## In Scope

- 在并发占用获取链路记录真实等待时长样本，按账号和分钟聚合 `wait_sum_ms`、`wait_sample_count`。
- 在请求聚合中新增 `rate_limited_count`，定义并测试平台 429、业务限制和上游 429 的归类边界。
- 扩展快照响应 `CapacitySummary`：四层水位、当前排队请求、平均等待、今日限流次数和可用状态。
- 当前排队请求从 Redis 当前账号等待计数批量读取并按账号去重。
- 平均等待只使用近 5 分钟真实样本；无样本时返回 `unavailable`，不得返回 0。
- 今日限流次数从本地自然日 00:00 起按小时聚合求和，不随页面查询窗口变化。
- 为新增分钟字段增加幂等迁移、整分钟 rollup 和保留清理，不写逐请求明细到观测表。

## Out of Scope

- 本阶段不实现容量卡的最终视觉布局。
- 不提供容量修改、阈值应用、排队取消或人工释放入口。
- 不用队列深度乘固定系数推算平均等待。

## Implementation Notes

- 等待时长应在请求真正进入并发等待队列到获得槽位之间采样；普通未排队请求不进入平均等待样本。
- 快照只查询当前 Redis 计数与最新 5 分钟聚合，不扫描原始 `usage_logs` / `ops_error_logs`。
- 今日限流次数按配置时区日界线查询小时桶，复用现有时间索引。
- 新字段和索引应合并到现有 rollup 表，避免新增每请求明细表。
- 任何子指标缺失只降级本指标并产生稳定 warning，不能让整个并发卡清空。

## Acceptance Criteria

- [x] `capacity_summary` 同时返回四层水位及三个底部指标，字段单位明确。
- [x] 单账号筛选时账号级水位只计算一次，不因多条路径重复放大。
- [x] 平均等待由真实等待样本的分子和样本数计算，无样本时返回 unavailable。
- [x] 今日限流次数在切换 1 小时、7 天和自定义范围后仍为同一自然日累计值。
- [x] 1000 个账号快照不会产生逐账号 N+1 查询。
- [x] 迁移可重复执行，分钟聚合、小时 rollup 与清理测试通过。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/service ./internal/service/observability ./internal/repository ./internal/handler/admin ./internal/server/routes
go test -tags=integration ./internal/repository
go build -p 1 ./cmd/server
```

使用本地 MySQL/Redis 构造排队账号，验证等待样本、429 日和跨日切分。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Snapshot contract | `backend/internal/service/observability/contract.go` | Added four-level `capacity` summary, queue depth, average wait samples/value and local-day rate-limit count |
| Wait sampling | `backend/internal/service/observability/concurrency_accumulator.go`, `observability_concurrency_history.go`, `handler/gateway_helper.go` | Account wait duration is sampled around the real slot wait loop and aggregated without changing periodic concurrency sample counts |
| Batch capacity query | `ConcurrencyHistoryService.CollectCurrent` | Reuses `GetAccountsLoadBatch.WaitingCount`; removed per-account wait-count calls |
| Rate-limit rollup | `backend/internal/repository/observability_request_rollup_repo.go` | Final status 429 or business-limited errors increment `rate_limited_count`; hourly records support local-day sums |
| Migration | `backend/sqlArchiving/179_add_observability_capacity_metrics.sql` | Adds request rate-limit and concurrency wait fields through idempotent information_schema guards |
| API types | `frontend/src/api/admin/observability.ts` | Snapshot type exposes `capacity` levels and bottom metrics |
| Wait unit test | `TestConcurrencyAccumulatorRecordsAccountWaitWithoutChangingConcurrencySamples` | PASS: two waits sum to 600ms while periodic sample count/current peak remain unchanged |
| Capacity API test | `TestObservabilityCapacitySummaryUsesRealWaitAndStableDailyLimit` | PASS: 70/80 global level, queue 29, average wait 420ms, and 126 daily limits across 1h/7d/custom |
| Migration/rollup integration | `go test -tags=integration ./internal/repository` | PASS: new migration rerun idempotently; 429 count, next-day reset, wait sums and rollups verified |
| Focused backend | `go test -tags=unit ./internal/service/observability ./internal/handler/admin ./internal/server/routes` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Frontend | `pnpm run typecheck`; focused observability tests; `pnpm run build` | PASS |

## Execution Notes

若 429 与业务限制的产品口径不同，必须在实现前用现有错误分类表确定分子，不得在 UI 中混成一个无法解释的数字。

本轮按现有错误分类将平台最终 429 与 `is_business_limited=true` 的最终失败计入“今日限流次数”。平均等待按账号级真实排队样本计算，路由样本不重复累加。`go test -tags=unit ./internal/service` 仍受工作区既有 `AccountRepository.ExistsByName` 接口/mock 不一致影响；观测相关测试、集成和构建均通过。
