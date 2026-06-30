# MVP-023: 完成每日配额与分组路由后端集成回归

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 只补计划列出的跨组件高风险用例并运行聚焦套件，功能实现已在前置 MVP 完成。
- Dependencies: `MVP-016`

## Outcome

通过数据库与 gateway 集成测试证明并发不丢增量、跨日切换正确、身份字段正确且旧路由不回归。

## Context

优先扩展 `backend/internal/integration`、migration tests 和现有 gateway record usage tests。

## In Scope

- 覆盖 migration 默认值/唯一索引、并发累加与跨日窗口。
- 覆盖新旧 model_routing、配额降级和 requested/upstream model。
- 记录可重复的测试命令与结果到 Completion Evidence。

## Out of Scope

- 全仓不相关 flaky test 修复和性能压测。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 计划中的后端单元与集成场景均有自动化覆盖。
- [ ] 旧 model_routing 行为不回归。
- [ ] 聚焦 integration/service/repository 测试全部通过。

## Verification Plan

- `cd backend; go test ./internal/repository ./internal/service ./internal/integration ./migrations -run 'ModelRouting|TokenQuota|RequestedModel|UpstreamModel'`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


