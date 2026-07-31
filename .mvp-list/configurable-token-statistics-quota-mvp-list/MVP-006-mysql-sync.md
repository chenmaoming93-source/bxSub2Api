# MVP-006：交付 Redis 到 MySQL 幂等同步

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `脏集合轮换、快照读取、版本UPSERT和失败恢复可以独立验证并形成完整持久化闭环。`
- Dependencies: `MVP-002, MVP-004`

## 预期成果

后台任务能把新Redis统计的绝对值和版本批量同步到 `token_stat_aggregates`，重复执行和并发写入均不重复计数或漏掉新脏标记。

## 背景

MySQL负责历史数据，模型请求不得同步写MySQL。同步必须使用新体系独立锁和配置。

## 范围内

- 当前/处理中脏集合原子轮换。
- 批量读取Redis计数、版本和维度内容。
- MySQL版本式UPSERT。
- 分布式同步租约。
- 失败重试和重新入队。
- 同步延迟与失败指标。

## 范围外

- 周期删除。
- 通用查询API。

## 实现说明

- 不清零Redis计数器。
- 处理中有新写入时，新脏标记必须留在current集合。
- 不复用旧 `TokenStatisticsSyncEngine`。

## 验收标准

- [x] Redis快照可写入通用聚合表。
- [x] 重复同步不增加统计值。
- [x] 旧版本不能覆盖新版本。
- [x] 同步期间新写入不会丢失脏标记。
- [x] MySQL失败后可安全重试。
- [x] 同步任务不影响模型请求。

## 验证计划

- `cd backend && go test ./internal/repository/... ./internal/service/... -run "DynamicToken.*Sync|TokenStat.*Sync"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 同步引擎 | `backend/internal/repository/tokenstat/sync_engine.go` | 独立租约、脏集合轮换、绝对值/版本快照、失败回队和定时运行已实现 |
| MySQL | `backend/internal/repository/tokenstat/repository.go` | 版本单调 UPSERT，重复和旧版本均不增加/覆盖 |
| 装配 | `backend/cmd/server/wire_gen.go` | 新配置启用时独立启动同步任务，不进入模型请求路径 |
| 测试 | `cd backend && go test ./internal/repository/... ./internal/service/... ./cmd/server -run "DynamicToken.*Sync\|TokenStat.*Sync"` | 通过；覆盖并发新脏标记、失败重试和绝对快照版本 |

## 执行记录

2026-07-30：完成 Redis→MySQL 幂等同步、后台调度和故障恢复测试。
