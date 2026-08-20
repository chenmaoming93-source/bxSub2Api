# MVP-004：请求侧 Redis 当前值读取与旧 key 回退

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 在现有并发缓存接口上增加新 Hash key 的优先读取和旧 key 回退，配合 Redis/单元测试即可独立验证请求路径无时间计算。
- Dependencies: `MVP-001`

## 预期成果

请求执行时通过一次 Redis 读取优先使用分时段当前值；新 key 不存在时使用原有 `concurrency:route-config:{route-key}`，两者都缺失时继续原有数据库兜底逻辑。请求路径不查新表、不计算时间。

## 背景

现有实现集中在 `backend/internal/repository/concurrency_cache.go`，旧配置 key 为 `concurrency:route-config:{route-key}`，实际槽位 key 为 `concurrency:route:{route-key}`。新 key 只为有分时段配置的候选建立：`concurrency:route-schedule:{route-key}`，Hash 至少包含 `limit` 和 `updated_at`。

## 范围内

- 增加新 key/hash 的常量、读写辅助方法及数值/`unlimited`解析。
- 修改候选级并发限制读取为“新 Hash 的 `limit` 优先、旧 key 回退”。
- 保持 `AcquireRouteSlot` 的槽位 key 和原有 `unlimited` 行为不变。
- 覆盖无分时段、新 key 存在、新 key 缺失、旧 key 也缺失、数值和 unlimited 场景。
- 明确 Redis 多次访问如何合并为一次 pipeline/Lua 操作，避免额外 DB/时间判断。

## 范围外

- 不负责从数据库计算有效时间段或定时写入新 key。
- 不改变旧 route-config key 的值，也不把所有候选强制写入新 key。
- 不实现刷新锁、调度器和立即刷新。

## 实现说明

- 新 Hash 缺少 `limit` 时视为当前值不可用并回退旧 key；不能把 Hash 元数据误当并发值。
- 新 key 和旧 key 同时缺失时沿用现有数据库兜底读取，保证升级期间兼容。
- 读路径只做 Redis 读取和既有并发槽位流程，不调用分时段表 repository。

## 验收标准

- [x] 新 Hash 存在时数值/`unlimited`优先生效，新 Hash 缺失时旧 key 生效。
- [x] 两个 key 都缺失时既有数据库兜底生效；无分时段候选不创建新 key也不改变旧行为。
- [x] 请求路径测试证明不查询分时段数据库且不依赖当前时间。
- [x] Redis 读写和并发获取定向测试通过，旧 key/实际槽位 key 名称未改变。

## 验证计划

- `go test ./internal/repository/... ./internal/service/...`（在 `backend` 目录执行）。
- 使用 miniredis 或现有 Redis 集成测试验证 key 优先级、回退和 `unlimited`。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 请求侧读取 | `backend/internal/repository/concurrency_cache.go` | `GetRouteConcurrencyLimit` 通过一次 Pipeline 读取新 Hash `limit` 和旧 route-config，优先新值并回退旧值 |
| Redis 扩展 | `backend/internal/service/concurrency_service.go` | 新增可选 `RouteScheduleCache`，为刷新任务提供 Hash 写入/删除/扫描能力，不改变旧缓存接口 |
| Redis 测试 | `go test ./internal/repository ./internal/service -run 'RouteConcurrencyLimit|RouteScheduleCache|Concurrency' -count=1` | 通过：repository 与 service 定向测试均通过 |
| 兼容检查 | `backend/internal/service/gateway_service.go`、`backend/internal/service/openai_gateway_service.go` | 两个请求路径继续调用既有 `GetRouteConcurrencyLimit`，未增加时间计算或分时段表查询 |

## 执行记录

- 2026-08-18：新 Hash 缺少 `limit` 或值格式异常时回退旧 key，避免 Redis 部分写入/升级期间阻断旧逻辑。
- 2026-08-18：新 Hash 的写入、删除和扫描能力作为可选扩展保留给后续刷新任务，未扩展已有 `ConcurrencyCache` 接口，避免破坏测试桩和旧调用方。
