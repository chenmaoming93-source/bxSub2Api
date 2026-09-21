# MVP-003：实现 UI V2 弹窗与数据组件

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦统一下钻弹窗和高频数据控件，能被多个页面独立复用和验证。
- Dependencies: `MVP-002`

## 预期成果

提供符合最终交互要求的居中弹窗、表格、分页、筛选条和横向排行榜组件。

## 背景

现有 `BaseDialog.vue` 已支持 Teleport、ESC、焦点和宽度；现有 `DataTable.vue`、`Pagination.vue`、`UsageFilters.vue` 可作为行为契约参考。所有详情必须使用居中弹窗，不使用抽屉和内嵌展开。

## 范围内

- `UiDialog`、`UiDetailDialog`、`UiConfirmDialog`。
- `UiDataTable`、`UiPagination`、`UiTableToolbar`、`UiFilterBar`。
- `UiColumnSettings`、`UiBulkActions`、`UiEmptyState`、`UiTableSkeleton`。
- `UiHorizontalRanking`。
- ESC、焦点恢复、滚动锁定、遮罩、响应式宽度。
- 兼容现有表格 slots、分页和弹窗关闭事件。

## 范围外

- 不迁移业务页面。
- 不删除或覆盖 `BaseDialog`、`DataTable`、`Pagination`。
- 不实现具体场景、部门或请求数据。

## 实现说明

- 详情组件只能以居中 Dialog 形态呈现。
- 排行组件面向高基数数据，支持搜索、排序、Top N、全部和分页。
- 视觉参考 `assets/final-admin-dashboard-scene-modal.png` 和 `assets/final-admin-dashboard-department-modal.png`。

## 验收标准

- [x] Dialog 不以抽屉形式出现。
- [x] 打开和关闭时焦点、滚动和 ESC 行为正确。
- [x] DataTable 支持 columns、data、具名 slot、加载和空状态。
- [x] HorizontalRanking 支持 Token、占比、行内条和详情操作。
- [x] 旧组件仍可被旧页面引用。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- 使用测试组件人工验证键盘、响应式和弹窗遮罩。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Unit tests | `pnpm run test:run -- src/components/ui-v2/__tests__/dialog-data.test.ts` | 3 个测试通过 |
| Build | `pnpm run build` | 通过；存在既有 chunk size 和 dynamic import 警告 |
| Changed files | `frontend/src/components/ui-v2/feedback/`、`frontend/src/components/ui-v2/data/` | 新增 V2 Dialog、数据组件和测试；旧组件未删除 |

## 执行记录

已完成居中 Dialog、详情/确认弹窗、表格、分页、筛选条、空状态、骨架屏、列设置、批量操作和横向排行。