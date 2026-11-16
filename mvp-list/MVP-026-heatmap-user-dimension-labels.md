# MVP-026: 修复热力图维度查询、对象名称与零并发样本展示

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `0.75 day`
- Estimate rationale: 后端需要修正用户维度限制和零值样本存储，前端补稳定键到展示名映射，并完成两层回归测试。
- Dependencies: `MVP-024`

## Outcome

Token 速率和错误率热力图支持“用户”维度，并始终用用户、分组、别名或模型账号名称展示对象；真实采样到 0 并发时显示 0%，不再误判为无采样。

## Context

A 环境切换“用户 + 错误率”时接口返回 `500`，页面显示 `Unable to load observability heatmap`。根因是 `requestHeatmap` 错误地拒绝了 `user_id`，尽管请求聚合事实本身包含用户维度。

另外，热力图接口返回稳定对象键（用户 ID、分组 ID、账号 ID 或路由别名）。前端此前直接把稳定键作为行标题，导致模型账号显示成 `1`、`3` 等数字。

采样器状态为 `healthy` 且历史桶已有有效 `effective_limit=10`，但当实际并发为 `0` 时，后端仅在 `ratio > 0` 时保存候选采样，最终把真实的 0 并发样本当成“没有数据”隐藏。

## In Scope

- 允许 Token 速率和错误率热力图使用 `user_id`。
- 保留接口行键作为筛选、联动和 drilldown 的稳定标识。
- 在热力图展示层把行键映射为用户邮箱、分组名称、路由别名或模型账号名称。
- 表头、hover tooltip 和 `aria-label` 使用展示名；选择事件继续发送原始键。
- 有效采样即便实际并发为 0，也返回 `available=true`、`actual_concurrency=0` 和真实上限。
- 补充后端用户维度查询测试和前端名称映射测试。

## Out of Scope

- 不改变并发热力图对用户维度“无采样”的降级语义。
- 不修改热力图颜色、阈值、时间桶或聚合表结构。
- 不新增配置、数据库迁移或数据回填。

## Implementation Notes

- `requestHeatmap` 只校验维度是否合法，不再单独排除 `user_id`。
- 并发热力图按 cell 区分为“有采样且值为 0”和“没有采样”，不再使用 map 零值推断采样存在性。
- `PressureHeatmap` 增加可选 `rowLabels`，仅负责展示转换。
- `ModelObservabilityView` 从快照 filter options 和 TopN 数据构造当前热力图维度名称表。
- 找不到名称时使用“用户/分组/账号 #ID”回退，避免直接裸露纯数字。

## Acceptance Criteria

- [x] “用户 + Token 速率”和“用户 + 错误率”接口返回成功。
- [x] 热力图行标题、tooltip 和 aria-label 优先展示对象名称。
- [x] 点击热力格仍发送原始对象键，不破坏筛选与链路联动。
- [x] 采样任务正常且存在真实 0 并发样本时，热力图显示 0% 而非“暂无数据”。
- [x] 后端用户维度测试、前端名称映射与页面行为测试通过。
- [x] 后端构建和前端类型检查通过。

## Verification Plan

```powershell
cd backend
go test -p 1 -tags=unit ./internal/handler/admin -run 'Observability.*Heatmap|Observability.*GroupBy'
go build -p 1 -o server.exe ./cmd/server

cd ..\frontend
pnpm exec vitest run src/components/observability/__tests__/PressureHeatmap.spec.ts src/views/admin/__tests__/ModelObservabilityView.behavior.spec.ts
pnpm run typecheck
```

真实 API 验证：

```text
GET /api/v1/admin/observability/heatmap?window=1h&bucket=5m&group_by=user_id&value_kind=error_rate
```

预期 HTTP `200`。

## Completion Evidence

| Type | Command or path | Result |
|---|---|---|
| Backend user dimension | `go test -p 1 -tags=unit ./internal/handler/admin -run 'Observability.*Heatmap|Observability.*GroupBy'` | PASS |
| Backend build | `go build -p 1 -o server.exe ./cmd/server` | PASS |
| Frontend focused tests | `PressureHeatmap.spec.ts`, `ModelObservabilityView.behavior.spec.ts` | PASS: 12 tests |
| Frontend typecheck | `pnpm run typecheck` | PASS |
| Live API | `group_by=user_id&value_kind=error_rate` | PASS: HTTP `200`, rows returned |
| Live sampled concurrency | `group_by=account_id&value_kind=sampled_concurrency` | PASS: `available=true`, 3 rows, 24 available zero-valued cells with limit `10` |
| Label behavior | `PressureHeatmap.spec.ts` | PASS: display name shown; raw row key preserved in selection event |

## Execution Notes

修复不涉及配置和数据库。历史 `Unable to load observability heatmap` 请求对应后端 `500`；修复后同一 API 返回 `200`。
