# MVP-007：全链路、兼容性与发布前验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 汇总前后端、数据库、Redis 和多实例时序验证，重点是发布安全和旧逻辑回归。
- Dependencies: `MVP-002, MVP-003, MVP-004, MVP-005, MVP-006`

## 预期成果

完成从页面保存到数据库、刷新任务、新 Redis 当前值、请求侧并发判断的闭环验证，并确认新表为空、旧版本忽略新 key、新版本回退旧 key 时均能安全运行。

## 背景

该功能要求老表和老逻辑完全保留。新版本升级后，未配置分时段的候选必须继续使用 `group_model_route_accounts.max_concurrency`；分时段配置删除后，新 key 清理并恢复旧逻辑。刷新是最终一致的预计算机制，允许约 1 分钟周期延迟，但立即刷新必须可抢锁立即启动。

## 范围内

- 端到端验证页面/API/新表/刷新/Redis/request acquire 的数据流。
- 验证三类时间行为：命中分时段、未覆盖回退旧默认、完全无配置旧逻辑。
- 验证数值、`unlimited`、边界 `00:00/24:00`、相邻区间和重叠拒绝。
- 验证多实例锁、续租失败、锁过期接管、长任务跳过整分钟和手动/定时互斥。
- 验证新旧 key 兼容、旧表 schema 未变、删除配置清理新 key、刷新失败保留最后成功值。
- 汇总定向测试、构建/类型检查和发布注意事项到各 MVP 的完成证据。

## 范围外

- 不增加按日期、星期或节假日的配置。
- 不引入版本号 key、修改旧表字段或做分时段总和合理性校验。
- 不以“测试通过”替代真实缺失依赖的证据；环境限制必须记录为阻塞。

## 实现说明

- 优先使用仓库现有 Go/前端/Redis 测试设施；跨服务场景使用可控时钟、miniredis 或现有集成环境。
- 对旧表执行 schema 对比或迁移回归检查，证明只有新表迁移被加入。
- 将关键兼容矩阵写入测试名称或证据表，便于后续发布和回滚排查。

## 验收标准

- [x] 页面配置的时间段最终能在请求侧生效，且请求处理没有分时段表查询或时间计算。
- [x] 空新表和完全未配置候选的旧行为回归通过；新 key 缺失能回退旧 key。
- [x] 分时段覆盖不全时未命中区间使用旧默认值，命中区间使用新值，`unlimited` 正确。
- [x] 多实例刷新、长任务、锁续租/接管、立即刷新冲突和日志验收全部通过。
- [x] 迁移/构建/后端定向测试/前端类型与组件测试结果均已记录，且无旧表结构变更。

## 验证计划

- `go test ./...`（在 `backend` 目录执行，若耗时或依赖受限则记录实际替代命令）。
- `pnpm --dir frontend run typecheck` 与 `pnpm --dir frontend run lint:check`。
- 运行新增端到端/集成测试，检查 Redis key、刷新日志和数据库迁移结果。
- 复核 `git diff -- backend/sqlArchiving backend/ent/schema backend/internal frontend/src`，确认改动范围符合 Plan。

## 完成证据

兼容性矩阵：

| 场景 | 结果 |
|---|---|
| 新表无配置/候选无分时段配置 | 不创建新 Hash，继续读取旧 `concurrency:route-config:{route-key}` |
| 新 Hash 缺失但旧 key 存在 | 请求侧一次 Pipeline 同时读取，新 Hash miss 后回退旧 key |
| 命中时间段 | 刷新任务写入新 Hash `limit`，请求侧优先读取新值 |
| 未覆盖时间段 | 刷新任务写入旧 `max_concurrency` 默认值；默认为 NULL 时写入 `unlimited` |
| 删除分时段配置 | 全量刷新清理对应新 Hash，旧 key 保留 |
| 立即刷新与定时刷新竞争 | 同一 Redis `SET NX PX` 锁决定唯一执行者，冲突返回 409，不排队 |
| 旧表/schema | 未修改 `group_model_route_accounts`，`backend/ent/schema` 无本次 diff；仅新增独立 SQL 表迁移 |

验证结果：

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 完整 Go 测试 | `go test ./...`（`backend`） | 运行完成；失败仅来自工作区既有无关改动：`internal/handler/admin/TestGroupRequestsAcceptLegacyAndCandidateModelRouting`、`internal/rbac` compatibility seed、`internal/repository/tokenstat` 统计测试；本次新增相关包定向测试均通过 |
| 后端定向测试 | `go test ./internal/handler/admin -run 'TestGroupConcurrencySchedule' -count=1`；`go test ./internal/service -run 'TestModelRouteConcurrencyScheduleRefresh' -count=1`；`go test ./internal/repository -run 'TestListModelRouteConcurrencyScheduleCandidates|TestRouteScheduleCache|TestGetRouteConcurrencyLimit' -count=1` | 通过 |
| 路由/配置/启动 | `go test ./internal/server/routes -count=1`；`go test ./internal/config -count=1`；`go test ./cmd/server -run 'Test.*Cleanup' -count=1` | 通过 |
| 前端测试 | `pnpm exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/components/admin/group/__tests__/ModelRouteConcurrencyScheduleEditor.spec.ts src/components/admin/group/__tests__/modelRouteConcurrencySchedule.spec.ts` | 通过，13 tests |
| 前端类型/lint | `pnpm run typecheck`；`pnpm run lint:check` | 类型检查通过；lint 退出成功，仅有既有 `UsageFilters.spec.ts` 未使用变量 warning |
| 迁移/schema 检查 | `backend/sqlArchiving/170_create_group_model_route_account_concurrency_schedules.sql`；`git diff --name-only -- backend/ent/schema` | 仅新增独立分时段配置表 SQL；旧表和 Ent schema 未改 |

| 类型 | 命令或路径 | 结果 |
|---|---|---|

## 执行记录

<记录执行过程中的偏差、阻塞项与决策。>
