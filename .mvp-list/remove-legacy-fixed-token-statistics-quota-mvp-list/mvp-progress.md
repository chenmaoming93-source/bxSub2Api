# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/remove-legacy-fixed-token-statistics-and-quota-implementation-plan.md`
- Target effort per MVP: `假设每个 MVP 为一个专注开发日`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-31T01:16:00+08:00`
- Overall: `7/7 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-legacy-asset-boundary.md](./MVP-001-legacy-asset-boundary.md) | DONE | none | 0.5 开发日 | 2026-07-30 | 旧专属、共享保留、新体系保护与 Redis/MySQL/API/前端边界完成分类 |
| MVP-002 | [MVP-002-remove-runtime-accounting-quota.md](./MVP-002-remove-runtime-accounting-quota.md) | DONE | MVP-001 | 1.5 开发日 | 2026-07-30 | 旧请求统计/限额、同步调度与 Wire 依赖已移除；service/repository/server 测试通过 |
| MVP-003 | [MVP-003-remove-backend-admin-api.md](./MVP-003-remove-backend-admin-api.md) | DONE | MVP-001, MVP-002 | 1 开发日 | 2026-07-30 | 旧管理路由、专属后端实现和 Wire 依赖已移除；新管理 API 与通用权限保留 |
| MVP-004 | [MVP-004-remove-legacy-frontend.md](./MVP-004-remove-legacy-frontend.md) | DONE | MVP-001, MVP-003 | 1 开发日 | 2026-07-30 | 旧页面/API/入口已删除；typecheck、build 与 10 个聚焦测试通过 |
| MVP-005 | [MVP-005-remove-ent-and-config.md](./MVP-005-remove-ent-and-config.md) | DONE | MVP-002, MVP-003 | 1.5 开发日 | 2026-07-31 | 六个旧 Ent 实体与旧配置删除；五个新实体保留；Ent/config/service/server 测试通过 |
| MVP-006 | [MVP-006-deliver-drop-sql-redis-audit.md](./MVP-006-deliver-drop-sql-redis-audit.md) | DONE | MVP-005 | 0.5 开发日 | 2026-07-31 | 交付 166 人工删表 SQL 与只读 Redis TTL 审计说明；未执行 SQL |
| MVP-007 | [MVP-007-final-isolation-regression.md](./MVP-007-final-isolation-regression.md) | DONE | MVP-004, MVP-005, MVP-006 | 1.5 开发日 | 2026-07-31 | 后端全量、前端专项/类型/构建与隔离扫描通过；全量前端既有 6 项认证跳转失败已记录 |

## 依赖说明

- 关键路径：MVP-001 → MVP-002 → MVP-003 → MVP-005 → MVP-006 → MVP-007。
- 前端清理在旧后端 API 边界确定后执行，并与 Ent/配置清理汇合到最终验收。
- 删除 SQL 只作为工程交付物，由用户在生产环境人工执行。

## 规划假设

- 新可配置体系已经完成并通过验收，是整个移除工作的保护基线。
- 不迁移旧历史数据、旧限额配置，不保留兼容 API。
- Redis 不批量删除，只交付只读 TTL 审计方法和异常 key 人工处理建议。
- 用户已明确 SQL 不需要实际运行测试；执行者只做静态方言、表名和保护清单验证。
