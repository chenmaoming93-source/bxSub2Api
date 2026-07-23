# MVP-004：完善多账号模型加载、缓存与重新校验

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `在已有基础交互上集中补齐一个候选的多账号状态机、缓存和失效处理，范围可通过组件测试隔离。`
- Dependencies: `MVP-003`

## 预期成果

选择多个账号时，下拉框只显示模型交集；账号增删会重新计算，模型加载失败或历史模型失效时不会产生错误保存。

## 背景

多账号是现有路由结构的兼容要求。任一账号模型未知时不能用部分结果冒充完整交集，相同账号也不应在一次编辑会话中重复请求。

## 范围内

- 并行加载新增账号的模型列表。
- 在编辑会话内按账号 ID 缓存成功结果。
- 计算所有已选账号的模型交集。
- 账号增删后重新计算并校验 `candidate.model`。
- 任一请求失败时禁用模型选择并提供可恢复路径。
- 无交集时禁用选择并暴露明确状态。
- 取消组件卸载或候选删除后的失效请求。

## 范围外

- 不修改后端账号模型接口。
- 不增加跨页面或持久化缓存。
- 不完成最终 i18n 文案。

## 实现说明

- 成功缓存与失败状态分离；失败不得缓存成永久空列表。
- 请求返回时校验候选与账号仍然有效，防止竞态覆盖。
- 历史模型若仍在交集内应保留，否则清空并标记需重选。

## 验收标准

- [x] 多账号下拉选项等于模型 ID 交集。
- [x] 相同账号在同一编辑会话内不重复请求。
- [x] 删除账号后交集按剩余账号立即更新。
- [x] 任一账号失败或无交集时不能保存无效模型。
- [x] 失效响应不会覆盖较新的账号选择状态。

## 验证计划

- `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `frontend/src/components/admin/group/GroupModelRoutingEditor.vue` | 增加会话级成功缓存、共享进行中请求、并行加载、交集重算、失败重试与版本化竞态保护。 |
| 测试 | `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 通过：1 个测试文件、9 个测试，覆盖交集、缓存、账号删除、失败重试和旧响应隔离。 |
| 类型检查 | `pnpm --dir frontend run typecheck` | 通过：`vue-tsc --noEmit` 退出码 0（新增竞态测试后生产类型未变化）。 |

## 执行记录

已完成多账号状态管理；失败不会写入成功缓存，组件卸载会取消仍在进行的账号模型请求。
