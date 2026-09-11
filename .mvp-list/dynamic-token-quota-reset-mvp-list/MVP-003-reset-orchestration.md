# MVP-003：交付可复用的分批限额重置服务

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发者日`
- Estimate rationale: `该切片将身份发现和 Redis 快照能力组合成完整业务服务，并覆盖分批、部分成功、日志与指标，工作量略高于一个开发日但边界完整。`
- Dependencies: `MVP-001, MVP-002`

## 预期成果

后端具备一个不依赖 HTTP 入口的完整限额重置服务：能够发现候选、分页/分批执行独立快照、持续处理单条失败，并返回 `RESET`、`PARTIAL_RESET`、`NO_QUOTA` 或 `NO_USAGE` 的可观察结果。

## 背景

- MVP-001 提供单条快照与 reset-aware 限额读取。
- MVP-002 提供最终一致的候选身份发现。
- 计划明确不要求多个额度整体原子，不因匹配数量多直接报错，也不回滚成功项。
- 运行指标可扩展 `backend/internal/service/tokenstat/observability.go`，但不得使用用户或模型等高基数标签。

## 范围内

- 实现统一 `ResetQuotaUsage` 业务入口和稳定的请求/响应结构。
- 将候选身份按固定批大小处理，并用 Redis pipeline 执行每条独立 Lua。
- 对成功、原统计缺失和失败分别累计 `reset_count`、`no_usage_count`、`failed_count`。
- 实现 `RESET`、`PARTIAL_RESET`、`NO_QUOTA`、`NO_USAGE` 状态判定。
- 单条失败后继续处理后续候选，不回滚成功快照。
- 增加固定标签观测计数和结构化日志；敏感维度不得明文输出。
- 增加服务级并发、部分失败、重复调用和同步兼容测试。

## 范围外

- 不实现管理员或 integrations HTTP Handler。
- 不实现前端页面。
- 不增加请求幂等或分布式事务。
- 不因总匹配数量设置 `TOO_MANY_MATCHES`。
- 不新增数据库状态或后台重置任务。

## 实现说明

- 分批大小应是内部安全参数或复用现有 Redis 批处理经验值；它限制每次 pipeline 大小，不限制总处理数量。
- 若请求 context 已取消或 Redis 整体不可用，应保留已完成计数并返回能够表达部分结果的错误/结果对象，供 Handler 映射。
- 若没有任何条目成功且核心依赖不可用，暴露 `TOKEN_QUOTA_RESET_UNAVAILABLE` 语义；已有成功项时优先表达 `PARTIAL_RESET`。
- 重置只影响快照 namespace，后续 Redis→MySQL 同步必须继续读取原 raw 和 version。

## 验收标准

- [x] 全部候选成功时返回 `RESET`，计数字段准确。
- [x] 部分条目失败时返回 `PARTIAL_RESET`，已成功项保留且后续项继续执行。
- [x] 全部候选原统计 field 缺失时返回 `NO_USAGE`，不创建零基线。
- [x] 身份发现返回无限额时直接返回 `NO_QUOTA`，不访问快照写入。
- [x] 大量候选通过多个 pipeline 批次处理，不因数量本身失败。
- [x] 相同请求再次执行会把快照更新为第二次调用时的 raw 值，不实现幂等。
- [x] 重置完成后原统计值和 MySQL 同步结果保持真实累计量。
- [x] 观测数据包含请求、成功条目、部分成功、无限额、无用量和失败计数，且没有高基数标签。
- [x] 日志包含入口可补充字段、周期、指标及汇总计数，不输出敏感 API Key 明文。

## 验证计划

- `cd backend && go test ./internal/service/tokenstat ./internal/repository/tokenstat`
- 使用故障注入 Stub 验证中间条目失败后后续条目仍执行、成功项不回滚。
- 使用 miniredis + sqlmock 验证 dirty/MySQL 身份联合发现、快照写入与原聚合同步隔离。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 编排 | `backend/internal/service/tokenstat/quota_reset_service.go` | 实现 runtime gate、固定批次、四状态及部分/全失败语义。 |
| Pipeline | `backend/internal/repository/tokenstat/quota_reset.go` | 普通 pipeline 内每条独立 EVAL；逐命令分类 RESET/NO_USAGE/FAILED。 |
| 可观测性 | `backend/internal/service/tokenstat/observability.go` | 增加请求、成功条目、部分成功、无限额、无用量和失败固定计数。 |
| 测试 | `quota_reset_service_test.go`、`quota_reset_test.go` | 覆盖多批、顺序、部分成功、全失败、NO_USAGE、NO_QUOTA、disabled 和同批 WRONGTYPE 隔离。 |
| 命令 | `cd backend && go test ./internal/service/tokenstat ./internal/repository/tokenstat`；`go test ./cmd/server` | 全部通过。 |
| 格式 | `git diff --check -- ...` | 通过。 |

## 执行记录

2026-09-09 完成。go-redis `Pipeline.Exec` 在单条 Lua WRONGTYPE 时返回 error，但各 `*redis.Cmd` 仍保留独立结果，因此实现始终逐命令分类，不把整批自动判失败；后续条目和批次继续执行，成功快照不回滚。日志仅记录周期、指标、输入维度数量和汇总计数，不输出维度值。
