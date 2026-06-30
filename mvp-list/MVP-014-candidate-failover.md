# MVP-014: 完成候选账号与跨模型 failover 及最终错误映射

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 聚焦选择循环的退出条件和错误优先级，不改转发/usage。
- Dependencies: `MVP-013`

## Outcome

候选账号失败或被排除后先尝试同候选其他账号，再降级下一模型；全部失败返回确定的 429/503。

## Context

需保留现有并发、RPM、window cost、模型映射、平台与账号健康检查。

## In Scope

- 让 excludedIDs 与候选边界正确协作。
- 区分全部配额耗尽和全部账号不可调度。
- 补混合失败原因与多候选测试。

## Out of Scope

- 实际发起上游请求后的重试策略改造。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 同候选仍有可用账号时不提前跨模型。
- [ ] 候选耗尽后按 priority 尝试下一模型。
- [ ] 全部配额耗尽为 429，不可调度为项目既有 503。

## Verification Plan

- `cd backend; go test ./internal/service -run 'GroupedModel.*Failover|QuotaExhausted'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


