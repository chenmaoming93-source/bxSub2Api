# MVP-002：交付当前三周期 Redis 精确读取能力

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `范围集中在共享 Key 规则、精确 HGET、结果语义及 miniredis 兼容测试，可独立实现和验证。`
- Dependencies: `MVP-001`

## 预期成果

提供一个可复用的动态统计当前周期 Reader，能按投影、维度身份和周期精确读取 `total_tokens`，并严格区分 Field 不存在、合法值与 Redis 故障。

## 背景

写入端位于 `backend/internal/repository/tokenstat/redis_accumulator.go`。读取端必须与其 Key、Field、周期格式、hash 和 shard 规则完全一致，不允许扫描 Redis。

## 范围内

- 抽取或复用读写双方共享的 Redis Key/Field Builder。
- 新增 `current_usage_reader.go` 及窄接口。
- 支持日、周、月精确读取。
- 将 `redis.Nil` 映射为 `exists=false,value=0`。
- 将合法非负整数映射为 `exists=true`。
- 将连接错误、超时、非法整数和负值返回为明确错误。
- 使用 miniredis 编写读写兼容测试。

## 范围外

- 不解析用户、分组或 API Key 名。
- 不匹配统计投影，不注册 HTTP 路由。
- 不查询 MySQL，不实现故障回退。

## 实现说明

- 复用 `BuildDimensionIdentity`、`RedisField`、`RedisShard` 和自然周期规则。
- 可用 pipeline 读取三个周期，但 Reader 的单周期结果必须可独立测试。
- 禁止 `KEYS`、`SCAN`、遍历所有 shard 或 Hash Field。
- 不访问旧 `sub2api:token_stats:*` namespace。

## 验收标准

- [x] Reader 可正确读取 D/W/M 三类当前周期值。
- [x] Redis Key 和 Field 与 `RedisAccumulator` 产出的数据完全兼容。
- [x] 缺失 Field、零值、正值、非法值和 Redis 故障语义正确。
- [x] 测试证明不使用旧 namespace 或扫描命令。
- [x] Reader API 不接受外部业务名称或任意 Redis Key。

## 验证计划

- `go test ./internal/repository/tokenstat -run 'Redis|CurrentUsage'`
- `go test ./internal/service/tokenstat`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/tokenstat/current_usage_reader.go` | 新增按内部 Period、projectionID、hash、metric 精确读取的 Reader |
| 共享规则 | `backend/internal/repository/tokenstat/redis_accumulator.go` | 写入端与读取端共享 `DynamicCountKey`、Field 和 shard 规则 |
| 聚焦测试 | `go test ./internal/repository/tokenstat -run 'Redis|CurrentUsage'` | PASS |
| 领域回归 | `go test ./internal/service/tokenstat` | PASS |

## 执行记录

- 2026-07-31：实现 D/W/M 兼容读取；`redis.Nil` 返回不存在和 0，非法/负数返回错误。
- Reader 只接受内部计算参数，不允许调用方传任意 Redis Key，且未引入任何扫描操作。

