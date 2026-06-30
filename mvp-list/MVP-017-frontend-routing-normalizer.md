# MVP-017: 增加前端新旧模型路由类型与归一化助手

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `15min`
- Estimate rationale: 纯 TypeScript 类型与函数，测试快速且不涉及 Vue 布局。
- Dependencies: `MVP-002`

## Outcome

管理端可把旧 map 和新候选数组归一为可编辑行，并稳定序列化为新格式。

## Context

更新 `frontend/src/types/index.ts`、`frontend/src/api/admin/groups.ts`，新增与现有 groups helper 同目录的纯函数。

## In Scope

- 定义候选与路由 map 类型。
- 实现旧格式加载、新格式加载、排序、校验和新格式保存。
- 补 Vitest 数据驱动测试。

## Out of Scope

- 修改 GroupsView.vue。

## Implementation Notes

- 优先复用上述现有路径与模式；若执行时发现接口名漂移，记录实际路径但不得扩大本 MVP 的结果边界。
- 本 MVP 的实现、测试和证据记录必须在估时内完成；超出时拆出后续 MVP，不以跳过验证换取完成。

## Acceptance Criteria

- [x] 旧数据加载后账号顺序不丢失。
- [x] 新候选所有字段往返不丢失。
- [x] 重复别名、模型、优先级或负限额给出校验结果。

## Verification Plan

- `cd frontend; pnpm exec vitest run src/views/admin/__tests__/groupsModelRouting.spec.ts`

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Implementation | `frontend/src/types/index.ts`, `frontend/src/views/admin/groupsModelRouting.ts`, `frontend/src/api/admin/groups.ts` | Added legacy/new routing types, deterministic normalization and serialization, stable priority sorting, and structured validation issues. |
| Focused tests | `cd frontend; pnpm exec vitest run src/views/admin/__tests__/groupsModelRouting.spec.ts` | PASS: 1 file, 3 tests; legacy account order, complete candidate round trip, and duplicate/negative validation covered. |
| Targeted lint | `cd frontend; pnpm exec eslint src/views/admin/groupsModelRouting.ts src/views/admin/__tests__/groupsModelRouting.spec.ts src/api/admin/groups.ts src/types/index.ts` | PASS. |
| Broader check | `cd frontend; pnpm run typecheck` | One expected MVP-018 integration error remains: legacy `GroupsView.vue` converter accepts only `Record<string, number[]>`; no errors were reported in the new helper/types. |

## Execution Notes

- Legacy account arrays normalize to one candidate whose model is the route alias, priority is `0`, limit is `null`, and account order is unchanged.
- Alias keys and candidate priority are serialized deterministically. Equal priorities retain input order.
- View integration is intentionally deferred to MVP-018 per this MVP's non-goal; its existing legacy converter is the sole current full-typecheck failure.
