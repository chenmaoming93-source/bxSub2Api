# MVP-004：交付四维三周期查询 Service

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `在现成对象解析和 Redis Reader 上编排投影匹配、周期状态和错误传播，形成完整但尚未暴露 HTTP 的业务切片。`
- Dependencies: `MVP-002, MVP-003`

## 预期成果

提供完整领域 Service：解析四维对象、匹配精确活动投影、计算当前日/周/月并返回可区分未配置、无数据、实际值和基础设施故障的结果。

## 背景

活动投影通过可配置 Token 统计快照维护。目标签名必须精确为 `user_id,api_key_id,group_id,route_alias`，指标必须包含 `total_tokens`。

## 范围内

- 定义查询响应领域模型和可空 Token 值表达。
- 只匹配精确四维 `ACTIVE` 投影。
- 检查活动投影快照初始化状态。
- 复用统计时区计算当前自然日、周、月。
- 三周期分别查询并生成状态。
- Redis 任一错误时返回统一不可用错误。
- 使用固定 Clock 或时间注入测试周期边界。

## 范围外

- 不实现 Gin Handler 和路由。
- 不新增统计投影或修改写入流程。
- 不查询 MySQL 历史聚合数据。

## 实现说明

- 规范化投影维度顺序后再比较签名。
- 未配置时仍返回三个周期及其边界，Token 为 `null`。
- 已配置但 Field 缺失时 Token 为 0、`data_present=false`。
- 不返回部分成功；Redis 故障必须与 `redis.Nil` 分离。

## 验收标准

- [x] 精确活动投影匹配成功，部分、额外维度或非活动投影不匹配。
- [x] 未配置时三个周期分别返回明确状态和 `null` Token。
- [x] 已配置无数据和 Field 中真实零值可通过 `data_present` 区分。
- [x] 日、周、月边界与写入端一致。
- [x] 任一 Redis 故障导致统一不可用错误。
- [x] Service 不依赖 Gin、JWT、RBAC 或旧统计类型。

## 验证计划

- `go test ./internal/service/tokenstat ./internal/service/... -run 'External.*TokenUsage|Current.*Usage|NaturalPeriods'`
- `go test ./internal/repository/tokenstat`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 领域实现 | `backend/internal/service/external_token_usage.go` | 精确投影匹配、三周期结果及统一不可用错误已实现 |
| Service 测试 | `go test ./internal/service/tokenstat ./internal/service -run 'External.*TokenUsage|Current.*Usage|NaturalPeriods'` | PASS |
| Reader 回归 | `go test ./internal/repository/tokenstat` | PASS |

## 执行记录

- 2026-07-31：通过固定 Clock 验证上海时区自然周从周一开始；未配置返回三周期 `nil` Token。
- Reader 返回真实零值时 `DataPresent=true`，Field 缺失时 `DataPresent=false`；第二个周期故障会终止完整请求。

