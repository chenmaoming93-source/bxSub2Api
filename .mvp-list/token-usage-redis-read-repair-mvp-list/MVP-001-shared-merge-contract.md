# MVP-001：建立当天消耗集合合并契约

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 只定义共享类型、业务键与纯合并规则，并用表驱动测试锁定四种数据状态，范围适合一次约 40 分钟的聚焦改动。
- Dependencies: `none`

## 预期成果

后端拥有一套与存储实现无关的当天 Token 消耗集合合并契约，能够稳定表达 Redis 覆盖、Redis-only 保留、MySQL-only 回退及修复候选。

## 背景

当前 `backend/internal/service/token_usage_report_service.go` 直接消费 MySQL 报表结果，`backend/internal/repository/token_statistics_quota_repo.go` 只支持逐项覆盖。后续三个维度需要共享相同合并语义。

## 范围内

- 在 `backend/internal/service` 定义模型、路由候选、用户模型的当天累计记录或等价共享类型。
- 实现三个维度稳定、无歧义的业务键生成规则。
- 实现不依赖 Redis/MySQL 客户端的纯集合合并函数，返回最终集合与 MySQL-only 修复候选。
- 补充四种状态、绝不相加、两边均无数据的单元测试。

## 范围外

- 不访问 Redis 或 MySQL。
- 不改报表接口和页面。
- 不实现读修复写入。

## 实现说明

- 优先复用现有 `ModelTokenUsageRow`、`RouteTokenUsageRow`、`UserTokenUsageRow`，避免重复 DTO。
- 合并函数必须保持 Redis `used_tokens`，同时允许用 MySQL 元数据补齐限额、用户名、分组名和优先级。

## 验收标准

- [x] 三种维度均可按批准 Plan 中的业务键完成集合去重。
- [x] Redis 与 MySQL 同时存在时最终值只取 Redis，不进行相加。
- [x] Redis-only 保留；MySQL-only 进入最终结果并被标记为修复候选。
- [x] 相关 service 单元测试通过。

## 验证计划

- `cd backend && go test ./internal/service -run 'Test.*TokenUsage.*Merge'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/service/token_usage_merge.go` | 已实现模型、路由候选、用户模型的结构化业务键与纯集合合并函数；Redis 值优先，MySQL 补齐元数据并产生修复候选。 |
| 测试 | `backend/internal/service/token_usage_merge_test.go` | 覆盖两边都有、Redis-only、MySQL-only、两边为空、不相加、复合键分隔符不碰撞与元数据补齐。 |
| 聚焦验证 | `cd backend && go test ./internal/service -run 'Test.*TokenUsage.*Merge'` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 5.860s`。 |
| 包回归 | `cd backend && go test ./internal/service` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/service 53.268s`。 |

## 执行记录

2026-07-14 完成。使用结构体作为业务键，避免字符串拼接分隔符导致的歧义；最终集合先保留 Redis 输入顺序，MySQL-only 项追加并作为后续读修复候选。未访问 Redis/MySQL，未修改报表接口或页面。
