# MVP-007：Token 配额读取优先使用 Redis

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 三维度当前值读取、MySQL 回退和配额回归测试可独立交付。
- Dependencies: `MVP-003`

## 预期成果

请求前 Token 配额检查读取 Redis 当日累计值；Field 不存在或 Redis 失败时按规则回退 MySQL，保留现有软限制语义。

## 背景

相关逻辑集中在 `billing_cache_service.go`、daily token quota cache/repository 和 quota-aware routing。

## 范围内

- 三类 Redis 当前累计值读取。
- Field 不存在回退 MySQL。
- Redis 异常回退 MySQL。
- 双重失败返回保留 cause 的错误。
- 配额和路由回归测试。

## 范围外

- 金额资格检查停用和请求后累计。

## 实现说明

- 不改变 `daily_limit_tokens` 来源。
- Redis 失败但 MySQL 回退成功与双重失败必须可区分。

## 验收标准

- [x] Redis 有值时优先使用 Redis。
- [x] Redis 无值或失败时正确回退。
- [x] Token 超限行为保持兼容。
- [x] 错误上下文包含统计类型、日期和回退状态。

## 验证计划

- `cd backend; go test ./internal/service ./internal/repository -run "TokenQuota.*Redis|QuotaAwareRouting"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_statistics_quota_repo.go` | 保留 MySQL 限额来源，以 Redis 三 Hash 当日值覆盖 `used_tokens`；缺失或故障时保留 MySQL 快照。 |
| 测试 | `cd backend; go test ./internal/service ./internal/repository -run "TokenQuota.*Redis|QuotaAwareRouting" -count=1` | 通过：service `5.176s`，repository `6.451s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：验证 Redis 值优先、Field 缺失回退、Redis 断连回退、限额保持 MySQL 来源、达到限额仍返回原分类错误，以及 MySQL 读取失败保留具体 cause。
