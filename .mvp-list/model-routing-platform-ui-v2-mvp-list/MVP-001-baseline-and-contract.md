# MVP-001：建立页面重整基线与接口契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 盘点范围明确，产出路由、复用和契约基线，独立可验证。
- Dependencies: `none`

## 预期成果

形成可供后续 UI 和页面开发使用的页面、路由、接口、组件及停用范围基线。

## 背景

源计划：`../../.plans/model-routing-platform-ui-v2-page-rework-implementation-plan.md`。当前路由集中在 `frontend/src/router/index.ts`，菜单集中在 `frontend/src/components/layout/AppSidebar.vue`，权限映射位于 `frontend/src/rbac/permissionMatrix.ts`。

## 范围内

- 固化 Part 1～10 页面路径和权限边界。
- 固化可复用接口、组件和必须新增接口清单。
- 固化六张原型图路径及视觉验收规则。
- 固化停用页面路径和 404 行为。
- 记录旧组件、旧页面不得删除的约束。

## 范围外

- 不修改业务代码、路由或接口。
- 不新增数据库表。
- 不实现 UI V2 组件。

## 实现说明

- 以已审核 Plan 和仓库只读审计结果为准。
- 对场景排行明确记录：现有数据需跨日期按 `group_id` 聚合，不能直接当作完成的场景汇总接口。

## 验收标准

- [x] 基线文档覆盖十个 Part、停用路由、六张原型图和暂停节点。
- [x] 每个 Part 均标注接口复用、组件复用和新增能力。
- [x] 场景聚合、个人统计和按模型耗时接口缺口被明确记录。

## 验证计划

- 人工逐项对照 Plan、`router/index.ts`、`AppSidebar.vue`、API 模块和原型图文件。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Plan | `.plans/model-routing-platform-ui-v2-page-rework-implementation-plan.md` | 已审核 Plan 存在且状态为 Final — user approved |
| 路由 | `frontend/src/router/index.ts` | 已核对管理员路由、`/admin` 重定向和 catch-all 404 |
| 菜单 | `frontend/src/components/layout/AppSidebar.vue` | 已核对管理员和个人导航声明 |
| 权限 | `frontend/src/rbac/permissionMatrix.ts` | 已核对有效页面和待停用页面权限映射 |
| API | `frontend/src/api/admin/dashboard.ts`、`frontend/src/api/admin/usage.ts`、`frontend/src/api/usage.ts` | 已核对 Part 1/2/9/10 现有接口与缺口 |
| 原型 | `assets/final-admin-dashboard-prototype-v2.png` 等六张原型图 | 六张文件均存在 |

## 执行记录

已完成基线核对；未修改业务源代码。
