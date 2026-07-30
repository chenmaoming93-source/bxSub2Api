# MVP-020：交付角色管理与用户多角色界面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `后端 API 和前端权限基础完成后，可交付一个完整可操作的 RBAC 管理体验。`
- Dependencies: `MVP-016, MVP-018`

## 预期成果

授权管理员可在界面查看和管理角色、按模块配置权限，并在用户管理页分配多个角色。

## 背景

系统权限不可通过界面创建、删除或修改编码；admin/user 角色需要特殊保护提示。

## 范围内

- 角色列表、创建、编辑、启停和软删除。
- 按模块展示权限、风险等级和描述。
- 角色权限全量保存。
- 用户管理页角色查看和多选分配。
- admin、user、`*`、最后超级管理员和高影响操作提示。

## 范围外

- RBAC1 角色继承。
- 自定义系统权限编码。

## 实现说明

- 内置 admin 禁止修改关键属性。
- 修改 user 权限和分配 admin 需要二次确认。
- 页面与操作自身也按 RBAC 权限控制。

## 验收标准

- [x] 可完成自定义角色创建、授权、分配用户和停用的完整流程。
- [x] admin 的 `*` 不可在界面移除。
- [x] 无相应权限的管理员看不到或无法执行管理操作。
- [x] API 错误和并发冲突有明确反馈。

## 验证计划

- `pnpm --dir frontend exec vitest run src/views/admin src/components/admin/user`
- `pnpm --dir frontend run lint:check`
- 人工执行角色创建→权限配置→用户分配→登录验证流程。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API Client | `frontend/src/api/admin/rbac.ts` | 覆盖角色 CRUD/启停、权限目录与全量替换、用户角色读取与全量替换。 |
| 角色页面 | `frontend/src/views/admin/RolesView.vue` | 角色列表、创建、启停、软删除、按模块/风险显示权限和保存；admin `*` 禁用编辑，修改 user 二次确认。 |
| 用户分配 | `frontend/src/components/admin/user/UserRolesModal.vue`、`frontend/src/views/admin/UsersView.vue` | 用户操作菜单按 `users.roles.assign` 展示多角色选择；授予 admin 二次确认并显示最后管理员/自提权约束提示。 |
| 类型检查 | `pnpm --dir frontend exec vue-tsc --noEmit` | 通过。 |
| 聚焦测试 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/RolesView.rbac.spec.ts src/rbac/permissionMatrix.spec.ts src/router/__tests__/guards.spec.ts` | 通过，3 文件 40 个测试。 |
| Lint | `pnpm --dir frontend run lint:check` | 通过。 |
| 后端回归 | `go test ./internal/rbac ./internal/handler/admin ./cmd/server -run 'RBAC|^$' -count=1` | 通过。 |

## 执行记录

权限以模块分组平铺，避免引入 RBAC1 树形继承误导。critical/high 风险着色；内置 admin 权限不可勾选，内置 user 修改、授予 admin、删除角色均二次确认。API 错误在页面或弹窗内保留并展示，服务端并发保护仍是最终裁决。
