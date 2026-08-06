# MVP-004：弹窗集成（创建/编辑账户弹窗接入模型属性卡片）

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: 1 个开发者日
- Estimate rationale: 两个弹窗体积大、平台分支多，需在每处「模型限制（可选）」区块下方接入组件并打通回显/打包/提交，工作量约一个工作日。
- Dependencies: `MVP-003`

## 预期成果

`/admin/accounts` 的创建账户与编辑账户弹窗中，在「模型限制（可选）」区块正下方出现「模型基本属性（可选）」卡片：创建时可配置，编辑时回显并可修改/清空；提交载荷包含 `model_attributes`（map），一行不填则传 `{}`。

## 背景

- 相关路径：`frontend/src/components/account/CreateAccountModal.vue`、`EditAccountModal.vue`。
- 两个弹窗均为平台/类型分支结构，「模型限制（可选）」区块存在多处（如 apikey、openai oauth、service_account、bedrock 等分支），需在每处区块结束后插入组件。
- 编辑回显数据源：`Account.model_attributes`（后端响应，可能为 null/undefined → 按空配置处理）。
- 创建弹窗同样需要该区块（用户明确要求）。

## 范围内

- `CreateAccountModal.vue`：每处「模型限制（可选）」区块下方插入 `ModelAttributesSection`；提交时构建 `model_attributes` 并入请求体（空配置传 `{}`）。
- `EditAccountModal.vue`：每处「模型限制（可选）」区块下方插入组件；打开弹窗时从 `props.account.model_attributes` 回显；提交时打包 `model_attributes`；未配置时显示空卡片。
- 与现有保存逻辑（credentials / extra 的合并语义）互不干扰：`model_attributes` 为独立顶层字段。
- 手工冒烟：创建（带属性/不带属性）→ 重新打开回显 → 编辑保存 → 清空保存。

## 范围外

- BulkEditAccountModal（批量编辑）不在本期范围。
- 后端任何改动。
- 模型属性的任何运行时/网关消费。

## 实现说明

- 提交载荷构造可复用 MVP-003 组件的序列化输出；清空 = 提交空 map `{}`（后端按覆盖清空处理）。
- 编辑回显需容错：`account.model_attributes` 为 undefined/null 时按空配置。
- 两个弹窗各自维护一份组件引用即可（组件内部状态自包含），注意在打开弹窗时同步回显，关闭/切换账户时重置。

## 验收标准

- [x] `npm run typecheck`（vue-tsc --noEmit）通过
- [x] 创建弹窗：所有 5 处 create 调用点（含 `submitCreateAccount` 统一入口）提交载荷均含 `model_attributes`；不填时传 `{}`（组件空态输出空 map）
- [x] 编辑弹窗：打开回显已存属性（含 value 类型）；提交载荷含 `model_attributes`（新增 2 个集成测试覆盖回显与提交）
- [x] 现有弹窗测试（EditAccountModal 等 account 组件 spec）不回归（9 文件 81 用例全绿）
- [x] 手工冒烟项已按用户指示移除：用户无法提供运行环境（前端 dev server / MySQL 测试库），该人工验证取消，改为以自动化测试（typecheck + 组件集成测试 + MVP-002 handler 测试）覆盖端到端链路

## 验证计划

- `cd frontend && npm run typecheck`
- `cd frontend && npx vitest run src/components/account/__tests__/`
- 手工冒烟（需后端 MVP-001/MVP-002 已就绪）：创建含属性账户 → 编辑回显 → 修改/清空保存

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 修改 | `frontend/src/components/account/CreateAccountModal.vue` | import 组件/类型；`modelAttributes` ref + `resetForm` 重置；5 处 create 调用点注入 `model_attributes`；4 处「模型限制」下方插入组件（含分支 v-if） |
| 修改 | `frontend/src/components/account/EditAccountModal.vue` | import 组件/类型；`modelAttributes` ref；`syncFormFromAccount` 回显；`handleSubmit` 提交载荷加 `model_attributes`；5 处「模型限制」下方插入组件（含分支 v-if） |
| 修改 | `__tests__/EditAccountModal.spec.ts` | 新增 2 用例：回显（2 行渲染、value 文本正确）、提交载荷含 `model_attributes` |
| npm typecheck | `cd frontend && npm run typecheck` | 通过（exit 0） |
| vitest | `npx vitest run src/components/account/__tests__/` | 9 文件 81 用例全绿（含新增 2 用例） |

## 执行记录

- 2026-08-05：MVP-004 实现完成。两个弹窗共 9 处「模型限制（可选）」区块下方接入 `ModelAttributesSection`（创建 4 处 + 编辑 5 处）。
- **实现中发现并修复的 bug**：初始插入时有 5 处组件位于模板顶层（无 v-if 保护），会导致所有账户类型都渲染多余卡片；已为这些插入点补充对应平台/类型分支条件（EditAccountModal 3 处、CreateAccountModal 2 处），并有测试锁定。
- 创建弹窗无既有 spec 基础，未新增组件级测试；其请求体 `model_attributes` 链路由 MVP-002 handler 测试（Create 透传）与本处代码审查共同覆盖。
- **阻塞解除（用户指示 2）**：手工冒烟需要运行环境（前端 dev server / MySQL 测试库），用户无法提供，已按用户指示移除该人工验证项（验收标准中对应条目已调整），以自动化测试覆盖端到端链路。
- **用户反馈 bug 修复（2026-08-05 追加）**：用户反馈「模型属性卡片 UI 为英文 + 添加属性按钮无效」。①「英文」为 dev server 运行时 locale 消息缓存所致（`loadedLocales` 无 HMR 处理，改动前 zh 已加载，新 key 回退 fallbackLocale=en），刷新页面恢复；②「按钮无效」为下拉 `absolute` 定位被 `modal-body`（`overflow-y-auto`）裁剪，已改为**内联展开面板**。
- **用户反馈 bug 修复 2（2026-08-05 追加）**：用户反馈「点击添加属性后不新增输入行」。根因：**v-model 回写循环**——添加行 → 组件 `emit` → 父组件 `v-model` 回写 `modelAttributes.value` → 组件 `watch` 触发重建行数组 → 新行被清掉（此前单测无真实父组件回写，未暴露）。修复：`watch` 改为**内容比较**（`JSON.stringify` 与当前行构建结果对比），父回写内容一致时跳过重建；新增「模拟真实 v-model 父组件回写」回归测试。修复后 account 组件测试 9 文件 82 用例全绿、typecheck 通过。
