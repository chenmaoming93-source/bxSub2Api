# MVP-003：路由账号批量 cache-first 加载与回源写回

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个专注开发日`
- Estimate rationale: `需要扩展缓存接口、处理部分命中、接入 fallback 限制并验证数据库调用次数，超过普通单日但仍是单一架构成果。`
- Dependencies: `none`

## 预期成果

模型路由显式账号通过现有 `sched:acc:<id>` 批量读取；只有 Redis 未命中的 ID 才一次性批量查库并写回缓存，热态请求不再查账号数据库。

## 背景

当前 `backend/internal/service/gateway_route_accounts.go` 直接调用 `accountRepo.GetByIDs`。Redis 实现位于 `backend/internal/repository/scheduler_cache.go`，已有分块 `MGET` 内部能力和单账号写入能力。

## 范围内

- 在 SchedulerCache/SchedulerSnapshotService 边界增加批量账号读取。
- 对账号 ID 去重并批量 MGET。
- 对未命中 ID 执行一次 `GetByIDs`。
- 将回源结果 best-effort 写回现有账号缓存。
- 接入现有 DB fallback 开关、QPS 限制和 timeout。
- 增加命中、未命中、回源及写回失败测试和指标。

## 范围外

- 新建 Redis key 或第二套账号缓存。
- 冷启动扫描模型路由账号。
- 修改账号快照现有同步机制。

## 实现说明

- 批量结果应支持高效构建 `accountByID`。
- Redis 写回失败不能使已成功回源的当前请求失败。
- Redis 与数据库均不可用时沿用现有调度失败语义。
- 不应循环调用 `SchedulerSnapshotService.GetAccount`，避免 N 次 Redis/DB 往返。

## 验收标准

- [x] Redis 全命中时 `GetByIDs` 调用次数为零。
- [x] 部分命中时只查询缺失 ID，且只调用一次 `GetByIDs`。
- [x] 回源结果写入 `sched:acc:<id>` 并供后续请求命中。
- [x] 写回 Redis 失败时当前请求仍可使用回源账号。
- [x] fallback 限制和错误语义保持有效。

## 验证计划

- `cd backend && go test ./internal/repository ./internal/service -run 'SchedulerCache|RouteAccount|MergeExplicitRouteAccounts'`
- 使用仓库集成测试环境验证一次冷读后第二次读取不触发账号数据库查询。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 缓存接口 | `SchedulerAccountBatchCache.GetAccounts/SetAccounts` | 作为现有 SchedulerCache 的可选批量扩展，生产 Redis 实现复用 `sched:acc:<id>` 与分块 MGET/批量 pipeline。 |
| 服务回源 | `SchedulerSnapshotService.GetAccounts` | ID 去重、部分命中只批量查询缺失 ID 一次，沿用 fallback QPS/timeout，写回失败仅记录日志。 |
| 路由接入 | `backend/internal/service/gateway_route_accounts.go` | 显式 account_ids 优先使用 SchedulerSnapshotService，结果按请求 ID 顺序合并。 |
| 测试 | `cd backend && go test ./internal/repository ./internal/service -run 'SchedulerCache|RouteAccount|MergeExplicitRouteAccounts' -count=1` | repository 与 service 均通过；覆盖全命中、部分命中、二次热读、写回失败和限流。 |

## 执行记录

- 批量接口返回 `map[int64]*Account`，缺失 ID 不占位，调用方可 O(1) 构建 accountByID。
- 兼容未实现批量扩展的测试替身：服务直接把全部 ID 视为未命中并执行一次受控回源；生产 Redis 实现必定走批量 MGET。
- 测试明确断言：全命中 DB 0 次；部分命中 DB 1 次且仅查询 `[2,3]`；回填后二次读取 DB 次数仍为 1。
