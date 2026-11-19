# MVP-016: 支持快捷与自定义绝对时间范围

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 需要同时修改共享契约、桶规划、所有查询入口和缓存键，并覆盖跨日与时区边界。
- Dependencies: `MVP-005`, `MVP-006`

## Outcome

观测台所有接口都能使用今天、昨天、最近 1 小时、6 小时、24 小时、7 天及自定义起止时间，并在 `Asia/Shanghai` 时区返回一致、可核对的真实起止时间和时间桶。

## Context

当前 `backend/internal/service/observability/contract.go` 只接受固定的 `1h/6h/24h/7d` 与固定桶组合。前端也只提供四个 select 选项，无法表达今天、昨天和自定义范围。

## In Scope

- 扩展观测查询契约，保留现有固定窗口并增加 `today`、`yesterday`、`custom`。
- 自定义范围使用 ISO-8601 起止时间；服务端统一转换为 UTC 查询，展示仍使用配置时区。
- 按范围长度选择受控桶：短范围 5 分钟，中范围 15 分钟，长范围 1 小时，最长 6 小时。
- 校验 `start < end`、未来结束时间、超过保留上限及非法桶组合。
- 将规范化后的 UTC 起止时间与桶加入 ETag、短缓存键和前端请求取消标识。
- 前端 API 类型与参数构造支持绝对范围，为后续页面重构提供稳定基础。

## Out of Scope

- 本阶段不重做页面视觉和时间选择器弹层。
- 不支持超过 7 天的交互式全明细查询。
- 不改变原始事实表保留周期。

## Implementation Notes

- 建议引入单一 `ResolveObservabilityRange`，所有 snapshot/timeseries/latency/heatmap/paths/relations 共用。
- 今天：本地 00:00 到当前时刻；昨天：昨日 00:00 到昨日 23:59:59.999。
- 单次查询最大跨度按 7 天处理，避免突破当前分钟/5 分钟/小时聚合保留范围。
- 限制桶数量上限，例如 288 个，防止恶意自定义范围放大 SQL 与响应体。
- 缓存键必须包含 UTC 起点、UTC 终点和实际桶大小，不能只使用 `window` 名称。

## Acceptance Criteria

- [x] 今天、昨天和自定义范围均返回准确的本地起止时间及对应 UTC 查询范围。
- [x] 同一请求重复执行时桶边界、标签和汇总完全一致。
- [x] `start >= end`、未来结束时间、超过 7 天和桶数量超限返回稳定的 4xx 错误。
- [x] 所有观测接口对相同绝对范围使用同一桶规划器。
- [x] ETag/缓存不会在相邻但不相同的时间范围之间串数据。
- [x] 跨日与 `Asia/Shanghai` 边界测试通过。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/service/observability ./internal/handler/admin ./internal/server/routes
go test -tags=integration ./internal/repository
go build -p 1 ./cmd/server

cd ..\frontend
pnpm run typecheck
pnpm run test:run -- src/api/admin/__tests__/observability.spec.ts
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Range resolver | `backend/internal/service/observability/contract.go` | Added fixed/today/yesterday/custom resolution, 7-day and 288-bucket limits, and exact UTC `start_at/end_at` metadata |
| Bucket planner | `backend/internal/service/observability/series.go` | Exact absolute range plan is deterministic and remains within the configured bucket cap |
| Shared handlers | `backend/internal/handler/admin/observability_{query,snapshot,paths}_handler.go` | Snapshot, timeseries/latency/heatmap, paths and relations all call the same range parser |
| Frontend API | `frontend/src/api/admin/observability.ts` | New presets, `startAt/endAt`, and absolute `start_at/end_at` request parameters |
| Focused backend tests | `go test -tags=unit ./internal/service/observability` | PASS |
| Focused handler tests | `go test -tags=unit ./internal/handler/admin -run Observability` | PASS |
| Route tests | `go test -tags=unit ./internal/server/routes` | PASS |
| Repository integration | `go test -tags=integration ./internal/repository` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Frontend tests | `pnpm run test:run -- src/api/admin/__tests__/observability.spec.ts` | PASS, 4 tests |
| Frontend typecheck/build | `pnpm run typecheck`; `pnpm run build` | PASS |

## Execution Notes

7 天最大范围是当前保留能力的规划假设；若未来扩大保留周期，应单独调整该契约和负载测试。

`go test -tags=unit ./internal/service` 当前被既有 `AccountRepository.ExistsByName` 接口变更导致的旧 mock 编译错误阻断；该失败不涉及观测契约文件。MVP 要求的观测服务、handler、routes、integration 和 build 均已通过。
