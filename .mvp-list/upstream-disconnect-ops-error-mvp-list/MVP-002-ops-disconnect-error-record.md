# MVP-002：写入上游服务断开的 Ops 错误记录

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个聚焦开发日`
- Estimate rationale: 复用现有 `OpsService` 和 `OpsInsertErrorLogInput`，补充专用分类、可靠提交、去重保护及后端单元测试。
- Dependencies: `none`

## 预期成果

当收到明确的调用方断开事件时，可以脱离已取消的入站 Context，向 `ops_error_logs` 写入一条 `client_disconnected` 记录；不调用 usage 记录、扣费或 Token 统计逻辑。

## 背景

`/admin/usage` 的“错误请求”菜单通过 `/api/v1/admin/ops/errors` 查询 `ops_error_logs`。现有 `OpsInsertErrorLogInput` 已提供请求、账号、模型、错误分类和时延字段，`OpsService` 已有监控开关、清理和写入能力。

## 范围内

- 定义 `client_disconnected` 的标准错误分类；
- 填充 `error_phase=network`、`error_owner=client`、`error_source=client_request`、`status_code=499`；
- 保留原请求的 `request_type`，不把它用于错误分类；
- 复用现有请求、用户、API Key、账号、模型、端点和时延字段；
- 使用脱离入站请求的短超时 Context 写入；
- 队列满时提供同步 fallback；
- 请求生命周期内只写一次；
- 保持 Ops monitoring 开关语义；
- 验证该路径不写 `usage_logs`、不扣费、不更新 Token/配额统计。

## 范围外

- 不新增数据库表；**不得**恢复 `is_retryable`：该列属于已下线的 Ops 重试功能，迁移 `136_remove_ops_retry_replay.sql` 已随 `ops_retry_attempts` 一并删除，生产库中不存在；
- 不改变普通 Ops 错误类型的分类规则；
- 不修改 `/admin/usage` 前端显示；
- 不处理具体协议如何检测断开；
- 不修改模型路由 failover。

## 实现说明

- 重点检查 `backend/internal/service/ops_service.go`、`backend/internal/service/ops_port.go`、`backend/internal/repository/ops_repo.go`；
- 复用现有 sanitize、截断和 monitoring gate；
- 设计一个集中调用入口，避免不同 handler 重复构造错误记录；
- 使用请求级 once/标记避免同一断开事件在多个层级重复写入；
- 对队列提交、同步 fallback、数据库失败分别记录可观测日志或计数。

## 验收标准

- [x] 生成的记录 `error_type` 为 `client_disconnected`；
- [x] 生成的记录 `status_code` 为 499，且 owner/source/phase 值符合约定；
- [x] 生成的记录不写入任何已下线字段（`is_retryable`、`retry_count` 等），INSERT 列与线上 schema 完全一致；
- [x] `request_type` 仍准确表示 sync/stream/ws_v2；
- [x] Ops monitoring 关闭时不违反现有跳过行为；
- [x] 客户端 Context 已取消时，记录写入仍使用独立短超时 Context；
- [x] 同一个请求生命周期最多写入一条断开记录；
- [x] 断开记录不会调用 usage、计费、Token 或配额更新服务；
- [x] 队列满时同步 fallback 测试通过。

## 验证计划

- `go test ./internal/service/... ./internal/repository/... ./internal/handler/...`
- 重点检查 `ops_error_logger_test.go`、`ops_error_logger_attribution_test.go` 和 Ops repository 测试；
- 使用 stub/mock 验证 Ops 写入调用以及 usage repository 未被调用。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 代码 | `backend/internal/service/ops_service.go`、`backend/internal/service/ops_port.go` | 新增 `client_disconnected` 标准分类，清理上游错误体和统计相关字段；**不再**涉及 `is_retryable`。 |
| 代码 | `backend/internal/handler/ops_error_logger.go` | 新增请求级去重、独立超时 Context、关键错误队列满时同步 fallback 和 fallback 计数。 |
| 数据库 | 无 | 纠正后不新增任何 DDL；曾新增的 `backend/migrations/149_ops_error_logs_add_is_retryable.sql` 已删除（该列线上不存在，且违反项目规约）。 |
| 持久化 | `backend/internal/repository/ops_repo.go` | INSERT 列/占位符/参数严格对应线上 `ops_error_logs` schema；`TestOpsErrorLogInsertColumnPlaceholderAndArgCountsMatch` 锁定三者计数一致。 |
| 测试 | `go test ./internal/handler -run 'Test(RecordClientDisconnectedOpsError|EnqueueCriticalOpsErrorLog)' -count=1` | 通过。 |
| 测试 | `go test ./internal/service -run 'Test(MarkClientDisconnectedErrorLog|RecordClientDisconnectedError)' -count=1` | 通过。 |
| 测试 | `go test ./internal/repository -run 'TestOpsRepositoryInsertErrorLogPersistsClientDisconnectShape|TestOpsErrorLogInsert' -count=1` | 通过，包含 sqlmock 持久化形状验证。 |
| 回归 | `go test ./internal/handler -run 'Test(EnqueueOpsErrorLog|OpsErrorLogger|MarkClientDisconnected)' -count=1` | 通过。 |
| 回归 | `go test ./internal/service/... ./internal/repository/... ./internal/handler/...` | 本次相关包代码可编译；全量命令仍有既有的 `TestGroupRequestsAcceptLegacyAndCandidateModelRouting` 失败，与本 MVP 无关，已保留为后续回归记录。 |

## 执行记录

- 2026-09-16：普通 Ops 错误仍保持队列满即丢弃；仅 `client_disconnected` 使用同步 fallback，避免该关键事件被丢失。
- 2026-09-16：专项测试、sqlmock 持久化验证和既有 Ops logger 回归通过；全量相关包唯一已观察到的失败为既有模型路由测试。
- 纠正（本次）：最初把批准方案字段表中的 `is_retryable` 误当作落库需求，新增 `migrations/149_...` 并把它写进 INSERT。但该列早已被迁移 136 随重试功能删除，**线上 `ops_error_logs` 不存在该列**，导致所有 Ops 写入以 MySQL 1054 静默失败（记录全丢）。现已：从 INSERT 列/占位符/参数、`OpsInsertErrorLogInput`、`MarkClientDisconnectedErrorLog` 中彻底移除 `is_retryable`；删除该无效迁移文件；把原先"断言存在该列"的测试反转为"禁止出现该列"，并新增列/占位符/参数计数一致性测试。教训：字段表只是逻辑记录内容，任何落库字段都必须先核对线上 schema。

