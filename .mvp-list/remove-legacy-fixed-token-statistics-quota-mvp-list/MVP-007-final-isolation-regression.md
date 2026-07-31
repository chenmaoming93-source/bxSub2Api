# MVP-007：完成残留审计与全系统回归

- Protocol: `mvp-list/v1`
- State: `DONE`
- Estimate: `1.5 开发日`
- Estimate rationale: `需要全仓静态门禁、后端全量、前端全量/类型/构建和新体系专项回归。`
- Dependencies: `MVP-004, MVP-005, MVP-006`

## 预期成果

生产代码、运行配置和前端不再残留旧固定统计限额；新体系与所有共享核心业务通过回归，可发布无旧依赖版本。

## 背景

这是人工执行删表前的代码门禁，不代表生产表已经删除或旧 Redis key 已人工处理。

## 范围内

- 全仓旧标识残留扫描和保护标识正向扫描。
- 所有协议、计费、余额、API Key、订阅、路由、调度、新统计/限额/查询回归。
- 前端页面、RBAC、用户/分组/账号和公共组件回归。
- 更新开发/操作文档中的旧体系状态与人工 SQL 指引。

## 范围外

- 执行生产 SQL。
- 自动删除 Redis key。

## 实现说明

- 测试失败必须区分本次回归与仓库既有失败并保留证据。
- 新权限 `token_usage.*`、`token_quota.*` 必须保留。

## 验收标准

- [x] 旧生产符号、API、配置、页面和 Redis 写入残留扫描通过。
- [x] 新动态体系保护标识和测试完整。
- [x] `usage_logs`、计费、余额、订阅、路由和调度测试通过。
- [x] 后端全量测试通过。
- [x] 前端专项、类型检查和构建通过；全量结果已如实记录。
- [x] 进度、SQL人工边界和 Redis自然过期说明完整。

## 验证计划

- `cd backend && go test ./...`
- `cd frontend && pnpm test:run`
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm build`
- 全仓 `rg` 旧标识与新保护标识扫描。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端全量 | `cd backend && go test ./...` | 全部通过 |
| 前端专项 | Token 统计、路由/RBAC、侧栏、分组路由共 7 个文件 59 个测试 | 全部通过 |
| 前端静态/构建 | `pnpm typecheck`、`pnpm build` | 全部通过 |
| 前端全量 | `pnpm test:run` | 本次相关测试通过；发现 7 项失败，其中路由守卫测试参数错误已修复，余 6 项为既有 OAuth/注册跳转断言与 `/welcome` 行为不一致 |
| 隔离扫描 | 旧生产标识负向 `rg`、新动态标识正向 `rg`、`git diff --check` | 旧标识无生产残留；新标识 65 处；diff check 通过 |
| 人工边界 | `backend/sqlArchiving/166_drop_legacy_fixed_token_statistics.sql`、`docs/legacy-fixed-token-redis-retirement.md` | SQL 未执行；Redis 不自动删除 |

## 执行记录

2026-07-31：开始最终隔离扫描、后端/前端回归与交付核对。

2026-07-31：删除最终扫描发现的分组候选每日 Token 限额残留 UI/持久化调用和旧文案；完成全量后端、前端专项、类型、构建、隔离与交付验收。
