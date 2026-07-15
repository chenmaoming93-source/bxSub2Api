# MVP-012：落实具体异常日志与故障测试

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 对核心失败阶段统一事件名、字段和错误链，并用故障 stub 验证日志契约。
- Dependencies: `MVP-003, MVP-005, MVP-006, MVP-007`

## 预期成果

Redis 累加、Field 解码、HSCAN、同步锁、MySQL 批量写入和配额回退异常均输出明确、结构化、可搜索的日志，不被通用错误吞掉。

## 背景

Plan 已定义 `token_statistics.*` 事件名和必填上下文；监控不得写入新数据库表。

## 范围内

- 固化所有 Plan 事件名。
- 保留底层 error chain 和 MySQL 错误码。
- 字段白名单、截断与敏感信息保护。
- 成功汇总日志，不逐 Field 打印成功。
- Redis/MySQL/解码/锁/回退故障测试。

## 范围外

- 引入新的监控系统或数据库指标表。

## 实现说明

- 同一错误避免多层重复完整堆栈。
- 锁被占用与获取锁失败必须区分级别和事件。

## 验收标准

- [x] 每类异常事件名与 Plan 一致。
- [x] 日志包含 stage、统计类型、日期、Key/表、重试及 cause。
- [x] 不只出现 `internal error` 等通用信息。
- [x] 日志不泄露 API Key 明文和请求正文。

## 验证计划

- `cd backend; go test ./internal/service ./internal/repository -run "TokenStatistics.*(Log|Error|Fault|Fallback)"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 事件契约 | Redis 累加/HSCAN/Field 解码/MySQL upsert/锁/配额回退 | 使用 `token_statistics.*` 固定事件名，并携带 stage、type、Key、游标/重试和底层 cause。 |
| 敏感信息 | `token_statistics_sync.go`、`token_statistics_quota_repo.go` | Field 最多记录 128 字符；不记录 API Key 或请求正文。 |
| 测试 | `cd backend; go test ./internal/service ./internal/repository -run "TokenStatistics.*(Log|Error|Fault|Fallback)" -count=1` | 通过：service `5.411s`，repository `9.526s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：故障测试验证 Redis 断连事件、Pipeline stage/Keys/cause、MySQL 1205 cause 及 max_retries；非法 Field 输出有界结构化 warning，Redis 配额读取失败输出回退 warning。
