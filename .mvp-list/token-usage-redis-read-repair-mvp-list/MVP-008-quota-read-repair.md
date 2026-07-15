# MVP-008：让请求限额读取具备缺失读修复

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 只调整现有逐项限额快照链路的缺失语义并补回归测试，约 40 分钟。
- Dependencies: `MVP-002, MVP-003`

## 预期成果

请求限额判断继续 Redis-first；当 Redis field 缺失而 MySQL 快照有累计值时，返回 MySQL 值并并发安全地修复 Redis。

## 背景

现有 `backend/internal/repository/token_statistics_quota_repo.go` 在 `redis.Nil` 时直接返回 MySQL snapshot，但不会反向修复。

## 范围内

- 在三种 `Get*DailyTokenQuota` 缺失分支接入共享原子修复能力。
- 严格区分 `redis.Nil` 与连接失败。
- Redis 连接失败继续使用 MySQL 且不误触发修复。
- 保持现有限额错误类型和 fail-open/fail-closed 语义。
- 补模型、路由候选、用户模型和并发覆盖测试。

## 范围外

- 不修改限额阈值和路由选择算法。
- 不修改 Token 累加。

## 实现说明

- 如果 base snapshot 不代表已持久化 usage，不应创建无意义 field。
- 修复失败不得改变本次限额判断使用的 MySQL 值。

## 验收标准

- [x] 三种限额读取在 Redis 缺失时均返回 MySQL 累计值。
- [x] 符合条件的缺失项被原子修复。
- [x] Redis 已有实时值不被覆盖。
- [x] 现有限额仓储与 quota-aware routing 测试通过。

## 验证计划

- `cd backend && go test ./internal/repository ./internal/service -run 'Test.*TokenQuota.*Redis|Test.*QuotaAware'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_statistics_quota_repo.go` | 仅在 `redis.Nil` 且 MySQL snapshot 表示正数持久化 usage 时触发共享原子修复；连接故障只回退 MySQL。 |
| 接线 | `backend/internal/repository/wire.go`、`backend/cmd/server/wire_gen.go` | 限额仓储注入既有 retention/batch 配置的 repairer。 |
| 测试 | `backend/internal/repository/token_statistics_quota_repo_test.go` | 覆盖三维度 miss 修复、已有实时值保留、连接失败回退不修复。 |
| 聚焦验证 | `cd backend && go test ./internal/repository ./internal/service -run 'Test.*TokenQuota.*Redis|Test.*QuotaAware'` | 通过。 |
| 包回归 | `cd backend && go test ./internal/repository ./internal/service ./cmd/server` | 通过。 |

## 执行记录

2026-07-15 完成。修复失败被刻意忽略，不改变本次 MySQL snapshot；零 usage 不创建无意义 Redis field。限额阈值、累加和 quota-aware routing 语义未修改。
