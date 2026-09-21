# MVP-008：交付部门用户 Token 详情弹窗

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 复用现有部门详情弹窗逻辑并替换 UI，范围独立可测。
- Dependencies: `MVP-003, MVP-005, MVP-006`

## 预期成果

点击管理员仪表盘部门排行后，打开居中弹窗显示该部门用户 Token 数据。

## 背景

现有 `/admin/usage` 已使用 `BaseDialog`、`DepartmentUserUsageChart` 和分页。接口为 `queryDepartmentUsers()`，路径为 `/admin/usage/department-stats/users`。

## 范围内

- `DepartmentUserTokenDialog`。
- 复用现有部门用户查询、汇总和分页逻辑。
- 使用 `UiDialog`、`UiDataTable`、`UiPagination`。
- 显示部门总 Token、用户数、人均 Token 和公司占比。
- 用户排行显示用户、邮箱、Token、占比和请求数。
- 搜索、Top12/全部、刷新、错误和空状态。
- 参考 `assets/final-admin-dashboard-department-modal.png`。

## 范围外

- 不改部门统计接口业务规则。
- 不在页面排行内展开用户。
- 不删除旧 `BaseDialog` 逻辑。

## 实现说明

将现有部门详情代码抽取为可复用业务弹窗，数据请求仍由页面或业务协调器控制。

## 验收标准

- [x] 点击部门只打开居中弹窗。
- [x] 页面没有用户内嵌展开内容。
- [x] 详情表格支持分页和刷新。
- [x] 原 `DepartmentUserUsageChart` 仍可复用。
- [x] 弹窗视觉符合指定原型图。

## 验证计划

- `pnpm --dir frontend typecheck`
- 部门点击、搜索、分页、刷新和关闭测试。
- 启动项目人工验证弹窗布局。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Unit tests | `pnpm run test:run -- src/features/admin-dashboard/__tests__/department-user-dialog.test.ts src/features/admin-dashboard/__tests__/scene-account-dialog.test.ts` | 2 个测试通过 |
| Build | `pnpm run build` | 通过 |
| Changed files | `frontend/src/features/admin-dashboard/DepartmentUserTokenDialog.vue`; `frontend/src/views/admin/AdminDashboardV2View.vue` | 部门排行点击后按需打开居中详情弹窗 |

## 执行记录

已复用 `queryDepartmentUsers()`，弹窗内提供 Token 汇总、用户表格、搜索、分页、刷新错误重试和关闭。