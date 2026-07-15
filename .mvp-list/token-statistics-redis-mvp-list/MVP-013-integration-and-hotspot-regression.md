# MVP-013：完成集成、数量级与热点回归

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 以现有测试工具完成端到端断言、数万 Field 扫描和并发热点回归，形成最终交付证据。
- Dependencies: `MVP-011, MVP-012`

## 预期成果

完整链路在正常条件下实现“请求明细保留、金额不变、Redis 三 Hash 累计、MySQL 最终覆盖”，并证明热点写入已从请求路径移除。

## 背景

这是列表最终集成关卡，验证源 Plan 的 AC-01 至 AC-08，不增加新功能。

## 范围内

- 端到端成功请求断言。
- 金额字段/表不变化断言。
- 每日三个 Hash 与 TTL 断言。
- 数万用户模型 Field 的 HSCAN 测试。
- 并发相同维度累计测试。
- 旧逐请求事务未调用的 spy/日志证据。
- 运行相关 package 回归测试并记录证据。

## 范围外

- 生产部署、迁移、恢复和对账。

## 实现说明

- 若无法在本地启动真实 MySQL/Redis，使用项目现有 integration harness；环境缺失必须如实记录，不得伪造通过。
- 热点验证至少证明请求路径不再触发 `IncrementDailyTokenQuotas` 三表事务。

## 验收标准

- [x] `usage_logs` 行为保持，金额数据不变化。
- [x] Redis 三 Hash 和 MySQL 三表数据符合预期。
- [x] 数万 Field HSCAN 完成且无 `HGETALL` 依赖。
- [x] 并发累计不丢失，旧热点事务不在请求路径执行。
- [x] 相关 Go 测试通过并记录证据。

## 验证计划

- `cd backend; go test ./internal/config ./internal/repository ./internal/service ./internal/handler ./cmd/server`
- 人工或测试 spy 验证压测路径不调用 `IncrementDailyTokenQuotas`。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 全量回归 | `cd backend; go test ./internal/config ./internal/repository ./internal/service ./internal/handler ./cmd/server -count=1` | 通过：config `3.000s`、repository `12.034s`、service `51.458s`、handler `25.668s`、server `6.642s`。 |
| 数量级/热点 | `cd backend; go test ./internal/repository -run "TokenStatisticsHScanSyncEngine|TokenStatisticsAccumulateConcurrent|DailyTokenQuotaAbsoluteUpsertBatch" -count=1` | 通过：`5.385s`；每类 10000 Field HSCAN、64 goroutine 原子累计及绝对值幂等覆盖。 |
| 请求路径 spy | `TestOpenAIRecordUsageTokenStatisticsSwitch`、`TestRecordUsageTokenQuotaAccountingCountsAllClaudeTokensOnce` | usage log 保留，金额 Repository 与旧 `IncrementDailyTokenQuotas` 均为 0 次，新 accumulator 为 1 次。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：首次五包全量回归发现两个旧 MySQL Token 记账测试仍断言旧语义，已改为 accumulator spy；第二次发现新增配置校验遮蔽既有 external API key 错误，已调整校验顺序。第三次完整运行五包全部通过。
- 端到端测试使用项目现有 miniredis 与 Ent SQLite harness；未启动外部真实 MySQL/Redis，所需 Redis Pipeline/HSCAN/TTL 与三表 upsert 行为均由对应集成 harness 验证。
