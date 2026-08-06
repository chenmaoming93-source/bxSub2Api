# MVP-007：限额 wildcard 与 API Key 搜索选择体验

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `在同一限额表单中交付 wildcard 配置和 API Key 可搜索选择，并复用现有限额提交契约。`
- Dependencies: `MVP-006`

## 预期成果

管理员可在限额表单为维度选择 wildcard，并可按名称或具体 Key 搜索 API Key、选择候选后以现有 `api_key_id` 提交规则。

## 背景

动态统计前端类型位于 `frontend/src/api/admin/dynamicTokenStatistics.ts`，表单位于 `frontend/src/views/admin/TokenStatisticsView.vue`。当前 API Key 维度要求直接输入 ID。

## 范围内

- 扩展前端 `DimensionValue` 联合类型。
- 为每个限额维度提供具体值/wildcard 选择。
- 将 API Key 输入替换为搜索选择组件。
- 400ms 防抖、最少 2 字符、取消旧请求并防止过期响应覆盖。
- 复用或扩展管理员 API Key 列表搜索，返回 ID、名称和脱敏 Key。
- 选择后仍按现有 int64 ID 契约提交。

## 范围外

- 在限额规则中保存 API Key 明文。
- 返回完整 API Key。
- 改变限额创建接口的 `api_key_id` 内部类型。

## 实现说明

- 自由文本与已选 ID 必须绑定；输入变化后清除旧选择。
- 如果现有 API Key 存储为哈希，具体 Key 搜索应复用认证的精确查找能力。
- 搜索原文不得写入应用日志。

## 验收标准

- [x] wildcard 选择提交 `{type:"wildcard"}`。
- [x] API Key 搜索不会逐字符立即请求，连续输入只保留最新结果。
- [x] 候选只展示名称和脱敏 Key。
- [x] 未明确选择候选时无法提交 API Key 限额。
- [x] 最终请求仍提交所选 API Key 的数据库 ID。

## 验证计划

- `cd frontend && pnpm test:run src/views/admin/__tests__/TokenStatisticsView.spec.ts`
- `cd frontend && pnpm typecheck`
- `cd backend && go test ./internal/handler/admin ./internal/repository -run 'APIKey|Quota'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 前端类型/表单 | `dynamicTokenStatistics.ts`、`TokenStatisticsView.vue` | 每个限额维度支持 exact/wildcard；wildcard 提交明确类型；API Key 必须选中候选后才有 ID。 |
| 搜索策略 | `searchApiKeys(..., {signal})` | 至少 2 字符、400ms 防抖、取消旧请求、版本号防止过期响应覆盖；输入变化立即清除旧 ID。 |
| 后端复用 | `/admin/usage/search-api-keys` | 沿用现有接口；名称模糊或具体 Key 精确匹配，返回 ID/名称/user_id/脱敏 Key，不返回明文。 |
| 前端验证 | `pnpm test:run src/views/admin/__tests__/TokenStatisticsView.spec.ts && pnpm typecheck` | 10 个测试通过，类型检查通过。 |
| 后端验证 | `go test ./internal/handler/admin ./internal/repository -run 'APIKey|Quota|MaskUsage' -count=1` | 两个包均通过。 |

## 执行记录

- 复用管理员 usage 的 API Key 搜索接口和既有权限，不改变 quota create 接口。
- Key 搜索仅允许精确等值，名称继续大小写不敏感模糊搜索；搜索原文未新增任何日志。
- 脱敏形式为前 4 位 + `****` + 后 4 位，短 Key 统一显示 `****`；测试确认不会包含中间 secret。
- UI 展示名称、脱敏 Key 和 ID，但保存 payload 仍为 `{type:"int64", int64:<selected id>}`。
