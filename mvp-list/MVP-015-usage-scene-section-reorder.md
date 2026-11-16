# MVP-015: 将场景 Token 用量移到使用记录页底部

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.5 day`
- Estimate rationale: 改动集中在一个 Vue 页面和组件测试，主要成本是确保页面顺序、加载行为与移动端布局不回退。
- Dependencies: `none`

## Outcome

“使用记录”页面优先展示使用记录、错误记录和筛选操作；“场景 Token 用量”作为低优先级区块移动到页面最低端，并延后加载。

## Context

`frontend/src/views/admin/UsageView.vue` 当前把 `data-test="scene-token-usage"` 放在页面顶部、筛选和 Tab 之前；`onMounted` 还会立即调用 `loadSceneUsage()`，导致低频内容占用首屏空间和接口请求。

## In Scope

- 将 `场景 Token 用量` section 移至页面最低端，位于使用记录表、错误 Tab、分页和其它主要区块之后。
- 首屏不再自动请求场景用量；用户展开/点击查询时再加载。
- 保留日期、分组筛选、加载中、空数据、错误和同步提示。
- 调整移动端布局，确保区块移动后不产生横向溢出。
- 更新 `UsageView.spec.ts`，验证 DOM 顺序和按需加载。

## Out of Scope

- 删除场景用量接口或业务数据。
- 修改统计口径。
- 改变使用记录、错误记录的排序和分页。

## Implementation Notes

- 使用折叠面板或底部默认折叠区块表达低优先级；只有展开或点击查询时请求。
- 避免复用顶部日期筛选状态造成隐式查询；场景用量保留自己的日期与分组选择。
- `applyFilters`、`refreshData` 不应自动触发场景用量请求；仅用户显式查询或已展开时刷新。
- 增加 `data-test="scene-token-usage"` 的 DOM 顺序断言，确认它位于主要 Tab 和表格之后。

## Acceptance Criteria

- [x] 场景 Token 用量是页面最低端的主要业务区块。
- [x] 页面进入时不自动请求场景用量，展开或点击查询后才请求。
- [x] 使用记录和错误记录仍可正常切换、筛选、分页和导出。
- [x] 场景用量的查询、空态、错误态和同步提示保持可用。
- [x] 桌面与移动端无横向溢出，高低优先级区块顺序正确。

## Verification Plan

```powershell
pnpm --dir frontend exec vitest run src/views/admin/__tests__/UsageView.spec.ts
pnpm --dir frontend run typecheck
pnpm --dir frontend run lint:check
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Page order | `UsageView.vue` DOM-order test | PASS: scene section appears after usage/error tabs and their tables instead of before filters |
| Lazy loading | mounted component test | PASS: page entry issues zero scene-usage requests; first expansion performs the request |
| Main filter isolation | `applyFilters` and `refreshData` tests | PASS: main filters never load scene usage while collapsed; expanded section refreshes only through an explicit user action |
| Independent range | dedicated `sceneUsageStartDate` / `sceneUsageEndDate` state | PASS: scene usage no longer reads or mutates the primary usage date range |
| Preserved states | mocked success, empty and rejected API results | PASS: grouped account/model table, empty state, error state and synchronization banner remain available |
| Existing behavior | usage/error filtering, pagination and export tests | PASS |
| Responsive layout | `flex-wrap` controls plus contained tables using `overflow-x-auto`; typecheck/build | PASS: bottom section wraps controls on narrow screens and keeps wide detail tables inside their own horizontal scroll container |
| Focused test | `pnpm exec vitest run src/views/admin/__tests__/UsageView.spec.ts` | PASS: 11 tests |
| Frontend verification | `pnpm run typecheck`, `pnpm run lint:check`, `pnpm run build` | PASS; lint has one pre-existing warning and zero errors |
| Changed paths | `frontend/src/views/admin/UsageView.vue`, `frontend/src/views/admin/__tests__/UsageView.spec.ts`, i18n files | Implemented |

## Execution Notes

若产品希望完全隐藏而不是置底，可在此 MVP 把区块默认折叠并保留入口；当前按需求实现“置于最低端”。
