# MVP-001：建立 Redis 重置快照与限额差值读取

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: `范围集中于 Redis Key Builder、两个小型 Lua 操作和限额读取兼容改造，可在一个开发日内独立实现并用 miniredis/单元测试验证。`
- Dependencies: `none`

## 预期成果

系统能够在不修改既有统计 Hash、版本 Key、dirty set 和 MySQL 数据的情况下，为单个具体统计身份创建 Redis 重置快照，并让限额检查使用 `max(0, raw_usage-reset_baseline)`。

## 背景

- 现有累计器位于 `backend/internal/repository/tokenstat/redis_accumulator.go`，统计前缀为 `sub2api:dynamic_token_stats:v1:`。
- 现有限额读取位于 `backend/internal/service/tokenstat/quota.go`、`backend/internal/service/tokenstat/quota_admin.go` 和 `backend/internal/repository/tokenstat/quota_reader.go`。
- 快照必须使用独立前缀 `sub2api:dynamic_token_quota_reset:v1:`，但周期、projection、shard 和 field 定位必须复用现有 Builder 语义。
- 快照只存 Redis；Redis 数据丢失后不从 MySQL 恢复。

## 范围内

- 增加重置快照 Key Builder，并复用 `NaturalPeriods`、`RedisPeriodStart`、`RedisShard` 和 `RedisField`。
- 实现单条原子快照操作：读取原统计 field，存在时写入同身份快照 field 并设置 `period.End + orphanTTL`。
- 实现 reset-aware 限额计数读取：快照缺失视为 0，结果小于 0 时钳制为 0。
- 将现有 `QuotaChecker` 接入有效用量读取，同时保持读取异常时 `fail-open`。
- 增加 Redis 单元/集成测试及限额判定回归测试。

## 范围外

- 不实现部分维度候选发现。
- 不实现批量重置服务、HTTP API 或页面。
- 不修改 `token_stat_aggregates` 或新增数据库结构。
- 不实现多个条目的整体原子性。

## 实现说明

- 优先在 `backend/internal/repository/tokenstat` 中放置 Redis 快照读写能力，避免 service 层自行拼接 Redis 字符串。
- 单条写入可使用 `HGET + HSET + EXPIREAT` Lua，保证同一条统计累加与快照建立之间存在明确顺序。
- 限额读取可使用短 Lua 同时取得 raw 与 baseline；接口应继续向 `QuotaChecker` 暴露一个有效计数，而不是让 Checker 了解快照持久化细节。
- 快照写入不得触碰 `sub2api:dynamic_token_stats_ver:v1:*` 和 `sub2api:dynamic_token_stats_dirty:v1:current`。

## 验收标准

- [x] 快照 Key 前缀独立，suffix 与对应统计 Key 的周期、projection 和 shard 完全一致，field 完全复用现有格式。
- [x] 原统计值 100000、快照缺失时，限额读取返回 100000。
- [x] 原统计值 120000、快照值 100000 时，限额读取返回 20000。
- [x] 快照值大于原统计值时返回 0，不产生负数。
- [x] 多次执行单条重置后，快照覆盖为最新 raw 值。
- [x] 原统计 field 不存在时不创建无意义快照，并返回可识别的 `NO_USAGE` 类结果。
- [x] 快照 TTL 等于对应周期结束时间加现有 orphan TTL 策略。
- [x] 重置不改变原统计值、version field 或 dirty set。
- [x] Redis 读取失败时，限额检查继续符合现有 `fail-open` 行为。

## 验证计划

- `cd backend && go test ./internal/repository/tokenstat ./internal/service/tokenstat`
- 使用 miniredis 测试 D/W/M Key、field、TTL、多次覆盖以及 raw/baseline 差值。
- 在测试中读取原统计 Hash、version Hash 和 dirty set，证明快照操作未修改既有数据。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/tokenstat/quota_reset.go`、`quota_reader.go` | 新增独立快照 namespace、单条原子 Snapshot 与 effective usage 读取。 |
| 接入 | `backend/cmd/server/wire.go`、`wire_gen.go` | 运行时 `QuotaChecker` 已改用 reset-aware repository reader。 |
| 测试 | `cd backend && go test ./internal/repository/tokenstat ./internal/service/tokenstat` | 通过。 |
| 编译 | `cd backend && go test ./cmd/server` | 通过。 |
| 格式 | `gofmt -d ...`、`git diff --check -- ...` | 无格式或空白错误。 |

## 执行记录

2026-09-09 完成。新增测试覆盖 D/W/M Key、TTL、多次覆盖、NO_USAGE、差值钳制及 Redis 错误；同时用 `mini.SetTime` 修复两个既有固定日期测试在当前日期下因 EXPIREAT 立即过期的问题。保留原 `QuotaCounterReader` 接口，测试桩无需变更。
