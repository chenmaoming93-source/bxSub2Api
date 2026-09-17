# MVP-002：发现部分维度匹配的可重置统计身份

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1.5 个开发者日`
- Estimate rationale: `需要组合投影超集、有效限额、单次 dirty identity 扫描、MySQL 分页与跨来源去重，是本计划中查询规则最密集的一个独立能力。`
- Dependencies: `none`

## 预期成果

给定具体维度子集、指标和自然周期，系统能够以最终一致方式找出所有包含这些维度、值相等且存在有效限额的具体统计身份，并明确区分 `NO_QUOTA` 与 `NO_USAGE`。

## 背景

- 投影和限额管理位于 `backend/internal/service/tokenstat/projection_admin.go`、`quota_admin.go`。
- MySQL 聚合结构位于 `backend/internal/repository/tokenstat/repository.go`，身份字段包括 `period_type`、`period_start`、`projection_id`、`dimension_hash`、`dimension_values` 和 `metric_code`。
- Redis dirty identity 结构当前定义于 `backend/internal/repository/tokenstat/sync_engine.go`，current set 为 `sub2api:dynamic_token_stats_dirty:v1:current`。
- Redis 统计 field 只有 hash，无法反推出维度值；身份发现必须联合 dirty identity 与 MySQL。

## 范围内

- 定义重置请求、统计身份、发现结果及 `NO_QUOTA`/`NO_USAGE` 所需领域类型。
- 校验指标允许限额、周期为 D/W/M、维度已注册、值类型合法且请求不接受 wildcard。
- 查找维度代码集合包含全部输入维度的 ACTIVE 投影。
- 匹配当前有效的 ENABLED 限额，包括具体值和 wildcard，覆盖 OBSERVE 与 ENFORCE。
- 仅对 `sub2api:dynamic_token_stats_dirty:v1:current` 做一次 SSCAN/等效渐进读取并过滤。
- 在 dirty 扫描后分页查询当前周期 `token_stat_aggregates`，使用冗余列或 `dimension_values` 完成精确过滤。
- 按周期、projection、dimension hash、metric 去重，输出可供重置服务消费的完整身份。
- 增加 repository 与 service 单元测试。

## 范围外

- 不读取或扫描 `processing:{token}` 集合。
- 不获取同步锁，不二次读取 dirty set，不提供强一致快照。
- 不写入重置快照。
- 不实现 Redis pipeline、HTTP API 或页面。
- 不为身份发现新增长期 Redis 反向索引。

## 实现说明

- 顺序固定为“读取一次 current dirty identity → 查询 MySQL”，缩小但不消除 processing 窗口。
- dirty identity 被读入内存后，即使同步任务随后删除 member，候选仍保留。
- MySQL 查询应稳定分页，避免一次载入全部当前周期聚合；动态筛选必须由已注册维度白名单生成，不能接受任意 SQL 字段。
- 可将 dirty identity 的共享结构从 `sync_engine.go` 中适度抽取，避免重置服务复制 JSON 协议。
- 限额确认完成后返回固定候选集合，不在后续写入前二次校验。

## 验收标准

- [x] 输入 `user_id+group_id` 能匹配同维投影及包含额外 `upstream_model` 的超集投影。
- [x] 缺少任一输入维度、值不同、指标不同或周期不同的身份不会返回。
- [x] 没有 ENABLED 且当前生效的适用限额时返回 `NO_QUOTA`。
- [x] 存在适用限额但 dirty/MySQL 均无当前周期身份时返回 `NO_USAGE`。
- [x] wildcard 限额可以匹配请求提供的具体维度值；请求自身不能提交 wildcard。
- [x] dirty-only、MySQL-only 和两者重复三种身份场景结果正确。
- [x] dirty 与 MySQL 重复身份只返回一次。
- [x] 测试证明 dirty 只读取一次、不获取同步锁、不读取 processing set。
- [x] processing 窗口和扫描后新增身份被明确保留为可接受的一致性边界。

## 验证计划

- `cd backend && go test ./internal/repository/tokenstat ./internal/service/tokenstat`
- 使用 sqlmock 验证当前周期、projection、metric 及分页参数，检查查询不修改聚合数据。
- 使用 miniredis 构造 current dirty identity，验证单次扫描、过滤和跨来源去重。

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 领域服务 | `backend/internal/service/tokenstat/reset_identity_discovery.go` | 实现请求校验、投影超集、有效 quota、单次 dirty→MySQL 分页、去重及完整规则匹配。 |
| 数据源 | `backend/internal/repository/tokenstat/reset_identity_sources.go` | 实现 current dirty 单遍 SSCAN、维度解码及 MySQL 白名单列 keyset 分页，无 DDL。 |
| 测试 | `reset_identity_discovery_test.go`、`reset_identity_sources_test.go` | 覆盖超集、wildcard/具体规则、跨来源去重、状态、调用顺序、无锁/processing 及 SQL 参数。 |
| 命令 | `cd backend && go test ./internal/repository/tokenstat ./internal/service/tokenstat` | 通过。 |
| 格式 | `gofmt -d ...`、`git diff --check -- ...` | 通过。 |

## 执行记录

2026-09-09 完成。身份发现严格先完整扫描一次 current dirty，再分页查询 MySQL；不获取同步锁、不读取 processing、不二扫。无效 dirty member 记录警告后跳过，避免单个损坏 member 阻断内部重置。全部现有注册维度使用静态冗余列白名单过滤，未新增数据库结构。
