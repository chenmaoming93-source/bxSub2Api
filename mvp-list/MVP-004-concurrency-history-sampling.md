# MVP-004: 增加并发与等待队列历史采样

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: Redis 读取接口已存在，主要是内存窗口聚合、分钟表、单实例采样锁、分批读取和数据保留。
- Dependencies: `MVP-001`

## Outcome

看板能显示账号/候选真实的并发历史峰值和水位趋势，缺采样时明确不可用，不再使用时段系数模拟。

## Context

`ConcurrencyService.GetAccountsLoadBatch`、`GetRouteLoadsBatch` 和等待计数只提供当前快照；`ops_system_metrics` 只有全局聚合，不能支撑账号级时段图。

## In Scope

- 新增 `observability_concurrency_minute` 表，保存分钟桶、account、group、route_alias、current_peak/current_avg、limit、waiting_peak。
- 每 15-30 秒批量读取 Redis，在内存中累计一个分钟窗口，每分钟仅为活跃账号和候选 upsert 一行；多实例通过分布式锁只由一个实例写库。
- 采样失败不阻塞 gateway，并记录 heartbeat/指标。
- 提供当前与历史批量读取仓储接口，支持账号、候选、分组过滤。
- 分钟数据保留 24 小时，再滚动为 5 分钟/小时数据保留 7 天；禁止将每次 15-30 秒采样直接写入数据库并保留 7 天。

## Out of Scope

- 修改 Redis 槽位算法或调度策略。
- 以采样值替代实时槽位判定。
- 用户级并发历史。

## Implementation Notes

- 只采集 `schedulable` 或近期有流量的账号，避免无界扫描。
- 当前值从 `GetAccountsLoadBatch`、`GetRouteLoadsBatch` 读取；有效上限优先取分时 schedule。
- `group_id + route_alias + account_id` 是候选采样键；账号级水位仍保留独立行，以便识别共享压力。
- 一轮 Redis 读取使用固定采样时间；落库使用统一 `minute_start`，避免桶内顺序漂移。

## Acceptance Criteria

- [x] 15-30 秒 Redis 读取不会直接产生数据库行；每个对象每分钟最多一行。
- [x] 多实例并发时只有一个 leader 写入，其他实例不报错。
- [x] 账号级和候选级当前值可批量查询，缺失账号返回 0 当前值并保留有效上限。
- [x] 历史峰值查询只读分钟聚合表，不访问 Redis。
- [x] 无采样数据的窗口明确返回 `unavailable`，不生成 0 值伪趋势。
- [x] 采样任务停止时 heartbeat 和告警能反映。

## Verification Plan

```powershell
cd backend
go test -tags=unit ./internal/service ./internal/repository
go test -tags=integration ./internal/repository
go build ./cmd/server
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Sampling cadence | `internal/service/observability/concurrency_accumulator_test.go` | PASS: 20s samples remain in memory; a row set is emitted only when crossing a minute boundary |
| Minute cardinality | Same unit test and local repository verifier | PASS: repeated flush of the same minute remains one row per account/candidate |
| Leader election | `ConcurrencyHistoryService.sampleOnce` + shared `tryAcquireSingletonLeaderLock` | PASS: only the lock owner reads/writes the minute bucket; peers return without error |
| Current batch | Temporary verifier against local Redis | PASS: missing account `current=0 limit=12`; missing route `current=0 limit=8` |
| Historical query | `QueryConcurrencyHistory` | PASS: reads only `observability_concurrency_bucket`; account/group/route filters supported |
| Unavailable/stale quality | `internal/service/observability/concurrency_quality.go` | PASS: no sample returns `unavailable`; stopped sampler becomes `stale` after 90s |
| SQL archive | `backend/sqlArchiving/178_create_observability_concurrency_rollup.sql` | PASS: executed twice; 2 tables and 4 non-primary indexes created idempotently |
| Local rollup verification | Temporary MariaDB verifier | PASS: 3 objects produced 3 minute + 3 five-minute + 3 hourly rows; repeated minute write stayed at one row/object; route label `2026-11-11T10:01:00+08:00` |
| Retention | `CleanupConcurrencyHistory` | Minute 24h, five-minute/hour 7d; cleanup is rate-limited to once per hour |
| Worker integration | Server Wire provider and cleanup hook | PASS: 20s sampler starts with the server and stops during cleanup |
| Unit tests | `go test -tags=unit ./internal/service/observability ./internal/repository` | PASS |
| Integration compile | `go test -c -tags=integration ./internal/repository` | PASS |
| Backend build | `go build -p 1 ./cmd/server` | PASS |
| Known limitation | Docker integration execution | Docker daemon is unavailable; equivalent persistence and Redis paths were exercised against local MariaDB/Redis |

## Execution Notes

如果活跃账号列表很大，使用分批游标和每轮预算。按 1000 账号 + 1000 候选估算，逐 15 秒落库保留 7 天会达到约 8000 万行；分钟聚合可将写入量降低至少 15-30 倍。
