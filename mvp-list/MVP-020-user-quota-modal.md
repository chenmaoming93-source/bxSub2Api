# MVP-020: 在用户管理页增加模型每日限额弹窗

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 复用现有平台额度单元格和弹窗交互，限定查询、编辑、保存与回显。
- Dependencies: `MVP-019`

## Outcome

管理员可从 UsersView 为单个用户配置多个实际上游模型的每日 Token 限额。

## Context

入口位于 `frontend/src/views/admin/UsersView.vue`，风格参考现有 balance_platform_quota 功能。

## In Scope

- 新增操作入口与 modal 状态。
- 支持模型行增删、整数校验、保存 loading/error。
- 保存成功后回显最新数据且不整页丢失筛选状态。

## Out of Scope

- 普通用户查看、自助修改、全局限额 UI。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 打开弹窗加载目标用户数据。
- [ ] 非法模型或限额不能提交。
- [ ] 保存只调用目标用户端点并显示后端结果。

## Verification Plan

- `cd frontend; pnpm exec vitest run src/views/admin/__tests__/UsersView.spec.ts; pnpm run typecheck`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


