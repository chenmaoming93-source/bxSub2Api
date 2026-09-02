# MVP-008：完成安全检查记录查询、详情和生命周期管理

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发者日`
- Estimate rationale: 包含后端分页/详情 API、按时间批量清理、熔断状态/恢复和前端大字段详情展示。
- Dependencies: `MVP-006, MVP-007`

## 预期成果

管理员可以分页查询安全检查记录、查看完整返回体和保存的请求体，并能查看采集熔断状态和手动恢复；后台按全局保留期限批量清理过期记录。

## 背景

安全记录保存在独立的 `security_check_logs` 表中。请求体可能很大，列表接口不能加载大字段，详情接口才读取并解压。默认保留期限为 3 天。

## 范围内

- 安全检查记录分页 API；
- 按时间、分组、决策、检查状态筛选；
- 记录详情 API；
- 请求体解压和截断状态展示；
- 完整 SingGuard 返回体展示；
- 命中规则、配置版本和耗时展示；
- 定时按 `created_at` 分批清理过期数据；
- 采集熔断状态查询；
- 手动重新开启采集；
- 页面权限、错误和空状态处理。

## 范围外

- 导出全部原始数据；
- 跨数据库归档；
- 对象存储迁移；
- 自动脱敏；
- 非管理员查询接口。

## 实现说明

- 列表接口限制最大 page size，不返回 `request_body` 和完整返回体；
- 详情接口按 ID 查询并解压请求体；
- 过期清理每批最多删除 1000 条，失败时停止本轮并等待下一次任务；
- 清理条件使用 `created_at < now - retention_days`；
- 采集恢复接口清理共享熔断状态并发布本地状态恢复通知；
- 页面明确标示请求体是否被截断。

## 验收标准

- [x] 管理员可以分页查看安全检查记录，并使用时间、分组、决策和状态筛选。
- [x] 列表接口不加载大字段，详情页能够查看请求体、完整 SingGuard 返回体和判定信息。
- [x] 被截断的请求体有明确提示，并展示原始/保存大小。
- [x] 默认保留期限为 3 天，过期清理使用时间索引并分批执行。
- [x] 采集熔断状态可以查询，手动恢复后采集 worker 能重新工作。
- [x] 记录查询 API、清理任务和前端页面构建通过。

## 验证计划

- `cd backend; go test ./internal/handler/... ./internal/service/... ./internal/repository/...`
- `cd frontend; pnpm run typecheck`
- `cd frontend; pnpm run test:run`
- 手工验证列表分页、筛选、详情大字段、截断提示、保留期限和恢复采集操作。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 查询/详情接口 | `backend/internal/repository/security_check_log_repo.go`, `backend/internal/handler/admin/group_security_check_handler.go` | 支持分页、时间/分组/决策/状态筛选；列表显式排除大字段，详情解压请求体并返回完整响应。 |
| 生命周期/熔断 | `backend/internal/service/security_check_collection.go` | 默认 3 天按 `created_at` 索引分批删除；状态查询和手动恢复接口可用。 |
| 路由 | `backend/internal/server/routes/admin.go` | 新增日志列表、详情、采集状态和恢复路由，分别复用 groups read/update 权限。 |
| 管理页面 | `frontend/src/components/admin/group/SecurityCheckLogsModal.vue`, `frontend/src/views/admin/GroupsView.vue` | 支持筛选、分页、详情大字段、截断提示、熔断状态和恢复操作。 |
| 后端验证 | `cd backend; go test ./internal/service ./internal/repository ./cmd/server` | 通过。 |
| 前端验证 | `cd frontend; pnpm run build` | 通过（含 vue-tsc 类型检查）。 |

## 执行记录

- MVP-008 首版清理 worker 每小时探测并按每批最多 1000 条删除；可配置每日调度和保留天数由 MVP-010 补充实现。
- MVP-010 完成后，保留期限和每日清理时间通过安全检查日志页面持久化配置。
- admin handler 全量测试仍受既有 `group_model_routing_test.go` 的不相关断言失败影响。
- 独立日志页面、菜单位置、状态列和风险维度可读化由后续 MVP-009 补充实现。
