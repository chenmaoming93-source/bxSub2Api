# MVP-003：前端基础（类型 + ModelAttributesSection 组件 + 预置清单 + i18n）

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: 1 个开发者日
- Estimate rationale: 涉及类型定义、卡片组件、18 条预置常量与 zh/en 文案、组件单测（覆盖预置/自定义/解析/回显），工作量约一个工作日。
- Dependencies: `none`

## 预期成果

前端具备独立的「模型基本属性」可复用组件：卡片区域 +「添加属性」下拉（18 条预置快捷项 + 自定义行）、行编辑与删除、value 智能解析、map 序列化与回显；类型与文案齐备，组件单测通过。

## 背景

- 项目前端：Vue 3 + TypeScript + vue-i18n + Vitest。
- 相关路径：`frontend/src/types/index.ts`、`frontend/src/components/account/`、`frontend/src/i18n/locales/zh.ts` 与 `en.ts`。
- 数据结构：`Record<string, { description?: string; value?: unknown }>`（map）。
- 后端信任前端：value 由前端解析为 JSON 类型（数字/布尔/数组/对象/字符串）后原样提交。

## 范围内

- `frontend/src/types/index.ts`：新增 `ModelAttributeItem` / `ModelAttributes`（map）类型；`Account`、`CreateAccountRequest`、`UpdateAccountRequest` 增加 `model_attributes` 字段。
- 新增 `frontend/src/components/account/ModelAttributesSection.vue`：
  - 卡片区域（标题「模型基本属性（可选）」+ 说明）；
  - 「添加属性」下拉：预置项（英文名 + 中文描述）+「自定义属性」；已添加的预置项置灰；
  - 每行：key / description / value 输入框 + 删除按钮；value placeholder「可填数字、true/false、文本或 JSON 数组」；
  - 提交前重复 key 检测并阻止；value 输入留空的行不纳入 map；
  - 对外暴露 `modelValue`（map）与 `update:modelValue`。
- 新增预置清单常量（前端）：18 条属性（key + i18n 描述 key）。
- i18n：zh.ts / en.ts 补充组件文案与 18 条预置描述。
- 组件单测：预置项添加、自定义行、重复 key 阻止、value 解析（数字/布尔/数组/字符串）、空 map 提交、map → 行回显往返。

## 范围外

- 创建/编辑弹窗的接入（见 MVP-004）。
- 后端任何改动。
- 预置清单的任何后端同步（仅前端维护）。

## 实现说明

- 组件仅依赖已有通用样式类（`input`、`input-label`、`input-hint`）与 i18n；不引入新的 UI 库。
- value 解析规则：`JSON.parse` 成功则用解析结果，失败则按字符串；显式空字符串 `""` 为合法值（与「输入框留空 = 未填」区分）。
- 行顺序由组件内部数组维护，序列化为 map 时以 key 为准；回显时按 map 的 key 顺序生成行。
- 预置常量建议独立文件（如 `modelAttributePresets.ts`），描述文案用 i18n key，保证 zh/en 一致。

## 验收标准

- [x] `npm run typecheck`（vue-tsc --noEmit）通过
- [x] 组件单测全绿：预置添加/自定义/重复 key 阻止/value 解析/空 map/回显往返（20 用例）
- [x] 组件对外契约（modelValue / update:modelValue）符合设计
- [x] zh.ts 与 en.ts 文案齐全且 key 对齐

## 验证计划

- `cd frontend && npm run typecheck`
- `cd frontend && npx vitest run src/components/account/__tests__/ModelAttributesSection.spec.ts`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 修改 | `frontend/src/types/index.ts` | 新增 `ModelAttributeItem`/`ModelAttributes` 类型；`Account`/`CreateAccountRequest`/`UpdateAccountRequest` 加 `model_attributes` 字段 |
| 新增 | `frontend/src/components/account/modelAttributePresets.ts` | 18 条预置清单（key + i18n descriptionKey） |
| 新增 | `frontend/src/components/account/modelAttributeUtils.ts` | 纯函数：`parseAttributeValue`/`displayAttributeValue`/`buildModelAttributes`/`rowsFromModelAttributes` |
| 新增 | `frontend/src/components/account/ModelAttributesSection.vue` | 卡片区域组件（添加下拉/预置/自定义/行编辑/删除/重复提示），v-model: map 契约 |
| 修改 | `frontend/src/i18n/locales/zh.ts`、`en.ts` | `admin.accounts.modelAttributes` 文案（标题/提示/占位/重复提示 + 18 条预置描述），zh/en 对齐 |
| 新增测试 | `__tests__/modelAttributeUtils.spec.ts` | 解析（数字/布尔/数组/字符串/空串）、回显、构建（空行丢弃/去空白/重复保第一份） |
| 新增测试 | `__tests__/ModelAttributesSection.spec.ts` | 空态/预置添加/预置置灰/自定义/解析输出/重复提示/删除/回显 |
| npm typecheck | `cd frontend && npm run typecheck` | 通过（exit 0） |
| vitest | `npx vitest run .../modelAttributeUtils.spec.ts .../ModelAttributesSection.spec.ts` | 2 文件 20 用例全绿 |

## 执行记录

- 2026-08-05：MVP-003 完成。组件与纯函数分离实现，单测覆盖全部验收场景；typecheck 与测试均通过。
- 组件遵循现有弹窗样式类（`input`/`input-label`/`border-t` 区块风格）与 i18n 惯例；重复 key 由组件内提示并保第一份，符合 plan A-02。
