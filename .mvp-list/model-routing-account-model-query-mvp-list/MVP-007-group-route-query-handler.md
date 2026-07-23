# MVP-007：实现分组路由查询 Handler 契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `聚焦请求绑定、统一响应、错误映射和审计日志，Service 已由前置 MVP 提供，可用 Handler 单测独立验收。`
- Dependencies: `MVP-006`

## 预期成果

HTTP Handler 接受 `group_name` JSON 请求体，调用只读 Service，并输出批准的统一响应与错误契约。

## 背景

现有 `ExternalProvisioningHandler` 已包含 `getOrCreate` 的请求处理和审计模式。新接口应保持相同响应封装，但不得返回账号内部信息或记录 Token。

## 范围内

- 定义请求及响应 DTO。
- 实现 `POST /model-routes/list` 对应 Handler 方法。
- 校验非法 JSON、缺失和空白分组名。
- 映射 `GROUP_NOT_FOUND` 与内部错误。
- 输出 `code/message/data`，空路由为 `[]`。
- 增加成功、空配置和错误 Handler 测试。
- 增加不含敏感信息的成功/失败审计日志。

## 范围外

- 不注册路由。
- 不重复实现鉴权中间件。
- 不改动 `getOrCreate` 请求响应。

## 实现说明

- 若扩展 Handler 的窄 Service interface，必须同步更新测试 stub。
- DTO 只暴露 `group_name`、`route_alias` 和 `upstream_models`。
- 不记录 Authorization Header。

## 验收标准

- [x] 合法请求返回批准的响应结构。
- [x] 缺失或空白 `group_name` 返回 `400 INVALID_REQUEST`。
- [x] 分组不存在返回 `404 GROUP_NOT_FOUND`。
- [x] 内部错误返回安全的 `500` 响应。
- [x] Handler 测试确认响应不含账号、优先级、限额和 Token。

## 验证计划

- `cd backend && go test ./internal/handler -run 'TestExternalProvisioningHandler.*Group.*Route'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/handler/external_provisioning_handler.go` | 新增请求/响应 DTO、统一响应、错误映射和仅记录分组名/IP/结果的审计日志。 |
| 测试 | `cd backend && go test ./internal/handler -run 'TestExternalProvisioningHandler.*Group.*Route'` | 通过：handler 包退出码 0；覆盖成功、空数组、非法请求、404、安全 500 与敏感字段缺失。 |

## 执行记录

Handler 已完成但尚未注册路由，符合本 MVP 边界；DTO 只输出 `group_name`、`route_alias` 与 `upstream_models`。
