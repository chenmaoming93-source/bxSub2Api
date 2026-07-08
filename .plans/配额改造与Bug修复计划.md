# 配额数据分离 + Bug 修复改造计划

## 背景

当前三层日 Token 配额（全局模型 / 用户模型 / 分组候选）存在两个结构性问题：

1. **配置与用量混在同一个表中**：`daily_limit_tokens` 字段嵌入在日用量记录里，导致跨日时限额丢失——第二天的记录需要被"触发创建"才会继承前一天的限额值。
2. **分组候选的两个 bug**：
   - Token 计费记录到了路由别名（`test`）而非实际上游模型（`deepseek-v4-flash`）
   - 上游调用异常时没有 failover 到下一个候选

## 执行顺序总览

| 序号 | 任务 | 涉及文件 | 状态 |
|---|---|---|---|
| 1 | 建 3 张 config 表 + 删 3 张 usage 表的 `daily_limit_tokens` 列 | `migrations/158_*.sql` | ✅ |
| 2 | 写 3 个新 ent schema + 修改 3 个旧 schema | `ent/schema/*.go` | ⏳ |
| 3 | ent 代码生成 | `ent/*.go` | ⏳ |
| 4 | 重写 `daily_token_quota_repo.go` | `internal/repository/` | ⏳ |
| 5 | 改 `daily_token_quota_port.go` | `internal/service/` | ⏳ |
| 6 | 改 `daily_token_quota_accounting.go` | `internal/service/` | ⏳ |
| 7 | 改 admin service | `internal/service/` | ⏳ |
| 8 | 改缓存层 | `internal/repository/daily_token_quota_cache.go` | ⏳ |
| 9 | 修复：配额记录到路由别名而非实际上游模型 | `internal/handler/openai_chat_completions.go` | ⏳ |
| 10 | 修复：上游异常时未 failover 到下一候选 | `internal/service/gateway_service.go` | ⏳ |
| 11 | 编译验证 + 跑测试 | — | ⏳ |

---

## 一、数据库层

### 新增 3 张 config 表

| 表 | 唯一索引 | 外键 |
|---|---|---|
| `model_token_daily_limit_configs` | `UNIQUE (model)` | — |
| `user_model_token_daily_limit_configs` | `UNIQUE (user_id, model)` | `user_id → users(id) ON DELETE CASCADE` |
| `group_candidate_token_daily_limit_configs` | `UNIQUE (group_id, route_alias, upstream_model)` | `group_id → groups(id) ON DELETE CASCADE` |

每张表字段：`id`, `daily_limit_tokens BIGINT DEFAULT NULL`, `created_at`, `updated_at`。

### 修改 3 张 usage 表

| 表 | 操作 |
|---|---|
| `model_token_daily_usages` | `DROP COLUMN daily_limit_tokens` |
| `user_model_token_daily_usages` | `DROP COLUMN daily_limit_tokens` |
| `group_candidate_token_daily_usages` | `DROP COLUMN daily_limit_tokens` |

迁移文件：`migrations/158_token_quota_config_and_usage_split.sql`

---

## 二、Ent Schema 层

### 新增

- `ent/schema/model_token_daily_limit_config.go`
- `ent/schema/user_model_token_daily_limit_config.go`
- `ent/schema/group_candidate_token_daily_limit_config.go`

### 修改

- `ent/schema/model_token_daily_usage.go` — 删除 `daily_limit_tokens` 字段
- `ent/schema/user_model_token_daily_usage.go` — 删除 `daily_limit_tokens` 字段
- `ent/schema/group_candidate_token_daily_usage.go` — 删除 `daily_limit_tokens` 字段

### 生成

```bash
cd backend
go generate ./ent
```

---

## 三、Repository 层

`internal/repository/daily_token_quota_repo.go` 完全重写：

| 操作 | 新逻辑 |
|---|---|
| `GetModelDailyTokenQuota` | 查 `model_token_daily_limit_configs` 拿 limit + 查 `model_token_daily_usages` 拿 used_tokens |
| `GetUserModelDailyTokenQuota` | 查 `user_model_token_daily_limit_configs` 拿 limit + 查 `user_model_token_daily_usages` 拿 used_tokens |
| `GetGroupCandidateDailyTokenQuota` | 查 `group_candidate_token_daily_limit_configs` 拿 limit + 查 `group_candidate_token_daily_usages` 拿 used_tokens |
| `IncrementDailyTokenQuotas` | 只写三张 usage 表的 `used_tokens`，ON CONFLICT 累加，删除 `latestXxxLimit` 辅助函数 |
| `ListModelDailyTokenQuotas` / `SetModelDailyTokenQuota` | 改为操作 `model_token_daily_limit_configs` |
| `ListUserModelDailyTokenQuotas` / `UpsertUserModelDailyTokenQuotas` | 改为操作 `user_model_token_daily_limit_configs` |

---

## 四、Service 层

### `daily_token_quota_port.go`

- `DailyTokenQuotaSnapshot` 结构不变（`Exists`、`UsedTokens`、`DailyLimitTokens` 等字段）
- 输入输出接口签名不变
- 删除 `latestModelLimit` / `latestUserModelLimit`

### `daily_token_quota_accounting.go`

- `IncrementDailyTokenQuotas` 调用时不再传 `ModelDailyLimitTokens`、`UserModelDailyLimitTokens`
- `GroupCandidateDailyLimitTokens` 从路由规则 JSON 取值写入 usage 表（仅用于展示）
- `IncrementDailyTokenQuotas` 内部不再调 `latestXxxLimit`

### `model_token_quota_admin_service.go`

- `ListModelDailyTokenQuotas` → 改为读 config 表 + usage 表（拼接 used_tokens）
- `SetModelDailyTokenQuota` → 改为写 config 表

### `user_model_token_quota_admin_service.go`

- 同上，改为操作 config 表
- `UpsertUserModelDailyTokenQuotas` → 写 config 表 + 只读当天 usage 的 used_tokens

---

## 五、缓存层

`internal/repository/daily_token_quota_cache.go`：

- 缓存 key 不变
- 缓存 miss 回源时，从 config 表 + usage 表重新组装 `DailyTokenQuotaSnapshot`

---

## 六、Bug 修复

### Bug 1：配额被记录到路由别名而非实际上游模型

**现象**：候选 1 `deepseek-v4-pro` 限额耗尽 → failover 到候选 2 `deepseek-v4-flash` → token 累加记到了 `test`（路由别名）上，而非 `deepseek-v4-flash`。

**排查路径**：

1. `SelectAccountWithLoadAwareness` 中候选选择逻辑（`gateway_service.go:1808-1839`）正确地使用 `candidateModel` 作为 `routingModelForSelection` 传给 `trySelectRouteCandidateAccounts`
2. `withSelectionModelIdentity`（`gateway_service.go:547`）正确地设置 `result.RequestedModel = "test"`、`result.UpstreamModel = "deepseek-v4-flash"`
3. `openai_chat_completions.go:330` 将 `result.UpstreamModel` 传给 `RecordUsage`
4. **关键**：需要追踪 `RecordUsage` 内部如何将 `result.UpstreamModel` 写入 `UsageLog.UpstreamModel`，以及 `incrementDailyTokenQuotasForUsage`（`daily_token_quota_accounting.go`）中 `usageLog.UpstreamModel` 是否被正确取值

**待办**：完整追踪 RecordUsage → UsageLog 的数据流，确认 `UpstreamModel` 是否正确到达 `incrementDailyTokenQuotasForUsage`。

### Bug 2：上游异常时未 failover 到下一候选

**现象**：候选 1 的账号调用上游返回错误，直接返回报错，未尝试候选 2。

**排查路径**：

1. `openai_chat_completions.go:419-452`：failover 循环存在——当 `err` 类型为 `*UpstreamFailoverError` 时，将账号加入 `failedAccountIDs` 并重试
2. `trySelectRouteCandidateAccounts`（`gateway_service.go:2333`）接收 `isExcluded` 回调，排除 `failedAccountIDs` 中的账号
3. 当候选 1 的所有账号被排除后，`trySelectRouteCandidateAccounts` 返回 `(nil, false, nil)`
4. 外层循环（`SelectAccountWithLoadAwareness:1808`）应继续到候选 2

**待办**：确认 `SelectAccountWithSchedulerForCapability` → `SelectAccountWithLoadAwareness` 的调用链中，failover 循环的外层是否正确重试到第 2+ 个候选。如果 `isExcluded` 过滤导致候选 1 账号全被排除但循环未进入候选 2，需定位外层循环的 continue 逻辑是否正确。

**疑点**：`routingAccountIDs`（`gateway_service.go:1785`）在 `routeCandidates` 被迭代后还保有第一批候选的账号 ID，可能在候选间 failover 时错误使用了旧的 `routingAccountIDs`。

---

## 七、不变的部分

- 前端所有页面和 API 调用
- Admin API 请求/响应 JSON 结构
- 路由选择的大框架
- `checkDailyTokenQuota` 判别逻辑核心
