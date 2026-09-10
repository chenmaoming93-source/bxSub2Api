# MVP-006：立即全量刷新入口及与定时任务互斥

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 复用刷新服务和同一把锁，增加手动触发 API、状态响应、权限及页面入口，边界清晰。
- Dependencies: `MVP-003, MVP-005`

## 预期成果

用户点击“立即刷新”时，如果没有刷新任务运行则立即启动全量刷新；如果已有刷新任务则立即返回“刷新任务正在执行”，不排队、不创建 pending 任务。手动刷新和定时刷新共享同一把全局 Redis 锁。

## 背景

定时刷新以每个整分钟为触发点，任务可能超过一分钟。立即刷新必须能绕过等待下一个整分钟，但不能与定时刷新重叠。既有页面在 MVP-003 已具备分时段配置区域，本 MVP 接入操作入口。

## 范围内

- 增加全量立即刷新管理接口和运维/分组管理权限校验。
- 抢锁成功后立即异步启动全量刷新，并返回已启动及 task_id；抢锁失败返回任务正在执行的明确业务响应。
- 与定时任务复用 MVP-005 的抢锁、续租、释放、日志和刷新实现，不复制第二套逻辑。
- 页面提供立即刷新按钮、执行中提示、成功启动提示和任务冲突提示。
- 覆盖手动/定时同时触发、刷新执行中再次点击、任务完成后的下一个整分钟行为。

## 范围外

- 不提供任务排队、取消或历史任务持久化。
- 不改变刷新失败后的保留旧值、下一轮重试策略。
- 不改变用户请求路径的 Redis 读取方式。

## 实现说明

- 接口应快速返回，不等待全量刷新完成；结果以日志和既有运行状态/响应中的 task_id 观察。
- 手动任务结束后，定时器继续按墙上时钟工作；例如 10:00:20 启动、10:01:40 完成，下一次为 10:02:00。
- 手动和定时发生竞态时由 Redis `SET NX` 决定唯一持有者，未抢到锁的一方不启动写任务。

## 验收标准

- [x] 无任务时手动触发会立即启动全量刷新并返回 task_id。
- [x] 有任务时手动触发立即返回“刷新任务正在执行”，不会重复写 Redis或排队。
- [x] 手动/定时竞争和多次点击均只有一个刷新执行者，使用同一锁和日志字段。
- [x] 页面按钮和权限行为可观察，前端类型检查及相关组件测试通过。

## 验证计划

- `go test ./internal/handler/... ./internal/service/... ./internal/server/...`（在 `backend` 目录执行）。
- `pnpm --dir frontend run typecheck`
- `pnpm --dir frontend exec vitest run <立即刷新入口相关测试文件>`
- 使用手工 API 请求或集成测试验证 202/业务冲突响应和 task_id。

## 完成证据

实现文件：

- `backend/internal/service/model_route_concurrency_schedule_refresh.go`
- `backend/internal/handler/admin/group_handler.go`
- `backend/internal/handler/wire.go`
- `backend/internal/server/routes/admin.go`
- `frontend/src/api/admin/groups.ts`
- `frontend/src/components/admin/group/GroupModelRoutingEditor.vue`
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

验证结果：

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 立即刷新行为 | `go test ./internal/handler/admin -run 'TestGroupConcurrencySchedule' -count=1` | 通过；覆盖无任务立即启动、返回 task_id、执行中再次触发返回 409 且不排队 |
| 刷新服务互斥 | `go test ./internal/service -run 'TestModelRouteConcurrencyScheduleRefresh' -count=1` | 通过；立即入口与定时入口共用同一 Redis 锁 |
| 路由注册/生命周期 | `go test ./internal/server/routes -count=1`；`go test ./cmd/server -run 'Test.*Cleanup' -count=1` | 通过；POST 入口使用 `PermissionGroupsUpdate`，刷新器纳入清理链路 |
| 前端组件与类型 | `pnpm exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/components/admin/group/__tests__/ModelRouteConcurrencyScheduleEditor.spec.ts src/components/admin/group/__tests__/modelRouteConcurrencySchedule.spec.ts`；`pnpm run typecheck` | 通过；13 个组件/工具测试通过，类型检查通过 |

| 类型 | 命令或路径 | 结果 |
|---|---|---|

## 执行记录

<记录执行过程中的偏差、阻塞项与决策。>
