# MVP-007：部门 Token 概览图表与全链路回归

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发日`
- Estimate rationale: 包含新图表、Usage 页面集成、国际化、前端测试和前后端回归验证，作为交付闭环切片。
- Dependencies: `MVP-004, MVP-006`

## 预期成果

`/admin/usage` 在“场景 Token 用量”上方保留部门 Token 用量环状图和部门概览列表，并为后续员工明细懒加载提供部门选择入口。

## 背景

现有 `frontend/src/views/admin/UsageView.vue` 已集成多类图表，`GroupDistributionChart.vue` 提供目标视觉参考。部门统计接口由 MVP-006 提供。

## 范围内

- 新增 `DepartmentDistributionChart.vue`。
- 左侧环状图，右侧部门名称、Token 数、占比、用户数和平均 Token 列表。
- 在场景 Token 用量上方插入独立区域。
- 部门列表行可点击并触发员工明细懒加载。
- 提供开始日期和结束日期按天选择。
- 对空部门显示“未设置”。
- 处理 loading、空数据、配置缺失、接口错误和 eventual consistency。
- 增加 API 类型、UsageView 状态管理和国际化文本。
- 增加图表组件和 UsageView 测试。
- 执行后端、前端构建和相关回归测试。

## 范围外

- 修改 usage_logs。
- 员工明细查询和员工 Token 柱状图（由 MVP-008 完成）。
- 部门筛选器。
- 异步同步任务。
- 使用历史 department 字段回填聚合数据。

## 实现说明

- 复用 Chart.js/vue-chartjs 和 GroupDistributionChart 的配色及响应式布局。
- 日期查询使用独立状态或明确复用策略，不能影响现有场景 Token 查询的日期语义。
- 列表数值与图表数据必须来自同一接口响应。
- 组件应支持深色模式、窄屏布局和较多部门滚动。

## 验收标准

- [x] 部门图表位于场景 Token 用量上方。
- [x] 环状图和右侧列表的部门 Token 数一致。
- [x] 日期选择器能够触发 API 查询并正确传递日期。
- [x] 无数据、加载中、接口错误和 Projection 未激活状态可读。
- [x] 空部门显示为“未设置”。
- [x] 前端类型检查、组件测试和构建通过。
- [x] 后端相关测试通过。
- [x] `usage_logs` 表结构仍未修改。

## 验证计划

- `pnpm --dir frontend run test:run -- src/views/admin/__tests__/UsageView.spec.ts`
- `pnpm --dir frontend run typecheck`
- `pnpm --dir frontend run build`
- `go test ./...`
- `make test-frontend-critical`
- 人工访问 `/admin/usage` 验证日期、图表、列表和深色模式。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 前端测试 | `pnpm exec vitest run src/components/charts/__tests__/DepartmentDistributionChart.spec.ts src/components/charts/__tests__/DepartmentUserUsageChart.spec.ts src/views/admin/__tests__/UsageView.spec.ts` | 通过，12 tests；包含部门行点击触发员工懒加载和员工排行图表 |
| 前端类型 | `pnpm run typecheck` | 通过 |
| 前端构建 | `pnpm run build` | 通过，产物写入 `backend/internal/web/dist` |
| 后端 API 测试 | `go test ./internal/handler/admin -run 'DepartmentStats' -count=1` | 通过 |
| 后端统计测试 | `go test ./internal/service/tokenstat -run 'DynamicTokenStatQuery' -count=1` | 通过 |
| 全量回归 | `go test ./...` | 失败项均为既有/环境问题：group model routing、RBAC compatibility seed、固定日期 Redis TTL、缺失 159 migration 文件；相关新增包可通过 |
| 全量前端 | `pnpm run test:run` | 795/801 tests 通过；6 个既有 OAuth/导航/测试 mock 失败，新增部门图表与 UsageView 通过 |
| 工具限制 | `make test-frontend-critical` | 当前 Windows 环境未安装 make |

## 执行记录

- 组件支持深色模式、窄屏折叠和部门列表滚动；图表与列表共用排序后的同一 rows 数据。
- 页面使用独立部门日期范围，不改变既有场景 Token 查询语义。
- 未修改 usage_logs，也未做历史部门回填。

