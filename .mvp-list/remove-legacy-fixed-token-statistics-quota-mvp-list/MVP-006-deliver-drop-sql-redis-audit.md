# MVP-006：交付人工删表 SQL 与 Redis TTL 审计

- Protocol: `mvp-list/v1`
- State: `DONE`
- Estimate: `0.5 开发日`
- Estimate rationale: `只交付精确 MySQL SQL、保护检查和只读 Redis 审计方法，不执行生产或测试 DDL。`
- Dependencies: `MVP-005`

## 预期成果

生产管理员获得可审阅、手动执行的六张旧表删除 SQL、执行前后验证查询，以及不会触碰新 key 的 Redis TTL 审计说明。

## 背景

用户明确要求 SQL 不实际运行；Redis 正常 TTL key 等待自然过期，异常 `TTL=-1` 只记录并人工处理。

## 范围内

- 下一编号 `backend/sqlArchiving/NNN_drop_legacy_fixed_token_statistics.sql`。
- 表存在性、保护表、执行顺序和执行后验证 SQL。
- 只读 TTL 审计命令、精确旧前缀、异常处理模板。

## 范围外

- 实际执行 SQL。
- Redis SCAN/DEL/UNLINK 自动清理。

## 实现说明

- MySQL 8 / GoldenDB 语法；每条语句有分号。
- SQL 必须显式排除 `token_stat_*`、`usage_logs` 和核心业务表。

## 验收标准

- [x] SQL 只 DROP 六张旧表且顺序合理。
- [x] SQL 包含执行前后及新表保护验证。
- [x] 文件编号符合项目规约且未接入运行时。
- [x] Redis 审计只读、旧前缀精确、新动态前缀明确排除。
- [x] 明确记录 SQL 未执行、由用户生产手工执行。

## 验证计划

- `rg -n "DROP TABLE|token_stat_|usage_logs|model_token_daily|user_model_token_daily|group_candidate_token_daily" backend/sqlArchiving/NNN_drop_legacy_fixed_token_statistics.sql`
- 人工核对 `backend/migrations/001_init.sql`、Ent 历史 schema 和 MySQL 8 语法。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 人工 SQL | `backend/sqlArchiving/166_drop_legacy_fixed_token_statistics.sql` | 仅含六条旧表 DROP，并提供执行前后与保护表查询；未执行、未测试 DDL |
| Redis 审计 | `docs/legacy-fixed-token-redis-retirement.md` | 七类旧模式只读 SCAN/TTL 审计，新动态前缀明确排除 |
| 静态核对 | `rg` + DROP 语句计数 | DROP 恰为 6 条，目标表名与保护清单正确 |

## 执行记录

2026-07-31：开始编制人工删表 SQL 与只读 Redis TTL 审计说明；SQL 仅静态核对，不执行。

2026-07-31：完成编号 166 的人工删表 SQL 和 Redis TTL 退役审计说明；未连接或变更任何数据库与 Redis。
