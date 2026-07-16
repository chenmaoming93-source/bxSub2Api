# MVP-005：完成路由编辑器校验、文案与兼容性测试

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `集中处理用户可见边界、保存阻断和中英文文案，并以聚焦回归测试收口前端需求。`
- Dependencies: `MVP-004`

## 预期成果

管理员能清晰识别加载中、加载失败、无共同模型和历史模型失效状态，且所有无效候选均无法通过现有保存校验。

## 背景

基础交互完成后仍需将状态转化为明确反馈，并保证现有优先级、每日限额和模型路由序列化行为不退化。

## 范围内

- 添加必要的中英文 i18n 文案。
- 完善加载中、失败、空交集和需重新选择提示。
- 将模型是否属于当前交集纳入候选有效性判断。
- 保留优先级和每日 Token 限额原有校验。
- 增加历史配置有效/失效、空交集和错误状态测试。
- 运行前端类型检查或构建验证。

## 范围外

- 不改动其他分组表单功能。
- 不修改模型路由数据结构。
- 不实施后端新查询接口。

## 实现说明

- 中英文 locale Key 保持结构对称。
- 不因加载尚未完成而静默清除历史模型；只有取得完整交集后才能判断失效。
- 保存失败提示应指向账号或模型问题，而非笼统报错。

## 验收标准

- [x] 所有新增状态均有中英文用户提示。
- [x] 模型不在完整交集内时保存校验失败。
- [x] 合法历史配置正常回显，失效配置要求重选。
- [x] 优先级、限额和提交结构现有测试保持通过。
- [x] 前端聚焦测试和类型检查通过。

## 验证计划

- `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/i18n/__tests__/modelTokenQuotaLocales.spec.ts`
- `pnpm --dir frontend run build`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `frontend/src/components/admin/group/GroupModelRoutingEditor.vue`、中英文 locale、两个保存入口 | 加载、失败、空交集和失效提示齐备；编辑器暴露完整有效性并阻断无效保存。 |
| 聚焦测试 | `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/i18n/__tests__/modelTokenQuotaLocales.spec.ts` | 通过：2 个测试文件、13 个测试。 |
| 构建 | `pnpm --dir frontend run build` | 通过：`vue-tsc -b` 与 Vite production build 均成功；仅有既有 chunk/dynamic import 警告。 |

## 执行记录

前端分支已收口；合法历史模型保留，完整交集确认失效后清空并显示重选提示，优先级、限额和序列化字段保持不变。
