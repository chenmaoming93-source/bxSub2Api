# MVP-002：基于动态统计的共享场景用量查询服务

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 在既有 tokenstat QueryUsage 之上增加固定 Projection 校验、名称批量补全和嵌套聚合，形成后续两个 HTTP 接口共用的独立业务成果。
- Dependencies: `MVP-001`

## 预期成果

后端具备统一的 `QuerySceneAccountDailyUsage` 查询用例，可按日期和可选技术分组名返回日期 → 场景 → 账号/上游模型统计结果。

## 背景

动态 Token 统计系统已经支持 `group_id`、`account_id`、`upstream_model` 和 `total_tokens`，查询入口为 `tokenstat.ProjectionAdminService.QueryUsage`。本 MVP 不扫描 `usage_logs`，也不创建新的报表表。

使用方必须手动配置并启用以下自定义统计项：

```text
指标：total_tokens
周期：D（日）
维度：group_id + account_id + upstream_model
状态：ACTIVE
```

统计项按维度签名精确识别，不按名称识别。

## 范围内

- 定义共享查询输入、结果和业务错误。
- 校验动态 Token 统计全局开关。
- 查找精确维度签名且指标为 `total_tokens` 的 Projection。
- 区分“统计项不存在”和“统计项未启用”，返回可操作原因。
- 校验日期格式、首尾包含语义和最长 366 天范围。
- 将 `group_name` 精确转换为 `group_id` 过滤条件。
- 调用现有动态统计查询并使用日周期。
- 批量补全 `group_name`、`scene_name`、`account_name`。
- 按 `group_id` 聚合场景总量，按 `account_id + upstream_model` 保留明细。
- 保留 `complete`、`consistency`、Projection 时间和最后同步时间。
- 增加服务单元测试。

## 范围外

- 不实现 HTTP 路由和前端页面。
- 不自动创建、发布或启用 Projection。
- 不补采 Projection 启用前的历史数据。
- 不按 `scene_name` 聚合或筛选。

## 实现说明

- 使用现有 `tokenstat.UsageQueryInput`，`GroupBy` 为 `group_id`、`account_id`、`upstream_model`。
- 对无数据返回成功的空结果；只有配置缺失、未启用或运行时关闭才返回配置类错误。
- 统一服务不携带调用方鉴权逻辑，由 HTTP 适配层负责。
- 避免对每条结果单独查分组和账号，必须批量加载。

## 验收标准

- [x] 已启用精确 Projection 时，服务返回按日、分组、账号和上游模型拆分的数据。
- [x] 两个相同 `scene_name` 的不同 `group_id` 不会被合并。
- [x] 结果包含 `account_name`、`upstream_model`、场景总量和账号/模型明细。
- [x] `group_name` 过滤按技术名称匹配，省略时查询全部分组。
- [x] 缺少 Projection、Projection 非 ACTIVE、全局统计关闭时分别返回明确业务错误。
- [x] 日期首尾包含，超过 366 天或格式错误时被拒绝。
- [x] 服务测试覆盖最终一致状态和无数据结果。

## 验证计划

- `cd backend && go test ./internal/service/... ./internal/service/tokenstat/...`
- 使用已有 tokenstat 测试数据或 stub 验证 Projection 签名、日期边界、分组聚合和账号/模型聚合。
- 人工核对错误响应包含“需要配置并启用 group_id + account_id + upstream_model、total_tokens、日统计项”的原因。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 共享服务 | `backend/internal/service/scene_token_usage.go` | 新增统一查询用例：日期解析、366 天限制、技术 `group_name` 过滤、批量补全和日期→分组/场景→账号模型聚合。未扫描 `usage_logs`。 |
| Projection 校验 | `backend/internal/service/tokenstat/usage_projection.go` | 按规范化维度签名 `account_id,group_id,upstream_model` 精确匹配，并区分未配置与非 ACTIVE/指标未启用。 |
| 服务聚焦测试 | `go test ./internal/service -run 'TestSceneAccountDailyUsageService' -count=1` | 通过；覆盖正常聚合、同场景不同分组、技术名过滤、空数据、最终一致状态、日期错误和三类配置错误。 |
| 动态统计回归 | `go test ./internal/service/tokenstat -run 'TestDynamicTokenStatQueryValidationAggregationAndPagination' -count=1` | 通过。 |
| 依赖注入 | `go generate ./cmd/server` | 通过，新增共享查询服务 Provider。 |
| 格式检查 | `git diff --check` | 通过。 |
| 变更路径 | `backend/internal/service/{scene_token_usage.go,scene_token_usage_test.go,wire.go}`, `backend/internal/service/tokenstat/usage_projection.go` | 实现和测试已落地。 |

## 执行记录

- 2026-08-26：完成共享动态统计查询服务；复用 `ProjectionAdminService.QueryUsage`，保留 `complete`/`consistency`/Projection 同步元数据，并按配置时区处理包含式日期范围。

