# MVP-009：切换 GatewayService 用量记录路径

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 聚焦 Anthropic/Gateway RecordUsage 家族，停用金额调用并接入公共 Redis 累计。
- Dependencies: `MVP-003, MVP-008`

## 预期成果

`GatewayService.RecordUsage` 与长上下文变体继续写 `usage_logs`，不再计算或落库金额，不再逐请求更新三张 Token 表，改为 Redis 累计。

## 背景

主要代码位于 `backend/internal/service/gateway_service.go` 和 `daily_token_quota_accounting.go`。

## 范围内

- 旁路价格计算、账号金额统计和 `applyUsageBilling`。
- 保持 `UsageLog` 构建和现有写入方式。
- 接入公共 Redis accumulator。
- 保留原金额和 MySQL 增量实现。
- RecordUsage 聚焦测试。

## 范围外

- OpenAI 专用 RecordUsage 和 handler wiring。

## 实现说明

- 避免 `CostBreakdown` 为 nil 时解引用。
- 金额字段使用现有零值，不改 `UsageLog` Schema。

## 验收标准

- [x] `usage_logs` 写入路径保持。
- [x] 金额 Repository 不再被调用。
- [x] 原逐请求三表增量不再被调用。
- [x] Redis 收到三个维度的累计。

## 验证计划

- `cd backend; go test ./internal/service -run "GatewayServiceRecordUsage|DailyTokenQuotaAccounting"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/gateway_service.go` | `recordUsageCore` 保留 usage log 构建/写入，金额成本归零并旁路 `applyUsageBilling`、账号金额统计和旧三表增量。 |
| 公共接入 | `backend/internal/service/daily_token_quota_accounting.go` | 新增 usage log 到三维 Redis 累计输入的公共适配，只有新 usage log 且 Token>0 时调用。 |
| 测试 | `cd backend; go test ./internal/service -run "GatewayServiceRecordUsage|DailyTokenQuotaAccounting" -count=1` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 5.144s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：Gateway 与长上下文变体共用的新核心路径已切换；保留成本计算、`applyUsageBilling` 和 `incrementDailyTokenQuotasForUsage` 具体实现，但正常路径不调用。
