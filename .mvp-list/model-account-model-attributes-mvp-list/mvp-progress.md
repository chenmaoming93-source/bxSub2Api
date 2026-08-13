# MVP 进度

- Protocol: `mvp-list/v1`
- Source plan: `.plans/model-account-model-attributes-implementation-plan.md`
- Target effort per MVP: 用户未指定目标工作量，按假设「每个 MVP 为一个聚焦的开发者工作日」执行
- Progress update cadence: `after every completed MVP`
- Last updated: `2026-08-05T10:09:12Z`
- Overall: `5/5 (100%)`

## 状态规则

- `PENDING`：尚未记录为已验证完成
- `BLOCKED`：无法继续，且不计入完成项
- `DONE`：已实现、验收标准已确认、测试已运行且证据已记录
- 每个 MVP 验证完成后必须立即更新进度文档，然后才能开始下一个 MVP。

## MVP 列表

| ID | MVP 文档 | 状态 | 依赖项 | 估算 | 完成时间 | 证据 |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-backend-data-layer.md](./MVP-001-backend-data-layer.md) | DONE | none | 1 个开发者日 | 2026-08-05T09:22:01Z | [证据](./MVP-001-backend-data-layer.md#完成证据) |
| MVP-002 | [MVP-002-backend-service-chain.md](./MVP-002-backend-service-chain.md) | DONE | MVP-001 | 1 个开发者日 | 2026-08-05T10:00:28Z | [证据](./MVP-002-backend-service-chain.md#完成证据) |
| MVP-003 | [MVP-003-frontend-section-component.md](./MVP-003-frontend-section-component.md) | DONE | none | 1 个开发者日 | 2026-08-05T09:38:34Z | [证据](./MVP-003-frontend-section-component.md#完成证据) |
| MVP-004 | [MVP-004-modal-integration.md](./MVP-004-modal-integration.md) | DONE | MVP-003 | 1 个开发者日 | 2026-08-05T10:05:00Z | [证据](./MVP-004-modal-integration.md#完成证据) |
| MVP-005 | [MVP-005-end-to-end-regression.md](./MVP-005-end-to-end-regression.md) | DONE | MVP-002, MVP-004 | 0.5 个开发者日 | 2026-08-05T10:09:12Z | [证据](./MVP-005-end-to-end-regression.md#完成证据) |

## 依赖说明

- 关键路径：MVP-001 → MVP-002 →（MVP-005）；MVP-003 → MVP-004 →（MVP-005）。
- 可并行分组：后端链路（MVP-001/MVP-002）与前端链路（MVP-003/MVP-004）互不依赖，可并行实施；两端完成后再执行 MVP-005 集成回归。
- 每个 MVP 均以「可独立编译/测试通过」为完成门槛，避免跨 MVP 的隐性耦合。

## 规划假设

- 目标工作量假设为「每个 MVP 一个聚焦的开发者工作日」（用户未指定具体值）。
- ent 生成采用全量 `go generate ./ent`；已验证仓库生成代码与 ent v0.14.5 匹配（干净状态零 diff），预期本次 diff 仅落在 account 实体相关文件；如实际 diff 超出预期，在 MVP-001 执行记录中说明并暂停确认。
- 后端信任前端：`value` 原样存储，不做类型解析或枚举校验；仅丢弃 key 为空白的条目。
- 预置属性清单仅存在于前端（常量 + i18n），后端不感知。
- 数据库 DDL 通过 `backend/sqlArchiving/168_add_account_model_attributes.sql` 归档，发布顺序固定为「先 SQL 后代码」。
