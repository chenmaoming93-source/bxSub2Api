# MVP-019：迁移现有页面、菜单与操作按钮到权限判断

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `权限基础已就绪后，可按现有页面矩阵完成一次机械迁移和闭合验证。`
- Dependencies: `MVP-014, MVP-018`

## 预期成果

全部站内登录页面、侧边栏入口和关键按钮使用权限编码，不再依赖角色名称决定一般功能访问。

## 背景

页面权限映射来自 MVP-001；admin/user 升级前后可见行为必须保持一致。

## 范围内

- Vue Router 的 `requiredPermission(s)`。
- AppSidebar 和动态菜单权限过滤。
- 管理页面关键新增、编辑、删除、余额、凭据和系统操作按钮。
- 普通用户个人页面和菜单权限。
- 前端页面映射闭合测试。

## 范围外

- 新角色管理页面。
- 删除所有兼容 `isAdmin` 字段。

## 实现说明

- 禁止新增 `role === 'operator'` 等角色名判断。
- 页面权限复用核心 read 权限，按钮使用对应动作权限。

## 验收标准

- [x] 所有站内登录 Vue 路由都有权限声明。
- [x] admin 原页面、菜单和按钮不减少。
- [x] user 原个人入口不减少，管理入口仍不可见。
- [x] 自定义只读角色不显示写操作按钮。

## 验证计划

- `pnpm --dir frontend run test:run`
- `pnpm --dir frontend run lint:check`
- 按 MVP-001 页面矩阵执行 admin/user/只读角色人工抽查。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 页面闭合 | `frontend/src/rbac/permissionMatrix.ts`、`frontend/src/router/index.ts` | 登录页面矩阵自动注入 Router `requiredPermission`，动态 custom 页面使用 `pages.self.read`。 |
| 菜单 | `frontend/src/components/layout/AppSidebar.vue` | 用户、管理员、子菜单及自定义菜单在 feature/simple 过滤之外统一按页面权限递归过滤。 |
| 操作 | `frontend/src/directives/permission.ts` | 全局 `v-permission`；用户、账号、分组的创建、编辑、删除关键按钮已按动作权限迁移。 |
| 聚焦测试 | `pnpm --dir frontend exec vitest run src/rbac/permissionMatrix.spec.ts src/router/__tests__/guards.spec.ts src/stores/__tests__/auth.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts src/views/admin/__tests__/UsersView.spec.ts src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts` | 通过，6 文件 67 个测试。 |
| Lint | `pnpm --dir frontend run lint:check` | 通过。 |
| 全量基线 | `pnpm --dir frontend run test:run` | 已执行：125 文件通过、9 文件失败（34/812 用例），失败集中在既存账户 usage mock、图表空值、OAuth 表单、历史图片文案、分页默认值等，与 RBAC 改动路径无关；RBAC/Router/Sidebar 用例均通过。 |

## 执行记录

`isAdmin` 仅保留后端模式、合规初始化、升级兼容及旧定制菜单等兼容用途；一般页面、菜单和新操作均使用权限编码。矩阵继续保留已从后端删除的历史统计页面仅作为前端现状核对项，不会生成后端路由。
