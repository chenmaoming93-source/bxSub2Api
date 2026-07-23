# MVP-018：建立前端权限 Store、守卫与组件能力

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `提供一个可被现有页面逐步采用的前端权限基础层，不同时迁移全部页面。`
- Dependencies: `MVP-017`

## 预期成果

前端可以从 `/auth/me` 加载角色和权限，并通过统一 API 判断单项、任一、全部和超级管理员权限。

## 背景

现有 `isAdmin` 需要保留兼容，但不得继续作为新增功能的一般授权方式。

## 范围内

- Auth Store 的 roles、permissions、permissionVersion。
- `can`、`canAny`、`canAll`、`isSuperAdmin`。
- Vue Router 权限元数据类型和守卫。
- `usePermission` 与可选 `v-permission`。
- 统一 403 或安全回退体验。

## 范围外

- 全部现有页面、菜单和按钮迁移。
- 角色管理页面。

## 实现说明

- `*` 不展开，前端判断时直接通配。
- 刷新和重新登录后权限状态正确恢复。
- 前端结果仅用于展示，后端仍独立鉴权。

## 验收标准

- [x] 四类权限判断均有单元测试。
- [x] 无权限路由被守卫阻止。
- [x] admin `*` 可访问所有声明页面。
- [x] 旧 `isAdmin` 调用仍能工作。

## 验证计划

- `pnpm --dir frontend exec vitest run src/stores/__tests__/auth.spec.ts src/router/__tests__/guards.spec.ts`
- `pnpm --dir frontend run lint:check`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Store | `frontend/src/stores/auth.ts` | 持久化 roles、permissions 和版本字段，提供 `can`、`canAny`、`canAll`、`isSuperAdmin`；`*` 运行时通配，旧 `isAdmin` 保留。 |
| 复用能力 | `frontend/src/composables/usePermission.ts`、`frontend/src/router/rbac-meta.d.ts` | 组件统一组合式 API，Router 支持单项/多项及 any/all 元数据。 |
| 守卫 | `frontend/src/router/index.ts` | 已认证但无权限的路由安全回退 `/dashboard`；后端仍独立执行 403 授权。 |
| 测试 | `pnpm --dir frontend exec vitest run src/stores/__tests__/auth.spec.ts src/router/__tests__/guards.spec.ts` | 通过，2 文件 57 个测试。 |
| Lint | `pnpm --dir frontend run lint:check` | 通过。 |

## 执行记录

路由层无权限时不展示后端错误页，统一回退用户 Dashboard；直接请求接口仍由后端返回 403。legacy admin 在刷新拿到 `*` 前仍由 `role=admin` 兼容视为超级管理员，避免升级瞬间丢菜单。
