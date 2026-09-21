# MVP-012：提供个人场景与耗时统计接口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦个人仪表盘缺失的三类后端聚合能力，接口契约和权限可独立验证。
- Dependencies: `MVP-001`

## 预期成果

新增个人场景 Token、平均耗时趋势和耗时百分位接口，严格限制为当前认证用户数据。

## 背景

当前个人仪表盘没有明确的个人场景汇总、耗时趋势和耗时百分位接口。现有请求日志和耗时聚合能力可作为数据来源。

## 范围内

- `GET /usage/dashboard/groups`。
- `GET /usage/dashboard/latency-trend`。
- `GET /usage/dashboard/latency-percentiles`。
- 日期范围、粒度和分页/Top N 参数。
- 按认证用户授权，不信任任意 `user_id`。
- 空结果、无数据和时区处理。
- 接口契约测试。

## 范围外

- 个人仪表盘 UI。
- 数据库表结构重构。
- 管理员范围接口。

## 验收标准

- [x] 个人场景结果只包含当前用户数据。
- [x] 趋势返回时间桶和平均耗时。
- [x] 百分位返回 P50/P90/P95/P99。
- [x] 日期范围和粒度参数生效。
- [x] 未授权或非法参数返回明确错误。

## 验证计划

- 后端项目现有测试命令（如仓库脚本未统一则按实际 Go 测试入口执行）。
- API contract 测试覆盖认证、越权、空数据和聚合口径。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| Backend tests | `go test ./internal/handler ./internal/service ./internal/repository ./internal/server/routes` | 通过 |
| Contract tests | `go test ./internal/handler -run "Test(Dashboard|UserDashboard)" -count=1`; `go test ./internal/server/routes -run "Test(PersonalDashboardRoutesRequireSelfUsagePermission|RBACUserRouteDeclarationCount)" -count=1` | 通过 |
| Frontend typecheck | `pnpm run typecheck` | 通过 |
| Routes | `backend/internal/server/routes/user.go` | 三个接口统一使用 `PermissionUsageSelfRead` |
| Frontend API | `frontend/src/api/usage.ts` | 新增三组类型和调用方法 |

## 执行记录

已接入当前认证用户隔离、日期范围/时区、粒度、limit 校验；百分位返回 P50/P90/P95/P99，空样本返回 null 和 sample_count=0。
