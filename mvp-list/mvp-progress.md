# MVP Progress

- Protocol: `mvp-list/v1`
- Source plan: `dashboard-demo/revision-v2/design-spec.md` + `prototype.html`（当前确认设计；`MVP-001`～`MVP-015` 保留历史基线）
- Target effort per MVP: 1 focused developer-day
- Update batch size: 2 completed MVPs
- Last updated: `2026-11-16T10:55:00+08:00`
- Overall: `24/24 (100%)`

## Status Rules

- `PENDING`: not yet recorded as verified complete
- `BLOCKED`: cannot proceed and is not counted as complete
- `DONE`: implemented, acceptance criteria checked, tests run, and evidence recorded
- Progress may lag verified work between batch updates; it must never lead verified work.

## MVP List

| ID | MVP document | Status | Dependencies | Estimate | Completed at | Evidence |
|---|---|---|---|---|---|---|
| MVP-001 | [MVP-001-observability-contract-rbac.md](./MVP-001-observability-contract-rbac.md) | DONE | none | 1 day | 2026-11-11 | [evidence](./MVP-001-observability-contract-rbac.md#completion-evidence) |
| MVP-002 | [MVP-002-account-model-and-route-attribution.md](./MVP-002-account-model-and-route-attribution.md) | DONE | MVP-001 | 1.5 days | 2026-11-11 | [evidence](./MVP-002-account-model-and-route-attribution.md#completion-evidence) |
| MVP-003 | [MVP-003-request-minute-rollup.md](./MVP-003-request-minute-rollup.md) | DONE | MVP-002 | 1.5 days | 2026-11-11 | [evidence](./MVP-003-request-minute-rollup.md#completion-evidence) |
| MVP-004 | [MVP-004-concurrency-history-sampling.md](./MVP-004-concurrency-history-sampling.md) | DONE | MVP-001 | 1 day | 2026-11-11 | [evidence](./MVP-004-concurrency-history-sampling.md#completion-evidence) |
| MVP-005 | [MVP-005-snapshot-filter-api.md](./MVP-005-snapshot-filter-api.md) | DONE | MVP-002, MVP-003, MVP-004 | 1 day | 2026-11-11 | [evidence](./MVP-005-snapshot-filter-api.md#completion-evidence) |
| MVP-006 | [MVP-006-timeseries-heatmap-quality-api.md](./MVP-006-timeseries-heatmap-quality-api.md) | DONE | MVP-003, MVP-004, MVP-005 | 1.5 days | 2026-11-11 | [evidence](./MVP-006-timeseries-heatmap-quality-api.md#completion-evidence) |
| MVP-007 | [MVP-007-relations-path-drilldown-api.md](./MVP-007-relations-path-drilldown-api.md) | DONE | MVP-002, MVP-003, MVP-005 | 1 day | 2026-11-11 | [evidence](./MVP-007-relations-path-drilldown-api.md#completion-evidence) |
| MVP-008 | [MVP-008-frontend-observability-foundation.md](./MVP-008-frontend-observability-foundation.md) | DONE | MVP-005, MVP-006, MVP-007 | 1 day | 2026-11-11 | [evidence](./MVP-008-frontend-observability-foundation.md#completion-evidence) |
| MVP-009 | [MVP-009-trend-rank-heatmap-ui.md](./MVP-009-trend-rank-heatmap-ui.md) | DONE | MVP-006, MVP-008 | 1.5 days | 2026-11-11 | [evidence](./MVP-009-trend-rank-heatmap-ui.md#completion-evidence) |
| MVP-010 | [MVP-010-drilldown-matrix-export-ui.md](./MVP-010-drilldown-matrix-export-ui.md) | DONE | MVP-007, MVP-008 | 1 day | 2026-11-11 | [evidence](./MVP-010-drilldown-matrix-export-ui.md#completion-evidence) |
| MVP-011 | [MVP-011-alert-and-ux-hardening.md](./MVP-011-alert-and-ux-hardening.md) | DONE | MVP-005, MVP-009 | 1 day | 2026-11-11 | [evidence](./MVP-011-alert-and-ux-hardening.md#completion-evidence) |
| MVP-012 | [MVP-012-e2e-performance-release.md](./MVP-012-e2e-performance-release.md) | DONE | MVP-009, MVP-010, MVP-011 | 1.5 days | 2026-11-11 | [evidence](./MVP-012-e2e-performance-release.md#completion-evidence) |
| MVP-013 | [MVP-013-non429-failure-request-logging.md](./MVP-013-non429-failure-request-logging.md) | DONE | none | 1 day | 2026-11-11 | [evidence](./MVP-013-non429-failure-request-logging.md#completion-evidence) |
| MVP-014 | [MVP-014-superadmin-user-limit-exemption.md](./MVP-014-superadmin-user-limit-exemption.md) | DONE | none | 1 day | 2026-11-11 | [evidence](./MVP-014-superadmin-user-limit-exemption.md#completion-evidence) |
| MVP-015 | [MVP-015-usage-scene-section-reorder.md](./MVP-015-usage-scene-section-reorder.md) | DONE | none | 0.5 day | 2026-11-11 | [evidence](./MVP-015-usage-scene-section-reorder.md#completion-evidence) |
| MVP-016 | [MVP-016-flexible-time-range-contract.md](./MVP-016-flexible-time-range-contract.md) | DONE | MVP-005, MVP-006 | 1.5 days | 2026-11-12 | [evidence](./MVP-016-flexible-time-range-contract.md#completion-evidence) |
| MVP-017 | [MVP-017-capacity-wait-rate-limit-metrics.md](./MVP-017-capacity-wait-rate-limit-metrics.md) | DONE | MVP-004, MVP-005 | 1.5 days | 2026-11-12 | [evidence](./MVP-017-capacity-wait-rate-limit-metrics.md#completion-evidence) |
| MVP-018 | [MVP-018-heatmap-concurrency-contract.md](./MVP-018-heatmap-concurrency-contract.md) | DONE | MVP-004, MVP-006 | 1 day | 2026-11-13 | [evidence](./MVP-018-heatmap-concurrency-contract.md#completion-evidence) |
| MVP-019 | [MVP-019-demo-shell-filter-kpi.md](./MVP-019-demo-shell-filter-kpi.md) | DONE | MVP-016, MVP-017 | 1.5 days | 2026-11-13 | [evidence](./MVP-019-demo-shell-filter-kpi.md#completion-evidence) |
| MVP-020 | [MVP-020-trend-quality-workbench.md](./MVP-020-trend-quality-workbench.md) | DONE | MVP-019 | 1.5 days | 2026-11-13 | [evidence](./MVP-020-trend-quality-workbench.md#completion-evidence) |
| MVP-021 | [MVP-021-pressure-capacity-ui.md](./MVP-021-pressure-capacity-ui.md) | DONE | MVP-017, MVP-018, MVP-019 | 1.5 days | 2026-11-13 | [evidence](./MVP-021-pressure-capacity-ui.md#completion-evidence) |
| MVP-022 | [MVP-022-inline-linked-drilldown.md](./MVP-022-inline-linked-drilldown.md) | DONE | MVP-021 | 1.5 days | 2026-11-13 | [evidence](./MVP-022-inline-linked-drilldown.md#completion-evidence) |
| MVP-023 | [MVP-023-interaction-responsive-hardening.md](./MVP-023-interaction-responsive-hardening.md) | DONE | MVP-020, MVP-021, MVP-022 | 1 day | 2026-11-13 | [evidence](./MVP-023-interaction-responsive-hardening.md#completion-evidence) |
| MVP-024 | [MVP-024-visual-e2e-performance-release.md](./MVP-024-visual-e2e-performance-release.md) | DONE | MVP-022, MVP-023 | 1.5 days | 2026-11-16 | [evidence](./MVP-024-visual-e2e-performance-release.md#completion-evidence) |

## Dependency Notes

- 关键路径：`MVP-001 → MVP-002 → MVP-003 → MVP-005 → MVP-006 → MVP-008 → MVP-009 → MVP-011 → MVP-012`。
- 关系明细支路：`MVP-001 → MVP-007 → MVP-010 → MVP-012`；该支路可与趋势支路部分并行。
- 可并行组：`MVP-003` 与 `MVP-004` 在事实字段确定后可并行；`MVP-009` 与 `MVP-010` 在页面骨架完成后可并行。
- 独立支路：`MVP-013`、`MVP-014`、`MVP-015` 与观测台主链路无强依赖，可作为并行需求插入任意可用开发档期。
- 观测台为纯只读页面，不包含切流、候选容量写入、阈值应用、处置审计或回滚。
- 当前修订批次的计划关键路径：`{MVP-016, MVP-017} → MVP-019 → MVP-021 → MVP-022 → MVP-023 → MVP-024`。
- 后端支路：`MVP-017` 与 `MVP-018` 可在 `MVP-016` 推进期间并行；`MVP-020` 可在 `MVP-019` 完成后与 `MVP-021` 并行。
- 历史 `MVP-001`～`MVP-015` 保留原编号、完成时间和证据；新增批次只描述当前设计相对现有实现的差额。

## Planning Assumptions

- 估算按 1 名熟悉 Go/Vue/MySQL/Redis 的开发者计算；包含实现、单测、必要迁移和文档，不含长时间等待代码评审或发布窗口。
- 1 个 focused developer-day 约等于 6-7 小时有效开发时间；1.5 天表示存在跨层联调或性能验证的 MVP。
- 历史 15 个 MVP 已投入估算为 17 个开发者工作日；当前 9 个新增 MVP 合计 12.5 个开发者工作日，总量为 29.5 个开发者工作日。
- 当前新增批次由 1 名开发者串行执行约需 12.5 个工作日（约 2.5-3 周）；按关键路径 8.5 个工作日，2 名开发者协作约需 1.5-2 周，前提是后端数据任务及时提供给前端。
- 产品约束：一个模型账号严格对应一个上游模型；模型名只作为账号属性，不单独成为筛选或聚合维度。
- 当前 tokenstat 的 D/W/M 周期和 `total_tokens` 指标保持不变；观测历史通过请求事实与分钟聚合实现。
- 现有 Ops 告警、RBAC 和并发配置能力优先复用；不引入 Prometheus/Grafana 等新基础设施。
- 主 KPI 使用最终客户端失败率；业务限制率、SLA 错误率和上游错误率单独展示。
- 管理员和运维只读角色可查看用户 ID、邮箱、部门，邮箱不脱敏；凭据、请求正文和 API Key secret 仍禁止返回。
- 分钟数据保留 24-48 小时，5 分钟数据保留 7 天；7 天用户明细只提供 TopN。
- 历史多模型账号只生成清单供人工修复，不自动修改，也不执行全表 JSON 回填。
- 告警复用现有 Ops 规则和通知渠道；新页面与现有 Dashboard/Ops 并存。
- 首期仅超级管理员访问，验证一天后通过 `observability.read` 开放给运维只读角色；Token/配额面板额外要求 `token_usage.read`。
- 新增批次的交互查询最大范围为 7 天，沿用当前分钟/5 分钟/小时聚合保留能力；若要支持更长自定义范围，需另建聚合与保留任务。
- 平均等待必须来自真实排队等待样本；当前没有该事实，因此 `MVP-017` 必须先补齐采样，不能使用队列深度推算。
- 正式页面只保留 `/admin/observability`；链路是同一页面的下钻区，不创建 `/admin/observability/paths`。
