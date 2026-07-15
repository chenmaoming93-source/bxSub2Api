# MVP-007：打通用户模型消耗报表的实时混合查询

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 围绕 user_id+model 和删除用户语义完成独立垂直链路，约 40 分钟。
- Dependencies: `MVP-001, MVP-002, MVP-003, MVP-004`

## 预期成果

用户模型 Token 消耗报表对今天使用 Redis 实时累计，同时保持用户信息、删除状态、筛选、汇总和分页正确。

## 背景

入口为 `/admin/token-usage/users`，现有查询支持 `user_id`、`model` 与 `include_deleted`。

## 范围内

- 实现用户模型报表的日期分流与混合查询。
- 按 user_id、model 业务键合并并关联用户元数据。
- 保持 `include_deleted`、用户筛选和模型筛选语义。
- 基于最终集合计算 summary、usage_rate、status、排序、total 和分页。
- 覆盖已删除用户、Redis-only、MySQL-only 和不存在组合。

## 范围外

- 不改用户模型限额配置页面。
- 不修改用户删除逻辑。

## 实现说明

- Redis 数据不得绕过 `include_deleted` 过滤。
- API 响应字段保持不变。

## 验收标准

- [x] 历史查询继续走 MySQL 快速路径。
- [x] 当天 Redis 实时值、汇总和状态正确。
- [x] 删除用户包含/排除行为与改造前一致。
- [x] 不存在 user/model 组合不触发额外 MySQL 点查或 Redis 写入。

## 验证计划

- `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*User.*TokenUsage'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/token_usage_report_service.go` | 用户模型日期分流、user_id+model 合并、筛选、删除用户语义、合并后状态/汇总/多字段排序/分页。 |
| 修复 | `backend/internal/service/current_token_usage_reader.go`、`backend/internal/repository/token_usage_read_repair.go` | 共享 repairer 端口增加用户模型原子修复。 |
| 测试 | `backend/internal/service/token_usage_user_hybrid_test.go` | 覆盖历史路径、实时覆盖、Redis-only、删除用户包含/排除、状态汇总和不存在组合。 |
| 聚焦验证 | `cd backend && go test ./internal/service ./internal/handler/admin -run 'Test.*User.*TokenUsage'` | 通过。 |
| 包回归 | `cd backend && go test ./internal/service ./internal/repository ./cmd/server` | 通过。 |

## 执行记录

2026-07-15 完成。Redis 数据进入最终集合后仍应用 user_id、model 与 `include_deleted` 过滤；MySQL 已知删除用户不会绕过排除规则。不存在组合只执行设计内集合读取，不触发修复。
