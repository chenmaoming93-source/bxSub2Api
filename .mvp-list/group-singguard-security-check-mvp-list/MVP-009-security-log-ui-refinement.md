# MVP-009：独立安全日志页面与可读状态/风险维度展示

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: 基于已完成的日志 API 和配置组件，补充独立页面、导航路由、状态映射与风险维度文案，不涉及数据库变更。
- Dependencies: `MVP-008`

## 预期成果

管理员可以从左侧菜单进入独立的安全检查日志页面，在列表中明确区分检查成功、超时、异常和未执行；配置安全检查规则时可以看到五个风险维度的中文名称和含义，同时保留英文 code 作为技术标识。

## 范围内

- 新增 `/admin/security-check-logs` 管理端页面；
- 左侧菜单将安全检查日志放在可配置 Token 统计上方；
- 复用既有日志列表、详情、采集状态和恢复 API；
- 日志列表增加 `check_status` 可读状态列和颜色；
- 日志详情命中规则显示中文名称、含义/技术 code；
- 分组安全检查配置中的五个风险维度显示中文名称和说明；
- 复用现有 `groups.read` 和 `groups.update` 权限；
- 补充路由、菜单和组件验证。

## 范围外

- 安全检查判定逻辑变化；
- `security_check_logs` 表、字段、索引或迁移变化；
- 新增后端日志接口或改变接口响应结构；
- 新增 RBAC 权限；
- 风险维度英文 code 变化。

## 实现说明

- `SecurityCheckLogsView.vue` 使用既有 `listSecurityCheckLogs`、`getSecurityCheckLog`、采集状态和恢复 API；
- 点击“查看详情”后通过 `BaseDialog` 弹窗展示详情，页面底部不再内嵌详情区域；
- 列表不加载请求体和完整 SingGuard 返回体，详情弹窗再加载大字段；
- 状态映射为 `success=检查成功`、`timeout=检查超时`、`error=检查异常`、`skipped=未执行`；
- 风险维度使用前端展示字典，提交给后端的 `dimension` 仍为原英文 code；
- 详情弹窗在存在 `exception_type` 或 `exception_message` 时展示完整异常信息；
- 移除分组管理页面的重复日志入口、弹窗挂载、import 和状态，日志统一从左侧菜单进入。

## 验收标准

- [x] 独立安全日志页面可以通过 `/admin/security-check-logs` 打开。
- [x] 左侧菜单中安全检查日志位于可配置 Token 统计上方。
- [x] 日志表格显示检查状态，并能区分成功、超时、异常和未执行。
- [x] 日志详情和分组日志弹窗中的检查状态使用可读中文名称。
- [x] 点击查看详情时使用独立弹窗，页面下方不再展开详情内容。
- [x] 异常记录详情弹窗显示 `exception_type` 和 `exception_message`。
- [x] 分组页面不再显示或挂载“安全日志”入口，日志统一从左侧菜单进入。
- [x] 五个风险维度显示中文名称和含义，保存时仍提交英文 code。
- [x] 页面和配置入口继续遵守现有 RBAC 权限。
- [x] 前端类型检查、相关测试和生产构建通过。

## 验证计划

- `cd frontend; pnpm run typecheck`
- `cd frontend; pnpm run test:run`
- `cd frontend; pnpm run build`
- 静态检查路由顺序、菜单顺序、状态映射和风险维度 code 映射。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 独立页面 | `frontend/src/views/admin/SecurityCheckLogsView.vue` | 新增 `/admin/security-check-logs` 页面，保留筛选、分页、采集恢复能力；详情使用 `BaseDialog` 弹窗展示异常和大字段信息。 |
| 菜单/权限 | `frontend/src/components/layout/AppSidebar.vue`, `frontend/src/router/index.ts`, `frontend/src/rbac/permissionMatrix.ts`, `frontend/src/views/admin/GroupsView.vue` | 安全检查日志位于可配置 Token 统计上方；分组页面移除重复入口；复用 `groups.read`，恢复操作使用 `groups.update`。 |
| 状态展示 | `frontend/src/views/admin/SecurityCheckLogsView.vue` | 新增 `success/timeout/error/skipped` 中文状态列和颜色。 |
| 异常展示 | `frontend/src/views/admin/SecurityCheckLogsView.vue`, `frontend/src/api/admin/groups.ts` | 详情弹窗显示后端返回的 `exception_type` 和完整 `exception_message`。 |
| 风险维度展示 | `frontend/src/components/admin/group/GroupSecurityCheckModal.vue`, `SecurityCheckLogsView.vue` | 五个风险维度显示中文名称、含义和英文 code；提交值保持英文 code。 |
| 类型检查 | `cd frontend; pnpm run typecheck` | 通过。 |
| 相关测试 | `cd frontend; pnpm exec vitest run src/views/admin/__tests__/SecurityCheckLogsView.spec.ts src/router/__tests__/guards.spec.ts` | 通过，38 tests。 |
| 构建 | `cd frontend; pnpm run build` | 通过。 |

## 执行记录

- 完整 `pnpm run test:run` 仍有仓库既有认证/账户相关测试失败，与本次变更无关；本次新增页面和路由相关测试已单独通过。
- 未修改数据库表、日志 API 响应结构或安全检查判定逻辑。
- 根据后续交互反馈，将详情从页面底部改为 `BaseDialog` 弹窗，详情大字段仍只在打开详情后请求。
- 根据错误排查反馈，详情弹窗增加异常信息区域，并移除分组页面的重复“安全日志”入口。
