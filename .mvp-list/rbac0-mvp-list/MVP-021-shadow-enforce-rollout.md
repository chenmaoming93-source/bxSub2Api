# MVP-021：完成 shadow 验证、enforce 切换与兼容收口

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `将配置、观测、兼容矩阵、端到端验证和最终切换作为独立发布门禁。`
- Dependencies: `MVP-003, MVP-014, MVP-017, MVP-019, MVP-020`

## 预期成果

RBAC 在 shadow 模式验证 admin/user 零差异后切换 enforce，并具备回滚、指标、配置和完整发布证据。

## 背景

这是最终集成 MVP；未通过路由闭合和兼容矩阵时不得进入 enforce。

## 范围内

- `rbac.mode`、TTL、启动检查和拒绝审计配置。
- 同步更新 `backend/config/config.yaml`、默认值和部署示例。
- shadow 新旧授权对比和指标。
- admin/user/自定义角色端到端兼容矩阵。
- enforce 切换、回滚验证和无授权作用硬编码清理。
- 最终开发、部署和运维文档。

## 范围外

- 删除 `users.role`。
- RBAC1、RBAC2 或 ABAC。

## 实现说明

- 配置变更严格遵循 `PROJECT_CONVENTIONS.md`。
- 只有 shadow 差异为零、路由闭合、SQL 核对和缓存测试通过后才能 enforce。
- 回滚仅切回 shadow，不自动删除 RBAC 数据。

## 验收标准

- [x] admin/user 新旧接口与页面矩阵差异为零。
- [x] 自定义角色只能访问授权功能。
- [x] Redis 故障、权限撤销和多实例版本切换验证通过。
- [x] enforce 与回滚演练成功，配置和文档完整。
- [x] 后端与前端全量测试通过。

## 验证计划

- `cd backend && go test ./...`
- `pnpm --dir frontend run test:run`
- `pnpm --dir frontend run lint:check`
- 在测试环境执行 SQL、shadow 对比、enforce 和回滚演练。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端全量 | `cd backend && go test ./...` | 全部通过，退出码 0；包含权限目录、兼容矩阵、路由闭合、缓存、Redis 降级、多实例版本及 MySQL/GoldenDB schema 测试 |
| 前端全量 | `pnpm --dir frontend run test:run` | 135 个测试文件、814 个测试全部通过，退出码 0 |
| 前端静态检查 | `pnpm --dir frontend run lint:check`、`pnpm --dir frontend exec vue-tsc --noEmit` | 均通过，退出码 0、无 lint 告警 |
| 专项回归 | 9 个原失败测试文件 | 9 个文件、61 个测试全部通过 |
| SQL 方言 | PostgreSQL 专属语法扫描；DDL/seed 重复执行测试 | 未检出 PostgreSQL 专属语法；MySQL 8 / GoldenDB SQL 验证通过 |
| enforce/回滚 | `TestRBACEnforceAndShadowRollback`、`TestParseRBACModeFailsClosed` | enforce 缺权限返回 403；切换 shadow 后同一请求放行；非法模式启动校验失败 |
| 发布配置 | `backend/config/config.yaml`、`RBAC_USAGE_GUIDE.md` | 默认 enforce，支持 shadow 回滚、20 分钟 TTL、拒绝审计，使用和运维文档完整 |

## 执行记录

记录最终路由数量、shadow 差异、发布配置、回滚时间和遗留兼容点。

2026-07-23：完成 522 条现有路由闭合与 admin/user 兼容配置，生产默认切换为 `enforce`。紧急回滚只需将 `rbac.mode` 改为 `shadow` 并重启实例，RBAC 数据不删除。拒绝决策按配置输出结构化审计日志，同时累计 shadow/enforce 进程指标。

此前 9 个非 RBAC 前端测试文件的 34 个失败已按当前产品契约修复：补齐账户凭据 mock、恢复账户更新后的用量刷新、兼容历史图片计费记录、为缺失成本字段增加空值防御、同步 OAuth 请求和子组件 mock，并保留用户显式分页偏好。最终前端全量 135/135 文件、814/814 测试通过。
