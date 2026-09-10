# MVP-003：接入并发查看刷新生命周期

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 天`
- Estimate rationale: 在已完成卡片上增加复用式轮询、手动刷新和关闭清理，属于独立的运行时行为切片。
- Dependencies: `MVP-002`

## 预期成果

并发查看卡片打开时加载并发数据并按账号页面相同节奏自动刷新，关闭时停止刷新和请求，且不影响编辑分组或账号页面。

## 背景

`admin/accounts` 使用前端定时轮询、后端读取 Redis 当前槽位数量的方式展示当前并发。本功能复用刷新节奏，不引入 WebSocket。

## 范围内

- 复用现有刷新间隔和刷新状态模式。
- 卡片打开启动刷新，关闭停止刷新。
- 手动刷新。
- 请求并发控制和未完成请求清理。
- 只更新当前并发数据，不重建路由结构。

## 范围外

- 不改账号页面的刷新实现。
- 不改后端并发槽位算法。
- 不新增 WebSocket。

## 实现说明

- 参考 `frontend/src/views/admin/AccountsView.vue` 的自动刷新和增量更新逻辑。
- 通过独立组件状态隔离刷新任务。
- 组件卸载和卡片关闭时清理定时器、监听器及请求。

## 验收标准

- [x] 卡片打开后按既有节奏刷新当前并发。
- [x] 支持手动刷新。
- [x] 刷新不覆盖路由和候选结构。
- [x] 卡片关闭后不再发起刷新请求。
- [x] 请求失败时保留结构并展示状态。
- [x] 编辑分组和 `admin/accounts` 行为不受影响。

## 验证计划

- 运行刷新相关组件测试。
- 使用定时器 mock 验证启动、刷新和清理。
- 运行受影响的前端测试和类型检查。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 类型检查 | `pnpm --dir frontend run typecheck` | 通过 |
| 测试 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/groupsModelRouting.spec.ts src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 2 个测试文件、10 个测试全部通过 |
| 实现路径 | `frontend/src/components/admin/group/GroupConcurrencyViewModal.vue` | 使用 `useAutoRefresh`，打开/关闭控制刷新生命周期 |

## 执行记录

执行前保持为空；记录实际偏差、阻塞项和决策。
