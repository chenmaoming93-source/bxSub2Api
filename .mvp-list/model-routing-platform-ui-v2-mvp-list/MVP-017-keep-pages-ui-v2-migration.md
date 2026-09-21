# MVP-017：迁移保留业务页面到 UI V2

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 将业务整体可保留的页面按模块迁移，保持接口和行为不变；如实际范围超出则按页面拆分。
- Dependencies: `MVP-002, MVP-003, MVP-004`

## 预期成果

账号管理、分组管理、角色权限、系统设置和可配置 Token 统计使用 UI V2，业务逻辑不变。

## 背景

这些页面已有完整 CRUD、表单、弹窗和接口，适合整体保留。旧页面和旧组件继续存在。

## 范围内

- Part 3 账号管理 UI 迁移。
- Part 4 分组管理 UI 迁移。
- Part 5 角色权限 UI 迁移。
- Part 7 系统设置 UI 迁移。
- Part 8 Token 统计 UI 迁移。
- 保持 API、权限、校验、slot、分页、导入导出和弹窗行为。

## 范围外

- 不改变业务规则和接口契约。
- 不迁移停用页面。
- 不删除旧组件。

## 实现说明

优先替换基础控件和布局；复杂业务组件保留数据和事件逻辑，只换视觉外壳。默认路由编辑组件仍保留在分组管理。

## 验收标准

- [x] 五类页面核心 CRUD 和查询行为不变。
- [x] 页面使用 UI V2 框架和控件。
- [x] 现有权限和错误处理不退化。
- [x] 旧页面与旧组件源文件仍存在。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- `pnpm --dir frontend build`
- 按页面执行 CRUD、权限、弹窗和暗色模式人工回归。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Wrapper tests | `pnpm run test:run -- src/views/admin/__tests__/keep-pages-v2.test.ts` | 6 个测试通过 |
| Routes | `frontend/src/router/index.ts` | 五类页面切换到 V2 wrapper |
| Legacy preservation | `frontend/src/views/admin/{Accounts,Groups,Roles,Settings,TokenStatistics}View.vue` | 旧页面保留并支持 embedded |

## 执行记录

账号、分组、角色权限、系统设置和 Token 统计均接入 `UiLegacyPageFrame`，复杂 CRUD 内容通过 embedded 模式保留。
