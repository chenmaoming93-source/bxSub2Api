# MVP-003：候选页面分时段并发配置编辑区

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 在已有候选最大并发配置下增加一个局部编辑区，主要工作是表单状态、时间输入、校验和 API 联调。
- Dependencies: `MVP-002`

## 预期成果

候选账号页面在现有“最大并发数”配置下方展示“分时段并发配置”，用户可以按分钟新增、编辑、删除每日区间并保存；未配置时明确显示使用旧默认值。

## 背景

前端为 Vue/TypeScript，已有分组路由候选并发配置页面和 admin API 封装。该功能必须保持旧字段配置入口和行为不变，分时段编辑是附加区域。

## 范围内

- 在候选最大并发设置下方增加分时段列表和编辑控件。
- 支持 `HH:mm` 分钟级输入、`24:00` 作为结束时间、正整数和“不限制”。
- 前端保存前提示反向/重叠区间，显示未覆盖全天是允许的。
- 调用 MVP-002 查询与整体替换 API；空列表可清除全部分时段配置。
- 增加必要的中英文/中文文案、加载/保存/错误状态和组件测试。

## 范围外

- 不在浏览器或请求路径执行每分钟刷新逻辑。
- 不重写现有默认最大并发控件。
- 不实现 Redis 刷新按钮；立即刷新入口在 MVP-006 完成。

## 实现说明

- 前端表单内部可使用分钟整数，展示层转换为 `HH:mm`；结束分钟 `1440` 显示为 `24:00`。
- 同一候选保存采用整列表提交，避免逐行保存造成短暂不一致。
- “未配置分时段，使用默认最大并发”只表达当前配置状态，不把默认值复制为新行。

## 验收标准

- [x] 页面能在默认并发控件下方加载、编辑、删除和保存分时段配置。
- [x] 相邻区间可保存，重叠/非法区间在提交前被阻止；未覆盖全天可保存。
- [x] 数值和“不限制”都能正确显示并往返 API；空配置恢复默认逻辑提示。
- [x] 现有候选默认并发配置测试/类型检查不回归。

## 验证计划

- `pnpm --dir frontend run typecheck`
- `pnpm --dir frontend exec vitest run <新增或相关组件测试文件>`
- 必要时运行 `pnpm --dir frontend run lint:check` 并手工验证候选页面。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API 封装 | `frontend/src/api/admin/groups.ts` | 新增候选分时段查询和整体替换方法，使用 `null` 往返 unlimited |
| 页面组件 | `frontend/src/components/admin/group/ModelRouteConcurrencyScheduleEditor.vue` | 在现有默认并发控件下方提供分钟级区间、数值/不限制、增删和保存 |
| 校验 | `frontend/src/components/admin/group/modelRouteConcurrencySchedule.ts` | 已覆盖 `00:00`、`24:00`、相邻/不完整覆盖、重叠及正整数校验 |
| 测试 | `pnpm --dir frontend exec vitest run src/components/admin/group/__tests__/ModelRouteConcurrencyScheduleEditor.spec.ts src/components/admin/group/__tests__/modelRouteConcurrencySchedule.spec.ts src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts` | 通过：3 个测试文件、12 个测试 |
| 类型检查 | `pnpm --dir frontend run typecheck` | 通过 |

## 执行记录

- 2026-08-18：分时段配置采用候选级独立保存按钮；这样新建分组尚未获得 group ID 时不会写入孤儿配置，编辑已有分组时仍可在默认并发字段下方立即配置。
- 2026-08-18：前端使用 `null` 表示“不限制”，后端/Redis 层继续保留 `unlimited` 的既有字符串语义；区间输入使用文本框以支持 `24:00` 结束边界。
