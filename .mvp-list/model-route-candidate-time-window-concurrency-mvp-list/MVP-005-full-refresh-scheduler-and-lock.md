# MVP-005：分时段并发全量刷新、整分钟调度与全局锁

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 包含每日有效值计算、Redis 批量写入/清理、整分钟调度、分布式租约续租和日志，工作量高于普通单点功能。
- Dependencies: `MVP-001, MVP-004`

## 预期成果

所有网关实例都可安全参与调度，但同一时刻只有一个实例执行全量刷新：按 Asia/Shanghai 默认时区计算当前分钟，将有配置候选的有效值写入新 Redis Hash；未覆盖区间使用旧 `max_concurrency`，无配置候选完全不创建新 key。

## 背景

刷新是请求侧的预计算机制，不能让每个请求检查时间。现有旧 route-config key 继续保存旧配置值。刷新锁使用 `concurrency:route-schedule:refresh-lock`，通过随机 token 的 `SET NX PX` 获取，默认初始 TTL 300 秒、每 30 秒续租；续租/释放必须校验 token。

## 范围内

- 实现按整分钟触发的全量刷新任务，所有实例尝试抢同一把 Redis 锁，失败者跳过。
- 批量读取新表，按候选匹配当前分钟；命中区间用区间值，未命中用旧字段默认值。
- 批量写入新 Hash 的 `limit`、`updated_at`，清理已删除分时段配置产生的旧新 key；保留旧 key。
- 实现锁续租、token 安全释放、续租失败停止后续写入，以及锁过期后的其他实例接管。
- 配置文件增加带注释的 `refresh_lock_ttl_seconds: 300` 与 `refresh_lock_renew_interval_seconds: 30` 默认值，并完成启动配置解析。
- 记录刷新开始/结束日志、task_id、trigger、实例、耗时、候选总数、更新/失败数和错误 route key。
- 覆盖刷新耗时跨过一分钟时不补历史轮次，完成后由下一个未来整分钟触发。

## 范围外

- 不修改旧表和旧 route-config key。
- 不在本 MVP 增加管理 API 的立即刷新入口。
- 不做候选账号分时段并发总和校验或跨日期调度。

## 实现说明

- 定时器应按墙上时钟对齐到每分钟 `00` 秒，而不是“任务完成后 sleep 60 秒”；任务执行期间到达的 tick 抢锁失败即跳过。
- 全量刷新失败保留上一次成功的新 Hash 值，下一轮重试；续租失败后禁止继续写，避免锁失效后覆盖接管实例结果。
- 任务结束释放锁，锁释放脚本以 token 比较后删除，防止误删别的实例的锁。

## 验收标准

- [x] 多实例同一整分钟只会有一个实例完成刷新，其他实例有明确跳过日志。
- [x] 命中/未命中时间段、未配置候选、数值/unlimited、删除清理和旧 key 保留均正确。
- [x] 续租失败不会继续写入；持锁实例异常后 TTL 到期，其他实例可以接管。
- [x] 刷新跨分钟不产生补历史任务，完成后的后续整分钟可再次刷新。
- [x] 刷新开始和结束日志字段齐全，配置默认值和注释可在运行配置中观察。

## 验证计划

- `go test ./internal/service/... ./internal/repository/... ./internal/config/...`（在 `backend` 目录执行）。
- 使用 miniredis/可控时钟或测试注入验证抢锁、续租、释放、长任务和批量写入/清理。
- 检查配置样例与启动后的默认配置值。

## 完成证据

实现文件：

- `backend/internal/service/model_route_concurrency_schedule_refresh.go`
- `backend/internal/repository/model_route_concurrency_schedule_refresh_repo.go`
- `backend/internal/repository/concurrency_cache.go`
- `backend/internal/config/config.go`
- `backend/config/config.yaml`
- `backend/resources/config.yaml`
- `backend/cmd/server/wire.go`
- `backend/cmd/server/wire_gen.go`

验证结果：

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Go 行为测试 | `go test ./internal/service -run 'TestModelRouteConcurrencyScheduleRefresh|TestRouteScheduleCache|TestGetRouteConcurrencyLimit' -count=1` | 通过；覆盖当前分钟匹配、默认值/unlimited、旧 key 保留、清理 stale key、抢锁冲突、续租和 token 安全释放 |
| Repository 测试 | `go test ./internal/repository -run 'TestListModelRouteConcurrencyScheduleCandidates|TestRouteScheduleCache|TestGetRouteConcurrencyLimit' -count=1` | 通过；覆盖新表与旧默认值聚合及批量 Redis projection |
| Config/启动链路 | `go test ./internal/config -count=1`；`go test ./cmd/server -run 'Test.*Cleanup' -count=1` | 通过；配置默认值、校验和刷新器生命周期接入正常 |
| 综合编译测试 | `go test ./internal/service ./internal/repository ./internal/config ./cmd/server` | 通过 |
| 配置检查 | `gateway.model_route_schedule.refresh_lock_ttl_seconds: 300`、`refresh_lock_renew_interval_seconds: 30` | 已写入带注释的运行配置样例 |

| 类型 | 命令或路径 | 结果 |
|---|---|---|

## 执行记录

<记录执行过程中的偏差、阻塞项与决策。>
