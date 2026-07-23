# MVP-012：补齐 Redis 故障降级与读修复可观测性

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 聚焦异常分类、结构化日志/计数与故障测试，不扩大功能范围，约 40 分钟。
- Dependencies: `MVP-003, MVP-005, MVP-006, MVP-007, MVP-008`

## 预期成果

Redis 连接失败、非法 field/value、修复失败和并发跳过均有可验证行为及诊断信息，且不会把故障误判成数据缺失。

## 背景

项目已有 `token_statistics_fault_test.go` 和结构化 logger 使用方式，可扩展现有事件而无需引入新监控系统。

## 范围内

- 为批量读取、MySQL 降级、MySQL-only/Redis-only 数量、修复结果增加结构化日志或现有指标接点。
- 保证正常逐条命中不刷日志。
- 覆盖 Redis 连接失败、空集合、非法 field/value、修复失败和并发跳过。
- 验证 Redis 故障时返回 MySQL，且不执行覆盖式修复。

## 范围外

- 不接入新的外部监控平台。
- 不新增用户可见错误码。

## 实现说明

- 日志不得输出敏感用户信息或完整编码内容。
- 事件名与现有 `token_statistics.*` 风格保持一致。

## 验收标准

- [x] Redis 故障和成功空集合可被明确区分。
- [x] 降级查询仍返回 MySQL 数据。
- [x] 故障时不触发缺失修复。
- [x] 关键日志字段至少包含 statistics_type、usage_date、stage、fallback/repair 结果。

## 验证计划

- `cd backend && go test ./internal/repository ./internal/service -run 'Test.*TokenStatistics.*Fault|Test.*Fallback|Test.*Repair'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 读取诊断 | `backend/internal/repository/current_token_usage_reader.go` | Redis 批量读取失败记录 `statistics_type/usage_date/stage/fallback_attempted`；非法条目仅聚合计数，不输出完整 field。 |
| 修复诊断 | `backend/internal/repository/token_usage_read_repair.go` | 修复批次记录 repaired/concurrent_skipped/failed/repair_succeeded；失败错误包含 type/date/stage。 |
| 故障测试 | `backend/internal/service/token_usage_fallback_test.go`、现有 repository fault/repair 测试 | 验证 Redis 故障返回 MySQL 且不修复、空集合与错误分离、非法条目、失败和并发跳过。 |
| 聚焦验证 | `cd backend && go test ./internal/repository ./internal/service -run 'Test.*TokenStatistics.*Fault|Test.*Fallback|Test.*Repair'` | 通过。 |
| 包回归 | `cd backend && go test ./internal/repository ./internal/service ./cmd/server` | 通过。 |

## 执行记录

2026-07-15 完成。正常逐条命中不记录日志；只记录故障、非法条目聚合和实际修复批次。日志不包含用户名、邮箱或完整编码 field。
