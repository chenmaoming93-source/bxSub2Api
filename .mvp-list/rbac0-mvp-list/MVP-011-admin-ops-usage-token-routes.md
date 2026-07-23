# MVP-011：迁移 Ops、用量与 Token 统计管理路由

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `集中处理观测查询和有限运维写操作，范围可按现有路由组独立验证。`
- Dependencies: `MVP-006`

## 预期成果

Ops、错误日志、用量清理、Token 统计和模型 Token 配额具备读写分离权限。

## 背景

主要路径为 `/admin/ops`、`/admin/usage`、`/admin/token-usage` 和模型配额。

## 范围内

- Ops 仪表盘、并发、实时流量和 WebSocket。
- 告警规则、事件、静默、运行时和通知配置。
- 错误、请求详情、系统日志及清理。
- 管理用量、清理任务、Token 统计和模型配额。

## 范围外

- 商业支付和渠道监控。
- 指标采集本身的业务改造。

## 实现说明

- Ops 读权限与规则/配置/清理写权限分离。
- WebSocket 在握手前完成同一权限校验。
- 查询其他用户用量属于管理权限，不复用 self 权限。

## 验收标准

- [x] 范围内所有路由进入 Registry。
- [x] Ops 只读角色不能修改规则、解决错误或清理日志。
- [x] Token 统计查询与配额修改权限分离。
- [x] WebSocket 无权限时拒绝建立连接。

## 验证计划

- `cd backend && go test ./internal/server/... -run 'Admin.*(Ops|Usage|Token)'`
- `cd backend && go test ./internal/handler/admin/... -run '(Ops|Usage|Token)'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/server/routes/admin.go` | Ops、WebSocket、用量管理、Token 统计和模型配额全部通过统一 registrar 注册；读、配置写、日志管理和配额写权限分离。 |
| 契约测试 | `backend/internal/server/routes/rbac_admin_ops_routes_test.go` | 验证 WebSocket 在握手处理器前绑定 `ops.read`，以及规则修改、错误解决、日志清理、用量清理和配额更新的独立权限。 |
| 测试 | `go test ./internal/server/... -run 'RBAC|Admin.*(Ops|Usage|Token)' -count=1` | 通过。 |
| 测试 | `go test ./internal/handler/admin/... -run '(Ops|Usage|Token)' -count=1` | 通过。 |
| 编译验证 | `go test ./cmd/server -run '^$'` | 通过。 |

## 执行记录

`/admin/ops/ws/qps` 使用与 HTTP 查询相同的 registrar，因此权限校验发生在 WebSocket 升级之前；清理和错误解决统一使用 `ops.logs.manage`，不会由 `ops.read` 放行。
