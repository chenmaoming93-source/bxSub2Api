# MVP-018：接入默认分组前端 API 与系统设置

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-004`

## 预期成果

管理员可在现有系统设置页面配置默认分组名并看到是否存在。

## 背景

前端设置 API位于 `frontend/src/api/admin/settings.ts`，主页面为 `SettingsView.vue`。

## 范围内

- 新增默认分组状态与更新 API类型及客户端。
- 在系统设置页增加默认分组名输入、状态提示与保存逻辑。
- 展示存在/不存在，不因缺失禁止保存。
- 增加 API和页面测试及 i18n文案。

## 范围外

- 默认模型路由独立页面。

## 实现说明

- 沿用现有设置表单交互和错误提示风格。

## 验收标准

- [x] 管理员可保存名称并看到后端返回的存在状态。
- [x] 前端测试和类型检查通过。

## 验证计划

- `cd frontend; pnpm test:run -- settings DefaultGroup`
- `cd frontend; pnpm typecheck`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 聚焦测试 | `cd frontend; pnpm exec vitest run src/api/__tests__/admin.defaultGroup.spec.ts src/components/admin/group/__tests__/DefaultGroupSettingsCard.spec.ts src/views/admin/__tests__/SettingsView.spec.ts` | 通过，3 个文件、22 个测试全部成功。 |
| 类型检查 | `cd frontend; pnpm typecheck` | 通过，`vue-tsc --noEmit` 退出码 0。 |
| API 客户端 | `frontend/src/api/admin/settings.ts` | 新增默认分组状态类型、GET 状态与 PUT 更新方法。 |
| 设置页面 | `frontend/src/components/admin/group/DefaultGroupSettingsCard.vue`、`frontend/src/views/admin/SettingsView.vue` | 通用设置页可加载、保存名称并展示存在/缺失状态。 |
| i18n | `frontend/src/i18n/locales/en.ts`、`frontend/src/i18n/locales/zh.ts` | 已增加中英文标题、说明、状态和保存提示。 |

## 执行记录

实现与验证完成。默认分组使用独立 API 保存，不混入庞大的通用设置提交载荷；后端返回 `exists=false` 时仍保留名称并显示提示。
