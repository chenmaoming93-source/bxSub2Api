# MVP-003：移除旧后端管理 API 与专属服务

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `旧统计报表、全局/用户/分组候选限额管理面可作为一个可验证的后端删除切片。`
- Dependencies: `MVP-001, MVP-002`

## 预期成果

旧 `/admin/token-usage/*`、旧模型和用户模型限额 API 返回 404，专属 Handler、Service、Repository、DTO 和 Wire 依赖被移除。

## 背景

通用 `token_usage.*` 与 `token_quota.*` 权限仍由新体系使用，不能删除。

## 范围内

- 删除旧统计查询、选项和默认目标 API。
- 删除全局、用户、默认、批量及分组候选旧限额 API。
- 删除专属 handler/service/repository/DTO 和测试。
- 更新路由与 Wire。

## 范围外

- 新 `/admin/token-statistics/*`。
- Ent schema 与生产表。

## 实现说明

- 旧 API 不提供兼容层或重定向。
- 权限代码保留，仅更新为新通用体系语义。

## 验收标准

- [x] 所有旧管理 API 路由已移除。
- [x] 旧专属后端服务和依赖注入已删除。
- [x] 通用权限和新管理 API 完整保留。
- [x] 旧 API 404 与新 API 可达测试通过。

## 验证计划

- `cd backend && go test ./internal/handler/... ./internal/server/... ./cmd/server/...`
- `rg -n '\"/token-usage\"|\"/model-token-quotas\"|default-model-token-quotas' backend/internal --glob "*.go"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 路由与管理面 | `backend/internal/server/routes/admin.go`、`backend/internal/handler/{handler.go,wire.go}` | 旧 token-usage、model-token-quotas、用户模型限额与默认限额路由/Handler 已移除 |
| 专属实现 | `backend/internal/{handler/admin,service,repository}` | 旧报表与固定限额专属 Handler、Service、Repository、DTO、Wire provider 已删除 |
| 权限与新 API | `backend/internal/rbac/permissions.go`、`backend/internal/server/routes/admin.go` | 通用 token 权限和 `/admin/token-statistics/*` 完整保留 |
| 测试 | `cd backend && go test ./internal/handler/... ./internal/server/... ./cmd/server/...` | 通过 |
| 静态扫描 | `rg -n '"/token-usage"|"/model-token-quotas"|default-model-token-quotas' backend/internal --glob "*.go"` | 生产代码零命中；移除断言测试保留旧路径字符串 |

## 执行记录

2026-07-30：开始删除旧路由、Handler、Service、Repository 与 Wire 依赖；新 `/admin/token-statistics/*` 保持不变。
2026-07-30：完成旧管理 API 与专属依赖删除，管理端、路由和 server 测试通过。
