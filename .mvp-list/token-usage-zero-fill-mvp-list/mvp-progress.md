# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `../token-usage-zero-fill-implementation-plan.md`
- Target effort per MVP: `1 个专注开发日（未指定时采用的规划假设）`
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-07-07T15:10:30+08:00`
- Overall: `4/4 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-global-model-zero-fill.md](./MVP-001-global-model-zero-fill.md) | DONE | none | 1 个专注开发日 | 2026-07-07T15:00:31+08:00 | [实现与测试证据](./MVP-001-global-model-zero-fill.md#完成证据) |
| MVP-002 | [MVP-002-user-model-zero-fill.md](./MVP-002-user-model-zero-fill.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-07T15:04:37+08:00 | [实现与测试证据](./MVP-002-user-model-zero-fill.md#完成证据) |
| MVP-003 | [MVP-003-route-candidate-zero-fill.md](./MVP-003-route-candidate-zero-fill.md) | DONE | MVP-001 | 1 个专注开发日 | 2026-07-07T15:06:35+08:00 | [实现与测试证据](./MVP-003-route-candidate-zero-fill.md#完成证据) |
| MVP-004 | [MVP-004-report-regression-verification.md](./MVP-004-report-regression-verification.md) | DONE | MVP-002, MVP-003 | 0.5 个专注开发日 | 2026-07-07T15:10:30+08:00 | [回归验证证据](./MVP-004-report-regression-verification.md#完成证据) |

## 依赖说明

- 关键路径：`MVP-001 → MVP-002/MVP-003 → MVP-004`。
- `MVP-002` 与 `MVP-003` 在公共日期序列和排序规则稳定后可以并行实施。

## 规划假设

- 当前配置条目对整个查询日期范围生效；本次不恢复配置历史生效区间。
- 不新增数据库表、字段或迁移，不修改用量入库、计费与限额拦截逻辑。
- 使用项目当前支持的 MySQL 8 递归 CTE；实施时必须通过 Repository 测试验证参数、汇总、排序和分页。
- API 请求及响应结构保持兼容，前端仅做回归检查，除非测试发现零用量行展示缺陷。
