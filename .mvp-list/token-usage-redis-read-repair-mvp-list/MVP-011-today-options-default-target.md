# MVP-011：让当天筛选候选和默认目标识别 Redis 实时项

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 集中处理 options/default-target 两类辅助接口及三个维度测试，约 40 分钟。
- Dependencies: `MVP-005, MVP-006, MVP-007`

## 预期成果

今天的筛选候选和默认统计目标能够识别尚未同步到 MySQL 的 Redis 数据；历史日期仍保持 MySQL 行为。

## 背景

现有 `SearchTokenUsageOptions` 与 `FindDefaultTokenUsageTarget` 位于 `token_usage_report_repo.go`，当前只查询 MySQL。

## 范围内

- 为今天的 models/routes/route_models/user_models 候选合并 Redis 实时业务键。
- 当天 default-target 基于最终合并消耗选择。
- 历史日期 default-target 保持 MySQL-only。
- 过滤无法关联有效用户/分组/路由的 Redis 孤立项并记录诊断。
- 保持 options limit、搜索文本和父级约束。

## 范围外

- 不改变前端选择器布局。
- 不新增公开 API。

## 实现说明

- users/groups 基础实体选项仍以 MySQL 为权威。
- 不得为不存在搜索词创建 Redis 数据或负缓存。

## 验收标准

- [x] Redis-only 的有效当天统计项可出现在对应候选中。
- [x] 当天默认目标选择合并后消耗最高的有效项。
- [x] 历史默认目标不访问 Redis。
- [x] 现有 options/default-target API 契约测试通过。

## 验证计划

- `cd backend && go test ./internal/service ./internal/repository ./internal/handler/admin -run 'Test.*TokenUsage.*Option|Test.*DefaultTarget'; pnpm --dir frontend exec vitest run src/api/__tests__/admin.tokenUsage.spec.ts`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/token_usage_report_service.go` | models/routes/route_models/user_models 合并 Redis 当天业务键并保持搜索、limit、parent 约束；users/groups 保持 MySQL 权威；当天默认目标复用最终混合报表。 |
| 测试 | `backend/internal/service/token_usage_options_realtime_test.go` | 覆盖 Redis-only 候选、父级约束、当天最高消耗目标和历史不读 Redis。 |
| 聚焦验证 | `cd backend && go test ./internal/service ./internal/repository ./internal/handler/admin -run 'Test.*TokenUsage.*Option|Test.*DefaultTarget'` | 通过。 |
| 前端契约 | `pnpm --dir frontend exec vitest run src/api/__tests__/admin.tokenUsage.spec.ts` | 通过：1 file、15 tests。 |
| 包回归 | `cd backend && go test ./internal/service ./internal/repository ./internal/handler/admin ./cmd/server` | 通过。 |

## 执行记录

2026-07-15 完成。当天 default-target 复用三维度合并后的排序结果；历史日期直接调用原 Repository。选项合并只读取 Redis，不创建 field 或负缓存；路由和用户模型严格受 parentID 约束。
