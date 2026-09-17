# MVP-004：提供管理员 Token 限额重置 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: `复用已完成的重置服务，工作集中于请求绑定、错误映射、RBAC 路由、依赖注入和契约测试。`
- Dependencies: `MVP-003`

## 预期成果

具备 `token_quota.update` 权限的管理员能够通过稳定的后台 API 提交具体维度子集、指标和周期，并获得明确、可机器识别的重置结果。

## 背景

- 管理 Handler 位于 `backend/internal/handler/admin/dynamic_token_statistics_handler.go`。
- 管理路由位于 `backend/internal/server/routes/admin.go` 的 `/admin/token-statistics` 分组。
- 现有权限 `token_quota.update` 已用于创建、修改和启停限额，本需求复用该权限，不新增 RBAC 权限点。
- 依赖注入入口涉及 `backend/internal/handler/wire.go` 及相应 provider wiring。

## 范围内

- 新增 `POST /api/v1/admin/token-statistics/quota-usage/reset`。
- 定义管理员请求 DTO，并通过注册表验证维度代码、值类型、metric 和 period。
- 禁止请求传入 projection、dimension hash、shard、Redis Key 或快照值。
- 将服务结果映射为 `RESET`、`PARTIAL_RESET`、`NO_QUOTA`、`NO_USAGE` 响应。
- 核心依赖不可用且没有成功项时映射为 503。
- 路由注册使用 `token_quota.update`。
- 增加 Handler、路由权限和 wiring 测试。

## 范围外

- 不实现管理员页面。
- 不实现 integrations 外部接口。
- 不新增管理员审计数据库表。
- 不改变其他 quota CRUD API。

## 实现说明

- 请求结构应复用 tokenstat 的 `DimensionValue` JSON 表达，避免管理员入口创造第二套类型协议。
- 合法但没有业务对象的 `NO_QUOTA`、`NO_USAGE` 使用 HTTP 200，并通过 `status` 区分。
- `PARTIAL_RESET` 使用 HTTP 200，包含准确的成功、失败及无用量计数。
- 结构化日志补充管理员身份时应复用现有上下文能力。

## 验收标准

- [x] 新路由准确注册在 `/api/v1/admin/token-statistics/quota-usage/reset`。
- [x] 路由要求管理员认证和 `token_quota.update` 权限。
- [x] 合法请求可以提交多个具体维度，并返回服务的完整汇总字段。
- [x] wildcard、未知维度、类型错误、非法周期或不允许限额的指标返回 400。
- [x] `NO_QUOTA`、`NO_USAGE`、`RESET`、`PARTIAL_RESET` 的 HTTP 和 JSON 契约稳定。
- [x] 核心依赖不可用且无成功项时返回 503 `TOKEN_QUOTA_RESET_UNAVAILABLE`。
- [x] 现有 projection、quota 和 query 管理路由不受影响。

## 验证计划

- `cd backend && go test ./internal/handler/admin ./internal/server/routes`
- `cd backend && go test ./internal/service/tokenstat ./internal/repository/tokenstat`
- 使用 Handler Stub 分别断言四种业务状态、400、403 和 503 映射。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Handler | `backend/internal/handler/admin/dynamic_token_statistics_handler.go` | strict JSON DTO、四状态响应、400 与 503 映射及脱敏管理员日志。 |
| Route/RBAC | `backend/internal/server/routes/admin.go`、`rbac_admin_ops_routes_test.go` | 注册 `/quota-usage/reset` 并复用 `token_quota.update`。 |
| Wiring | `backend/cmd/server/wire.go`、`wire_gen.go` | `go generate ./cmd/server` 生成并通过 `go test ./cmd/server`。 |
| 定向验证 | `go test ./internal/handler/admin -run 'Test(DynamicTokenStatistics|AdminQuotaReset|QuotaUpdateRequest)'`；`go test ./internal/server/routes`；tokenstat 两包 | 全部通过。 |
| 广泛验证 | `go test ./internal/handler/admin ./internal/server/routes` | routes 通过；admin 包仅既有 `TestGroupRequestsAcceptLegacyAndCandidateModelRouting` 失败（与本功能无关，定向测试通过）。 |

## 执行记录

2026-09-09 完成。未知内部字段由 `DisallowUnknownFields` 直接拒绝；领域注册表验证由共享 reset service 执行。生产 wiring 通过仓库 Wire 命令生成，管理员与后续 integrations 入口可复用同一服务实例。
