# MVP-002：实现可复用的多账号模型交集逻辑

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `聚焦纯 TypeScript 数据标准化、去重、交集与排序，测试无需挂载 Vue 组件，适合一个短周期完成。`
- Dependencies: `MVP-001`

## 预期成果

给定一个或多个账号的模型列表，可以稳定计算以模型 `id` 为准的去重交集，并生成可供下拉框使用的选项。

## 背景

路由候选支持多个账号，只有全部账号共同支持的模型才能成为合法上游模型。展示名可选，业务相等性必须严格使用模型 `id`。

## 范围内

- 在 `groupModelRoutingEditor.ts` 或相邻纯逻辑模块实现模型标准化和交集函数。
- 支持单账号、多账号、重复模型、空列表和无交集场景。
- 结果按模型 `id` 稳定排序。
- 增加纯函数单元测试。

## 范围外

- 不发起 HTTP 请求。
- 不修改 Vue 模板。
- 不处理加载失败或缓存状态。

## 实现说明

- 保留第一个可用的 `display_name` 仅用于展示，不参与相等性判断。
- 任一已选账号返回空模型列表时，整体交集为空。
- 输入不得被原地修改。

## 验收标准

- [x] 单账号结果正确去重和排序。
- [x] 多账号只返回所有账号共有的模型 ID。
- [x] 空账号集合、空模型列表和无交集均返回明确的空数组。
- [x] 原始输入保持不变，纯函数测试全部通过。

## 验证计划

- `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `frontend/src/components/admin/group/groupModelRoutingEditor.ts` | 新增 `intersectAccountModels`，按 ID 标准化、去重、求交并稳定排序。 |
| 测试 | `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 通过：1 个测试文件、5 个测试。 |

## 执行记录

已覆盖单账号、多账号、重复项、空输入、空列表、无交集和输入不可变场景。
