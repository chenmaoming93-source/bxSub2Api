# MVP-004：测试与回归验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 天`
- Estimate rationale: 集中补齐功能验收、边界状态和账号/分组页面回归证据，形成可交付闭环。
- Dependencies: `MVP-003`

## 预期成果

并发查看功能具备完整测试证据，覆盖展示、刷新、异常、权限和原有页面回归。

## 背景

该功能同时涉及分组路由结构和账号实时并发数据，需要验证实时数据刷新不会破坏已有编辑和账号列表行为。

## 范围内

- 组件测试。
- 数据组装测试。
- 自动刷新和请求清理测试。
- 分组编辑回归。
- `admin/accounts` 回归。
- 中英文文案检查。

## 范围外

- 不扩展新的并发产品能力。
- 不进行无关页面重构。

## 实现说明

- 补齐必要测试夹具和 mock。
- 验收多账号候选、候选上限、`∞`、失败和空数据状态。
- 记录实际测试命令和结果，不使用预期结果作为证据。

## 验收标准

- [x] 所有并发查看相关测试通过。
- [x] 路由别名—候选—账号层级和容量规则通过验证。
- [x] 自动刷新启动、停止和失败状态通过验证。
- [x] 分组编辑回归通过。
- [x] `admin/accounts` 回归通过。
- [x] 中英文界面文案和状态通过检查。

## 验证计划

- 运行前端相关 Vitest 测试。
- 运行项目可用的前端类型检查和构建检查。
- 手工执行分组查看、编辑和账号页面回归。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 类型检查 | `pnpm --dir frontend run typecheck` | 通过 |
| 回归测试 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/groupsModelsListLayout.spec.ts src/views/admin/__tests__/groupsModelsListCandidates.spec.ts src/views/admin/__tests__/groupsModelsList.spec.ts src/views/admin/__tests__/groupsModelRouting.spec.ts src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 5 个测试文件、22 个测试全部通过 |
| 代码检查 | `frontend/src/components/admin/group/GroupConcurrencyViewModal.vue`、`frontend/src/views/admin/GroupsView.vue` | 已确认入口、层级展示、容量规则、刷新生命周期和中英文文案 |

## 执行记录

执行前保持为空；记录实际偏差、阻塞项和决策。
