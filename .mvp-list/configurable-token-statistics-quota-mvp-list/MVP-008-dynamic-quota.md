# MVP-008：交付通用动态限额

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 开发日`
- Estimate rationale: `包含规则管理、精确Redis检查、调度前后两阶段接入和fail-open，工作量略高但仍是一个完整业务切片。`
- Dependencies: `MVP-003, MVP-004, MVP-005`

## 预期成果

管理员能够为任意已注册投影和指标创建观察或强制限额；模型请求在调度前和账号选择后执行精确Redis检查，基础设施失败时fail-open。

## 背景

限额采用简单后结算，允许少量异步和并发超额。投影可独立存在，限额必须依赖投影。

## 范围内

- 限额列表、创建、更新、启用、停用API。
- `token_quota.read/update` RBAC。
- 自动创建缺失投影或启用指标。
- 不完整周期的默认下周期生效。
- 规则缓存和配置刷新。
- 调度前、账号选择后精确Redis检查。
- OBSERVE/ENFORCE和fail-open。
- 被有效限额引用的投影停用保护。

## 范围外

- 额度预占。
- 严格零超额。
- 旧固定限额迁移或兜底。

## 实现说明

- 账号维度超限优先排除候选账号并继续调度。
- 只有成功读取且确认超限时才限制。
- 禁止调用旧DailyTokenQuota逻辑。

## 验收标准

- [x] 投影可无额度独立运行。
- [x] 缺失投影可自动创建并使规则等待生效。
- [x] OBSERVE只记录不阻断。
- [x] ENFORCE在确认超限时限制请求或候选。
- [x] Redis超时、错误和规则缓存失败均fail-open。
- [x] 调度前后两阶段维度检查正确。
- [x] 不读取旧限额缓存或表。

## 验证计划

- `cd backend && go test ./internal/service/... ./internal/handler/admin/... ./internal/server/... -run "DynamicToken.*Quota|TokenStat.*Quota"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 限额领域与管理 | `backend/internal/service/tokenstat/quota.go`、`quota_admin.go`、`projection_admin.go` | 完成规则缓存、精确 Redis 读取、OBSERVE/ENFORCE、自然周期生效、缺失投影自动创建、等待规则自动启用和投影引用保护 |
| 请求路径接入 | `backend/internal/service/dynamic_token_quota.go`、`gateway_service.go` | 调度前检查用户/分组维度；候选阶段检查账号/上游模型维度并跳过超限账号；所有基础设施错误 fail-open |
| API 与权限 | `backend/internal/handler/admin/dynamic_token_statistics_handler.go`、`backend/internal/server/routes/admin.go` | 完成限额列表、创建、更新、启停 API，并绑定 `token_quota.read/update` |
| 自动化测试 | `cd backend && go test ./internal/service/... ./internal/handler/admin/... ./internal/server/... -run "DynamicToken.*Quota\|TokenStat.*Quota\|ProjectionLifecycle\|RBACAdminOps"` | 通过；覆盖观察/强制模式、Redis 错误、两阶段匹配、投影与限额生命周期及路由权限 |

## 执行记录

2026-07-30：完成通用动态限额。新体系仅访问 `token_stat_*` 表和 `sub2api:dynamic_token_stats:*` Redis 命名空间，不调用旧固定 DailyTokenQuota 逻辑。
