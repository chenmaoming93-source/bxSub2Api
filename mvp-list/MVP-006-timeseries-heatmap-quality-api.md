# MVP-006: 提供时序、耗时质量与热力图 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 要同时保证多维度汇总一致、分位数合并、桶对齐和缺失数据语义，联调成本高于普通 CRUD。
- Dependencies: `MVP-003`, `MVP-004`, `MVP-005`

## Outcome

前端可一次获得对齐的 Token/请求质量时序和真实压力热力图，所有维度切换保持相同汇总，缺样本时明确降级。

## Context

Tokenstat 仅支持总量和 D/W/M，耗时/失败来自两套事实表，当前 Redis 只有瞬时并发。设计稿的合成曲线与固定系数必须移除。

## In Scope

- 新增 `timeseries`、`latency-timeseries`、`heatmap` 三类 GET 接口或一个可复用 service 下的三个端点。
- 支持 `1h/5m`、`6h/15m`、`24h/1h`、`7d/6h` 固定窗口组合。
- Token/请求数来自分钟聚合；耗时/TTFT 从直方图合并；失败率拆分最终失败、业务限制和 SLA 错误。
- 热力图值类型支持 sampled_concurrency、token_rate、error_rate；缺失采样返回 unavailable。
- 服务端 TopN + 其他；返回完整 meta 和数据质量 warning。
- 添加跨维度一致性、时区、缺桶和缓存测试。

## Out of Scope

- 任意时间区间编辑器。
- SSE/WebSocket 推送。
- 用模拟种子生成生产序列。

## Implementation Notes

- 统一桶边界 UTC，响应标签按配置时区格式化；相邻接口共用同一桶生成器。
- 分位数不能对每桶均值再聚合；直方图桶按 count 累加后计算。
- 最终失败按 request_id 去重，业务限制与 SLA 错误分别计数。
- 过滤组合为空返回合法空序列和 `complete=true`；数据源失败返回 `complete=false` 和 warning。
- TopN 以所选主指标排序，其他组保留聚合值，确保总量不丢。

## Acceptance Criteria

- [x] 同一筛选下，任何 group_by 的序列总和等于 summary；差异为 0（允许浮点尾差）。
- [x] 切换 group_by 不改变 summary 的均值、分位和比率。
- [x] 无数据窗口、缺采样、聚合延迟、查询超时分别有稳定状态和 warning。
- [x] 时区跨日、夏令时和 `Asia/Shanghai` 桶标签有测试。
- [x] P95 从直方图与原始样本对比满足预设误差。
- [x] 7d/6h 查询只访问聚合表。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/service ./internal/repository ./internal/handler
go test -tags=integration ./internal/repository
go build ./cmd/server
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Endpoints | `GET /api/v1/admin/observability/{timeseries,latency-timeseries,heatmap}` | Implemented as read-only `observability.read` routes |
| Bucket alignment | `internal/service/observability/series.go` + tests | PASS: fixed 1h/5m, 6h/15m, 24h/1h, 7d/6h plans; Asia/Shanghai crosses `23:00 -> 00:00` correctly |
| DST labels | `TestBuildBucketPlanHandlesDSTChange` | PASS: 24h New York plan skips nonexistent 02:00 and labels 03:00 after the spring-forward transition |
| Summary consistency | `TestObservabilityQueryGroupByKeepsCanonicalSummary` | PASS: request/token series sums equal summary before and after `group_by=group_id` |
| TopN + other | `groupSeriesByBucket` / `otherSeries` | Selected dimensions use the same storage buckets; the residual is exposed as `:other` without dropping totals |
| Failure split | Request rollup + query service | Final failure, SLA 529 error, and business-limited counts/rates are separate metrics |
| Latency | Histogram merge + `internal/service/observability/histogram_test.go` | P50/P95 are computed from summed fixed buckets; P95 interpolation stays within the documented half-bucket tolerance |
| Heatmap | `Heatmap` handler/service tests | `sampled_concurrency`, `token_rate`, `error_rate` implemented; sampled peak test returns persisted value 5 |
| Missing states | Handler/service tests | No data -> `request_rollup_unavailable`; gap -> `missing_request_bucket`; missing concurrency -> `concurrency_samples_unavailable`; latency gaps -> `partial_latency_histogram`; timeout -> HTTP 504 `OBSERVABILITY_QUERY_TIMEOUT` |
| 7d storage access | Service calls only `observability_request_bucket` and `observability_concurrency_bucket` for 7d/6h | PASS: no raw `usage_logs` / `ops_error_logs` query path |
| Cache | Query service cache key includes window/bucket/filter/group_by/value_kind/user scope | 5s cache covered by repeated Timeseries test (no second series query) |
| Unit tests | `go test -tags=unit ./internal/handler/admin ./internal/service/observability ./internal/server/routes ./internal/repository` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Known limitation | Docker integration execution | Docker daemon is unavailable; SQL series aggregation was executed against local MariaDB and integration tests compile |

## Execution Notes

如果热力图需要负载特征而非当前饱和度，可把 value_kind 扩展为 `peak_sampled_concurrency`；不得继续使用无来源的“时段系数”。
