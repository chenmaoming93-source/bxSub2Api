# MVP-013：实现平台专属 Key 幂等服务

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-001, MVP-002, MVP-003`

## 预期成果

内部服务可按 `(user_id, platform)` 查询或创建平台专属 Key。

## 背景

平台名仅允许小写英文和下划线；已有非 active Key也应返回而不是另建。

## 范围内

- 扩展领域实体、Repository和映射支持 `platform`、`purpose`。
- 新增按用户与平台查询。
- 实现平台名校验、命名、默认分组绑定和唯一冲突回读。
- 覆盖 active、disabled、expired、quota_exhausted及软删除场景。

## 范围外

- HTTP接口和 Bearer认证。
- LDAP用户查询。

## 实现说明

- Key名为 `<platform> API Key`，purpose为 `platform`。
- 不得自动恢复或重置已有 Key。

## 验收标准

- [x] 重复及并发语义最终返回同一平台 Key。
- [x] 平台服务与 Repository测试通过。

## 验证计划

- `cd backend; go test ./internal/service/... ./internal/repository/... -run 'PlatformAPIKey|PlatformKey'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 平台验证 | `ValidatePlatform`（platform_api_key_service.go）| 仅允许小写字母开头的字母+下划线，最长50字符 |
| 仓储接口 | `PlatformAPIKeyRepository`（api_key_service.go:92-95）| 新增 `GetByUserIDAndPlatform` 窄接口 |
| 仓储实现 | `apiKeyRepository.GetByUserIDAndPlatform`（api_key_repo.go）| 按 userID+platform 查询未删除 key（任意状态） |
| 核心服务 | `PlatformAPIKeyService.GetOrCreatePlatformKey`（platform_api_key_service.go）| 幂等 get-or-create，命名 `<platform> API Key`，purpose=`platform` |
| 并发处理 | 与 `EnsureDefaultAPIKey` 相同模式：创建冲突→回读胜出者 | 数据一致性保证 |
| 默认分组 | 与 DefaultGroupResolver 集成 | platform key 自动绑定默认分组 |
| 验证测试 | `TestValidatePlatform`（13个子测试）| ALL PASS |
| 服务测试 | `TestGetOrCreatePlatformKey_*`（6个测试）| ALL PASS：创建/幂等/隔离/平台区分/非法输入/缺接口 |
| 回归测试 | `go test ./internal/...` | service/repository/handler 全 PASS |

## 执行记录

- **2026-07-08**：新增文件 `platform_api_key_service.go` 和 `platform_api_key_service_test.go`，修改 `api_key_service.go`（接口）和 `api_key_repo.go`（查询方法）。完全复用现有 `EnsureDefaultAPIKey` 的幂等模式。

