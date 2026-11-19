# MVP-018: 输出可解释的采样并发热力格

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 改动集中在并发聚合与热力图查询，但必须保证分子、分母和采样时刻来自同一点。
- Dependencies: `MVP-004`, `MVP-006`

## Outcome

采样并发热力格返回“实际并发、并发上限、采样时刻、时间桶、可用状态”，支持全局、分组、别名和账号维度，并在单账号筛选时使用该账号的生效上限作为唯一分母。

## Context

当前热力图只返回单一 `value`，`PressureHeatmap.vue` 也只能展示数值。分钟并发表已有 `current_peak`、`effective_limit`、`first/last_sampled_at`，但峰值和上限可能来自不同采样点，不能直接配对计算。

## In Scope

- 在并发分钟聚合中保存“最高占用率采样”的 `actual_concurrency`、`concurrency_limit`、`sample_at`。
- rollup 在合并多个分钟时比较已保存的最高占用率，保留同一采样点的三字段。
- 扩展热力图 cell 契约及前端 TypeScript 类型，保留原 `value` 作为比率或兼容字段。
- 分组/别名聚合对唯一账号并发上限去重求和；单账号筛选时分母固定为该账号生效上限。
- 采样并发 tooltip 展示对象、时间桶、峰值采样时刻和 `实际并发 / 账号并发上限`。
- 无采样时返回 `available=false` 与明确 warning，不返回伪造 0 值。

## Out of Scope

- 不改变 token_rate/error_rate 热力图的业务口径。
- 不在本任务实现热力图视觉或点击滚动。
- 不根据历史平均值估算缺失采样。

## Implementation Notes

- 如果继续使用现有聚合表，应增加幂等字段迁移；清空或重建聚合时保留可重算能力。
- 查询按账号、分组合并后再计算行级分母，避免先按别名聚合造成账号容量重复。
- 热力查询优先命中 `(window_seconds, bucket_start)` 与维度索引，单次只访问聚合表。
- 单元测试必须覆盖“峰值与上限来自同一采样点”和“单账号筛选分母不放大”。

## Acceptance Criteria

- [x] 每个可用热力格同时返回 `actual_concurrency`、`concurrency_limit`、`sample_at` 和时间桶。
- [x] 单账号筛选时所有行的分母都等于该账号对应采样时刻的生效上限。
- [x] 同一账号被多路径复用时，范围分母只计一次。
- [x] 缺采样格返回 unavailable 和稳定 warning，不渲染 0。
- [x] 7 天热力图只访问聚合表并通过查询计划检查。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/service/observability ./internal/repository ./internal/handler/admin
go test -tags=integration ./internal/repository
go build -p 1 ./cmd/server

cd ..\frontend
pnpm run test:run -- src/components/observability/__tests__/PressureHeatmap.spec.ts src/api/admin/__tests__/observability.spec.ts
pnpm run typecheck
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Paired sample contract | `backend/internal/service/observability/contract.go` | `HeatmapCell` now returns actual concurrency, concurrency limit, sample time, ratio and availability |
| Sample accumulator | `backend/internal/service/observability/concurrency_accumulator.go` | Tracks the highest-ratio sample and keeps actual/limit/time from that same point |
| Rollup | `backend/internal/repository/observability_concurrency_history_repo.go` | Minute/5-minute/hour rollups retain the paired peak-ratio sample instead of combining independent maxima |
| Migration | `backend/sqlArchiving/180_add_concurrency_peak_ratio_sample.sql` | Idempotently adds paired peak-ratio columns |
| Query semantics | `ObservabilityQueryService.concurrencyHeatmap` | Account rows are used only for account grouping; route rows are used for group/route grouping; single-account denominator stays the account limit |
| Frontend contract | `frontend/src/api/admin/observability.ts` | Heatmap cell TypeScript type includes actual/limit/sample time |
| Accumulator test | `TestConcurrencyAccumulatorWritesOnlyAtMinuteBoundary` | PASS: peak pair is 5/10 at the exact sample timestamp |
| Integration | `go test -tags=integration ./internal/repository` | PASS: migration rerun, wait fields and paired peak-ratio fields survive minute rollup |
| Focused tests | Observability, admin handler and route unit suites | PASS |
| Frontend | Typecheck, heatmap/API tests, production build | PASS |

## Execution Notes

现有 `current_peak / effective_limit` 不能直接组合成“最高压力率”，必须保存配对后的采样点。

聚合与查询均只访问 `observability_concurrency_bucket`，没有回退到原始请求表。缺少采样时保留 `available=false` 和 `concurrency_samples_unavailable` warning。
