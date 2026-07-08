# MVP-020：完成缺失默认分组的锁名创建流程

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-019`

## 预期成果

默认模型路由页面可通过复用创建卡片创建名称锁定的默认分组。

## 背景

创建过程需复用现有 Group创建逻辑，避免第二套表单；并发同名冲突后应刷新已有分组。

## 范围内

- 在缺失态提供创建按钮并打开共享创建卡片。
- 锁定名称为 `default_group_name`，其他字段保持原规则。
- 创建成功后刷新并切换至路由编辑态。
- 处理并发同名分组冲突的刷新路径。
- 增加交互测试。

## 范围外

- 自动隐式创建默认分组。

## 实现说明

- 如现有创建卡片不可复用，做最小组件化抽取并保持原页面行为。

## 验收标准

- [x] 名称不可修改，创建成功立即进入编辑态。
- [x] 并发冲突不重复创建且页面能加载已有分组。

## 验证计划

- `cd frontend; pnpm test:run -- DefaultGroupRouting GroupCreate`
- `cd frontend; pnpm typecheck`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 定向测试 | `cd frontend; pnpm exec vitest run src/views/admin/__tests__/DefaultGroupRoutingView.spec.ts` | 通过，1 个测试文件、5 个测试全部通过 |
| 类型检查 | `cd frontend; pnpm typecheck` | 通过 |
| 实现 | `frontend/src/views/admin/DefaultGroupRoutingView.vue` | 缺失态锁定默认分组名称；创建成功或并发同名冲突后刷新进入已有分组编辑态 |
| 交互测试 | `frontend/src/views/admin/__tests__/DefaultGroupRoutingView.spec.ts` | 覆盖锁名、创建成功和并发冲突仅创建一次并刷新已有分组 |

## 执行记录

已完成：缺失态提供名称锁定的默认分组创建流程；创建成功立即进入编辑态，并发同名冲突时不重复创建而是刷新加载已有分组。
