# MVP-019：建设默认模型路由页面基础态

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-004, MVP-017, MVP-018`

## 预期成果

管理员可进入独立页面，在未配置、分组缺失和分组存在三种状态间获得明确反馈。

## 背景

新增管理路由与页面，复用共享路由编辑器和默认分组状态 API。

## 范围内

- 新增页面、管理员路由和导航入口。
- 实现未配置时前往设置引导。
- 实现缺失时提示默认分组不存在。
- 存在时加载并保存模型路由字段。
- 增加三态页面测试。

## 范围外

- 缺失分组创建卡片。

## 实现说明

- 保存优先复用现有分组更新 API，只提交路由相关字段。

## 验收标准

- [x] 三种状态渲染正确，存在态可保存路由。
- [x] 页面测试、导航测试与类型检查通过。

## 验证计划

- `cd frontend; pnpm test:run -- DefaultGroupRouting navigation`
- `cd frontend; pnpm typecheck`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 页面与导航测试 | `cd frontend; pnpm exec vitest run src/views/admin/__tests__/DefaultGroupRoutingView.spec.ts src/__tests__/integration/navigation.spec.ts src/router/__tests__/guards.spec.ts` | 通过，3 个文件、47 个测试全部成功。 |
| 类型检查 | `cd frontend; pnpm typecheck` | 通过，`vue-tsc --noEmit` 退出码 0。 |
| 页面实现 | `frontend/src/views/admin/DefaultGroupRoutingView.vue` | 覆盖未配置、分组缺失、分组存在三态；存在态加载共享编辑器并保存路由字段。 |
| 路由与导航 | `frontend/src/router/index.ts`、`frontend/src/components/layout/AppSidebar.vue` | 新增管理员路由 `/admin/default-group-routing` 及侧边栏入口。 |
| i18n | `frontend/src/i18n/locales/en.ts`、`frontend/src/i18n/locales/zh.ts` | 新增页面、三态、保存与导航中英文文案。 |

## 执行记录

实现与验证完成。未配置态引导前往设置，缺失态仅提示且不创建分组，存在态只提交 `model_routing_enabled` 与 `model_routing`。
