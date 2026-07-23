# MVP-003：实现并发安全的 Redis 缺失读修复

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 只实现绝对值、仅缺失时写入和 TTL 保持，并验证并发竞态，属于边界清晰的约 40 分钟任务。
- Dependencies: `MVP-001, MVP-002`

## 预期成果

MySQL-only 当天累计记录可以批量、幂等地修复到 Redis，且不会覆盖查询期间并发产生的实时 Redis 值。

## 背景

批准 Plan 要求 Redis 缺失且 MySQL 存在时反向修复，但无条件 `HSET` 会用旧 MySQL 快照覆盖并发 `HINCRBY` 结果。

## 范围内

- 实现 Lua 或等价原子“field 不存在才写入”的操作。
- 使用绝对累计值写入，禁止 `HINCRBY`。
- 为 Redis key 设置或保持 `TokenStatisticsExpireAt` 对应 TTL。
- 支持受控批次或 pipeline，返回成功、并发跳过、失败计数。
- 覆盖首次修复、重复修复、并发创建、TTL 和部分失败测试。

## 范围外

- 不改变 Redis → MySQL 同步器。
- 不实现报表合并。
- 不新增配置。

## 实现说明

- Redis 整体读取失败时调用方不应触发修复。
- 已存在 field 的值无论大小都必须保留。

## 验收标准

- [x] 缺失 field 被写入 MySQL 绝对累计值。
- [x] 已存在或并发出现的 field 不被覆盖。
- [x] 重复执行结果幂等且不累加。
- [x] 修复后的 key 具有现有保留策略要求的 TTL。

## 验证计划

- `cd backend && go test ./internal/repository -run 'Test.*TokenUsage.*Repair|Test.*ReadRepair'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_usage_read_repair.go` | Lua 原子执行 `HEXISTS`/`HSET`/`EXPIREAT`，绝对值写入，支持三维度、受控 pipeline 批次和 repaired/skipped/failed 计数。 |
| 测试 | `backend/internal/repository/token_usage_read_repair_test.go` | 覆盖首次修复、已有/并发值保留、重复幂等、TTL、三维度及批次内部分校验失败。 |
| 聚焦验证 | `cd backend && go test ./internal/repository -run 'Test.*TokenUsage.*Repair|Test.*ReadRepair'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 5.399s`。 |
| 包回归 | `cd backend && go test ./internal/repository` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 13.431s`。 |

## 执行记录

2026-07-15 完成。修复复用现有 `redis_retention_days`，未新增配置。首次聚焦测试因 pipeline 中 `EVALSHA` 未预加载脚本而失败，改为 pipeline 直接 `EVAL` 后通过；该调整仍保持每个 field 的判断、写入和 TTL 更新在单个 Lua 操作内原子完成。
