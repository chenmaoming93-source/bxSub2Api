# MVP-010：迁移设置、系统、数据管理与备份路由

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `系统配置与灾备同属高风险运维域，可作为一个受控管理切片。`
- Dependencies: `MVP-006`

## 预期成果

系统设置、数据管理、备份、升级和安全配置接口按查看、修改和关键操作分权。

## 背景

主要涉及 admin settings、data-management、backups、system、错误透传和 TLS 指纹。

## 范围内

- 系统设置和默认分组。
- 邮件模板、管理员 API Key、限流和模型默认配额。
- 数据源、S3、备份、恢复和下载。
- 系统版本、更新、回滚和重启。
- 错误透传规则、TLS 指纹模板。

## 范围外

- Ops 监控路由。
- 配置文件 shadow/enforce 开关。

## 实现说明

- `settings.read` 与 `settings.update` 分离。
- 管理员 API Key、恢复、更新、回滚和重启使用 critical 权限。
- 数据备份下载与恢复分别授权。

## 验收标准

- [x] 范围内所有路由进入 Registry。
- [x] 只读设置角色不能修改配置或执行系统操作。
- [x] 高风险操作拥有独立权限和拒绝测试。
- [x] admin 仍可访问全部功能。

## 验证计划

- `cd backend && go test ./internal/server/... -run 'Admin.*(Setting|System|Backup|Data)'`
- `cd backend && go test ./internal/handler/admin/... -run '(System|Backup|Data|ErrorPassthrough|TLS)'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/server/routes/admin.go` | settings、data-management、backups、system、错误透传和 TLS 指纹路由均通过统一 RBAC registrar 声明权限；读取、修改、密钥、恢复及系统操作分别授权。 |
| 测试 | `go test ./internal/server/... -run 'RBAC|Admin.*(Setting|System|Backup|Data)' -count=1` | 通过；覆盖路由权限映射、只读权限拒绝写操作及 admin 通配权限。 |
| 测试 | `go test ./internal/handler/admin/... -run '(System|Backup|Data|ErrorPassthrough|TLS)' -count=1` | 通过。 |
| 编译验证 | `go test ./cmd/server -run '^$'` | 通过，运行时依赖注入与路由装配可编译。 |

## 执行记录

管理员 API Key 使用 `settings.secrets.manage`；备份恢复使用 `backups.restore`；系统升级、回滚和重启使用 `system.operate`。普通读取权限不会隐式获得这些 critical 操作权限。
