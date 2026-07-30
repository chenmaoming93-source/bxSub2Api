# MVP-009：迁移账号、代理与上游 OAuth 管理路由

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `围绕上游凭据和调度资源形成一个高敏感权限切片。`
- Dependencies: `MVP-006`

## 预期成果

账号、代理、OpenAI、Gemini、Antigravity OAuth 接口按读取、修改、凭据和运维动作授权。

## 背景

该模块包含大量敏感凭据和状态恢复操作，不能只使用一个笼统的 accounts 权限。

## 范围内

- accounts 全部 CRUD、批量、测试、刷新、配额和模型同步路由。
- proxy CRUD、质量测试、批量和账号关联。
- OpenAI/Gemini/Antigravity OAuth 路由。
- 凭据读取、导入、交换和恢复等高风险权限。

## 范围外

- 系统设置和数据备份。
- 前端账号页面按钮改造。

## 实现说明

- `accounts.credentials.read/update` 与普通 `accounts.read/update` 分离。
- OAuth code 交换、Cookie 导入和批量凭据更新按 critical 风险登记。

## 验收标准

- [x] 范围内所有路由进入 Registry。
- [x] 无凭据权限的账号管理员无法读取或修改凭据。
- [x] 普通查询角色不能执行刷新、恢复、测试和批量操作。
- [x] admin 行为保持一致。

## 验证计划

- `cd backend && go test ./internal/server/... -run 'Admin.*(Account|Proxy|OAuth)'`
- `cd backend && go test ./internal/handler/admin/... -run '(Account|Proxy|OAuth)'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 路由迁移 | `cd backend && go test ./internal/server/... -run 'RBAC|Admin.*(Account|Proxy|OAuth)' -count=1` | 通过；72 条账号、代理及 OAuth 路由进入 Registry。 |
| Handler 回归 | `cd backend && go test ./internal/handler/admin/... -run '(Account|Proxy|OAuth)' -count=1` | 通过。 |
| 应用构建 | `cd backend && go test ./cmd/server -run '^$'` | 通过。 |
| 敏感映射 | `rbac_admin_account_routes_test.go` | 凭据读取、Codex 导入、批量凭据更新、OAuth code 交换、账号测试和代理质量测试均使用独立高风险权限。 |

## 执行记录

共迁移 72 条。账号普通查询使用 `accounts.read`，凭据导出/预览使用 `accounts.credentials.read`，导入、交换、Cookie/OAuth 凭据和批量凭据更新使用 critical 的 `accounts.credentials.update`；刷新、恢复、测试、配额和调度状态操作使用 `accounts.operate`。代理查询、CRUD 与测试/质检分别使用 read/create/update/delete/operate。
