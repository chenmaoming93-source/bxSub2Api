# MVP-005：实现固定 Key 双版本 Redis 权限缓存

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `缓存、版本一致性和故障回源是一个边界清晰的可靠性成果。`
- Dependencies: `MVP-004`

## 预期成果

多实例应用通过共享 Redis 安全缓存用户最终权限，角色变更后无需逐节点通知即可及时失效。

## 背景

固定 Key 为 `rbac:user:{user_id}:permissions`；Value 保存 user/policy 双版本。

## 范围内

- 权限缓存编码、读取、覆盖和 TTL。
- 用户版本与全局策略版本比较。
- Redis 不可用时数据库回源。
- 用户角色变更和角色策略变更后的版本递增接口。
- 并发缓存回填保护与指标。

## 范围外

- 进程内 L1 缓存。
- HTTP 鉴权中间件。

## 实现说明

- 版本递增与权限写操作必须处于同一数据库事务。
- 删除用户 Key 只是优化，正确性依赖版本校验。
- 不允许使用无法确认版本的新旧缓存放行。

## 验收标准

- [x] 版本相同命中缓存，任一版本不同均回源并覆盖固定 Key。
- [x] 修改角色权限不产生按策略版本扩张的用户 Key。
- [x] Redis 故障时可安全回源数据库。
- [x] 超级管理员缓存不展开全部权限。

## 验证计划

- `cd backend && go test ./internal/rbac/... -run Cache`
- 使用项目 Redis 测试设施验证并发、故障和版本切换。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 缓存测试 | `cd backend && go test ./internal/rbac/... -run 'Cache|Evaluate' -count=1` | 通过；覆盖命中、任一版本变化回源、固定 Key 覆盖、Redis 故障回源、并发请求合并和版本无法确认时拒绝使用缓存。 |
| Key 设计 | `rbac:user:{user_id}:permissions` | 策略版本变化前后 Key 数保持 1，不按版本派生新 Key。 |
| 超管缓存 | `EffectivePermissions{permissions:["*"], is_super_admin:true}` | 只缓存通配权限，不展开 100 个普通权限。 |
| 事务接口 | `NewRBACRepositoryTx(*sql.Tx)` + `IncrementUserVersion` / `IncrementPolicyVersion` | 后续写服务可在同一数据库事务内完成授权修改和版本递增。 |

## 执行记录

默认 TTL 为 20 分钟，可由构造参数覆盖；TTL 只回收冷用户 Key，不承担一致性。每次命中都先从数据库确认 user/policy 双版本，任一不一致即回源并覆盖同一 Key。`singleflight` 按用户合并并发回填；Redis 读写故障只增加降级指标并回源数据库，数据库版本或授权事实无法确认时返回错误而不是使用旧缓存放行。
