# MVP-002：关系表重建接口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 单组/全量重建、保留配置和权限校验。
- Dependencies: `MVP-001`

## 预期成果

提供单组和全量关系重建接口，可从现有 JSON 修复或初始化关系表。

## 范围内

- 单组重建接口。
- 全量重建接口。
- 保留已有 `max_concurrency`。
- 新关系默认为 `NULL`，失效关系删除。
- 幂等性和权限测试。

## 范围外

- Redis 实时限流。

## 验收标准

- [x] 两个接口均可调用并返回处理统计。
- [x] 重复调用结果一致。
- [x] 关系表以 JSON 为准且已有并发配置保留。
- [x] 非管理员不能调用。
- [x] 接口测试通过。

## 验证计划

- 运行 handler/service/repository 相关测试。
- 使用构造数据验证新增、删除和保留配置。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| API | `POST /api/v1/admin/model-route-references/rebuild` | 已接入全量重建 |
| API | `POST /api/v1/admin/groups/:id/model-route-references/rebuild` | 已接入单组重建 |
| 测试 | `GOCACHE=.gocache go test ./internal/handler/admin/... ./internal/repository/...` | 本任务相关 handler/repository 通过；tokenstat 中两项无关 Redis 测试失败 |

## 执行记录

- 2026-08-10：新增管理员重建路由、服务转发和仓储重建逻辑；保留关系表已有 `max_concurrency`。
