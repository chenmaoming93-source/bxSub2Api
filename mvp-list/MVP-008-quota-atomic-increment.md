# MVP-008: 实现三类用量的原子累加与跨日切换

- Protocol: `mvp-list/v1`
- State: `ACTIVE`
- Estimate: `20min`
- Estimate rationale: 集中实现一条事务/UPSERT 写路径并用数据库测试验证，不接入业务调用点。
- Dependencies: `MVP-007`

## Outcome

成功 usage 能按日期原子增加三类 Token 用量，并在新日期写入新窗口。

## Context

总量使用 `UsageLog.TotalTokens()`；并发安全必须由数据库原子语句保证。

## In Scope

- 实现候选、全局模型和用户模型的批量或事务性增量接口。
- 使用 UPSERT/原子加法，避免读改写竞争。
- 补同日并发和跨日测试。

## Out of Scope

- 调用 RecordUsage、Redis 同步。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 并发 N 次累加后的 used_tokens 等于输入总和。
- [ ] 次日写入不污染前一日行。
- [ ] 任一步失败时不会留下计划未允许的部分更新。

## Verification Plan

- `cd backend; go test ./internal/repository -run 'TokenQuota.*(Concurrent|Rollover|Increment)'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes

