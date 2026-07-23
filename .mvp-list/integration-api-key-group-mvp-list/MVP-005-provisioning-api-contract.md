# MVP-005：外部供应接口暴露分组请求与稳定响应

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40 分钟`
- Estimate rationale: `只处理 Handler 契约、错误映射、响应和审计字段，可通过独立 HTTP 测试验证。`
- Dependencies: `MVP-004`

## 预期成果

`POST /api/v1/integrations/api-keys/getOrCreate` 强制接收 `group_name`，成功响应返回实际 `group_id` 和 `group_name`，并按 Plan 输出稳定 HTTP 状态与错误码。

## 背景

当前 `backend/internal/handler/external_provisioning_handler.go` 只接收 `user` 和 `platform`，除用户不存在外的大部分业务失败都映射为 500。

## 范围内

- 请求 DTO 增加必填 `group_name`。
- Trim `group_name` 并传给供应服务。
- 响应 DTO 增加 `group_id` 和 `group_name`。
- 映射 `GROUP_NOT_FOUND`、`GROUP_INACTIVE`、`SUBSCRIPTION_GROUP_NOT_SUPPORTED`、`GROUP_NOT_ALLOWED`。
- 保持创建返回 201、幂等命中返回 200。
- 保持 `Cache-Control: no-store` 和 `Pragma: no-cache`。
- 审计增加分组维度但不记录 API Key 或 Bearer Token。
- 新增 Handler/路由契约测试。

## 范围外

- 不修改 Bearer Token 认证方式。
- 不修改 Content-Type、4 KB 请求体或限流中间件。
- 不实现仓储逻辑。

## 实现说明

- 缺少或空白 `group_name` 返回 400 `INVALID_REQUEST`。
- 用户无专属授权返回 403；分组不存在返回 404；inactive 返回 409；订阅分组返回 400。
- 失败审计不得包含明文 Key。

## 验收标准

- [x] 缺少 `group_name` 返回 400。
- [x] 成功响应包含正确的分组 ID 和名称。
- [x] 四类分组领域错误映射到计划中的 HTTP 状态和错误码。
- [x] 201/200 语义及禁止缓存响应头保持正确。
- [x] 审计字段包含分组上下文且不泄露凭据。
- [x] Handler 契约测试通过。

## 验证计划

- `cd backend && go test ./internal/handler -run 'Test.*EnsureAPIKey|Test.*ExternalProvisioning'`
- 若新增测试归属 server 路由包，追加 `cd backend && go test ./internal/server/... -run 'Test.*Integration.*APIKey|Test.*GetOrCreate'`。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Handler | `backend/internal/handler/external_provisioning_handler.go` | `group_name` 必填并 Trim；响应增加 `group_id/group_name`；领域错误映射为 404/409/400/403；保留 201/200 和禁止缓存头。 |
| 审计 | `backend/internal/handler/external_provisioning_handler.go` | 成功审计增加 `group_id/group_name`，失败审计增加 `group_name`；日志参数不包含 API Key 或 Bearer Token。 |
| 测试 | `backend/internal/handler/external_provisioning_handler_test.go` | 覆盖缺失/空白请求、成功响应分组、201/200、缓存头及四类错误映射。 |
| 验证 | `cd backend && go test ./internal/handler -run 'Test.*EnsureAPIKey|Test.*ExternalProvisioning'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/handler 5.289s`。 |

## 执行记录

2026-07-16：完成外部供应 HTTP 分组契约、错误映射、响应与审计，并通过 Handler 定向测试。
