# MVP-001：建立旧资产分类与新体系保护门禁

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.5 开发日`
- Estimate rationale: `以静态调用链、资源清单和可重复保护测试为单一审计成果。`
- Dependencies: `none`

## 预期成果

形成可执行的旧专属、共享能力、新体系三分类，并用自动化测试锁定不得删除的关键新体系与公共业务入口。

## 背景

工作树同时包含刚完成的新体系和大量旧固定逻辑；只有确认旧专属的资产才允许删除。

## 范围内

- 旧文件、符号、API、配置、表、Redis、页面调用者清单。
- `usage_logs`、计费、路由、调度、新体系、通用权限保护清单。
- 可重复静态边界测试或审计文档。

## 范围外

- 删除运行时代码、表或页面。

## 实现说明

- 优先使用 `rg` 和调用者检查；不因名称相似删除共享组件。
- 保护 `dynamic_token_*`、`tokenstat`、`token_stat_*` 和 `/admin/token-statistics`。

### 资产分类

| 分类 | 资产 |
|---|---|
| 旧运行时专属 | `token_statistics_{codec,accumulator,sync,quota_repo}.go`、`token_usage_read_repair.go`、`daily_token_quota_{port,accounting}.go`、`token_statistics_scheduler.go`、旧 Redis cache/repository |
| 旧报表专属 | `token_usage_report_{repo,service,handler}.go`、`token_usage_merge.go`、`current_token_usage_reader.go`、`frontend/src/views/admin/token-usage/` 及其专属组件/API |
| 旧限额管理专属 | `model_token_quota_admin_service.go`、`user_model_token_quota_admin_service.go`、对应 handler；全局/用户/默认/批量/分组候选旧前端组件 |
| 旧持久化专属 | 六个 `*_token_daily_{usage,limit_config}` Ent schema、生成实体、旧配置和旧六张 MySQL 表 |
| 共享保留 | `gateway_service.go` 的路由/调度/重试主体、统一 usage 结果、计费与余额、`usage_logs`、API Key/订阅额度、通用图表/日期/分页/权限组件 |
| 新体系保护 | `internal/service/tokenstat`、`internal/repository/tokenstat`、`dynamic_token_usage.go`、`dynamic_token_quota.go`、五张 `token_stat_*` schema、`/admin/token-statistics/*`、`TokenStatisticsView.vue`、三份新文档 |

### 资源边界

- 旧 Redis：`sub2api:token_stats:{model,user_model,group_candidate}:*`、`quota:daily_token:*`、`sub2api:token_stats:sync_lock`，仅停止访问并等待 TTL。
- 新 Redis：所有 `sub2api:dynamic_token_stats*`，禁止扫描、删除或改义。
- 旧 MySQL：`model_token_daily_usages`、`user_model_token_daily_usages`、`group_candidate_token_daily_usages` 及对应三个 `*_limit_configs`。
- 保护 MySQL：`token_stat_*`、`usage_logs`、`users`、`api_keys`、`groups`、`accounts`。
- 通用权限：`token_usage.read/manage`、`token_quota.read/update` 保留，只删除旧路由绑定。

## 验收标准

- [x] 三套旧固定功能的生产资产与调用者已分类。
- [x] 共享计费、用量明细、路由调度和通用 RBAC 明确标记为保留。
- [x] 新体系代码、Redis、MySQL、API、前端入口和文档保护清单完整。
- [x] 后续 MVP 的删除范围能由该清单逐项核对。

## 验证计划

- `rg -n "TokenStatisticsModel|ModelTokenDaily|UserModelTokenDaily|GroupCandidateTokenDaily|/admin/token-usage/|/admin/model-token-quotas" backend frontend/src --glob "!backend/internal/web/dist/**"`
- `rg -n "dynamic_token_statistics|DynamicTokenStat|token_stat_projections|/admin/token-statistics" backend frontend/src`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 旧资产扫描 | 验证计划第一条 `rg` | 找到旧运行时、报表、限额、Ent、API 和前端调用者并完成逐类归档 |
| 新体系正向扫描 | 验证计划第二条 `rg` | 确认 `dynamic_token_statistics`、`tokenstat`、五张新表和新前端入口均存在 |
| 分类门禁 | 本文“资产分类”“资源边界” | 明确旧专属、共享保留、新体系保护以及后续逐项删除范围 |

## 执行记录

2026-07-30：完成资产与调用者审计。未删除任何生产代码；未知或共享资产均归入保护范围。
