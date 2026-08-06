# MVP-010：交付投影、限额和同步状态管理页面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 开发日`
- Estimate rationale: `三个相互关联的管理页签、动态表单、RBAC和测试构成一个完整管理员配置体验。`
- Dependencies: `MVP-003, MVP-008`

## 预期成果

管理员可在新 `/admin/token-statistics` 入口管理统计投影、通用限额和同步状态，不调用任何旧统计限额API。

## 背景

新体系页面需动态读取维度和指标注册信息，并按权限控制读写操作。

## 范围内

- 新前端API模块和类型。
- 新路由、菜单、标题和RBAC矩阵。
- 投影列表、创建、编辑、发布和停用。
- 限额列表、创建、观察/强制、启停。
- 同步及周期状态展示。
- 不完整周期和生效时间提示。
- 组件和API测试。

## 范围外

- 通用查询图表页签。
- 删除旧页面。

## 实现说明

- 页面入口可采用单页多页签。
- 配置选项完全来自后端注册接口。
- 旧 `/admin/token-usage/*` 和旧model quota API不得调用。

## 验收标准

- [x] 管理员可完成投影全生命周期操作。
- [x] 管理员可创建不依赖旧逻辑的通用限额。
- [x] 页面正确提示新投影统计起点和限额生效时间。
- [x] `token_usage.read/manage` 与 `token_quota.read/update` 控件权限正确。
- [x] 同步和周期状态可见。
- [x] 新前端API和页面测试通过。

## 验证计划

- `cd frontend && pnpm test:run`
- `cd frontend && pnpm typecheck`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 前端 API | `frontend/src/api/admin/dynamicTokenStatistics.ts` | 新增注册表、投影、限额、同步状态 API 与完整类型；仅使用 `/admin/token-statistics` |
| 管理页面 | `frontend/src/views/admin/TokenStatisticsView.vue` | 单页三页签完成投影生命周期、动态维度限额、权限控件、统计起点/下周期生效提示和同步周期状态 |
| 路由与菜单 | `frontend/src/router/index.ts`、`frontend/src/rbac/permissionMatrix.ts`、`frontend/src/components/layout/AppSidebar.vue` | 新增 `/admin/token-statistics`，页面要求 `token_usage.read` |
| 定向测试 | `cd frontend && pnpm vitest run src/api/admin/__tests__/dynamicTokenStatistics.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts src/rbac/permissionMatrix.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts` | 4 个测试文件、9 项测试全部通过 |
| 类型检查 | `cd frontend && pnpm typecheck` | 通过 |
| 全量测试说明 | `cd frontend && pnpm test:run` | 已执行；本次新增用例均通过，仓库既有 OAuth redirect/EmailVerify 6 项断言因期望旧跳转地址而失败，与本 MVP 改动路径无交集 |

## 执行记录

2026-07-30：完成可配置统计管理入口。页面未调用旧 `/admin/token-usage/*` 或旧 model quota API。
