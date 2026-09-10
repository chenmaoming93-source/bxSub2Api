# MVP-010：安全日志保留期限与每日清理时间配置

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发者日`
- Estimate rationale: 同时覆盖 settings 持久化、后台每日调度、管理 API、日志页面配置和跨层验证，属于一个完整可操作的生命周期配置切片。
- Dependencies: `MVP-008, MVP-009`

## 预期成果

管理员可以在“安全检查日志”页面设置日志保留天数和每日清理时间。后台按服务器本地时区每日执行清理，默认保留 3 天、每日 03:00，配置修改无需重启即可生效。

## 范围内

- 使用现有 `settings` 表保存保留天数和清理时间；
- 保留天数默认 3 天，合法范围 1～3650 天；
- 清理时间默认 `03:00`，格式为 `HH:mm`；
- 按服务器本地时区每日清理；
- 严格在配置的每日时间执行；服务启动或配置保存晚于当天时间时不补偿，等待下一次计划时间；
- 保留每批最多删除 1000 条的批量清理策略；
- 新增安全日志清理配置查询和保存 API；
- 在独立安全日志页面增加配置卡片、权限控制和下一次清理时间；
- 清理失败不阻塞模型请求。

## 范围外

- 不修改 `security_check_logs` 表、字段、索引或迁移；
- 不新增独立时区持久化配置；
- 不支持分组级保留期限；
- 不引入消息队列或新的调度服务；
- 不改变日志采集和安全判定逻辑。

## 实现说明

- settings keys：`security_check_log_retention_days`、`security_check_log_cleanup_time`；
- `SecurityCheckCollector` 只在配置的本地时间分钟执行，错过该分钟不补偿，失败则等待下一次计划时间；
- 检测到保留天数或清理时间变化时重新计算下一次计划时间，不会因保存配置而立即清理；
- 配置服务每分钟刷新一次，保存配置后后台最多约一分钟使用新配置；
- 管理 API 复用 `groups.read` 查询权限和 `groups.update` 修改权限；
- 配置异常时回退到默认值并记录 warning；
- 多实例重复清理保持幂等，不新增分布式锁。

## 验收标准

- [x] 默认保留 3 天、每日 03:00 按服务器本地时间执行。
- [x] 管理员可以在安全检查日志页面修改保留天数和清理时间。
- [x] 修改配置后无需重启即可生效。
- [x] 同一天修改保留天数或清理时间后，不会被旧的当天完成标记阻止。
- [x] 非法保留天数或清理时间无法保存。
- [x] 清理按每批最多 1000 条执行并支持多批次。
- [x] 启动晚于清理时间时不执行补偿，等待下一次计划时间。
- [x] 配置在当天清理时间之后保存时不执行补偿，等待下一次计划时间。
- [x] 清理失败不会阻塞模型请求。
- [x] 无 `groups.update` 权限时不能修改配置。
- [x] 后端服务测试、路由编译、前端类型检查、相关测试和生产构建通过。

## 验证计划

- `cd backend; go test ./internal/service`
- `cd backend; go test ./internal/server/routes ./cmd/server`
- `cd frontend; pnpm run typecheck`
- `cd frontend; pnpm exec vitest run src/views/admin/__tests__/SecurityCheckLogsView.spec.ts src/router/__tests__/guards.spec.ts`
- `cd frontend; pnpm run build`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 设置服务 | `backend/internal/service/security_check_retention.go` | 实现默认值、范围校验、settings 持久化、配置回退和下一次清理时间计算。 |
| 清理调度 | `backend/internal/service/security_check_collection.go` | 从硬编码 3 天/每小时改为每日按服务器本地时间和动态保留天数清理；配置变化会重置当天完成标记。 |
| 管理 API | `backend/internal/handler/admin/group_security_check_handler.go`, `backend/internal/server/routes/admin.go` | 新增清理配置 GET/PUT，复用 `groups.read` / `groups.update`。 |
| 管理页面 | `frontend/src/views/admin/SecurityCheckLogsView.vue`, `frontend/src/api/admin/groups.ts` | 增加保留天数、清理时间、时区和下一次清理时间展示及保存。 |
| 后端测试 | `backend/internal/service/security_check_retention_test.go`, `cd backend; go test ./internal/service` | 通过，包含同日修改清理配置的回归测试。 |
| 路由/编译验证 | `cd backend; go test ./internal/server/routes ./cmd/server` | routes/cmd server 通过；admin handler 包存在既有无关的模型路由测试失败。 |
| 前端验证 | `cd frontend; pnpm run typecheck` | 通过。 |
| 前端相关测试 | `cd frontend; pnpm exec vitest run src/views/admin/__tests__/SecurityCheckLogsView.spec.ts src/router/__tests__/guards.spec.ts` | 通过，40 tests。 |
| 前端构建 | `cd frontend; pnpm run build` | 通过。 |

## 执行记录

- 当前完整 admin handler 测试仍受既有 `group_model_routing_test.go` 断言失败影响，与本 MVP 无关。
- 未修改数据库 schema；仅新增 settings key-value 配置。
- 根据验证反馈取消启动和同日保存配置后的补偿清理，调度严格等待下一次配置时间。
