# 可配置 Token 统计与限额验收报告

日期：2026-07-30

## 验收结论

新体系已完成注册表、投影配置、异步 Redis 统计、MySQL 同步、自然周期封账、动态限额、管理员查询与操作页面。新代码使用独立的表、Redis 命名空间和管理 API。旧三套固定统计与限额的应用逻辑、管理 API 和前端页面已按独立移除计划删除；旧表由生产管理员审核并手工执行 `backend/sqlArchiving/166_drop_legacy_fixed_token_statistics.sql` 删除，旧 Redis Key 按 `docs/legacy-fixed-token-redis-retirement.md` 审计并自然过期。

## 自动化验证

| 范围 | 命令 | 结果 |
|---|---|---|
| 后端全量 | `cd backend && go test ./...` | 通过 |
| 核心性能 | `cd backend && go test ./internal/service/tokenstat -run '^$' -bench DynamicTokenStatisticsQueueFullFailOpen -benchtime=200ms` | `13,915,462` 次，`19.25 ns/op`（Windows amd64，i5-1340P）；队列满路径无阻塞 I/O |
| 新前端功能 | `cd frontend && pnpm vitest run src/api/admin/__tests__/dynamicTokenStatistics.spec.ts src/views/admin/__tests__/TokenStatisticsView.spec.ts src/rbac/permissionMatrix.spec.ts` | 3 文件、9 项通过 |
| 前端类型 | `cd frontend && pnpm typecheck` | 通过 |
| 前端构建 | `cd frontend && pnpm build` | 通过；仅既有 chunk 警告 |
| 前端全量 | `cd frontend && pnpm test:run` | 已执行；新增用例通过，6 项既有 OAuth/EmailVerify 跳转断言失败，与本功能无关 |

## 故障演练映射

自动化测试以可控替身完成非破坏演练：

- `async_pipeline_test.go`：队列满立即丢弃、Redis writer 失败重试后丢弃，模型主路径不等待。
- `redis_accumulator_test.go`：Lua 原子累计、D/W/M、并发、TTL 与脏集合。
- `sync_engine_test.go`：同步失败重新归还脏集合，版本快照幂等。
- `period_finalizer_test.go`：pending 阻止删除、最终同步失败保留 key、校验成功后 UNLINK。
- `quota_test.go`：OBSERVE、ENFORCE、两阶段维度匹配和 Redis 错误 fail-open。
- `projection_admin_test.go`：自动投影、PENDING→ENABLED 和限额引用保护。
- `query_test.go`：投影白名单、范围/分页/排序注入保护。

生产环境未执行破坏性 Redis/MySQL 断连；操作手册提供预发布演练步骤。

## 隔离审计

可重复命令：

```text
rg -n "TokenStatisticsType|model_token_daily_|user_model_token_daily_|group_candidate_token_daily_|/admin/token-usage/|dailyTokenQuotaRepo" \
  backend/internal/service/tokenstat \
  backend/internal/repository/tokenstat \
  backend/internal/service/dynamic_token_quota.go \
  backend/internal/service/dynamic_token_usage.go \
  backend/internal/handler/admin/dynamic_token_statistics_handler.go \
  frontend/src/api/admin/dynamicTokenStatistics.ts \
  frontend/src/views/admin/TokenStatisticsView.vue
```

结果：无匹配。共享网关仅在两个独立的 `dynamic_token_*.go` 入口接入新体系；未让新体系调用旧统计或旧限额 repository。

独立资源核对：

- MySQL：仅 `token_stat_projections`、`token_stat_projection_metrics`、`token_stat_quota_rules`、`token_stat_aggregates`、`token_stat_period_states`。
- Redis：仅 `sub2api:dynamic_token_stats*`。
- API：仅 `/admin/token-statistics/*`。
- 前端：统一使用新的 `/admin/token-statistics` 页面；旧固定统计与限额页面及 API 已删除。

## 已知边界

- 统计与限额采用异步后结算，允许少量并发超额。
- Redis/规则异常时限额 fail-open。
- 新投影不回填启用前数据。
- 通用历史查询只读 MySQL，因此存在明确展示的最终一致延迟。
- SQL 只交付归档文件，由生产管理员手动执行，本次未实际运行生产 DDL。
