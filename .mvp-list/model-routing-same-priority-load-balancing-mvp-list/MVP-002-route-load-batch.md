# MVP-002：路由槽位批量负载与回退负载

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦 Redis route slot 读取、LoadRate 计算和无分配并发时的账号全局回退。
- Dependencies: `MVP-001`

## 预期成果

ConcurrencyService 能批量返回 `group + route_alias + account` 维度的当前并发和 LoadRate，供后续 priority 账号池选择使用。

## 背景

当前模型路由排序使用账号全局 `GetAccountsLoadBatch`。路由槽位已经通过 `AcquireRouteSlot` 存储，但缺少批量读取 route slot 当前并发的接口。

## 范围内

- 增加路由槽位批量负载读取接口；
- 使用 Redis pipeline 和现有过期槽位清理机制；
- `max_concurrency` 有值时计算 route LoadRate；
- `max_concurrency` 为空时计算账号全局回退 LoadRate；
- 增加 Redis/cache/service 单元或集成测试。

## 范围外

- 不切换 Gateway 主选择流程；
- 不实现路由等待队列；
- 不改变 route slot 的原子获取接口。

## 实现说明

- 增加 `RouteLoadCache`、`RouteLoadRequest`、`RouteLoadInfo` 和 `GetRouteLoadsBatch`；
- 路由 LoadRate 使用当前活跃并发数，不计等待数；
- `LoadRate >= 100` 由调用方视为不可用；
- 读取失败返回明确错误，不伪装为低负载；
- Redis 使用 pipeline 批量清理过期 route slot 并读取 ZCARD。

## 验收标准

- [x] 能批量读取多个 route slot 的当前活跃并发。
- [x] 配置 `max_concurrency=10`、当前并发 4 时返回 LoadRate=40。
- [x] `max_concurrency=null` 时按 `EffectiveLoadFactor` 计算账号全局回退负载。
- [x] 过期槽位不会继续计入当前并发。
- [x] 批量读取异常有可观测错误结果。

## 验证计划

- `cd backend; go test -tags unit ./internal/service -run 'TestGetRouteLoadsBatch' -count=1`
- `cd backend; go test ./internal/repository -run '^$'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `cd backend; go test -tags unit ./internal/service -run 'TestGetRouteLoadsBatch' -count=1` | 通过；route allocation 和 account fallback 两个测试通过 |
| 编译 | `cd backend; go test ./internal/repository -run '^$'` | 通过；仓储包编译成功 |
| 代码 | `backend/internal/service/concurrency_service.go`、`backend/internal/repository/concurrency_cache.go` | 新增 route load 批量接口、pipeline 读取和 LoadRate 计算 |

## 执行记录

2026-08-17：完成 route slot 批量读取及账号全局回退负载；聚焦 unit 测试和 repository 编译检查全部通过。
