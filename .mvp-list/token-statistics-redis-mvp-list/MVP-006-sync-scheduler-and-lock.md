# MVP-006：接入同步调度、分布式锁与跨天生命周期

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 单 worker 定时调度、现有 Redis 锁模式复用和当天/前一天同步构成一个运行时闭环。
- Dependencies: `MVP-001, MVP-005`

## 预期成果

应用运行期间仅有一个实例周期同步当天和前一天三个 Hash，并在关闭时安全停止。

## 背景

项目已有 `leader_lock_cache.go` 和 server/wire 生命周期模式可复用。

## 范围内

- 独立单 worker 定时器。
- Redis 分布式锁和合理锁 TTL。
- 当天、前一天同步顺序。
- 启停与正常关闭。
- 锁占用和锁失败的差异化处理。

## 范围外

- 历史扫描、恢复和对账。

## 实现说明

- 锁被其他实例持有属于正常跳过。
- 只处理当天和前一天，旧 Key 按配置 TTL 自然过期。

## 验收标准

- [x] 多实例竞争时最多一个同步执行者。
- [x] 同步间隔来自配置。
- [x] 当天和前一天均被处理，不扫描更早日期。
- [x] Stop 不泄漏 goroutine。

## 验证计划

- `cd backend; go test ./internal/service -run "TokenStatistics.*(Scheduler|Lock|Lifecycle|DayBoundary)"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/token_statistics_scheduler.go` | 单 ticker worker、Redis `SET NX` 所有权锁、安全释放、当天/前一天顺序同步及幂等 Stop。 |
| 测试 | `cd backend; go test ./internal/service -run "TokenStatistics.*(Scheduler|Lock|Lifecycle|DayBoundary)" -count=1` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 5.625s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：验证两个 scheduler 竞争同一锁时第二实例正常跳过、跨天仅同步当前与前一天、5ms 配置间隔驱动 worker，以及重复 Stop 安全返回。
