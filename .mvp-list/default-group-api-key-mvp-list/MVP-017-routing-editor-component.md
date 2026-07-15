# MVP-017：抽取共享模型路由编辑组件

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `none`

## 预期成果

分组编辑页使用可复用的模型路由编辑组件，现有行为不回归。

## 背景

模型路由 UI和转换逻辑集中在 `frontend/src/views/admin/GroupsView.vue`，需先抽取再建设默认分组页面。

## 范围内

- 抽取 `GroupModelRoutingEditor`。
- 抽取或集中 normalize、validate、API格式转换函数。
- 让现有创建与编辑卡片复用组件。
- 增加组件及转换函数测试。

## 范围外

- 默认分组新页面。
- 后端默认分组 API。

## 实现说明

- 保持现有 i18n key和账号搜索行为。
- 避免顺手重构无关分组表单。

## 验收标准

- [x] 现有分组创建/编辑模型路由行为一致。
- [x] 组件测试、相关既有测试和类型检查通过。

## 验证计划

- `cd frontend; pnpm test:run -- GroupsView GroupModelRoutingEditor`
- `cd frontend; pnpm typecheck`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 聚焦测试 | `cd frontend; pnpm exec vitest run src/views/admin/__tests__/GroupsView.modelTokenQuota.spec.ts src/views/admin/__tests__/groupsModelRouting.spec.ts src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 通过，3 个文件、6 个测试全部成功。 |
| 类型检查 | `cd frontend; pnpm typecheck` | 通过，`vue-tsc --noEmit` 退出码 0。 |
| 计划测试命令 | `cd frontend; pnpm test:run -- GroupsView GroupModelRoutingEditor` | 脚本实际运行全套测试；相关 GroupsView/组件测试通过，但全套存在 16 个与本 MVP 无关的 usage/chart/auth 既有失败。 |
| 共享组件 | `frontend/src/components/admin/group/GroupModelRoutingEditor.vue` | 创建与编辑表单共同使用；保留 i18n、账号搜索、候选增删和校验交互。 |
| 转换逻辑 | `frontend/src/views/admin/groupsModelRouting.ts` | normalize、validate、serialize 继续集中复用，既有转换测试通过。 |

## 执行记录

实现与验证完成。`GroupsView.vue` 的创建、编辑运行路径均接入共享组件；原内联模板暂以不可达分支保留，降低本轮大文件删除造成的合并噪声，后续可独立清理，不影响运行或类型检查。
