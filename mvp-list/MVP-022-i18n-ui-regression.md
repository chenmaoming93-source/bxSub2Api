# MVP-022: 补齐中英文文案与管理端 UI 回归

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `15min`
- Estimate rationale: 文案键和三处聚焦测试可在短周期内完成，不再增加功能。
- Dependencies: `MVP-018`, `MVP-020`, `MVP-021`

## Outcome

新路由与配额界面在中英文环境均无裸 key，关键交互有回归保护。

## Context

locale 位于 `frontend/src/i18n/locales/zh.ts`、`en.ts`；项目已有 locale key 测试模式。

## In Scope

- 补分组候选、全局限额、用户限额及错误提示文案。
- 增加中英文 key 对称性测试。
- 运行相关 UI 测试、lint check 和 typecheck。

## Out of Scope

- 产品文档大改或视觉重设计。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [ ] 新增 key 在 zh/en 均存在。
- [ ] 页面测试中不出现裸 i18n key。
- [ ] 相关 Vitest、lint:check、typecheck 通过。

## Verification Plan

- `cd frontend; pnpm exec vitest run src/i18n/__tests__ src/views/admin/__tests__/groupsModelRouting.spec.ts src/views/admin/__tests__/UsersView.spec.ts; pnpm run lint:check; pnpm run typecheck`

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|

## Execution Notes


