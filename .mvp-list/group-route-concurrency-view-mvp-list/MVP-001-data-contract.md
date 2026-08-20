# MVP-001：确认并发查看数据契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 天`
- Estimate rationale: 聚焦现有接口、类型和组件数据流确认，形成后续实现可直接使用的数据契约。
- Dependencies: `none`

## 预期成果

完成分组路由候选并发查看所需的数据契约确认，并在代码中补齐必要的类型/API封装；后续卡片可以按“路由别名—候选—账号”获得展示数据。

## 背景

现有分组路由编辑器已有候选账号和候选级最大并发配置，账号列表接口已有 `current_concurrency`。需要确认并发查看功能的最小数据来源，避免修改账号页面逻辑。

## 范围内

- 检查现有分组路由配置和账号并发接口。
- 定义并发查看所需的前端类型或数据转换边界。
- 明确候选最大并发未配置时使用 `∞` 的展示语义。
- 为后续卡片提供稳定的数据组装入口。

## 范围外

- 不实现并发查看卡片界面。
- 不实现自动刷新。
- 不修改账号列表接口的既有行为。

## 实现说明

- 重点检查 `frontend/src/views/admin/GroupsView.vue`、`frontend/src/components/admin/group/GroupModelRoutingEditor.vue`、`frontend/src/api/admin/groups.ts`、`frontend/src/api/admin/accounts.ts`。
- 优先复用已有路由引用查询和账号并发字段。
- 如需新增接口，仅增加并发查看所需的最小读取能力。

## 验收标准

- [x] 已确认路由别名、候选顺序、候选账号和候选上限的来源。
- [x] 已确认账号当前并发量的读取方式与 `admin/accounts` 一致。
- [x] 已形成可被卡片复用的类型或数据转换入口。
- [x] 相关类型检查或目标测试通过。

## 验证计划

- `rg` 检查数据字段和调用点。
- 运行受影响的前端类型检查或相关测试命令。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/groupsModelRouting.spec.ts src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 2 个测试文件、10 个测试全部通过 |
| 代码核查 | `frontend/src/types/index.ts`、`frontend/src/api/admin/groups.ts`、`frontend/src/api/admin/accounts.ts`、`frontend/src/views/admin/GroupsView.vue` | 已确认路由配置、候选上限、账号并发字段和现有接口 |

## 执行记录

执行前保持为空；记录实际偏差、阻塞项和决策。
