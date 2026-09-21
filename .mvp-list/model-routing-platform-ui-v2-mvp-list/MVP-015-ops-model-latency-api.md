# MVP-015：提供运维按模型耗时聚合接口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦按模型百分位和趋势两个接口，能独立进行后端契约和性能验证。
- Dependencies: `MVP-001`

## 预期成果

运维页面可按模型查询请求耗时 P50/P90/P95/P99、平均耗时和按时间桶趋势。

## 背景

现有 `/admin/ops/dashboard/overview` 只提供全局耗时百分位；已有预聚合逻辑包含耗时字段和百分位计算，可在其基础上扩展。

## 范围内

- `GET /admin/ops/dashboard/model-latency-percentiles`。
- `GET /admin/ops/dashboard/model-latency-trend`。
- 支持时间、平台、分组、模型和查询模式参数。
- 返回模型、请求数、P50/P90/P95/P99、平均值。
- 返回时间桶、模型、请求数和平均耗时。
- 权限、空数据、错误和性能测试。

## 范围外

- 运维页面图表 UI。
- 修改原有全局百分位接口。
- 保存用户自定义报表。

## 验收标准

- [x] 按模型过滤和多模型结果正确。
- [x] 百分位字段口径与既有 Ops 聚合一致。
- [x] 趋势时间桶和平均耗时正确。
- [x] 超出允许范围的查询被拒绝或降级。

## 验证计划

- 后端现有 Go 测试入口。
- API contract、聚合口径和查询性能测试。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Backend tests | `go test ./internal/service ./internal/repository ./internal/server/routes -count=1` | 通过 |
| Handler tests | `go test ./internal/handler/admin -run 'Test(Ops|ParseOps|PickThroughput|RBAC)' -count=1` | 通过 |
| Routes | `backend/internal/server/routes/admin.go` | 两个接口使用 `PermissionOpsRead` |
| API client | `frontend/src/api/admin/ops.ts` | 新增类型与调用方法 |

## 执行记录

按模型耗时百分位与趋势接口已完成，支持时间、平台、分组、模型及 query mode，自动选择时间桶并安全处理空数据和非法范围。
