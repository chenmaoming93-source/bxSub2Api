# MVP-004：交付用户模型统计 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `成果边界单一，包含实现与针对性验证，预计落在目标工作量的 0.5 至 1.5 倍内。`
- Dependencies: `MVP-001`

## 预期成果

管理员可分页查看指定用户在各实际上游模型上的每日用量。

## 背景

来源为 `../Token消耗统计页面实施Plan.md`。用户不存在与软删除用户需区分；有历史用量的软删除用户应能展示并标注。

## 范围内

- 实现用户模型统计查询及汇总
- 关联当前用户模型限额并兼容软删除用户
- 注册 `GET /api/v1/admin/token-usage/users`
- 覆盖必选用户、可选模型、分页、汇总与权限测试

## 范围外

- 不实现用户搜索和前端页面。

## 实现说明

- 保持管理员只读边界、项目全局时区和总 Token 既有口径。
- 所有列表必须有界；不要为了本 MVP 改动网关、计费或 Token 记账主链路。

## 验收标准

- [x] 指定用户可获得各模型的有界分页结果
- [x] 模型过滤、汇总和当前限额正确
- [x] 软删除用户历史用量可查，越权请求被拒绝
- [x] 相关 Go 测试通过

## 验证计划

- `cd backend; go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/...`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `token_usage_report_{repo,service,handler}.go` | 用户模型日报、可选模型、分页、汇总和当前限额 |
| 软删除兼容 | `TestTokenUsageReportRepositoryUserIncludesSoftDeleted` | 原始用户关联保留 `deleted_at` 标记与历史用量 |
| 权限及完整验收 | Admin 路由中间件；`go test ./internal/handler/admin/... ./internal/service/... ./internal/repository/...` | 全部通过 |

## 执行记录

用户关联不附加软删除过滤；响应通过 `user_deleted` 明确标记软删除用户。
