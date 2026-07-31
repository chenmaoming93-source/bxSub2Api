# MVP-004：移除旧前端页面、入口与 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `页面、专属组件、API、路由、菜单、i18n 与测试构成一个完整前端删除成果。`
- Dependencies: `MVP-001, MVP-003`

## 预期成果

用户无法再访问旧统计和旧限额界面，所有旧前端 API 调用消失，新 `/admin/token-statistics` 页面及公共组件不受影响。

## 背景

公共趋势图、日期选择、分页和权限组件可能被其他页面使用，不能按目录名盲删。

## 范围内

- 删除三张旧统计页面及专属组件/API/测试。
- 删除旧模型、用户、默认和批量限额入口与弹窗。
- 删除旧路由、菜单、权限矩阵项、标题和 i18n。
- 更新用户、分组等页面的旧按钮。

## 范围外

- 新可配置统计页面和通用公共组件。

## 实现说明

- 删除文件前用导入者扫描区分专属和共享组件。
- 不自动把旧按钮改造成行为不同的新配置。

## 验收标准

- [x] 旧统计页面、限额入口和 API 模块已移除。
- [x] 旧路由和菜单不可见。
- [x] 公共组件仍被保留且测试通过。
- [x] 新页面、RBAC、类型检查和构建通过。

## 验证计划

- `cd frontend && pnpm test:run`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- `rg -n "/admin/token-usage/|modelTokenQuotas|UserModelTokenQuota" frontend/src`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 删除范围 | `frontend/src/{api/admin,components/admin,views/admin,composables}` | 三张旧报表、固定限额弹窗/API/专属组件与测试已删除 |
| 导航与权限 | `frontend/src/{router/index.ts,components/layout/AppSidebar.vue,rbac/permissionMatrix.ts}` | 旧路由、菜单和矩阵项已移除；新 `/admin/token-statistics` 保留 |
| 类型与构建 | `pnpm typecheck`、`pnpm build` | 通过 |
| 聚焦测试 | `pnpm vitest run src/views/admin/__tests__/UsersView.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts src/rbac/permissionMatrix.spec.ts` | 4 文件、10 测试通过 |
| 全量测试 | `pnpm test:run` | 本次旧模块引起的导入失败已修复；仍有 6 个既有 OAuth/注册重定向断言失败，与本 MVP 无关 |

## 执行记录

2026-07-30：开始移除旧统计与固定限额页面、API、导航和组件，保护新的可配置统计页面。
2026-07-30：完成旧前端删除；类型检查、生产构建及新页面/导航/RBAC 聚焦测试通过。
