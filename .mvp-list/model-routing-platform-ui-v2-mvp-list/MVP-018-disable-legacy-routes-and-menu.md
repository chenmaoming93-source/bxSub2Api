# MVP-018：停用旧菜单与不可用路由

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 集中处理菜单、路由、权限映射、重定向和失效链接，结果可通过路由测试独立验证。
- Dependencies: `MVP-006, MVP-010, MVP-013`

## 预期成果

停用页面不再显示在菜单中，也不能通过 URL 访问；有效新页面和分组管理依赖组件继续正常。

## 背景

菜单位于 `frontend/src/components/layout/AppSidebar.vue`，路由位于 `frontend/src/router/index.ts`，权限矩阵位于 `frontend/src/rbac/permissionMatrix.ts`。

## 范围内

- 移除默认路由、渠道、订阅、公告、IP/代理、兑换码、优惠码管理员入口。
- 移除 `/admin/channels` 父级重定向。
- 清理系统设置内失效链接。
- 保留旧页面源码和旧组件源码。
- 验证有效路径 `/admin/dashboard`、`/admin/usage`、`/dashboard`、`/usage` 等正常。

## 范围外

- 不删除停用页面文件。
- 不禁用用户侧同名功能。
- 不清理未明确列入清单的其他管理员页面。

## 实现说明

直接访问停用路径进入 catch-all 404；权限守卫不能继续为这些页面提供可访问路由。

## 验收标准

- [x] 停用路径全部进入 404。
- [x] 停用菜单项不再渲染。
- [x] `/admin/channels` 不再重定向到定价页。
- [x] 分组管理仍能使用 `GroupModelRoutingEditor`。
- [x] 旧源文件未删除。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- 路由守卫和导航集成测试。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Route/menu audit | `frontend/src/router/index.ts`; `frontend/src/components/layout/AppSidebar.vue` | 停用入口及 `/admin/channels` 重定向已移除 |
| Legacy preservation | 旧页面文件 | 源文件保留 |

## 执行记录

停用管理员旧入口及失效路由，保留有效 V2 页面、分组模型路由编辑器和旧源文件。
