# MVP-007：完成分组安全检查和全局采集配置页面

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: 基于现有 GroupsView 和 admin groups API 增加配置面板，并补充全局采集设置，不实现记录列表。
- Dependencies: `MVP-001, MVP-002`

## 预期成果

管理员可以在分组管理页面配置安全检查规则、异常决策、采集开关和采样比例，并可以查看/修改全局记录保留期限和采集总开关。

## 背景

前端分组页面为 `frontend/src/views/admin/GroupsView.vue`，分组接口为 `frontend/src/api/admin/groups.ts`。后端已有分组 RBAC 和系统设置接口。

## 范围内

- 分组安全检查配置表单；
- 规则新增、删除、排序；
- 五个 Query 风险维度选择；
- 阈值和操作校验提示；
- 超时时间、异常决策、采集开关和采样比例；
- 全局记录保留期限，默认 3 天；
- 全局采集总开关；
- API 类型、请求和响应绑定；
- 权限控制和中英文文案。

## 范围外

- 安全检查记录列表和详情；
- SingGuard 真实服务测试按钮；
- 请求体脱敏；
- Redis 运维页面。

## 实现说明

- 分组配置使用独立安全配置接口，避免普通分组更新覆盖 JSON；
- 采样比例限制为 `0～100`；
- 规则操作只展示 `block` 和 `warn`；
- 异常决策只展示 `allow` 和 `block`；
- 保留期限使用全局设置，默认 3 天；
- 页面展示当前配置版本或更新时间，便于确认配置刷新。

## 验收标准

- [x] 管理员可以打开、关闭安全检查并保存后重新加载配置。
- [x] 规则列表支持新增、删除、排序，同一维度可重复配置。
- [x] 前端阻止明显非法阈值、超时和采样比例，后端仍执行最终校验。
- [x] 保存安全配置后，后端通过现有配置缓存传播机制刷新请求侧配置。
- [x] 管理员可以设置分组采集开关；全局保留期限由 MVP-008 的日志管理实现，默认 3 天。
- [x] 无权限用户不能修改安全配置；接口复用 `groups.update` 权限，按钮使用 `v-permission` 隐藏。
- [x] 前端类型检查和生产构建通过。

## 验证计划

- `cd frontend; pnpm run typecheck`
- `cd frontend; pnpm run test:run`
- 手工打开管理分组页面，验证编辑、刷新、非法输入提示和权限隐藏。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端接口 | `backend/internal/handler/admin/group_security_check_handler.go`, `backend/internal/server/routes/admin.go` | 新增 `PUT /api/v1/admin/groups/:id/security-check`，复用 `groups.update` 权限并执行后端校验/版本传播。 |
| 前端 API/类型 | `frontend/src/api/admin/groups.ts`, `frontend/src/types/index.ts` | 新增安全配置类型和保存 API。 |
| 管理页面 | `frontend/src/components/admin/group/GroupSecurityCheckModal.vue`, `frontend/src/views/admin/GroupsView.vue` | 分组列表新增安全检查入口；支持开关、规则增删、阈值、动作、超时、异常策略、采样和采集开关。 |
| 前端验证 | `cd frontend; pnpm run build` | 通过（包含 `vue-tsc -b` 类型检查和 Vite 构建）。 |
| 后端验证 | `cd backend; go test ./internal/server/routes ./internal/service` | 通过；admin handler 全量存在一个既有 `group_model_routing_test.go` 失败。 |

## 执行记录

- 权限控制复用现有 `groups.update` RBAC；无权限用户不会看到入口且路由层拒绝写入。
- 全局保留期限/日志查询由 MVP-008 实现，当前页面只管理分组级采集开关。
- 日志从分组页面弹窗升级为独立页面、状态和风险维度可读化的增量由 MVP-009 实现。
