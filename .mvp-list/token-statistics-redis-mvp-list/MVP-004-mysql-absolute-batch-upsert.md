# MVP-004：实现三表绝对值批量 Upsert

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 三种记录共享批量覆盖模式，可在同一 Repository 增量中完成并用聚焦集成测试验证。
- Dependencies: `none`

## 预期成果

DailyTokenQuota Repository 支持将三类 Redis 绝对累计值分批写入对应 MySQL 表，且不会重复累加或修改限额。

## 背景

现有逐请求事务位于 `backend/internal/repository/daily_token_quota_repo.go`，接口位于 `backend/internal/service/daily_token_quota_port.go`。

## 范围内

- 扩展 Repository port 的三类批量覆盖能力。
- 使用 `GREATEST(used_tokens, incoming)`。
- 更新 `updated_at`，保持 `daily_limit_tokens` 不变。
- 三表唯一键冲突与旧值覆盖测试。

## 范围外

- HSCAN、调度和业务路径切换。

## 实现说明

- 原 `IncrementDailyTokenQuotas` 保留。
- 每类写入使用独立小事务，不引入新表。

## 验收标准

- [x] 三张表均支持批量绝对值写入。
- [x] 重复提交同一值不增加 Token。
- [x] 较小值不能覆盖较大值。
- [x] `daily_limit_tokens` 保持不变。

## 验证计划

- `cd backend; go test ./internal/repository -run "DailyTokenQuota.*Batch|AbsoluteUpsert"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Port | `backend/internal/service/daily_token_quota_port.go` | 新增三类绝对累计记录与独立 Repository port。 |
| 实现 | `backend/internal/repository/daily_token_quota_repo.go` | 三类记录分别在小事务中 upsert；MySQL 使用 `GREATEST`，SQLite 测试方言使用等价 `MAX`。 |
| 测试 | `cd backend; go test ./internal/repository -run "DailyTokenQuota.*Batch|AbsoluteUpsert" -count=1` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 5.467s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：验证三类唯一键首次写入、同值重放、较小值重放以及独立限额配置不变；原 `IncrementDailyTokenQuotas` 未删除或改写。
