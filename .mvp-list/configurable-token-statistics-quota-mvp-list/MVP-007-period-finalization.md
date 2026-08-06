# MVP-007：交付周期封账和 Redis 安全清理

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 开发日`
- Estimate rationale: `周期状态机、最终同步门禁和UNLINK清理可作为独立运维能力交付和故障注入验证。`
- Dependencies: `MVP-005, MVP-006`

## 预期成果

已结束的日、周、月周期只有在异步写入排空、最终同步和版本确认后才删除Redis数据；失败时保留并重试。

## 背景

Redis只需保留当前周期，但不能因TTL或定时删除导致未同步数据丢失。

## 范围内

- `OPEN/CLOSING/FINAL_SYNC/PERSISTED/DELETED` 状态机。
- Worker水位或等效排空证明。
- 最终同步。
- Redis/MySQL版本核验。
- `UNLINK`清理新体系旧周期Key。
- 兜底TTL和孤儿Key策略。
- 管理状态和手动触发后端能力。

## 范围外

- 前端同步状态页。
- 旧Redis Key清理。

## 实现说明

- 只清理 `sub2api:dynamic_token_stats:v1:*` 对应已持久化周期。
- MySQL失败、脏集合非空或版本不一致时不得删除。

## 验收标准

- [x] 周期边界后进入正确状态。
- [x] 队列仍有旧周期事件时不会删除。
- [x] 最终同步失败时不会删除。
- [x] 版本核验通过后使用UNLINK并记录状态。
- [x] 重复封账幂等。
- [x] 不触碰当前周期或旧体系Redis Key。

## 验证计划

- `cd backend && go test ./internal/service/... ./internal/repository/... -run "PeriodFinal|DynamicToken.*Period"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 状态与清理 | `backend/internal/repository/tokenstat/period_finalizer.go` | 五态门禁、队列排空、最终同步、版本核验、UNLINK 和定时检查已实现 |
| 持久化 | `backend/internal/repository/tokenstat/repository.go` | 周期状态幂等 UPSERT 和 Redis/MySQL 版本逐项核验已实现 |
| 测试 | `cd backend && go test ./internal/service/... ./internal/repository/... ./cmd/server -run "PeriodFinal\|DynamicToken.*Period"` | 通过；覆盖当前周期保护、pending、同步失败、版本失败和成功删除 |

## 执行记录

2026-07-30：完成周期封账、版本门禁、安全清理、调度装配和故障测试。
