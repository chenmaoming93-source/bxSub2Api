# MVP-001：平台 Key 服务支持三元组幂等

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40 分钟`
- Estimate rationale: `只调整平台 Key 服务的输入、查询语义、创建字段及其单元测试，范围集中且可独立验证。`
- Dependencies: `none`

## 预期成果

`PlatformAPIKeyService` 能够按 `(user_id, platform, group_id)` 获取或创建平台 Key，相同三元组复用 Key，不同分组产生不同 Key，并且不再解析默认分组。

## 背景

当前 `backend/internal/service/platform_api_key_service.go` 按 `(user_id, platform)` 幂等，并通过默认分组解析器决定 `group_id`。目标接口要求调用方指定分组，因此服务必须接收有效 `groupID`，并将其写入新 Key。

## 范围内

- 将服务方法调整为接收 `groupID`。
- 将仓储窄接口调整为三元组查询。
- 删除平台 Key 服务中的默认分组解析依赖。
- 新建 Key时写入指定 `group_id`、`purpose = platform` 和调用系统 `platform`。
- 更新 `platform_api_key_service_test.go` 的 stub 和单元测试。
- 覆盖相同三元组幂等及相同用户/平台不同分组创建不同 Key。

## 范围外

- 不实现真实仓储查询。
- 不实现分组名称解析或专属授权。
- 不修改 HTTP 请求契约。
- 不新增数据库索引。

## 实现说明

- 目标方法形态：`GetOrCreatePlatformKey(ctx, userID, platform, groupID)`。
- `groupID <= 0` 必须被拒绝。
- 发生创建冲突时，按同一三元组重新读取，不能退回二元组读取。
- 保留现有 platform 格式校验和未软删除旧 Key 状态语义。

## 验收标准

- [x] 新建 Key 的 `GroupID` 等于传入 `groupID`。
- [x] 相同 `(userID, platform, groupID)` 连续调用返回同一 Key。
- [x] 相同用户和 platform、不同 groupID 返回不同 Key。
- [x] 不同 platform、相同 groupID 返回不同 Key。
- [x] 无效 groupID 返回错误且不创建 Key。
- [x] 服务不再依赖 `ResolveDefaultGroup`。
- [x] 定向单元测试通过。

## 验证计划

- `cd backend && go test ./internal/service -run 'TestGetOrCreatePlatformKey'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/platform_api_key_service.go` | 服务显式接收并校验 `groupID`，使用三元组查询和冲突重读，创建时写入 `GroupID`，已移除默认分组 resolver。 |
| 接口 | `backend/internal/service/api_key_service.go` | `PlatformAPIKeyRepository` 已调整为三元组查询契约。 |
| 测试 | `cd backend && go test ./internal/service -run 'TestGetOrCreatePlatformKey'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 6.017s`。 |

## 执行记录

2026-07-16：完成平台 Key 三元组幂等服务及单元测试；供应服务暂以无效占位分组调用，后续由 MVP-004 的分组解析接通，当前 MVP 的定向测试已通过。
