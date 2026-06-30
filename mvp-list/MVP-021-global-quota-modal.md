# MVP-021: 增加全局模型每日限额管理弹窗

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `20min`
- Estimate rationale: 一个独立弹窗与入口，范围不包含统计图表。
- Dependencies: `MVP-019`

## Outcome

管理员可在分组模型路由旁或管理员设置中维护全局实际模型每日 Token 限额。

## Context

为减少导航改动，优先放在 `GroupsView.vue` 路由编辑区域旁；最终位置可按现有布局选择。

## In Scope

- 新增入口、列表编辑弹窗和保存状态。
- 展示模型、当日 used_tokens、daily_limit_tokens。
- 校验重复模型和限额语义。

## Out of Scope

- 用户模型限额、趋势图和批量导入。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 打开时从全局配额端点加载。
- [ ] 保存后回显新限额和保留用量。
- [ ] 重复/空模型和非法限额不能提交。

## Verification Plan

- `cd frontend; pnpm exec vitest run src/views/admin/__tests__/GroupsView.modelTokenQuota.spec.ts; pnpm run typecheck`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


