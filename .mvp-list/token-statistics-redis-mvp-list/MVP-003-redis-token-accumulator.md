# MVP-003：实现三 Hash 原子累计入口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 一个 Pipeline 写入接口、TTL 设置和 miniredis 测试构成单一可验证成果。
- Dependencies: `MVP-002`

## 预期成果

一次用量记录可通过单个 Redis Pipeline 对三个每日 Hash 执行 `HINCRBY`，并发累加结果正确。

## 背景

当前逐请求入口为 `incrementDailyTokenQuotasForUsage`，本 MVP 先提供可替换的 Redis 累计能力，不切换业务路径。

## 范围内

- 定义 service port 和 Redis Repository 实现。
- 跳过 `total_tokens <= 0`。
- Pipeline 三个 `HINCRBY` 与固定 `EXPIREAT`。
- Redis 失败返回保留底层 cause 的上下文错误。
- 并发与 TTL 集成测试。

## 范围外

- RecordUsage 调用切换和 MySQL 同步。

## 实现说明

- 三个 Hash 的 value 均为当天绝对累计值。
- 重复设置过期时间不得延长既定保留边界。

## 验收标准

- [x] 单次调用只执行一个 Pipeline。
- [x] 三个维度累计值正确。
- [x] 多 goroutine 更新同一 Field 不丢增量。
- [x] Redis 错误包含 stage、Key 和底层原因。

## 验证计划

- `cd backend; go test ./internal/repository ./internal/service -run "TokenStatistics.*(Accumulate|Pipeline|Concurrent)"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Port | `backend/internal/service/token_statistics_port.go` | 定义公共累计输入及 `TokenStatisticsAccumulator`。 |
| 实现 | `backend/internal/repository/token_statistics_accumulator.go` | 单 Pipeline 执行三次 `HINCRBY` 与固定 `EXPIREAT`，非正 Token 直接跳过。 |
| 测试 | `cd backend; go test ./internal/repository ./internal/service -run "TokenStatistics.*(Accumulate|Pipeline|Concurrent)" -count=1` | 通过：repository `7.099s`，service `5.133s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：使用 miniredis 验证三个维度值、64 goroutine 同 Field 原子累计、绝对 TTL、单应用 Pipeline、零 Token 跳过及连接失败错误链。
- 初次测试暴露 go-redis 内部握手 Pipeline 也会经过 Hook，计数器已限定为包含 `HINCRBY` 的业务 Pipeline；随后修正动态 TTL 断言并通过完整聚焦测试。
