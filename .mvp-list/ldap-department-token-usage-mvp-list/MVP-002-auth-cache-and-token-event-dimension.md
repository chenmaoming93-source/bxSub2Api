# MVP-002：认证缓存部门快照与 Token 事件输入

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 只覆盖缓存快照和现有事件入口，边界清晰，可独立验证“无每请求额外用户查询”。
- Dependencies: `MVP-001`

## 预期成果

模型调用可以直接从当前 API Key 认证得到的用户对象获取部门，并把部门作为 Token 统计事件维度；缓存命中时不额外查询 users 表。

## 背景

API Key 认证在 `backend/internal/service/api_key_auth_cache.go` 和 `api_key_auth_cache_impl.go` 中使用 L1/L2 缓存。当前 `APIKeyAuthUserSnapshot` 不包含 department。动态统计事件由 `dynamic_token_usage.go` 生成，必须继续保持非阻塞、失败不影响模型请求。

## 范围内

- 保留内部 `DimensionDepartment` 定义，使兼容事件和旧 Projection 可以被现有 pipeline 校验；该维度不再出现在新 Projection 可配置列表。
- 为 `APIKeyAuthUserSnapshot` 增加 Department 字段。
- 修改快照生成、还原和快照版本，确保旧 Redis 快照自动失效回源。
- 修改网关现有 Token 统计调用点，显式将 `user.Department` 传入事件生成函数。
- 空部门转为非空统计值 `未设置`。
- 通过现有缓存失效接口保证用户部门更新后可重新加载。
- 增加事件、缓存序列化和缓存命中路径测试。

## 范围外

- LDAP 批量同步接口本身。
- Token 统计聚合表和 Projection 查询实现。
- 修改 `usage_logs` 或在事件生成函数中增加数据库查询。
- `/admin/usage` 页面。

## 实现说明

- 推荐调整为 `submitDynamicTokenUsage(usageLog, department)`，不在该函数内查询数据库。
- 覆盖 Gateway 和 OpenAI Gateway 的已有调用点。
- 缓存版本升级必须兼容旧 Redis 数据：旧版本快照应回源而不是错误解析。
- department 只作为当前请求事件的内存快照传入，不落 usage_logs。

## 验收标准

- [x] API Key 缓存快照包含 department，L1/L2 命中时可还原到 user。
- [x] 旧版本缓存快照不会导致错误部门数据。
- [x] Token 事件包含 department；空值为 `未设置`。
- [x] 事件提交函数不调用 UserRepository 或其他数据库查询。
- [x] 部门变化后的缓存失效路径可使后续请求获得新部门。
- [x] 相关 Go 测试通过。

## 验证计划

- `go test ./internal/service/...`
- 运行动态 Token 统计相关测试并检查事件维度断言。
- 代码审查确认 `submitDynamicTokenUsage` 没有新增数据库访问。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 测试 | `backend: go test ./internal/service -run 'DynamicToken|AuthCache|APIKeyService' -count=1` | 通过 |
| 检查 | `api_key_auth_cache.go`, `api_key_auth_cache_impl.go` | 快照增加 department，版本从 12 升至 13，旧快照自动回源 |
| 检查 | `dynamic_token_usage.go`, Gateway/OpenAI Gateway 调用点 | 部门显式传入事件，提交函数无数据库访问 |

## 执行记录

- 由于事件必须先能通过 registry 校验，本 MVP 同步注册了 `DimensionDepartment`；MVP-005 将继续完成聚合表和报表查询接入。

