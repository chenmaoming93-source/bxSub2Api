# MVP-009：完成管理员仪表盘验收材料并暂停开发

- Protocol: `mvp-list/v1`
- State: `ACTIVE`
- Estimate: `1 个开发日`
- Estimate rationale: 这是首个交付门，集中完成 Part 1 回归、截图和暂停前证据。
- Dependencies: `MVP-006, MVP-007, MVP-008`

## 预期成果

管理员仪表盘完成可检查版本，并按计划暂停后续开发，等待用户亲自启动项目验收 UI 风格。

## 背景

用户明确要求：完成 UI V2 组件和首个 Part（管理员仪表盘）后暂停，不继续开发其他 Part，直到用户确认风格。

## 范围内

- Part 1 功能回归。
- 统一时间范围、四类图表和两类详情弹窗验证。
- 亮色、暗色、桌面宽度和基础响应式验证。
- 对照六张原型图中与 Part 1 相关的三张图片。
- 记录测试命令、页面截图和已知问题。
- 更新 MVP 进度为 DONE 前的证据材料，并停止开发。

## 范围外

- 不开发 Part 2～10。
- 不迁移其他页面。
- 不清理停用路由。

## 实现说明

完成后必须将后续 MVP 保持 PENDING，并向用户报告已暂停，不得自动继续。

## 验收标准

- [x] Part 1 所有范围数据由统一日期范围控制。
- [x] 场景和部门详情均通过居中弹窗展示。
- [x] 无抽屉、无页面内展开。
- [ ] 自动测试和构建通过。（完整测试套件仍有既有失败，见完成证据）
- [ ] 已完成页面截图和视觉检查记录。（等待用户启动项目检查）
- [x] 开发流程在本 MVP 后暂停。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- `pnpm --dir frontend build`
- 用户启动项目进行人工 UI 验收。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm --dir frontend typecheck` | 通过 |
| Focused tests | `pnpm --dir frontend test:run -- src/features/admin-dashboard/__tests__/adminDashboardData.test.ts src/features/admin-dashboard/__tests__/scene-account-dialog.test.ts src/features/admin-dashboard/__tests__/department-user-dialog.test.ts` | 5 个相关测试通过 |
| Full tests | `pnpm --dir frontend test:run` | 未通过：137 个文件通过，3 个既有文件失败，共 6 个既有失败断言及 1 个既有 AccountsView 未处理错误 |
| Build | `pnpm --dir frontend build` | 通过；存在既有 chunk size 和 dynamic import 警告 |
| Route | `frontend/src/router/index.ts` | `/admin/dashboard` 已指向 `AdminDashboardV2View.vue` |
| Reference | `assets/final-admin-dashboard-prototype-v2.png`; `assets/final-admin-dashboard-scene-modal.png`; `assets/final-admin-dashboard-department-modal.png` | 已作为实现参考；用户视觉检查待完成 |
| Feedback regression | `pnpm run test:run -- src/components/ui-v2/__tests__/shell-charts.test.ts src/features/admin-dashboard/__tests__/scene-account-dialog.test.ts src/features/admin-dashboard/__tests__/department-user-dialog.test.ts` | 5 个相关测试通过；覆盖搜索、Top 12/全部、居中弹窗和详情排行数据加载 |
| Dialog width fix | `frontend/src/components/ui-v2/styles.css` | 修正 dialog 基础宽度固定为 32rem 的问题；wide/extra-wide/full 现在使用真实 width，不再仅设置 max-width |

## 执行记录

已根据用户对原型差异的反馈完成二次修正：排行标题/关键名称加粗放大；所有排行和用户趋势增加专属搜索及 Top 12/全部切换；指标和排行数字列设置防重叠宽度并支持至少 100 亿口径；性能指标改为当前日期范围平均 RPM/TPM；场景详情汇总严格沿用主页面跨日期聚合值；两个详情弹窗放大并改为横向 Token 柱状排行。随后修复弹窗基础 `width: 32rem` 覆盖尺寸变体的问题，`full` 弹窗现在实际占用视口宽度并可完整展示明细表。自动验证已完成，但完整测试套件和用户视觉验收尚未全部通过。按照计划在此暂停，不执行 MVP-010 及之后的开发。
