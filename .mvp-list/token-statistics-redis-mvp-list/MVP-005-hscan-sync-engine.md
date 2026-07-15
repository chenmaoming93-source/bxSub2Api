# MVP-005：实现 HSCAN 到 MySQL 的同步引擎

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 聚焦一个 Hash 的分页、解码、分批写入与有限重试，可形成独立同步核心。
- Dependencies: `MVP-002, MVP-004`

## 预期成果

同步引擎能够用 `HSCAN` 完整遍历任一统计 Hash，将数据按配置批量覆盖到对应 MySQL 表。

## 背景

用户模型 Hash 可达数万 Field，禁止以 `HGETALL` 作为周期同步实现。

## 范围内

- 通用统计类型分派。
- HSCAN 游标循环与 COUNT 配置。
- Field 解码、非法 Field 跳过。
- MySQL 分批与有限重试。
- 重复 Field 安全测试。

## 范围外

- 定时器、分布式锁和服务 wiring。

## 实现说明

- 游标归零才算本轮结束。
- MySQL 失败不得隐藏底层错误码和 cause。

## 验收标准

- [x] 三类 Hash 均能复用同一同步核心。
- [x] 数万 Field 通过分页处理而非 `HGETALL`。
- [x] 批量大小与重试次数受配置控制。
- [x] 扫描、解码、写入失败可区分。

## 验证计划

- `cd backend; go test ./internal/service ./internal/repository -run "TokenStatistics.*(HScan|SyncEngine|Retry)"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_statistics_sync.go` | 使用 `HSCAN` 游标循环复用三类 Hash，同步时按配置分批并有限重试。 |
| 测试 | `cd backend; go test ./internal/service ./internal/repository -run "TokenStatistics.*(HScan|SyncEngine|Retry)" -count=1` | 通过：service `5.182s`，repository `5.194s`。 |
| 数据量 | `TestTokenStatisticsHScanSyncEngine` | 每类 10000 Field、`COUNT=97`、批量 113；三类均完整写入，非法 Field 安全跳过。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：实现游标归零终止、三类解码分派、数值校验、配置化批次和 `retries+1` 次有限尝试；错误按 `redis_hscan`、`parse_value`、`mysql_batch_upsert` 分阶段包装并保留 cause。
