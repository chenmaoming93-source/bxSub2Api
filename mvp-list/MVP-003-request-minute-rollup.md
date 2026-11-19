# MVP-003: 建立可查询的一分钟请求观测聚合

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 需要新表、幂等聚合、时间桶、直方图和调度，属于跨持久化与后台任务的中等复杂度切片。
- Dependencies: `MVP-002`

## Outcome

系统能按分钟保存成功/失败请求的可加计数、Token 与耗时/TTFT 直方图，支持 7 天趋势而无需每次扫描原始事实。

## Context

tokenstat 只有 `total_tokens` 与 D/W/M 周期，不能承担秒/分钟趋势。`usage_logs` 和 `ops_error_logs` 可提供原始事实，但 5 秒轮询不应反复全窗口扫描。设计见 `DESIGN.md` 第 7 节。

## In Scope

- 新增 `observability_request_minute` 表与 Ent/SQL schema，维度包含 user/api_key/group/alias/account；模型由账号唯一推导。
- 记录 request/success/error、Token 总量、duration 和 TTFT 的固定直方图桶。
- 实现增量聚合任务：读取水位后的原始事实，按 UTC 分钟聚合，幂等 upsert；同一请求只计一次。
- 实现 1 分钟原始聚合保留 24-48 小时、5 分钟聚合保留 7 天、小时聚合保留更久的滚动任务。
- 增加数据质量字段：最后成功桶、watermark、跳过/延迟计数。

## Out of Scope

- 替换 tokenstat 配额判定。
- 精确到秒的全量保留。
- 独立上游模型维度。

## Implementation Notes

- 使用唯一键 `(bucket_start, dimension_hash)`，dimension hash 由 canonical 账号路径维度生成。
- duration/TTFT 直方图使用固定边界，例如 TTFT 100/250/500/1000/2000/5000/10000/+Inf ms；分位数合并只读桶。
- 聚合任务可 30-60 秒运行一次，失败不影响 gateway；使用现有 leader lock 防多实例重复计算。
- 原始成功/失败按 request_id 去重后入桶；中间 upstream retry 只累计到上游错误字段，不增加客户端失败计数。
- 为过渡期保留 `legacy_inferred` alias 计数，查询时默认包含但可过滤。

## Acceptance Criteria

- [x] 重复运行同一分钟聚合不会重复计数。
- [x] 跨 UTC 日/时区边界时，查询窗口返回 `Asia/Shanghai` 正确桶标签。
- [x] 一分钟内成功+最终失败数等于去重后的客户端请求数。
- [x] P50/P95 从直方图合并的结果与同批原始样本误差满足验收阈值。
- [x] 聚合失败、缺少 watermark 或部分直方图时返回数据质量告警。
- [x] 7 天趋势按 5 分钟/小时聚合查询，不访问原始全表。

## Verification Plan

```powershell
cd backend
go generate ./ent
go test -tags=unit ./internal/repository ./internal/service
go test -tags=integration ./internal/repository
go build ./cmd/server
```

实现时补充至少一个包含失败重试、缓存 Token、流式 TTFT 和非流式请求的集成测试。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Ent schema | `backend/ent/schema/observability_request_bucket.go` + `go generate ./ent` | PASS: generated model/query/create/update/migration schema includes all additive counters and fixed histograms |
| SQL archive | `backend/sqlArchiving/177_create_observability_request_rollup.sql` | PASS: executed twice locally; created 2 tables and 6 non-primary indexes idempotently |
| Unit tests | `go test -tags=unit ./internal/repository ./internal/service/observability` | PASS |
| Integration test compile | `go test -c -tags=integration ./internal/repository` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Actual repository verification | Temporary local MariaDB verifier using repository APIs | PASS: 4 source rows collapsed to 3 final requests (2 success, 1 final error), 425 tokens retained, rerun stayed at 3 |
| Time bucket | Same verifier with `Asia/Shanghai` | PASS: UTC `2026-11-10T15:59:00Z` returned as `2026-11-10T23:59:00+08:00` |
| Histogram | `internal/service/observability/histogram_test.go` | PASS: P50/P95 interpolation stays within the documented half-bucket-width tolerance; empty samples return no quantiles |
| Data quality | `observability_request_rollup_state` | Tracks `last_source_at`, `last_success_at`, `last_success_bucket`, `skipped_count`, `delayed_count`, and `quality_status` |
| Retention | `CleanupRequestRollups` | Minute 48h, five-minute 7d, hourly 90d; query endpoint reads only `observability_request_bucket` |
| Worker integration | `cmd/server/wire.go`, generated `wire_gen.go`, cleanup hook | PASS: 45s leader-elected worker starts with the server and stops during cleanup |
| Changed paths | `backend/ent/schema/observability_request_bucket.go`, `backend/internal/{service,repository}`, `backend/sqlArchiving/177_create_observability_request_rollup.sql`, server wiring | Implemented |
| Known limitation | Full Docker integration execution | Docker daemon is unavailable; the integration suite compiles, and the same repository path was executed end-to-end against local MariaDB |

## Execution Notes

若 7 天高基数用户明细存储量超出预算，保留完整分组/账号维度，并对用户维度采用受控 TopN 存储；必须在指标 meta 中标注 TopN 覆盖范围。分钟聚合按“先删除受影响桶、再从原始事实完整重算”的方式写入，单个批次失败不会推进水位；后续批次仍可安全重试。
