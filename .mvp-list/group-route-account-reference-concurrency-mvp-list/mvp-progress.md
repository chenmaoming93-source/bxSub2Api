# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `../../.plans/group-route-account-reference-and-concurrency-implementation-plan.md`
- Target effort per MVP: 默认每个 MVP 约一个开发日
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-11T00:00:00+08:00`
- Overall: `6/6 (100%)`

## 状态规则

- `PENDING`：尚未验证完成
- `BLOCKED`：无法继续，且不计入完成
- `DONE`：已实现、测试通过并记录证据

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-relation-table-sync.md](./MVP-001-relation-table-sync.md) | DONE | none | 1 day | 2026-08-10 | [MVP-001-relation-table-sync.md](./MVP-001-relation-table-sync.md) |
| MVP-002 | [MVP-002-rebuild-references-api.md](./MVP-002-rebuild-references-api.md) | DONE | MVP-001 | 1 day | 2026-08-10 | [MVP-002-rebuild-references-api.md](./MVP-002-rebuild-references-api.md) |
| MVP-003 | [MVP-003-account-reference-ui.md](./MVP-003-account-reference-ui.md) | DONE | MVP-001 | 1 day | 2026-08-10 | [MVP-003-account-reference-ui.md](./MVP-003-account-reference-ui.md) |
| MVP-004 | [MVP-004-concurrency-config-ui.md](./MVP-004-concurrency-config-ui.md) | DONE | MVP-001 | 1 day | 2026-08-10 | [MVP-004-concurrency-config-ui.md](./MVP-004-concurrency-config-ui.md) |
| MVP-005 | [MVP-005-redis-concurrency-control.md](./MVP-005-redis-concurrency-control.md) | DONE | MVP-001, MVP-004 | 1.5 days | 2026-08-11 | [MVP-005-redis-concurrency-control.md](./MVP-005-redis-concurrency-control.md) |
| MVP-006 | [MVP-006-integration-verification.md](./MVP-006-integration-verification.md) | DONE | MVP-002, MVP-003, MVP-005 | 1 day | 2026-08-11 | [MVP-006-integration-verification.md](./MVP-006-integration-verification.md) |

## 依赖说明

MVP-003 与 MVP-004 在 MVP-001 完成后可并行；MVP-005 依赖并发配置数据结构；MVP-006 负责全链路验证、迁移校验和运维证据。

## 规划假设

- 使用现有 MySQL/GoldenDB 迁移体系、Go 后端和现有前端组件。
- 具体测试命令以仓库现有脚本和项目说明为准；若运行环境缺少工具，必须记录限制，不得伪造完成证据。
