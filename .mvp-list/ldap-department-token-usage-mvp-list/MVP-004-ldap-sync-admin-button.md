# MVP-004：用户管理页 LDAP 同步按钮

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.75 个开发日`
- Estimate rationale: 复用现有 UsersView 操作区、API client、toast 和权限体系，范围集中但包含交互测试。
- Dependencies: `MVP-003`

## 预期成果

管理员可以在 `/admin/users` 点击“同步 LDAP 用户信息”，看到同步中的状态和完成汇总。

## 背景

`frontend/src/views/admin/UsersView.vue` 已有刷新、筛选、列设置和属性配置等操作。API client 位于 `frontend/src/api/admin/users.ts`，页面使用现有权限指令和 `appStore` 提示机制。

## 范围内

- 在 users API client 增加同步接口方法和响应类型。
- 在 UsersView 操作区增加同步按钮。
- 使用 `users.update` 权限控制。
- 同步期间按钮禁用并显示 loading。
- 完成后显示成功、部分失败或失败提示。
- 增加同步结果的国际化文本。
- 增加 UsersView/API mock 测试。

## 范围外

- 异步任务轮询。
- 修改用户编辑表单的部门手工编辑能力。
- 部门 Token 图表。

## 实现说明

- 采用已确认的同步批处理模式，前端等待接口返回汇总。
- 错误提示优先显示后端 detail/message，缺失时使用本地化默认文案。
- 不在前端保存或传递 LDAP 密码。
- 可在同步成功后刷新用户列表，使 department 列数据及时显示。

## 验收标准

- [x] `/admin/users` 显示同步按钮并位于现有操作区。
- [x] 无 `users.update` 权限时按钮不可执行或不显示。
- [x] 点击后按钮进入 loading，重复点击被阻止。
- [x] 成功和部分失败响应均显示可读汇总。
- [x] 接口异常显示错误提示并恢复按钮状态。
- [x] 中英文语言包和前端测试通过。

## 验证计划

- `pnpm --dir frontend run test:run -- src/views/admin/__tests__/UsersView.spec.ts`
- `pnpm --dir frontend run typecheck`
- 人工访问 `/admin/users` 验证按钮布局和 loading 状态。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `frontend: pnpm run test:run -- src/views/admin/__tests__/UsersView.spec.ts` | 2 tests passed |
| 类型检查 | `frontend: pnpm run typecheck` | 通过 |
| 检查 | `frontend/src/api/admin/users.ts`, `frontend/src/views/admin/UsersView.vue` | 增加同步 API、权限指令、loading/防重入、成功/部分失败/异常提示和完成后刷新 |
| 检查 | `frontend/src/i18n/locales/{zh,en}.ts` | 中英文同步文案已加入 |

## 执行记录

- 同步接口不接受或发送 LDAP 密码；按钮由 `v-permission="'users.update'"` 控制。

