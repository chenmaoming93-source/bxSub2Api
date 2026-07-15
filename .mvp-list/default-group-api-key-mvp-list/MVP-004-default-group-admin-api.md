# MVP-004：提供默认分组管理员 API

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `MVP-003`

## 预期成果

管理员可读取默认分组状态并更新默认分组名。

## 背景

管理员设置现有入口位于 `backend/internal/handler/admin/setting_handler.go` 和路由注册代码。

## 范围内

- 实现 `GET /api/v1/admin/default-group`。
- 实现或接入 `PUT /api/v1/admin/settings/default-group`。
- 校验去空格后非空及长度。
- 返回 configured、name、exists、group 信息。
- 添加管理员鉴权和 Handler测试。

## 范围外

- 模型路由编辑页面。
- 创建默认分组。

## 实现说明

- 沿用现有 response/error DTO风格。
- 普通用户及匿名请求不得访问。

## 验收标准

- [x] 管理员可保存不存在的分组名并看到 `exists=false`。
- [x] 匿名或普通用户访问被拒绝，Handler测试通过。

## 验证计划

- `cd backend; go test ./internal/handler/admin/... -run 'DefaultGroup'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Handler 测试 | `cd backend; go test ./internal/handler/admin/... -run 'DefaultGroup' -count=1` | 通过；覆盖不存在分组名的保存、去空格、`exists=false` 和空名称拒绝。 |
| 鉴权测试 | `cd backend; go test ./internal/server/middleware -run 'Admin' -count=1` | 通过；路由挂载于统一 `/admin` 鉴权组，匿名请求返回 401，非管理员 JWT 返回 403。 |
| 服务回归 | `cd backend; go test ./internal/service/... -run 'DefaultGroup|Setting' -count=1` | 通过。 |
| 路由与装配 | `cd backend; go test ./internal/server/routes -count=1; go test ./cmd/server -run '^$' -count=1` | 通过；新增 GET/PUT 路由与依赖装配可编译。 |
| 实现路径 | `backend/internal/handler/admin/setting_handler.go`、`backend/internal/server/routes/admin.go` | 实现 `GET /api/v1/admin/default-group` 与 `PUT /api/v1/admin/settings/default-group`。 |

## 执行记录

实现与验证完成。两个接口复用 `/api/v1/admin` 路由组的统一管理员鉴权；更新接口允许引用暂不存在的分组，但拒绝去空格后为空或超过 100 字符的名称。
