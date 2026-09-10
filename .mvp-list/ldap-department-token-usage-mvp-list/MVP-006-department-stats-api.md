# MVP-006：按当前部门聚合的 Token 用量查询 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.75 个开发日`
- Estimate rationale: 复用现有 ProjectionAdminService 和 MySQL 聚合查询，只增加部门汇总契约及路由。
- Dependencies: `MVP-005`

## 预期成果

管理端可以按日期从 Token 统计 MySQL 聚合数据中获取各部门实际 Token 用量。

## 背景

现有 admin usage 路由和 handler 位于 `backend/internal/handler/admin/usage_handler.go`。部门报表使用激活的 user_id Token Projection，并通过 user_id 关联当前 `users.department`；不访问 `usage_logs`，也不读取聚合记录中的历史 department 字段。

## 范围内

- 新增 `GET /api/v1/admin/usage/department-stats`。
- 支持 `start_date`、`end_date`、`timezone`。
- 使用按天半开区间，结束日期包含整天。
- 校验日期范围和 Projection 激活状态。
- 通过 Token 统计服务执行 MySQL users/aggregate 关联查询和内存部门聚合。
- 返回部门、total_tokens、percentage、user_count、average_tokens、总量、complete、last_synced_at、consistency。
- 新增部门员工明细接口，返回当前部门全部用户（包括零 Token 用户）、部门内占比和分页数据。
- 按 Token 数倒序返回部门和员工。
- 增加 handler、service 和契约测试。

## 范围外

- 部门图表组件。
- 修改现有 usage 日志 API。
- Redis 实时查询。
- 修改用户部门或回写历史 Token 聚合记录。

## 实现说明

- 在 `ProjectionAdminService` 使用一条 MySQL users/aggregate 关联查询取得每个用户当前部门和 user_id Token 汇总，再在内存中聚合部门。
- 只使用激活的 user_id + total_tokens + day Projection。
- 部门人数来自 users 关联结果，LEFT JOIN 保证零 Token 用户计入平均值。
- Projection 未配置或未激活时返回明确的配置状态/错误。
- 保留 MySQL eventual consistency 信息，并支持部门员工分页明细接口。
- 日期范围遵循现有 Token 统计查询限制。

## 验收标准

- [x] 正确解析开始日期、结束日期和 timezone。
- [x] 结束日期整天被包含，且日期范围边界正确。
- [x] 接口使用 user_id Projection 并关联当前 users.department，不查询 usage_logs。
- [x] 部门汇总数、人数、平均值和占比计算正确，按 Token 降序返回。
- [x] `未设置` 部门和零 Token 用户正常返回。
- [x] 员工明细接口支持部门筛选、分页和员工占比。
- [x] Projection 缺失、未激活、日期非法和无权限均有明确响应。
- [x] 后端 API 测试通过。

## 验证计划

- `go test ./internal/handler/admin/... ./internal/service/tokenstat/...`
- 使用 MySQL/Ent 测试数据验证跨多日汇总和日期边界。
- 代码审查确认查询没有读取 `usage_logs`。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `backend: go test ./internal/service/tokenstat -run 'DynamicTokenStatQuery' -count=1` | 通过，验证 user_id 数据、当前部门、`未设置`、零 Token 用户、平均值和员工聚合 |
| 测试 | `backend: go test ./internal/handler/admin -run 'DepartmentStats|DepartmentUsers' -count=1` | 通过，验证日期边界、timezone、部门汇总和员工 HTTP 响应 |
| 编译 | `backend: go test ./internal/server/routes ./cmd/server -run 'Route|NonExistent' -count=1` | 通过，路由和 Wire 注入可编译 |
| 检查 | `backend/internal/service/tokenstat/query.go`, `backend/internal/handler/admin/usage_handler.go` | 使用 user_id Projection 关联 users.department，内存聚合部门和员工，未读取 usage_logs |

## 执行记录

- `department-stats` 使用激活的 user_id + total_tokens 日 Projection，按当前 users.department 聚合并返回 `mysql_eventual` 一致性标识；`department-stats/users` 提供懒加载员工明细。

