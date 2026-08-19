# MVP-005：候选 Redis 并发控制

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 days`
- Estimate rationale: 配置缓存、无限制哨兵、原子获取释放、TTL 和请求链路接入。
- Dependencies: `MVP-001, MVP-004`

## 预期成果

请求在最终候选账号维度执行 Redis 并发限制，多实例共享同一上限。

## 范围内

- 配置缓存：未命中、`unlimited`、正整数三态。
- 查不到关系或 `NULL` 时缓存 `unlimited`。
- 配置更新后刷新 Redis。
- 原子获取/释放、TTL 异常回收。
- 达到上限直接拒绝。

## 范围外

- 排队等待策略。

## 验收标准

- [x] 每笔请求正常路径不直接查数据库。
- [x] `NULL` 和查不到均不会被限制且会缓存 `unlimited`。
- [x] 配置修改为数字或 `NULL` 后 Redis 立即反映新值。
- [x] 达到上限时新请求被拒绝。
- [x] 完成、失败、超时后占用可释放或自动过期。
- [x] Redis 不可用时按项目既有降级策略处理并记录日志。

## 验证计划

- 运行限流单元测试、Redis 集成测试和请求链路测试。
- 验证并发边界、多实例共享和 TTL 回收。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Redis | `backend/internal/repository/concurrency_cache.go` | 新增 route slot 和三态配置缓存 |
| 请求链路 | `backend/internal/service/gateway_service.go` | 候选选择前获取，释放复用现有请求生命周期 |
| 配置同步 | `backend/internal/service/admin_service.go` | 更新数据库后刷新 Redis 配置 |
| 测试 | `go test ./internal/service ./internal/repository ./internal/handler` | 通过 |

## 执行记录

- 2026-08-11：完成 Redis 配置缓存、候选维度原子槽位、TTL 回收和管理端配置刷新。
