# MVP-002：候选分时段并发配置管理 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 复用现有分组路由管理接口和权限体系，增加查询、整体保存及输入错误响应。
- Dependencies: `MVP-001`

## 预期成果

管理员可以通过分组路由候选管理 API 查询和整体替换某个候选的每日分时段配置，服务端统一校验时间格式、分钟边界、重叠和 `NULL/unlimited` 表示。

## 背景

现有管理路由位于 `backend/internal/server/routes/admin.go`，已有分组模型路由和候选并发接口。本 MVP 在不改变旧接口行为的前提下增加独立的分时段配置接口，并沿用分组读/更新权限。

## 范围内

- 增加候选分时段配置查询接口。
- 增加候选分时段配置整体替换接口，保存请求以候选为原子边界。
- 增加参数校验和明确的 4xx 错误：时间不是 `HH:mm`、`24:00` 使用错误、区间为空/反向/重叠、并发值小于 1。
- 外部 JSON 使用正整数或 `null` 表示并发上限；`null` 持久化为 `unlimited` 语义。
- 接入分组管理权限，并覆盖 handler/service/repository 的接口测试。

## 范围外

- 不在请求处理路径读取该配置或判断当前时间。
- 不负责 Redis 当前值刷新和立即刷新。
- 不修改旧的候选最大并发接口及 `max_concurrency` 字段。

## 实现说明

- 路由命名沿用现有 admin group 路由风格，具体 URL 可在实现时按仓库约定落地。
- 保存请求以 `route_alias + account_id` 标识候选，先校验完整列表，再调用 MVP-001 的整体替换仓储方法。
- 响应包含 `start`、`end`、`max_concurrency`；未配置时返回空数组，并可带“使用默认并发值”的状态信息。

## 验收标准

- [x] 有权限用户可查询、整体替换和清空某个候选的分时段配置。
- [x] 越权请求被拒绝；旧的候选默认并发 API 回归测试保持通过。
- [x] 重叠、非法边界、非法并发值和合法 `null` 的响应符合约定。
- [x] 保存失败时不会部分写入，且接口测试覆盖空配置和不完整全天覆盖。

## 验证计划

- `go test ./internal/handler/... ./internal/server/... ./internal/repository/...`（在 `backend` 目录执行，按实际包调整）。
- 使用现有 admin API 测试工具或手工请求验证读/写/清空及权限响应。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 服务/API | `backend/internal/service/model_route_concurrency_schedule_admin.go` | 已接入分组存在性检查、候选分时段查询和整体替换，不扩展旧 `AdminService` 接口契约 |
| 路由/权限 | `backend/internal/server/routes/admin.go` | 新增 GET/PUT `/:id/model-route-references/concurrency-schedules`，分别沿用 `PermissionGroupsRead/Update` |
| Handler | `backend/internal/handler/admin/group_handler.go` | 支持 `HH:mm`、结束 `24:00`、数值/null，并将响应格式化为 `start/end` |
| 测试 | `go test ./internal/handler/admin -run 'TestGroupConcurrencyScheduleHandler' -count=1` | 通过 |
| 回归 | `go test ./internal/service ./internal/repository -run 'ModelRoute|ConcurrencySchedule|Concurrency' -count=1` | 通过；既有并发相关服务/仓储测试通过 |
| 路由编译 | `go test ./internal/server/routes` | 通过 |

## 执行记录

- 2026-08-18：采用 `GET` query 参数 `route_alias/account_id` 与 `PUT` body 整体替换，避免将路由别名编码进 URL。
- 2026-08-18：完整 `go test ./internal/handler/admin` 受工作区已有的 `TestGroupRequestsAcceptLegacyAndCandidateModelRouting` 失败影响；该失败发生在本次改动之外的模型路由校验，新增 handler 定向测试及相关 service/repository 回归测试均通过。
