# MVP-008：当前部门员工 Token 明细与消耗排行

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发日`
- Estimate rationale: 在既有部门概览上增加当前部门关联查询、平均值、员工明细接口、懒加载和员工图表，包含前后端契约及回归测试。
- Dependencies: `MVP-006, MVP-007`

## 预期成果

部门 Token 概览按查询时 `users.department` 归类，并支持点击部门后懒加载全部当前员工的 Token 消耗、部门内占比和消耗排行。

## 背景

部门报表不再读取聚合记录中的历史 `department`，而是使用激活的 `user_id + total_tokens + day` Projection，通过 `token_stat_aggregates.user_id = users.id` 取得当前部门。部门人数来自 users 关联结果，零 Token 用户也必须返回并计入平均值。管理员手动配置 Projection/指标，不创建 SQL 初始化配置。

## 范围内

- 将部门概览 API 改为 user_id 聚合关联当前 `users.department`。
- 返回部门 Token 总量、占比、用户数和平均 Token。
- 增加 `GET /api/v1/admin/usage/department-stats/users`。
- 员工接口返回当前部门所有未软删除用户，包括零 Token 用户。
- 返回员工 Token 数、部门内占比和分页信息。
- 部门列表行点击后立即打开员工明细弹窗，并在弹窗内懒加载员工数据。
- 增加员工 Token 横向柱状图，按 Token 倒序展示前 20 名。
- 增加后端、前端和维度配置回归测试。

## 范围外

- 创建 department 或 department + user_id Projection。
- 新增 Projection/指标初始化 SQL。
- 修改 `usage_logs`、`auth_identities` 或 `user_attribute_values`。
- 修改或回填历史 Token 聚合记录。
- LDAP 用户字段、LDAP 同步和 API Key 部门缓存逻辑改造。

## 实现说明

- `ProjectionAdminService` 使用一条 users 与 user_id 聚合的 LEFT JOIN 查询取得用户当前部门和 Token 汇总，再在内存中计算部门汇总。
- 员工明细接口复用相同数据源，按当前部门过滤，排序后分页。
- `department` 从新的 Projection 可配置列表中下架，但内部保留兼容旧事件、旧 Projection 和旧聚合数据。
- 部门概览和员工明细保留 `mysql_eventual` 一致性信息。

## 验收标准

- [x] 只需要配置 `user_id + total_tokens + day` Projection。
- [x] 部门概览按查询时 `users.department` 归类。
- [x] 零 Token 用户计入部门人数和平均 Token。
- [x] 部门列表显示 Token、占比、用户数和平均 Token。
- [x] 点击部门后立即打开员工明细弹窗，并请求员工明细接口。
- [x] 员工明细包含当前部门全部用户和零 Token 用户。
- [x] 员工列表返回部门内 Token 占比并支持分页。
- [x] 员工柱状图按 Token 倒序展示高消耗用户。
- [x] 不创建 Projection/指标初始化 SQL，且未修改 usage_logs。
- [x] 相关 Go 测试、前端类型检查和 Vitest 测试通过。

## 验证计划

- `backend: go test ./internal/service/tokenstat ./internal/handler/admin ./internal/server/routes -run 'Department|DynamicTokenStat|TokenHandlerEndpoints' -count=1`
- `frontend: pnpm run typecheck`
- `frontend: pnpm exec vitest run src/components/charts/__tests__/DepartmentDistributionChart.spec.ts src/components/charts/__tests__/DepartmentUserUsageChart.spec.ts src/views/admin/__tests__/UsageView.spec.ts`
- 使用包含有 Token、零 Token、空部门和部门变更用户的测试数据验证接口结果。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端测试 | `backend: go test ./internal/service/tokenstat ./internal/handler/admin ./internal/server/routes -run 'Department|DynamicTokenStat|TokenHandlerEndpoints' -count=1` | 通过；验证当前部门聚合、零 Token 用户、员工接口和配置维度列表 |
| 前端类型 | `frontend: pnpm run typecheck` | 通过 |
| 前端测试 | `frontend: pnpm exec vitest run src/components/charts/__tests__/DepartmentDistributionChart.spec.ts src/components/charts/__tests__/DepartmentUserUsageChart.spec.ts src/views/admin/__tests__/UsageView.spec.ts` | 通过，12 tests |
| 实现检查 | `backend/internal/service/tokenstat/query.go` | user_id Projection 关联 users.department，内存聚合部门/员工并计算平均值和占比 |
| 实现检查 | `backend/internal/handler/admin/usage_handler.go`, `backend/internal/server/routes/admin.go` | 新增部门员工明细 API 和受保护路由 |
| 实现检查 | `frontend/src/views/admin/UsageView.vue`, `frontend/src/components/charts/DepartmentUserUsageChart.vue` | 部门点击懒加载、员工列表和排行图表 |
| 配置检查 | `backend/internal/handler/admin/dynamic_token_statistics_handler.go` | department 不出现在新 Projection 可配置维度列表 |
| SQL 边界 | `backend/sqlArchiving/` | 未新增 Projection/指标初始化 SQL，未修改 usage_logs |

## 执行记录

- 2026-09-04：按批准的补充方案在旧部门概览上增量实现员工明细；没有采用之前被否决的 department + user_id Projection。
- 2026-09-04：部门统计切换为 user_id Projection 与当前 users.department 关联，旧 department 字段仅保留兼容。
