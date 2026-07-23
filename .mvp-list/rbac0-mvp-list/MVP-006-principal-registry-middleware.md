# MVP-006：建立统一 Principal、路由 Registry 与鉴权中间件

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `交付一个可在少量试验路由上工作的声明式鉴权基础设施。`
- Dependencies: `MVP-001, MVP-004, MVP-005`

## 预期成果

Gin 路由可通过统一包装器声明权限，自动登记并挂载 `RequirePermission`，业务 Handler 无需编写通用鉴权逻辑。

## 背景

现有 `AdminAuth` 混合认证与 admin 授权，需要先建立可复用的认证主体和授权层。

## 范围内

- Principal 抽象和 Context 存取。
- `rbac.GET/POST/PUT/PATCH/DELETE`。
- Route Registry 与权限编码校验。
- `RequirePermission` 的 401/403、`*` 和审计钩子。
- 最小 shadow/enforce 决策接口。

## 范围外

- 全量路由迁移。
- 管理页面和角色管理 API。

## 实现说明

- 包装器一次完成路由、权限、中间件和 Registry 登记。
- 管理员 API Key 作为 `*` Principal，但保留原认证。
- Registry 应保存 Method、完整模板路径、权限或排除原因。

## 验收标准

- [x] 试验路由可按普通权限和 `*` 正确放行或拒绝。
- [x] 空权限和未知权限在注册阶段失败。
- [x] Handler 未执行时返回统一 401/403。
- [x] Registry 可输出已声明路由集合。

## 验证计划

- `cd backend && go test ./internal/server/middleware/... -run RBAC`
- `cd backend && go test ./internal/rbac/... -run Registry`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Registry/包装器 | `cd backend && go test ./internal/rbac/... -run Registry -count=1` | 通过；登记 Method、完整模板路径和权限，空权限、未知权限及重复路由在注册阶段失败。 |
| 鉴权中间件 | `cd backend && go test ./internal/server/middleware/... -run RBAC -count=1` | 通过；普通权限、角色 `*`、管理员 API Key Principal、401、403 和 shadow 审计均覆盖。 |
| Handler 边界 | `rbac_auth_test.go` | 未认证或无权限时业务 Handler 未执行；enforce 返回统一错误，shadow 只记录差异并继续。 |
| 路由声明 API | `rbac.NewRouteRegistrar(...).GET/POST/PUT/PATCH/DELETE` | 一次调用完成权限校验、Registry 登记、中间件挂载和原 Handler 注册。 |

## 执行记录

当前只新增 `PrincipalFromAuthenticatedSubject` 适配层：JWT/管理员 API Key 仍由原中间件认证，管理员 API Key 映射为 `SuperAdmin` Principal。原 `AdminAuth` 中 JWT 的 `role == admin` 限制将在管理路由迁移时移除；通用授权已经完全位于 `RequirePermission`，业务 Handler 不需要写权限判断。
