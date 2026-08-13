# MVP-005：全链路验证

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Dependencies: `MVP-002, MVP-003, MVP-004`

## 目标

验证配置更新、额度重算、页面展示、数据库事务和 Redis 同步的整体行为，并确认请求级并发执行逻辑未被修改。

## 范围

- 检查数据库字段和已有迁移兼容性；
- 验证账号并发变更的候选分配结果；
- 验证分组配置超限拒绝；
- 验证页面 0/正数/`null` 状态；
- 运行后端和前端相关测试；
- 检查 diff，确认未改动请求限流核心路径。

## 验收标准

- [x] 所有前置 MVP 已完成并有证据；
- [x] 后端相关测试通过；
- [x] 前端类型检查和相关测试通过；
- [x] 候选具体并发总和不会超过账号正数并发；
- [x] 账号不限和候选不限展示正确；
- [x] 数据库与 Redis 同步路径有验证证据；
- [x] 请求级并发逻辑未发生范围外改动；
- [x] MVP 清单文档状态和证据完整。

## 验证计划

- 运行 `go test ./...` 或受环境限制时运行相关包测试；
- 运行 `npm run typecheck` 和相关 Vitest；
- 执行 MVP 清单最终一致性检查。

## 证据

完成证据：

- `go test ./internal/domain ./internal/service ./internal/repository ./internal/handler/admin` 通过；
- `npm run typecheck` 通过；
- `npm run test:run -- src/views/admin/__tests__/groupsModelRouting.spec.ts src/api/admin/__tests__/accounts.spec.ts` 通过，4 个测试通过；
- `go test ./...` 已执行，RBAC compatibility seed 和 tokenstat/Redis 现有测试仍失败，失败项与本次改造无关；本次涉及的后端包均通过；
- 代码检查确认本次改造未修改网关请求级并发执行路径。
