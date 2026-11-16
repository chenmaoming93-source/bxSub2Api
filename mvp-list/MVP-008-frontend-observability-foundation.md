# MVP-008: 落地观测页骨架、全局筛选与首屏 KPI

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 使用已完成的 snapshot API，聚焦页面容器、状态管理、权限、KPI 和数据质量提示。
- Dependencies: `MVP-005`, `MVP-006`, `MVP-007`

## Outcome

授权管理员可从导航进入观测页，选择四个 canonical 维度的 filters，看到实时 KPI、TopN 摘要和数据质量状态。

## Context

现有管理端使用 Vue 3、`AppLayout`、Pinia/权限 composable、Tailwind 语义色与 Chart.js。需要复用现有布局、状态、主题和 i18n，不另做独立前端。

## In Scope

- 新增 `ModelObservabilityView.vue`、`frontend/src/api/admin/observability.ts`、`useObservability.ts`。
- 添加路由、侧边栏/菜单项和 `observability.read` 权限控制。
- 实现全局 filter context：user/group/alias/account 多选、搜索、chips、清空、仅高压；账号详情展示唯一绑定模型。
- 用户选择器与排行展示用户 ID、邮箱和部门，不做邮箱脱敏；敏感凭据字段不进入前端数据模型。
- 首屏按 snapshot API 渲染 KPI、TopN 摘要和 `DataQualityBanner`。
- 分层轮询：KPI 5 秒、趋势 30-60 秒、页面隐藏暂停、请求可取消、失败退避。
- 支持深浅色、窄屏和首屏空/错/部分数据状态。

## Out of Scope

- 复杂图表、关系矩阵和详情抽屉。
- 容量写操作。
- 自由文本任意搜索所有用户；过滤选项受 API 上限和权限约束。

## Implementation Notes

- filter 值全部保存 canonical ID/string，不在 URL 或 SQL 中依赖显示名。
- 快速改变筛选时使用 AbortController 取消前序请求；只接受最新 `request_seq`。
- 缓存命中 304 时不重建 DOM，仅更新时间/新鲜度提示。
- 每个卡片展示自己的 loading/empty/error，不用全页 spinner 覆盖已有数据。
- 中英文案沿用 i18n 结构；看板英文名与现有导航风格一致。

## Acceptance Criteria

- [x] 无权限角色看不到菜单，直接访问路由得到禁止页/重定向，不渲染数据请求。
- [x] 新页面以独立路由和菜单入口上线，现有 Dashboard/Ops 页面路由保持可用。
- [x] 四个 filter 维度可搜索、多选、清空并生成 canonical 参数。
- [x] 用户 ID、邮箱和部门均可见，且页面数据中不存在凭据、请求正文或 API Key secret。
- [x] KPI 和 TopN 来自 snapshot，不在组件内伪造或按比例缩放。
- [x] 数据延迟、聚合滞后、采样缺失和对账差异均有可见提示。
- [x] 页面隐藏后不轮询，恢复可见时立即刷新一次。
- [x] 深浅色、移动端和键盘操作基本通过。

## Verification Plan

```powershell
pnpm --dir frontend run typecheck
pnpm --dir frontend run lint:check
pnpm --dir frontend run test:run
pnpm --dir frontend run build
```

## Completion Evidence

> Leave this section empty until work has actually been performed.

| Type | Command or path | Result |
|---|---|---|
| Route/menu | `frontend/src/router/index.ts`, `AppSidebar.vue` | Independent `/admin/observability` route and menu item; existing `/admin/dashboard` and `/admin/ops` remain |
| Permission | Router guard + `permissionMatrix.spec.ts` | `observability.read` required; unauthenticated/unauthorized users are redirected before view mount, so no request starts |
| API client | `frontend/src/api/admin/observability.ts` | Snapshot/series/heatmap clients, repeated canonical IDs, ETag/304 handling and abort signals |
| Canonical filters | `src/api/admin/__tests__/observability.spec.ts` | PASS: repeated `user_id/group_id/route_alias/account_id`, department and search params use IDs/aliases only |
| Filter UX | `ModelObservabilityView.vue` | Search, four multi-select groups, removable chips, clear-all, department chip and high-pressure toggle |
| Identity privacy | View source test + API types | PASS: user ID/email/department and bound model visible; no credential, request-body or API-key secret fields |
| Snapshot KPI/TopN | `ModelObservabilityView.spec.ts` | PASS: KPI and rankings bind directly to snapshot; no ratio/scale fabrication |
| Data quality | Snapshot view | Warning codes are rendered in a visible banner with lag and complete/partial state |
| Polling | `useObservability.spec.ts` | PASS: immediate load, no polling while hidden, immediate snapshot+trend refresh on visible |
| Layered refresh | `useObservability.ts` | Snapshot 5s; trend 45s; failure exponential backoff capped at 60s; AbortController + sequence guard |
| 304 behavior | API/composable | 304 retains previous object references and only updates freshness, avoiding DOM rebuild |
| Responsive/accessibility | View templates | Responsive `sm/md/xl` grids, dark variants, labeled checkboxes, keyboard-focus styles and scrollable filter groups |
| Typecheck/build | `pnpm run typecheck`, `pnpm run build` | PASS; built `ModelObservabilityView` chunk |
| Lint | `pnpm run lint:check` | PASS with one pre-existing warning in `UsageFilters.spec.ts` |
| Focused tests | `vitest` observability, permission, sidebar and view tests | PASS, 13 tests |
| Full tests | `pnpm run test:run` | 805/811 passed; 6 pre-existing auth redirect URL failures in EmailOAuth/WechatOAuth/EmailVerify, unrelated to observability |

## Execution Notes

原 Demo 的“生产环境”下拉没有对应 API，真实页面应删除或改为现有 platform/instance 标识。
