# MVP-012: 按优先级选择分组模型候选

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 只把已解析候选接入选择入口，暂不检查 Token 配额。
- Dependencies: `MVP-002`, `MVP-003`

## Outcome

请求 model 命中新格式分组别名时，调度器按 priority 尝试实际模型，并限制候选账号集合。

## Context

入口是 `GatewayService.SelectAccountWithLoadAwareness`；旧模式仍通过 `GetRoutingAccountIDs` 等现有行为。

## In Scope

- 识别精确分组别名并构造候选序列。
- 把候选实际 model 与 account_ids 传给现有账号筛选。
- 保留旧通配路由行为并加单元测试。

## Out of Scope

- Token 配额检查和 usage 写入。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 新别名优先选择最小 priority 候选。
- [ ] 只会返回候选 account_ids 中可调度账号。
- [ ] 旧路由测试继续通过。

## Verification Plan

- `cd backend; go test ./internal/service -run 'ModelRouting|GroupedModelCandidate'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


