# MVP-005：Token 统计 user_id 维度与部门报表数据源

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发日`
- Estimate rationale: 需要扩展维度注册、Token 统计聚合冗余字段、查询映射和迁移，属于统计核心切片。
- Dependencies: `MVP-002`

## 预期成果

现有可配置 Token 统计使用 `user_id` 作为唯一报表维度，将模型调用按用户累计到 MySQL 聚合表；部门报表后续通过当前 users.department 归类。

## 背景

现有 `tokenstat` 使用 Dimension registry、Projection、Redis accumulator 和 `token_stat_aggregates`。新的部门报表只要求事件显式提供 user_id，并在查询时关联 users 表取得当前 department。`usage_logs` 不得修改。

## 范围内

- 使用现有 `DimensionUserID = "user_id"` 作为部门报表唯一配置维度，指标为 `total_tokens`、周期为 day。
- 确认用户 Token 事件包含 user_id，继续保持异步事件和现有 Projection 行为。
- 保留 `token_stat_aggregates.user_id` 及通用 dimension_values 作为报表数据源。
- 将 `department` 从管理员新 Projection 可选列表中下架；内部定义、事件字段、冗余字段和旧 Projection 兼容能力保留。
- 不创建 department 或 department + user_id Projection，不增加 Projection/指标初始化 SQL。
- 增加 user_id 聚合查询、当前 users.department 关联、零 Token 用户和多用户测试。

## 范围外

- 部门同步接口和用户页面按钮。
- `/admin/usage` 前端交互和员工图表。
- 新建或初始化可配置 Projection/指标的 SQL。
- 使用聚合记录中的历史 department 字段作为新报表归类依据。
- 修改 `usage_logs`。

## 实现说明

- `AsyncPipeline` 继续使用现有通用维度遍历逻辑，不为部门复制统计流程。
- 事件中的 user_id 由 MVP-002 提供；部门报表不依赖事件中的历史 department。
- 具体 user_id Projection 由管理员在界面创建、发布和激活，服务端不自动创建。
- Projection 的发布/激活继续使用现有状态机和配置广播机制。
- `department` 冗余字段和旧 Projection 保留兼容，但不作为新部门报表数据源。

## 验收标准

- [x] 管理端可看到并选择 `user_id` 维度，`department` 不出现在新 Projection 可选列表。
- [x] user_id Projection 能接收事件并写入用户聚合。
- [x] 聚合记录的 user_id JSON 和冗余字段一致。
- [x] 现有 Projection 的统计结果和运行方式不变。
- [x] 不创建 Projection/指标初始化 SQL。
- [x] `usage_logs` 表结构未修改。
- [x] 相关 Go 测试通过。

## 验证计划

- `go test ./internal/service/tokenstat/... ./internal/repository/tokenstat/...`
- `go test ./internal/repository/...`
- 使用测试 Projection 验证 user_id 事件从 Redis 到 MySQL 的链路。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `backend: go test ./internal/service/tokenstat -count=1` | 通过 |
| 测试 | `backend: go test ./internal/repository/tokenstat -run 'TestUpsertAggregateUsesMonotonicSourceVersionSQL|TestDynamicTokenSyncPersistsAbsoluteSnapshotAndPreservesConcurrentDirty' -count=1` | 通过 |
| 检查 | `backend/ent/schema/token_stat_aggregate.go`, `backend/internal/repository/tokenstat/{repository,sync_engine}.go` | 保留 JSON/冗余 department 兼容能力；新部门报表使用 user_id 聚合 |
| 检查 | `backend/sqlArchiving/174_add_token_stat_aggregate_department.sql` | 仅增加聚合字段和 department 索引；Projection/指标由管理员后续在界面配置，未修改 usage_logs |
| 回归限制 | `backend: go test ./internal/repository/tokenstat -count=1` | 固定 2026-07 测试日期在当前 2026-09 运行时已过 Redis orphan TTL，既有 Redis 过期测试失败；非本改动逻辑 |

## 执行记录

- 未修改或回填历史聚合记录；新报表查询时通过 users.department 重新归类。

