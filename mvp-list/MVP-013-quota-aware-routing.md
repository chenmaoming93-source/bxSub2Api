# MVP-013: 在候选选择前检查三类 Token 配额

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 在既有候选循环增加统一 preflight，错误分支数量有限且可用 stub 测试。
- Dependencies: `MVP-007`, `MVP-009`, `MVP-012`

## Outcome

任一候选、全局模型或当前用户模型额度耗尽时，路由自动跳到下一个候选。

## Context

耗尽错误仅用于内部降级，不能在仍有候选时直接返回客户端。

## In Scope

- 注入 quota checker 到 GatewayService/Wire。
- 按候选→全局模型→用户模型顺序执行 preflight。
- 用 stub 覆盖三种耗尽与其他 repository 错误。

## Out of Scope

- 账号请求失败后的跨模型 failover、用量累加。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 三类耗尽均跳过当前候选。
- [ ] 用户 A 耗尽不影响用户 B。
- [ ] 非耗尽基础设施错误不会被伪装成配额耗尽。

## Verification Plan

- `cd backend; go test ./internal/service -run 'QuotaAwareRouting'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


