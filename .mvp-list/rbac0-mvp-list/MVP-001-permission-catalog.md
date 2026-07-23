# MVP-001：建立可审计的权限目录与接口页面矩阵

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `产出一个完整目录和机器可检查的分类基线，不涉及运行时改造，可独立评审。`
- Dependencies: `none`

## 预期成果

形成稳定的权限编码目录，并将现有 522 条接口及全部 Vue 路由映射为“受 RBAC 控制”或“明确排除”。

## 背景

源清单为 `RBAC_ENDPOINT_INVENTORY.md`；权限应按业务动作而不是逐 URL 定义。

## 范围内

- 新增 `backend/internal/rbac/permissions.go` 权限定义。
- 建立接口、页面、菜单和高风险按钮的权限矩阵。
- 固化当前 `427` 条受控、`95` 条排除的迁移基线。
- 为每个排除类别记录原因和现有认证方式。

## 范围外

- 数据库表和运行时中间件。
- 修改现有路由行为。

## 实现说明

- 权限编码遵循 `<resource>.<scope>.<action>` 或 `<resource>.<action>`。
- 列表与详情可共用 read；余额、凭据、角色分配等单独授权。
- 目录应包含 code、name、module、description、risk level。

## 验收标准

- [x] 522 条现有接口均有唯一分类，不存在未分类条目。
- [x] 受控和排除数量与基线一致。
- [x] 所有现有 Vue 路由具有权限或公开分类。
- [x] 权限编码无重复，`*` 仅作为系统通配权限。

## 验证计划

- `cd backend && go test ./internal/rbac/...`
- 人工抽查 `users`、`accounts`、`settings`、个人 API Key 和支付 webhook 映射。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端测试 | `cd backend && go test ./internal/rbac/...` | 通过；目录唯一性、字段完整性、通配权限和 `522 = 427 + 95` 闭合检查全部通过。 |
| 前端测试 | `cd frontend && pnpm exec vitest run src/rbac/permissionMatrix.spec.ts` | 通过；1 个测试文件、4 个测试全部通过，并从 `router/index.ts` 自动抽取静态路由检查无漏项。 |
| 实现 | `backend/internal/rbac/permissions.go`、`backend/internal/rbac/baseline.go` | 建立集中权限目录以及受控/排除接口基线；排除项记录数量、认证方式与原因。 |
| 页面矩阵 | `frontend/src/rbac/permissionMatrix.ts` | 全部现有 Vue 静态路由均归类为 `public`、`authenticated` 或带权限编码的 `rbac`。 |

## 执行记录

以 `RBAC_ENDPOINT_INVENTORY.md` 的 522 条接口作为兼容迁移基线：427 条进入 RBAC，95 条按公开认证、支付回调、外部集成和模型网关四类明确排除。本 MVP 只建立目录与机器可检验矩阵，没有改变运行时鉴权行为。
