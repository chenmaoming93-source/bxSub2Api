# MVP-001：提供账号可用模型的前端 API 客户端

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `只增加一个已有后端接口的类型化前端封装及聚焦测试，边界单一，可在目标时间内独立验证。`
- Dependencies: `none`

## 预期成果

前端可通过一个类型明确、支持取消请求的方法取得指定账号的可用模型，为模型路由联动提供稳定数据源。

## 背景

后端已注册 `GET /api/v1/admin/accounts/:id/models`；当前 `frontend/src/api/admin/accounts.ts` 尚无对应的专用调用方法。返回项至少保证模型 `id`，`display_name` 为可选展示信息。

## 范围内

- 在账号 Admin API 中定义可用模型的最小前端类型。
- 增加按账号 ID 查询模型的方法，并支持 `AbortSignal`。
- 确保该方法通过现有 `adminAPI.accounts` 导出。
- 增加针对请求路径、返回数据和取消参数的聚焦测试。

## 范围外

- 不修改后端账号模型接口。
- 不实现模型交集或路由编辑器 UI。
- 不调整管理员 JWT 鉴权。

## 实现说明

- 主要位置：`frontend/src/api/admin/accounts.ts`、`frontend/src/api/admin/index.ts` 及相应测试。
- 只强依赖 `id: string`；不同平台的其他字段保持可选。
- 沿用 `apiClient` 已有响应解包约定。

## 验收标准

- [x] 可调用 `adminAPI.accounts` 中的类型化方法查询 `/admin/accounts/:id/models`。
- [x] 调用方可传入 `AbortSignal`，且返回模型项至少具有 `id`。
- [x] 聚焦测试验证路径、账号 ID 和响应解包行为。

## 验证计划

- `pnpm --dir frontend exec vitest run src/api/admin/__tests__/accounts.spec.ts`
- `pnpm --dir frontend exec vue-tsc --noEmit -p tsconfig.app.json`（若仓库配置不接受该参数，则执行 `pnpm --dir frontend run build` 并记录替代命令）

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `frontend/src/api/admin/accounts.ts` | 新增最小 `AvailableAccountModel` 类型、取消参数和兼容旧调用方的重载。 |
| 测试 | `pnpm --dir frontend exec vitest run src/api/admin/__tests__/accounts.spec.ts` | 通过：1 个测试文件、1 个测试。 |
| 类型检查 | `pnpm --dir frontend run typecheck` | 通过：`vue-tsc --noEmit` 退出码 0。 |

## 执行记录

已完成账号模型 API 客户端增强；保留无选项调用的 `ClaudeModel[]` 返回类型，避免现有账号页面类型回归。
