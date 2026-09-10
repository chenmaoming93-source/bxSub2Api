# MVP-002：实现分组安全配置的 Redis 与多实例本地缓存同步

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发者日`
- Estimate rationale: 聚焦配置读取、缓存回源、版本控制和 Pub/Sub 失效，不接入模型请求判定。
- Dependencies: `MVP-001`

## 预期成果

请求侧可以低延迟读取分组安全配置；配置更新后，各服务实例的本地缓存能够失效并重新加载最新版本。

## 背景

项目已使用 `github.com/redis/go-redis/v9`，并有多种 Redis cache 实现。安全检查配置应按分组 ID 整体缓存，数据库仍是持久化权威来源。

## 范围内

- 实现 `SecurityConfigProvider`；
- 本地缓存、Redis 缓存和数据库回源；
- Redis 配置 Key 和完整 JSON value；
- 配置版本号；
- 配置更新后的 Redis Pub/Sub 失效通知；
- 本地缓存 TTL 和 Pub/Sub 丢消息兜底；
- Redis/数据库不可用时使用最近有效配置或安全关闭策略；
- 配置更新接口完成缓存同步。

## 范围外

- SingGuard 调用和规则判定；
- 采集队列；
- 管理页面；
- 业务消息队列。

## 实现说明

- Redis Key：`sub2api:security-check:group:{group_id}`；
- Pub/Sub 频道：`sub2api:security-check:config-change`；
- 消息至少包含 `group_id` 和 `version`；
- 配置写库成功后再更新 Redis 和发布通知；
- 服务实例收到通知后删除或标记本地缓存；
- 本地缓存默认 TTL 为 5 秒；
- 配置更新使用行锁或等价并发控制，避免 JSON read-modify-write 覆盖；
- API Key 认证路径只需提供 `group_id`，不要强制在每次 API Key 查询中加载完整安全 JSON。

## 验收标准

- [x] 本地缓存命中时不访问 Redis 或数据库。
- [x] 本地缓存未命中时按本地缓存 → Redis → 数据库顺序加载并回填。
- [x] 配置版本较旧时不会覆盖较新本地配置。
- [x] 配置更新成功后，订阅通知的模拟服务实例能够删除对应本地缓存。
- [x] Redis 暂不可用时，已有最近有效配置的请求仍可获得配置；无有效配置时不阻塞主流程。
- [x] 缓存和版本同步测试通过。

## 验证计划

- `cd backend; go test ./internal/repository/... ./internal/service/...`
- 使用现有 Redis 测试工具或 miniredis 测试缓存命中、回源、版本和 Pub/Sub 行为。
- 人工检查 Redis Key、频道名和失效消息格式未包含敏感信息。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `cd backend; go test ./internal/service ./internal/repository` | 通过；配置 provider、分组仓储配置读写相关包测试通过。 |
| 缓存实现 | `backend/internal/service/security_check_config.go` | 已实现本地 TTL、Redis 回源、数据库回源、最近有效配置降级、版本化 Pub/Sub 和更新写入。 |
| 测试覆盖 | `backend/internal/service/security_check_config_test.go` | miniredis 验证本地命中、Redis 回源、数据库降级、过期配置、版本失效和更新写入。 |
| 持久化接口 | `backend/internal/repository/group_repo.go` | 已增加按分组 ID 读取/校验和更新安全配置的方法。 |

## 执行记录

- 使用 `SecurityConfigProvider.Update` 作为配置更新与缓存同步边界：数据库成功后写 Redis、更新本地缓存并发布包含 `group_id`/`version` 的通知。
- 实际 Redis Pub/Sub 订阅由 `Start` 管理；版本判断由 `InvalidateIfOlder` 覆盖，避免旧通知清除新配置。
