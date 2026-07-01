# MVP-018: 升级 GroupsView 分组模型路由编辑器

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `20min`
- Estimate rationale: 复用现有两套创建/编辑路由区域和账号搜索组件，只扩展候选行字段。
- Dependencies: `MVP-003`, `MVP-017`

## Outcome

管理员能在创建和编辑分组时维护别名下的候选模型、账号、优先级与每日限额。

## Context

`GroupsView.vue` 已有 create/edit ModelRoutingRule、账号搜索和保存逻辑，应共用 helper 避免两套规则漂移。

## In Scope

- 把规则 UI 改为别名 + 候选列表。
- 支持候选增删、模型、账号、priority、daily_token_limit。
- 保存时始终写新格式，加载时接受旧格式。

## Out of Scope

- 全局/用户模型限额弹窗。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] 创建与编辑表单行为一致。
- [x] 候选最少字段、非负整数和账号选择校验可见。
- [x] 保存 payload 与 MVP-017 序列化结果一致。

## Verification Plan

- `cd frontend; pnpm exec vitest run src/views/admin/__tests__/groupsModelRouting.spec.ts; pnpm run typecheck`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `frontend/src/views/admin/GroupsView.vue` | Create/edit forms now share alias/candidate structures and handlers for candidate model, accounts, priority, daily limit, add/remove, validation, loading, and saving. |
| Focused tests | `cd frontend; pnpm exec vitest run src/views/admin/__tests__/groupsModelRouting.spec.ts` | PASS: 1 file, 3 tests. Both save paths call the tested MVP-017 serializer, and the load path calls its legacy/new normalizer. |
| Typecheck | `cd frontend; pnpm run typecheck` | PASS (`vue-tsc --noEmit`). |
| Targeted lint | `cd frontend; pnpm exec eslint src/views/admin/GroupsView.vue src/views/admin/groupsModelRouting.ts src/views/admin/__tests__/groupsModelRouting.spec.ts src/api/admin/groups.ts src/types/index.ts` | PASS. |

## Execution Notes

- Account search state is keyed per candidate rather than per alias, so multiple candidates under one alias can be edited independently.
- Empty daily limits serialize as `null`; priority and positive/zero limits serialize as numbers. Validation rejects missing model/accounts, non-integer or negative values, duplicate aliases/models/priorities.
- Existing legacy account arrays normalize into one editable candidate; every save emits the new candidate-array format.
