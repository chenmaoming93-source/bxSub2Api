# MVP-003：交付统计投影管理 API 与配置发布

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `覆盖投影 CRUD、状态机、注册信息和多实例配置刷新，形成可独立调用的管理能力。`
- Dependencies: `MVP-001, MVP-002`

## 预期成果

管理员能够通过新 `/admin/token-statistics/*` API 查询注册信息、创建投影、发布、激活和停用投影，运行实例能够自动刷新配置。

## 背景

已有维度和指标的组合应纯配置生效。投影管理需要新增 `token_usage.manage`，查询复用 `token_usage.read`。

## 范围内

- 维度与指标注册信息 API。
- 投影列表、创建、详情、修改、发布和停用 API。
- `DRAFT/PUBLISHED/ACTIVE/DISABLED` 状态校验。
- 维度签名去重。
- Redis 配置版本、通知和本地缓存刷新。
- RBAC 权限、路由和合约测试。

## 范围外

- Redis Token累计。
- 限额管理。
- 前端页面。

## 实现说明

- Pub/Sub 丢消息时通过周期版本检查恢复。
- 被有效限额引用的投影停用规则在 MVP-008 完成。
- API 路径不得复用旧 `/admin/token-usage/*`。

## 验收标准

- [x] 注册信息接口只返回代码注册的维度和指标。
- [x] 相同维度不同顺序不能创建重复投影。
- [x] 状态非法跳转被拒绝。
- [x] 发布投影后配置版本递增并通知实例。
- [x] `token_usage.read` 与 `token_usage.manage` 权限边界正确。
- [x] 路由、Service、Repository 和 API 合约测试通过。

## 验证计划

- `cd backend && go test ./internal/handler/admin/... ./internal/service/... ./internal/server/routes/... ./internal/rbac/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API | `backend/internal/handler/admin/dynamic_token_statistics_handler.go`、`backend/internal/server/routes/admin.go` | 注册信息和投影 CRUD/状态路由已接入独立 `/admin/token-statistics/*` |
| 状态与发布 | `backend/internal/service/tokenstat/projection_admin.go` | DRAFT/PUBLISHED/ACTIVE/DISABLED 校验、签名去重、Redis 版本和 Pub/Sub 已实现 |
| RBAC | `backend/internal/rbac/permissions.go`、`backend/sqlArchiving/163_seed_rbac_compatibility.sql` | 新增 `token_usage.manage`，读写路由权限分离 |
| 测试 | `cd backend && go test ./internal/handler/admin/... ./internal/service/... ./internal/server/routes/... ./internal/rbac/... ./cmd/server` | 通过 |

## 执行记录

2026-07-30：完成投影管理 API、状态机、配置版本发布、RBAC 和服务装配；完整验收测试通过。
