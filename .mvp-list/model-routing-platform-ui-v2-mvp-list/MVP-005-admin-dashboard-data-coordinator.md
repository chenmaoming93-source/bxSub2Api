# MVP-005：完成管理员仪表盘数据协调与场景聚合

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦 Part 1 的数据边界、并行请求和高基数场景聚合，不包含完整视觉页面。
- Dependencies: `MVP-001`

## 预期成果

为新管理员仪表盘提供统一时间范围、数据并行加载、字段适配和场景跨日期聚合模型。

## 背景

现有接口位于 `frontend/src/api/admin/dashboard.ts`、`frontend/src/api/admin/usage.ts`。管理员 Dashboard Stats、范围 Usage Stats、用户趋势、模型分布、分组数据和部门数据已有能力，但场景数据当前按日期和 `group_id` 组织。

## 范围内

- 统一 `start_date/end_date/granularity` 状态。
- 并行加载总览、用户趋势、模型、场景和部门数据。
- 复用 `getStats`、`getSnapshotV2`、`getUserUsageTrend`、`getModelStats`、`getGroupStats`、`queryDepartmentStats`。
- 按 `group_id` 跨日期聚合场景 Token，并计算占比和排序。
- 处理加载、取消过期请求、空数据、部分失败和同步状态。
- 明确范围值、累计值和性能指标口径。

## 范围外

- 不制作最终页面视觉。
- 不实现场景账号和部门用户弹窗。
- 不删除旧 Dashboard。
- 暂不新增场景后端接口，除非验证后端聚合确有必要。

## 实现说明

性能指标需要使用范围平均 RPM/TPM；如果后端无法提供，则保留明确的待实现接口适配点。

## 验收标准

- [x] 日期、粒度变化能生成统一查询参数。
- [x] 场景排行按跨日期 Token 汇总值排序。
- [x] 过期请求不会覆盖最后一次筛选结果。
- [x] 部分接口失败时其他数据仍能展示。
- [x] 累计和范围指标字段没有混用。

## 验证计划

- `pnpm --dir frontend typecheck`
- 针对跨日期同一 `group_id` 聚合、空数据和请求竞态编写单元测试。
- 使用固定 fixture 验证统计字段。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Typecheck | `pnpm run typecheck` | 通过 |
| Unit tests | `pnpm run test:run -- src/features/admin-dashboard/__tests__/adminDashboardData.test.ts` | 2 个测试通过 |
| Changed files | `frontend/src/features/admin-dashboard/adminDashboardData.ts` | 新增统一查询、并行加载、序列保护、部分失败降级和场景聚合 |

## 执行记录

已根据实际接口实现数据协调器；场景排行按 `group_id` 跨日期聚合后排序。