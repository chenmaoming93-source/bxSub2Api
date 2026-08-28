# MVP-004：在 `/admin/usage` 展示场景 Token 用量

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 在稳定管理端接口基础上增加 API 类型、筛选控件、嵌套结果展示和前端测试，形成管理员可直接使用的完整页面成果。
- Dependencies: `MVP-003`

## 预期成果

管理员可以在 `/admin/usage` 选择日期范围和技术分组名，查看每天每个场景的 Token 总量，以及场景内每个模型账号和上游模型的明细。

## 背景

现有管理用量页面为 `frontend/src/views/admin/UsageView.vue`，API 封装为 `frontend/src/api/admin/usage.ts`。现有 `/admin/token-statistics` 页面提供 Projection 管理，但本 MVP 不改变其功能。

## 范围内

- 在 `/admin/usage` 增加场景 Token 用量区域或 Tab。
- 增加开始日期、结束日期、分组名筛选和查询/刷新操作。
- 分组筛选选项可以展示 `scene_name（name）`，但提交值必须为 `name`。
- 展示日期 → 场景 → 账号/上游模型的嵌套数据。
- 展示场景总 Token、账号/模型明细、统计时区和同步时间。
- 展示无数据、同步不完整和缺少配置时的明确状态。
- 增加 API 类型、页面测试和类型检查。
- 保持现有用量记录、统计和清理功能不受影响。

## 范围外

- 不在前端调用外部 integrations 接口。
- 不允许通过 `scene_name` 过滤查询。
- 不新增独立导航页面。
- 不在前端实现 Token 统计聚合，页面使用后端返回的场景总量。

## 实现说明

- 调用 `/admin/usage/scene-account/daily`。
- 对账号/模型明细按 `account_id + upstream_model` 渲染，避免同一账号多个上游模型被覆盖。
- 配置错误提示应明确指向 `/admin/token-statistics`，并说明需要启用 `group_id + account_id + upstream_model`、`total_tokens`、日统计项。
- 使用现有页面国际化和组件风格；必要时补充中英文文案。

## 验收标准

- [x] 管理员可以在 `/admin/usage` 选择日期范围并查询。
- [x] 分组筛选提交的是 `name` 而不是 `scene_name`。
- [x] 页面正确展示每天的场景总 Token。
- [x] 页面正确展示 `account_name`、`upstream_model` 和对应 Token。
- [x] 同名场景对应的不同分组仍分别展示。
- [x] 页面正确展示无数据、最终一致和配置缺失状态。
- [x] 前端页面测试和类型检查通过。

## 验证计划

- `cd frontend && pnpm test:run -- src/views/admin/__tests__/UsageView.spec.ts`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- 人工打开 `/admin/usage`，验证筛选、嵌套展开、空数据和配置错误提示。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API 类型 | `frontend/src/api/admin/usage.ts` | 新增 `querySceneAccountDaily` 及日期/分组/嵌套场景账号模型响应类型。 |
| 页面实现 | `frontend/src/views/admin/UsageView.vue` | `/admin/usage` 新增场景 Token 区域：日期范围、技术 `group_name` 选择、查询、日期→场景→账号/上游模型展示。筛选值提交技术 `name`。 |
| 状态提示 | `frontend/src/views/admin/UsageView.vue` | 覆盖无数据、`complete=false` 同步提示、配置错误及 `/admin/token-statistics` 操作指引；同名场景按 `group_id` 保持分离。 |
| 国际化 | `frontend/src/i18n/locales/{zh.ts,en.ts}` | 增加中英文场景用量、同步和配置提示文案。 |
| 页面测试 | `pnpm test:run -- src/views/admin/__tests__/UsageView.spec.ts` | 通过，7 个测试；新增场景/账号模型展示断言。 |
| 类型检查 | `pnpm typecheck` | 通过。 |
| 构建 | `pnpm build` | 通过；Vite 产物生成成功，仅有既有 chunk size/Browserslist 警告。 |
| 格式检查 | `git diff --check` | 通过。 |

## 执行记录

- 2026-08-26：在既有 `/admin/usage` 页面内增加场景 Token 用量区域，复用管理端接口和全局日期范围，不新增导航页面。

