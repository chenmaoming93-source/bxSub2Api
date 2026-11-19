# Growth SQL

Growth 投产脚本按用途分目录，避免执行脚本时混入不同类型语句。

- `ddl/`：仅包含 `CREATE`、`ALTER`、`DROP` 等结构变更。
- `dml/`：仅包含用于初始化或修复数据的 `INSERT`、`UPDATE`、`DELETE`。

## 新环境

1. 执行 `ddl/182_create_growth_tables.sql`。
2. 执行仓库现有 RBAC DML：`../163_seed_rbac_compatibility.sql`。
3. 执行 `dml/183_add_growth_rbac_permissions.sql`。

## 已执行旧版 182 的环境

按实际旧结构选择增量脚本：

1. 有旧版数据库校验约束时，执行 `ddl/187_growth_drop_legacy_constraints.sql`。
2. 如果曾执行旧版 185，且仍存在 `chk_growth_reward_claims_sequence`，执行 `ddl/188_growth_drop_sequence_constraint.sql`。
3. 奖励领取仍是单次唯一键时，执行 `ddl/185_growth_multi_reward_claims.sql`。
4. 存在 `status`、`description`、`action` 旧列名时，执行 `ddl/186_growth_goldendb_schema_compat.sql`。
5. 5 个 Growth JSON 字段仍是 `LONGTEXT` 时，执行 `ddl/189_growth_restore_json_columns.sql`。
6. 执行 `dml/183_add_growth_rbac_permissions.sql`。

`182` 和 `189` 中的 JSON 字段均为可空原生 `JSON`。数据库不创建 `CHECK` 约束，枚举与金额边界由应用层校验。
