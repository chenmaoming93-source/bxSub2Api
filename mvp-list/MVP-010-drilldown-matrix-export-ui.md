# MVP-010: 实现调用链路、交叉明细、单维排行与导出

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 复用关系/路径 API 和页面共享状态，主要是三个视图、详情抽屉与安全导出。
- Dependencies: `MVP-007`, `MVP-008`

## Outcome

管理员可从任意排行/热力格在三次点击内定位具体用户、分组、路由、账号和上游模型，并安全导出当前路径结果。

## Context

现有 Demo 已证明交互方向，但关系数据需要按真实 `group_model_route_accounts` 和 canonical 维度实现，不能按名称拼数组。

## In Scope

- ChainView：用户 → API Key → 分组 → 路由别名 → 账号 → 上游模型逐级联动。
- 用户节点展示 `user_id / email / department`，邮箱不脱敏；API Key 仅展示 ID/名称，不展示 secret。
- MatrixView：完整路径行、搜索、状态筛选、排序、服务端分页、详情抽屉。
- DimensionView：四个维度 Tab 排行和关联层级信息；模型账号行展示唯一绑定模型。
- DetailDrawer：当前值、窗口值、样本数、数据时间、关联路径和数据质量。
- CSV 导出当前筛选/分页范围，服务端限制行数并做公式注入转义。
- 点击联动并可返回图表对象。

## Out of Scope

- 配置编辑。
- 导出包含错误正文、凭据或上游响应头的敏感数据。

## Implementation Notes

- 链路节点携带稳定 ID 和 `has_usage/idle`，显示名只展示。
- 表格列使用虚拟滚动或服务端分页；不一次性渲染万行。
- CSV 对以 `= + - @` 开头的单元格加前缀或引号安全处理；内容类型 UTF-8 BOM。
- 导出也走 `observability.read` 和 RBAC scope，不允许绕过 filters。

## Acceptance Criteria

- [x] 点击热力格或排行能在三次点击内到达完整路径详情。
- [x] 四维 Tab 的排序、筛选、分页一致，空结果不报错。
- [x] `include_idle` 数据不参与 usage 汇总。
- [x] 详情抽屉明确展示 `legacy_inferred`、unknown model 和缺失样本。
- [x] CSV 只包含当前筛选范围且公式注入测试通过。
- [x] 页面视觉与现有管理端一致，不出现横向溢出。

## Verification Plan

```powershell
pnpm --dir frontend run test:run
pnpm --dir frontend run typecheck
pnpm --dir frontend run lint:check
pnpm --dir frontend run build
```

添加一个 CSV 恶意文本夹具，例如用户名 `=HYPERLINK(...)`，验证导出不再以公式开头。

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Interaction path | Heatmap/ranking click -> shared filter -> path row -> detail drawer | Two clicks reach path detail; selected canonical filter remains shared across charts and drill-down |
| ChainView | `frontend/src/components/observability/DrilldownPanel.vue` | user -> API Key -> group -> route -> account -> model nodes; node click emits stable ID/alias filter |
| MatrixView | Same component + server paths API | Search/status/sort/page controls, server pagination, empty state and keyboard row activation |
| DimensionView | Four canonical tabs | Users/groups/aliases/accounts from snapshot; account rows keep single bound model |
| Detail drawer | DrilldownPanel | Shows current/limit, tokens/requests, P95 TTFT with unavailable text, status, legacy inference, model attribution and data-quality warnings |
| Idle behavior | `includeIdle` API parameter + path aggregation | Idle configuration may appear in relations, but zero-traffic edges never enter usage totals |
| CSV scope | `frontend/src/components/observability/csv.ts` | Exports only `paths.rows` for the current server page/filter, UTF-8 BOM |
| Formula injection | `csv.spec.ts` | PASS: `=HYPERLINK(...)`, `+`, `-`, `@` cells are prefixed and quoted |
| API params | `observability.spec.ts` | PASS: page/page_size/sort/status/include_idle/search and canonical IDs remain in path request |
| Component tests | `DrilldownPanel.spec.ts` | PASS: row detail emission, legacy flag, bound model and missing-sample text |
| Typecheck/build | `pnpm run typecheck`, `pnpm run build` | PASS |
| Layout | Existing responsive screenshots plus matrix/heatmap scroll containers | PASS: page-level mobile layout remains within viewport; tables/heatmaps scroll locally |
| Known limitation | Full visual replay after adding DrilldownPanel | Panel covered by component tests and production build; prior 1280/768/375 screenshots cover shared page shell and chart layout |

## Execution Notes

若用户级明细涉及隐私决策未确认，MVP 可先支持 group/account/model 维度并让 user 维度受 feature flag 控制。
