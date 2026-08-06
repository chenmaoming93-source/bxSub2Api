# MVP-004：模型路由账号冷启动预热

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `复用 MVP-003 批量缓存能力，在启动流程增加有限范围扫描、分批写入和恢复测试。`
- Dependencies: `MVP-003`

## 预期成果

Redis 清空或重启后，服务启动会将所有已启用 `model_routing.account_ids` 引用的账号分批预热到现有单账号缓存。

## 背景

现有 `SchedulerSnapshotService.runInitialRebuild` 主要恢复正常 bucket 账号，无法覆盖没有 `account_groups` 关系的纯路由账号。

## 范围内

- 获取已启用模型路由的分组。
- 解析、去重全部路由账号 ID。
- 有界分批加载账号并写入缓存。
- 预热失败时记录日志/指标但不阻止服务启动。
- 验证请求路径仍能为遗漏账号回源兜底。

## 范围外

- 预热数据库中所有账号。
- 重写正常 bucket 初始重建。
- 依赖 outbox 重放全部历史账号。

## 实现说明

- 批次大小应固定或配置有界，避免启动数据库峰值。
- 预热只处理有效、启用路由配置中引用的 ID。
- 非法历史路由应可观测且不能中断其他分组预热。

## 验收标准

- [x] Redis 空状态启动后，纯模型路由账号进入 `sched:acc:<id>`。
- [x] 无 `account_groups` 关系的账号也能预热。
- [x] 账号按有限批次查询，不产生无界并发。
- [x] 单批或单组失败不阻止服务启动，并有可诊断记录。
- [x] 预热遗漏时 MVP-003 的请求回源仍可恢复。

## 验证计划

- `cd backend && go test ./internal/service ./internal/repository -run 'SchedulerSnapshot|Warm|InitialRebuild'`
- 使用集成 Redis 清空缓存后启动服务，检查路由账号 key 和预热指标。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 启动接入 | `SchedulerSnapshotService.runInitialRebuild` | 初始 bucket 重建结束或提前返回时均 defer 执行路由账号预热，失败不阻止启动。 |
| 有界预热 | `SchedulerSnapshotService.warmRouteAccounts` | 只读取活动且启用 routing 的分组，解析/去重 account_ids，以 128 个 ID 串行分批查库并批量写现有账号 key。 |
| 降级 | 日志前缀 `[Scheduler] route account warmup ...` | 分组 JSON 非法、单批 DB 或 Redis 失败均记录 group/batch 上下文并继续。 |
| 测试 | `cd backend && go test ./internal/service ./internal/repository -run 'SchedulerSnapshot|Warm|InitialRebuild' -count=1` | 通过；130 个无 account_groups 的纯路由账号触发 2 个有界 DB 批次并全部回填。 |

## 执行记录

- 固定批次大小为 128，不启动并发 goroutine，避免启动瞬时数据库峰值。
- 测试同时放入一个非法历史路由分组，确认它不会中断其他分组预热。
- MVP-003 的请求时批量 cache-first 回源仍保留，因此预热遗漏或失败不会造成永久不可用。
