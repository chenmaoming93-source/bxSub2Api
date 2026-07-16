# MVP-003：实现先选账号再选上游模型的基础交互

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `限定为单账号成功路径和模板顺序调整，复用已完成的 API 与交集函数，避免混入缓存和异常状态。`
- Dependencies: `MVP-001, MVP-002`

## 预期成果

模型路由候选先显示账号选择；没有账号时模型选择禁用，选择一个账号并加载成功后可从下拉框选择模型，不再允许手填。

## 背景

当前 `GroupModelRoutingEditor.vue` 先提供自由文本模型输入，再搜索账号。本 MVP 建立新的主流程，但暂不覆盖多账号并发与失败恢复。

## 范围内

- 调整候选区域控件顺序，账号选择位于模型选择之前。
- 将上游模型文本框替换为下拉选择控件。
- 无账号时禁用模型下拉框。
- 单账号选择后调用账号模型 API 并展示选项。
- 保持 `candidate.model` 的保存字段不变。
- 增加组件成功路径测试。

## 范围外

- 不实现多账号缓存、并发请求和失败重试。
- 不修改持久化结构。
- 不处理最终中英文文案完善。

## 实现说明

- 使用候选稳定 Key 保存临时模型状态。
- 下拉值绑定模型 `id`，标签使用 `display_name || id`。
- 组件卸载时不得留下会更新已销毁状态的请求。

## 验收标准

- [x] 页面顺序为账号选择在前、上游模型在后。
- [x] 未选择账号时模型控件禁用且不能手填。
- [x] 选择单个账号后展示该账号模型，并能写入 `candidate.model`。
- [x] 提交数据中的 `model` 和 `account_ids` 结构未变化。

## 验证计划

- `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `frontend/src/components/admin/group/GroupModelRoutingEditor.vue` | 账号控件前置；模型自由文本框改为按账号加载的下拉框；卸载及候选删除会取消请求。 |
| 测试 | `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 通过：1 个测试文件、6 个测试，含单账号加载、禁用状态与字段写回。 |
| 类型检查 | `pnpm --dir frontend run typecheck` | 通过：`vue-tsc --noEmit` 退出码 0。 |

## 执行记录

已完成单账号成功路径，保留 `candidate.model` 与 `candidate.accounts` 数据结构。
