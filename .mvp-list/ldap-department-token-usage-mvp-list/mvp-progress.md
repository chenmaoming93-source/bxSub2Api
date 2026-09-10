# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `../../.plans/ldap-department-token-usage-implementation-plan.md`
- Target effort per MVP: 假设每个 MVP 约 1 个聚焦开发日；允许 0.5–1.5 个开发日，以保证每个切片可独立实现和验证。
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-09-04T17:26:28+08:00`
- Overall: `8/8 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-user-department-and-ldap-login.md](./MVP-001-user-department-and-ldap-login.md) | DONE | none | 1.5 个开发日 | 2026-09-03T20:47:29+08:00 | [evidence](./MVP-001-user-department-and-ldap-login.md#完成证据) |
| MVP-002 | [MVP-002-auth-cache-and-token-event-dimension.md](./MVP-002-auth-cache-and-token-event-dimension.md) | DONE | MVP-001 | 1 个开发日 | 2026-09-03T20:54:16+08:00 | [evidence](./MVP-002-auth-cache-and-token-event-dimension.md#完成证据) |
| MVP-003 | [MVP-003-ldap-full-sync-backend.md](./MVP-003-ldap-full-sync-backend.md) | DONE | MVP-001, MVP-002 | 1.5 个开发日 | 2026-09-03T21:06:15+08:00 | [evidence](./MVP-003-ldap-full-sync-backend.md#完成证据) |
| MVP-004 | [MVP-004-ldap-sync-admin-button.md](./MVP-004-ldap-sync-admin-button.md) | DONE | MVP-003 | 0.75 个开发日 | 2026-09-03T21:16:35+08:00 | [evidence](./MVP-004-ldap-sync-admin-button.md#完成证据) |
| MVP-005 | [MVP-005-tokenstat-department-projection.md](./MVP-005-tokenstat-department-projection.md) | DONE | MVP-002 | 1.5 个开发日 | 2026-09-03T21:27:03+08:00 | [evidence](./MVP-005-tokenstat-department-projection.md#完成证据) |
| MVP-006 | [MVP-006-department-stats-api.md](./MVP-006-department-stats-api.md) | DONE | MVP-005 | 0.75 个开发日 | 2026-09-04T17:26:28+08:00 | [evidence](./MVP-006-department-stats-api.md#完成证据) |
| MVP-007 | [MVP-007-department-usage-chart-and-regression.md](./MVP-007-department-usage-chart-and-regression.md) | DONE | MVP-004, MVP-006 | 1.5 个开发日 | 2026-09-03T22:01:01+08:00 | [evidence](./MVP-007-department-usage-chart-and-regression.md#完成证据) |
| MVP-008 | [MVP-008-department-user-usage-details.md](./MVP-008-department-user-usage-details.md) | DONE | MVP-006, MVP-007 | 1.5 个开发日 | 2026-09-04T17:26:28+08:00 | [evidence](./MVP-008-department-user-usage-details.md#完成证据) |

## 依赖说明

- MVP-001 → MVP-002 → MVP-003 → MVP-004 是 LDAP 用户资料与管理员同步主链路。
- MVP-002 → MVP-005 → MVP-006 是 Token 事件、user_id 聚合和当前部门报表主链路。
- MVP-004 与 MVP-006 完成后，MVP-007 集成部门概览；MVP-008 在其上增加部门员工明细和懒加载图表。

## 规划假设

- 同步接口采用同步批处理模式，使用有限并发并返回汇总结果。
- `users.department` 和 Token 统计自身的聚合字段已允许迁移；本次补充不新增 SQL 初始化配置，`usage_logs`、`auth_identities`、`user_attribute_values` 不修改。
- 现有 Go、Ent、Vitest 和前端构建工具可用于验证；若环境缺少服务依赖，必须在对应 MVP 中记录限制，不能据此标记 DONE。
