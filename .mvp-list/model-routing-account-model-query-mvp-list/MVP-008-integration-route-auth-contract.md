# MVP-008：注册 integrations 路由并验证鉴权契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: `只接入一个现有路由组并补充中间件/路由级契约测试，不新增认证机制或配置，工作量集中。`
- Dependencies: `MVP-007`

## 预期成果

`POST /api/v1/integrations/model-routes/list` 可用，并与 `getOrCreate` 完全共享固定 Bearer Token 鉴权、安全加固和限流链路。

## 背景

`backend/internal/server/routes/integrations.go` 已在 `/integrations` 组应用 `provAuth` 与 `provHardening`。新接口必须注册在同一组内，不能使用管理员 JWT 代替固定 Token。

## 范围内

- 注册 `/model-routes/list` POST 路由。
- 验证缺失、格式错误和错误 Token 返回 `401 INVALID_ACCESS_TOKEN`。
- 验证 integrations 未启用或 Token 未配置时返回 `404 NOT_FOUND`。
- 验证正确 Token 可到达 Handler。
- 回归 `getOrCreate` 仍使用相同中间件且行为不变。
- 验证不新增配置字段。

## 范围外

- 不增加新的 Token、权限模型或数据库状态。
- 不改动普通管理员 JWT 中间件。
- 不加入分页或缓存。

## 实现说明

- 优先增加路由级测试；若当前路由包缺少测试基架，可在 server 或 middleware 现有测试模式中建立最小 Gin Router。
- 继续依赖常量时间 Token 比较和现有 hardening 中间件。

## 验收标准

- [x] 新路径、HTTP 方法和 Handler 注册正确。
- [x] 缺失或错误固定 Token 无法访问。
- [x] 正确固定 Token 可成功访问，管理员登录不是必要条件。
- [x] 禁用 integrations 时接口保持隐藏。
- [x] `getOrCreate` 鉴权回归通过，配置文件无需新增字段。

## 验证计划

- `cd backend && go test ./internal/server/middleware ./internal/server/routes ./internal/handler`
- 若 `routes` 无独立可测试包：`cd backend && go test ./internal/server/...`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/server/routes/integrations.go` | 在现有 `/integrations` 组注册 `POST /model-routes/list`，共享 `provAuth` 与 `provHardening`。 |
| 测试 | `cd backend && go test ./internal/server/middleware ./internal/server/routes ./internal/handler` | 通过：三个包均退出码 0；覆盖 Token 错误、隐藏开关、正确 Token 和 `getOrCreate` 回归。 |
| 配置检查 | `git diff -- backend/config/config.yaml deploy/config.example.yaml backend/internal/config/config.go` | 无输出；未新增或修改配置字段，因此无需配置文件变更。 |

## 执行记录

新查询接口与原 `getOrCreate` 共享相同固定 Token 鉴权及安全加固链路，未引入管理员 JWT 依赖。
