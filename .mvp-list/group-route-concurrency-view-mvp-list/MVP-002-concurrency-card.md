# MVP-002：实现分组并发查看卡片

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 天`
- Estimate rationale: 实现一个独立可见的分组并发查看入口和层级卡片，范围集中且可单独验收。
- Dependencies: `MVP-001`

## 预期成果

分组列表操作列出现“查看并发”按钮，点击后打开独立卡片，按路由别名—候选—账号展示当前并发和候选最大并发。

## 背景

管理员需要查看路由候选的实际容量，但不希望把实时信息混入编辑分组表单。

## 范围内

- 分组操作列按钮。
- 独立并发查看卡片。
- 路由别名、候选、账号三级展示。
- 多账号候选逐账号展示。
- 候选最大并发和 `∞` 展示。
- 加载、空数据和失败状态。

## 范围外

- 不实现自动刷新和定时器。
- 不修改 `admin/accounts` 页面。
- 不修改路由调度和并发计算。

## 实现说明

- 使用 MVP-001 确认的数据契约。
- 卡片展示结构与编辑表单状态隔离。
- 候选上限只读取候选配置，未配置显示 `∞`。

## 验收标准

- [x] 操作列按钮可打开和关闭卡片。
- [x] 卡片按路由别名—候选—账号展示。
- [x] 多账号候选分别展示。
- [x] 已配置上限显示数值，未配置显示 `∞`。
- [x] 加载和错误状态可观察且不破坏路由结构。
- [x] 相关组件测试或类型检查通过。

## 验证计划

- 运行并发卡片组件测试。
- 运行受影响的前端测试和类型检查。
- 手工检查分组列表操作列和卡片层级。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 类型检查 | `pnpm --dir frontend run typecheck` | 通过 |
| 测试 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/groupsModelRouting.spec.ts src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 2 个测试文件、10 个测试全部通过 |
| 实现路径 | `frontend/src/components/admin/group/GroupConcurrencyViewModal.vue`、`frontend/src/views/admin/GroupsView.vue` | 已实现操作列入口和独立层级卡片 |

## 执行记录

执行前保持为空；记录实际偏差、阻塞项和决策。
