# MVP-019: 增加管理端模型 Token 配额 API 与共享类型

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `15min`
- Estimate rationale: 只封装四个端点和共享 DTO，使用 mock client 快速验证。
- Dependencies: `MVP-010`, `MVP-011`

## Outcome

前端有类型安全的全局模型与用户模型限额查询/保存方法。

## Context

分别放入 `frontend/src/api/admin` 的合适模块，并从 admin index 导出。

## In Scope

- 定义 quota item、list/update payload。
- 封装全局 GET/PUT 与用户 GET/PUT。
- 补 URL、method、payload API 单测。

## Out of Scope

- 任何弹窗或页面 UI。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] API 路径与后端 routes 完全一致。
- [ ] null/0/正整数不会被客户端错误转换。
- [ ] 类型导出可被 UsersView 和 GroupsView/SettingsView 使用。

## Verification Plan

- `cd frontend; pnpm exec vitest run src/api/__tests__/admin.modelTokenQuotas.spec.ts`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


