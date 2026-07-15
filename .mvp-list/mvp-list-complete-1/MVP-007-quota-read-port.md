# MVP-007: 实现三类每日配额查询与耗尽判断

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 只做 service port、repository 读取和纯判断，不混入缓存与写入。
- Dependencies: `MVP-004`, `MVP-005`, `MVP-006`

## Outcome

调用方可用统一结果检查候选、全局模型和用户模型当日是否耗尽。

## Context

参考 `backend/internal/service/user_platform_quota_port.go` 与 repository adapter 解耦方式。

## In Scope

- 定义三类 key/snapshot、sentinel error 和 repository port。
- 实现按 StartOfDay 的 DB 查询与缺省不限额语义。
- 补边界测试：等于上限、超过上限、无记录。

## Out of Scope

- 原子累加、Redis 和路由循环。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] usage >= positive limit 返回对应耗尽错误。
- [x] 0/null 按 MVP-001 决策处理。
- [x] 用户耗尽错误携带足够上下文且不影响其他用户。

## Verification Plan

- `cd backend; go test ./internal/repository ./internal/service -run 'ModelTokenQuota|DailyTokenQuota'`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Service port | `backend/internal/service/daily_token_quota_port.go` | 定义三类 key、统一 snapshot、repository port、三种 sentinel 与带上下文耗尽错误。 |
| Repository | `backend/internal/repository/daily_token_quota_repo.go` | 三类查询统一按 `timezone.StartOfDay`，无记录返回不存在快照（不限额）。 |
| Verification | `cd backend; go test ./internal/repository ./internal/service -run 'ModelTokenQuota|DailyTokenQuota' -count=1` | PASS（repository 4.807s，service 4.953s）。 |
| Hygiene | `git diff --check` | PASS；仅有既有行尾提示。 |

## Execution Notes

- 读取端口不把“无记录”当异常：返回 `Exists=false`、用量 0、限额 nil，调用方自然按不限额处理。
- 正数限额在 `used_tokens >= daily_limit_tokens` 时耗尽；nil/0 均放行。
- `DailyTokenQuotaExhaustedError` 保留 scope、user/group、route、model、日期、用量与限额，并通过 `Unwrap` 暴露对应 sentinel；用户 42 的耗尽状态不会影响用户 43 的独立 key。
