# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: [model-routing-same-priority-load-balancing-implementation-plan.md](../../.plans/model-routing-same-priority-load-balancing-implementation-plan.md)
- Target effort per MVP: 约 1 个开发日（用户未指定，按技能默认假设）
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-17T05:00:00+08:00`
- Overall: `5/5 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成。
- `BLOCKED`：无法继续，不计入完成数。
- `DONE`：实现已完成、验收标准已确认、测试已运行且证据已记录。
- 每个 MVP 验证完成后，必须立即更新本文件，再开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-route-tier-validation.md](./MVP-001-route-tier-validation.md) | DONE | none | 1 个开发日 | 2026-08-17 | `MVP-001-route-tier-validation.md` |
| MVP-002 | [MVP-002-route-load-batch.md](./MVP-002-route-load-batch.md) | DONE | MVP-001 | 1 个开发日 | 2026-08-17 | `MVP-002-route-load-batch.md` |
| MVP-003 | [MVP-003-priority-pool-selection.md](./MVP-003-priority-pool-selection.md) | DONE | MVP-001、MVP-002 | 1.5 个开发日 | 2026-08-17 | `MVP-003-priority-pool-selection.md` |
| MVP-004 | [MVP-004-model-compatibility-and-retry.md](./MVP-004-model-compatibility-and-retry.md) | DONE | MVP-003 | 1 个开发日 | 2026-08-17 | `MVP-004-model-compatibility-and-retry.md` |
| MVP-005 | [MVP-005-admin-observability-regression.md](./MVP-005-admin-observability-regression.md) | DONE | MVP-003、MVP-004 | 1 个开发日 | 2026-08-17 | `MVP-005-admin-observability-regression.md` |

## 依赖关系

```text
MVP-001 → MVP-002 → MVP-003 → MVP-004 → MVP-005
                                      ↘──────── MVP-005
```

## 执行约束

- MVP-001 定义 priority 分层、重复账号和兼容字段规则。
- MVP-002 提供 route LoadRate 数据基础。
- MVP-003 使用前两个 MVP 完成 Gateway 核心调度切换。
- MVP-004 补齐上游模型兼容语义和失败重试路径。
- MVP-005 集中完成管理端提示、调度日志和跨层回归验证。
