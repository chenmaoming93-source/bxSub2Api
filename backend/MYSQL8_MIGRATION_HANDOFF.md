# Backend PostgreSQL → MySQL 8 迁移交接计划

## 1. 目标与边界

- 检查范围：`backend` 中的生产运行代码。
- 迁移目标：将残余 PostgreSQL SQL 语法、驱动 API 和数据库相关实现统一迁移为 MySQL 8。
- 业务要求：保持查询结果、事务边界、并发控制、幂等行为和返回值语义不变。
- 编码要求：涉及中文时只做局部补丁，避免整文件重写造成乱码。
- 明确排除：`backend/migrations`。该目录已经完成 MySQL 8 迁移，不得修改。
- 明确排除：测试文件和测试工作。本计划不包含测试代码迁移、测试执行或测试修复。
- 服务操作：不启动或重启前后端服务。

## 2. 状态说明

| 状态 | 含义 |
|---|---|
| 未开始 | 已定位 PostgreSQL 残留，尚未修改 |
| 进行中 | 已修改部分实现，但模块尚未完成 |
| 基本完成 | 运行代码中的主要 PostgreSQL 实现已替换，仍需人工复核 |
| 完成 | 迁移和静态复扫均已结束 |

> 2026-06-29 拉取并合并至 `544d6b1` 后，当前工作区为干净状态。后续每完成一个生产文件，都必须同步更新本文档。

> 2026-06-29 续作复核发现 `544d6b1` 将乱码换行损坏和未解决冲突标记提交进多个生产文件；已恢复 `account_repo.go` 代码边界，并将 `affiliate_repo.go`、`channel_monitor_repo.go`、`content_moderation_repo.go`、`user_repo.go`、`user_group_rate_repo.go` 恢复到父提交 `b8b0dfa` 的可格式化 MySQL 版本。处理后 `go build ./...` 通过。

> 2026-06-30 针对 MySQL 8 保留字/旧 PostgreSQL 语法完成追加扫描：生产 Go 代码中未再发现未转义的 `JOIN/FROM/UPDATE/INTO groups`、`AS range`、运行时 `ON CONFLICT/RETURNING/ILIKE/date_trunc/FILTER/FULL OUTER JOIN/IS NOT DISTINCT FROM/NULLS FIRST/LAST` 等语法残留；当前剩余命中为注释或科学计数法等误报。

## 3. 核心迁移清单

### 3.1 尚未完成的生产模块

| 优先级 | 模块/文件 | PostgreSQL 残留 | MySQL 8 迁移方案 | 状态 | 估算进度 | 注意事项 |
|---:|---|---|---|---|---:|---|
| P0 | `internal/repository/account_repo.go` | 已完成：JSON、配额递增、时间周期和返回值逻辑均已迁移为 MySQL 8/Go 事务实现 | 静态复扫未发现 PostgreSQL 运行语法；已修复既有乱码注释吞掉字段、函数声明和 SQL 起始符造成的语法损坏 | 完成 | 100% | 2026-06-29 重新 `gofmt` 并复核；该文件已恢复 Go 语法可解析 |
| P0 | `internal/repository/usage_billing_repo.go` | 已完成：幂等写入、余额/API Key/订阅/账户配额更新已迁移为 MySQL 8；2026-06-30 修复订阅用量累加中的 PostgreSQL `UPDATE ... FROM groups` 残留和 cost 参数数量 | 订阅用量更新使用 MySQL `UPDATE ... JOIN \`groups\``，三个 usage 字段分别传入 cost 参数 | 完成 | 100% | 2026-06-30 已 `gofmt` 且 `go build ./...` 通过；后续仍需留意多字段累加时重复占位符必须重复传参 |
| P0 | `internal/repository/usage_log_repo.go` | 已完成：移除 `ON CONFLICT ... RETURNING`、PostgreSQL 批量 CTE/JSON 返回、`FILTER` 与 `::timestamptz` | 单条写入使用 MySQL insert + `LastInsertId`/冲突后查询；创建批次在事务内逐条保持 inserted/ID/创建时间语义；best-effort 使用多值 `INSERT IGNORE`；聚合改为边界 CTE + `CASE WHEN` | 完成 | 100% | 2026-06-29 完成，已 `gofmt` 并静态复扫；基线修复后 `go build ./...` 通过 |
| P1 | `internal/repository/ops_repo_alerts.go` | 已完成：移除 4 处 `RETURNING`、`$n`、JSON `->>`、`IS NOT DISTINCT FROM` | 创建/更新使用事务内 `ExecContext + LastInsertId + SELECT`；统一 `?` 并补齐游标重复参数；JSON 使用 `JSON_UNQUOTE(JSON_EXTRACT(...))`；NULL 比较使用 `<=>` | 完成 | 100% | 2026-06-29 完成并静态复扫；同时修正规则更新、事件状态和邮件状态写入的参数顺序 |
| P1 | `internal/repository/ops_repo_dashboard.go` | 已完成：移除 `percentile_cont`、`FILTER`、`date_trunc` 和 `FULL OUTER JOIN`；2026-06-30 修复动态 usage filter 中 `LEFT JOIN groups` 未转义导致 MySQL 1064 | 原始延迟样本在 Go 端排序并连续线性插值；条件聚合使用 `CASE WHEN`；分钟桶使用 `DATE_FORMAT`；双侧桶使用 `UNION ALL` 汇总；`groups` 表名统一用反引号 | 完成 | 100% | 2026-06-30 已修复当前生产报错链路；后续如新增 SQL 仍需检查 MySQL 保留字表名 |
| P1 | `internal/repository/ops_repo_metrics.go` | 已完成：心跳 upsert 从 `ON CONFLICT/EXCLUDED` 迁移为 MySQL 8 `ON DUPLICATE KEY UPDATE/VALUES` | 保留成功时清错、失败时保留最近成功结果的 CASE 语义 | 完成 | 100% | 2026-06-29 完成并静态复扫 |
| P1 | `internal/repository/ops_repo_openai_token_stats.go` | 已完成：移除 PostgreSQL `::bigint/::numeric/::float8`，保留原聚合与 ROUND 精度 | 使用 MySQL 8 原生 `COUNT/SUM/AVG/ROUND` | 完成 | 100% | 2026-06-29 完成并静态复扫 |
| P1 | `internal/repository/ops_repo_preagg.go` | 已迁移：移除 `percentile_cont`、`FILTER`、`date_trunc`、类型转换、`FULL OUTER JOIN`、`ON CONFLICT/EXCLUDED` | 小时聚合改为 Go 端读取原始行、UTC 小时分桶、按 overall/platform/group 三维聚合并复用连续百分位算法；写入使用 MySQL `ON DUPLICATE KEY UPDATE`；日聚合改为 MySQL `DATE()`、`CASE WHEN` 条件聚合和 upsert | 基本完成 | 95% | 2026-06-29 已 `gofmt`、本文件静态残留扫描通过，且使用工作区 Go 缓存执行 `go build ./...` 通过；仍需人工核对 MySQL 唯一索引/RowsAffected 语义 |
| P1 | `internal/repository/ops_repo_realtime_traffic.go` | 已完成：`date_trunc` 改为 MySQL `DATE_FORMAT + CAST`，不支持的 `FULL OUTER JOIN` 改为双向 `LEFT JOIN + UNION ALL` | 保持分钟桶合并、峰值 QPS/TPS 统计语义 | 完成 | 100% | 2026-06-29 完成并静态复扫 |
| P1 | `internal/repository/ops_repo_trends.go` | 已完成：移除 `split_part`、JSONB lateral 展开、`date_trunc`、`FILTER` 和 `FULL OUTER JOIN`；2026-06-30 修复 throughput breakdown 查询中的 `JOIN groups` 未转义问题 | 使用 `SUBSTRING_INDEX` + `JSON_TABLE` 保持 failover 分类；MySQL `DATE_FORMAT/TIMESTAMP` 分桶；`CASE WHEN` 条件聚合；`UNION ALL` 汇总双侧数据；`groups` 表名统一用反引号 | 完成 | 100% | 2026-06-30 已补齐 MySQL 保留字表名转义，`gofmt` 且 `go build ./...` 通过 |
| P1 | `internal/repository/ops_repo_request_details.go` | 已完成：移除 UNION 查询中的 PostgreSQL 类型转换和 `NULLS LAST`；2026-06-30 修复 UNION 两个分支中的 `LEFT JOIN groups` 未转义问题 | 使用 MySQL 原生字面量/NULL 类型推断及 `duration_ms IS NULL` 排序；为两个 UNION 分支分别传入时间参数；`groups` 表名统一用反引号 | 完成 | 100% | 2026-06-30 已补齐 MySQL 保留字表名转义，`gofmt` 且 `go build ./...` 通过 |
| P2 | `internal/service/backup_service.go` | 已完成：`BackupType` 元数据和注释由 `postgres` 改为 `mysql` | 全生产代码搜索未发现读取端依赖旧值 | 完成 | 100% | 2026-06-29 完成；`go build ./...` 通过 |

### 3.2 已经修改但尚未完成最终复核的模块

| 模块/文件 | 已完成内容 | 状态 | 估算进度 | 接班注意事项 |
|---|---|---|---:|---|
| `internal/repository/ops_repo_histograms.go` | 2026-06-30 修复 MySQL 8 下 `/api/v1/admin/ops/dashboard/latency-histogram` 查询报错：`range` 作为 SELECT 别名触发 1064 语法错误，已改为反引号别名 `` `range` `` | 完成 | 100% | 已 `gofmt`，并使用工作区 Go 缓存执行 `go build ./...` 通过；接口 JSON 字段仍由 service model 的 `Range` 输出，未改 API 结构 |
| `internal/repository/ops_repo.go` | 已替换 `pq.CopyIn`、`RETURNING`、`$n`、`ANY`、`ILIKE`、PostgreSQL 类型转换；已处理 MySQL 保留表名 | 基本完成 | 90% | 复核所有动态条件的参数数量和顺序；检查 `UpdateErrorResolved` 参数顺序 |
| `internal/repository/affiliate_repo.go` | 已替换主要类型转换、冻结时间、配额解冻/转账 CTE、分页、批量数组和 upsert；2026-06-29 已移除合并提交误带入的嵌套冲突标记并恢复父提交可格式化版本 | 进行中 | 75% | 重点检查事务中的 `SELECT ... FOR UPDATE`、唯一邀请码冲突和批量参数数量 |
| `internal/repository/channel_monitor_repo.go` | 已替换 `DISTINCT ON`、`pq.Array/unnest`、条件聚合、日期 interval、批删和 upsert；2026-06-29 已移除合并提交误带入的冲突标记并恢复父提交可格式化版本 | 进行中 | 80% | `UpsertDailyRollupsFor` 的参数数量、MySQL upsert RowsAffected 语义需要人工复核；运行 SQL 静态复扫无 PostgreSQL 残留 |
| `internal/repository/channel_repo.go` | 2026-06-30 修复 `GetGroupPlatforms` 中 `FROM groups` 未转义导致 MySQL 1064 的隐患 | 完成 | 100% | 已改为反引号表名，`gofmt` 且 `go build ./...` 通过 |
| `internal/repository/content_moderation_repo.go` | 已替换 JSONB、`RETURNING`、`ILIKE`、类型转换和分页占位符；2026-06-29 已移除合并提交误带入的冲突标记并恢复父提交可格式化版本 | 基本完成 | 90% | 复核插入列/参数数量以及 `CountFlaggedByUserSince` 参数顺序；静态复扫无 PostgreSQL 运行语法 |
| `internal/repository/dashboard_aggregation_repo.go` | 已启用 MySQL 聚合；替换 PostgreSQL 驱动判断、`ctid`、条件 upsert、时间分桶和分区系统表 | 进行中 | 70% | 时区表依赖、MySQL 分区命名、批量归档与删除一致性是主要风险 |
| `internal/repository/group_repo.go` | 已处理 MySQL `groups` 保留字和已有 JSON_TABLE 查询 | 基本完成 | 90% | 复核所有原生 SQL 是否都使用反引号包裹 `groups` |
| `internal/repository/proxy_repo.go` | 已替换动态日期 interval，并补齐重复时间参数 | 基本完成 | 95% | 只需静态复扫 |
| `internal/repository/user_repo.go` | 已移除 `pq.Array`，改为 `JSON_TABLE`；替换 `ILIKE` 和 `NULLS FIRST/LAST`；2026-06-29 已移除合并提交误带入的冲突标记并恢复父提交可格式化版本 | 进行中 | 80% | 检查 Ent 自定义排序表达式生成的 MySQL SQL；静态复扫无 PostgreSQL 运行语法 |
| `internal/repository/user_group_rate_repo.go` | 已将数组、`unnest`、`ANY/ALL`、`ON CONFLICT` 改为 `JSON_TABLE` 和 MySQL upsert；2026-06-29 已移除合并提交误带入的冲突标记并恢复父提交可格式化版本 | 进行中 | 75% | 核对数组按 ordinality 配对、空列表和 NULL 清理语义；静态复扫无 PostgreSQL 运行语法 |
| `internal/repository/user_platform_quota_repo.go` | 已迁移主要 upsert，并修正部分历史参数数量问题；2026-06-29 已复核占位符数量、软删除复活和批量快照参数，并清理 PostgreSQL 术语注释 | 完成 | 100% | `gofmt` 后本文件 PostgreSQL 运行语法/旧术语静态扫描均无命中；当前 MySQL 唯一键为 `(user_id, platform)`，upsert 会复活软删除记录 |
| `internal/repository/user_subscription_repo.go` | 2026-06-30 修复订阅用量累加中的 PostgreSQL `UPDATE ... FROM groups` 残留和 cost 参数数量 | 使用 MySQL `UPDATE ... JOIN \`groups\``，三个 usage 字段分别传入 cost 参数 | 完成 | 100% | 已 `gofmt` 且 `go build ./...` 通过；该文件此前未列入迁移清单，本次因 MySQL 1064 同类扫描补录 |
| `internal/repository/user_profile_identity_repo.go` | provider grant 改为 `INSERT IGNORE`，avatar 改为 MySQL upsert | 基本完成 | 90% | 检查 RowsAffected 在 `INSERT IGNORE` 下是否仍满足业务判断 |
| `internal/service/admin_service.go` | 已移除金额 PostgreSQL cast，修正 MySQL `LIMIT/OFFSET` 顺序 | 基本完成 | 95% | 静态复扫即可 |
| `internal/service/auth_oauth_first_bind.go` | provider grant 改为 `INSERT IGNORE` | 基本完成 | 95% | 检查重复写入时 RowsAffected 语义 |
| `internal/service/ops_cleanup_executor.go` | 日期转换和 CTE 批删改为 MySQL 派生表；补充 MySQL 缺表识别 | 基本完成 | 90% | 检查动态表名仅来自受控白名单 |

## 4. 仅需清理描述性 PostgreSQL 注释的文件

以下文件当前静态扫描只发现 PostgreSQL 术语注释，不属于功能 SQL 阻塞项。完成核心迁移后统一更新注释即可。

| 文件 | 待清理术语 |
|---|---|
| `internal/repository/affiliate_repo.go` | `ILIKE` |
| `internal/repository/channel_monitor_repo.go` | `DISTINCT ON`、`unnest`、`ON CONFLICT`、`::date`、PostgreSQL |
| `internal/repository/channel_repo_pricing.go` | `LIKE/ILIKE` |
| `internal/repository/group_repo.go` | `ON CONFLICT DO NOTHING` |
| `internal/repository/user_attribute_repo.go` | `ON CONFLICT DO UPDATE` |
| `internal/repository/migrations_runner.go` | PostgreSQL advisory lock 旧说明；实现已使用 MySQL `GET_LOCK` |
| `internal/service/balance_notify_service.go` | `RETURNING` |
| `internal/service/channel_monitor_service.go` | `ON CONFLICT` |
| `internal/service/gateway_service.go` | `RETURNING` |
| `internal/service/leader_lock.go` | Postgres advisory lock |
| `internal/service/scheduler_outbox.go` | PostgreSQL advisory lock |
| `internal/service/ops_models.go` | `ILIKE` |
| `internal/service/ops_service.go` | `ILIKE` |

## 5. 执行计划（不包含测试）

| 阶段 | 工作内容 | 目标产物 | 当前状态 |
|---:|---|---|---|
| 1 | 保护并审阅当前未提交差异；对已修改文件执行格式化和人工参数核对 | 可继续工作的稳定基线 | 完成（冲突残片已清理，`go build ./...` 通过） |
| 2 | 完成 `account_repo.go` 与 `usage_billing_repo.go`，抽取共享的账户配额事务逻辑 | 配额与计费完全 MySQL 8 化 | 未开始 |
| 3 | 完成 `usage_log_repo.go` 的插入去重、批量返回值和聚合迁移 | 使用记录链路完全 MySQL 8 化 | 完成 |
| 4 | 依次完成 Ops alerts、metrics、OpenAI stats、realtime、trends | 中低复杂度 Ops 查询迁移 | 完成 |
| 5 | 单独完成 Ops dashboard 和 preagg 百分位及聚合迁移 | Ops 聚合完全 MySQL 8 化 | 进行中（dashboard 已完成，剩余 preagg） |
| 6 | 复核当前已修改的 14 个文件，修正参数数量、事务和 RowsAffected 语义 | 已修改模块达到完成状态 | 未开始 |
| 7 | 修改备份类型标识，统一更新 PostgreSQL 描述性注释 | 生产代码术语统一 | 进行中（备份类型已完成，描述性注释待统一） |
| 8 | 对 `backend` 生产代码进行最终静态复扫；排除 `migrations` 和测试文件 | PostgreSQL SQL/API 残留清单归零 | 未开始 |
| 9 | 删除不再使用的 `github.com/lib/pq` 生产依赖；如仅测试仍引用则暂不删除 | 依赖清理结果 | 未开始 |

## 6. 迁移规则速查

| PostgreSQL | MySQL 8 |
|---|---|
| `$1, $2` | `?, ?`，重复引用必须重复传参 |
| `RETURNING` | `LastInsertId()`、`RowsAffected()` 或事务内更新后查询 |
| `ON CONFLICT DO NOTHING` | `INSERT IGNORE`，但需检查是否会吞掉非目标唯一键冲突 |
| `ON CONFLICT DO UPDATE` | `ON DUPLICATE KEY UPDATE` |
| `EXCLUDED.column` | `VALUES(column)`，并关注目标 MySQL 版本的弃用提示 |
| `ILIKE` | `LOWER(column) LIKE LOWER(?)` 或确认列使用大小写不敏感 collation |
| `= ANY(array)` | `IN (...)` 或 `JSON_TABLE` |
| `unnest(array)` | `JSON_TABLE`，多数组配对使用 `FOR ORDINALITY` |
| `FILTER (WHERE condition)` | `SUM/COUNT(CASE WHEN condition THEN ... END)` |
| `percentile_cont` | MySQL 8 窗口函数实现，或 Go 端连续线性插值 |
| `date_trunc` | `DATE_FORMAT`、`DATE`、`TIMESTAMP`、`CONVERT_TZ` |
| `value::type` | `CAST(value AS type)` 或删除无必要转换 |
| JSON `->>` | `JSON_UNQUOTE(JSON_EXTRACT(json_col, '$.key'))` |
| JSONB 拼接/删除 | `JSON_SET`、`JSON_MERGE_PATCH`、`JSON_REMOVE` |
| `UPDATE ... FROM` | `UPDATE ... JOIN` |
| `ctid` 批删 | 按主键 `id` 的派生表分批删除 |
| PostgreSQL advisory lock | MySQL `GET_LOCK/RELEASE_LOCK` |

## 7. 接班前必须知道的风险

1. 当前修改尚未形成可确认完成的稳定版本，不能把“已修改”直接视为“已完成”。
2. `user_platform_quota_repo.go` 曾存在 PostgreSQL 编号占位符改成 `?` 后参数没有同步复制的问题；2026-06-29 已逐条复核并清理旧注释，后续如修改该文件仍需重新计数。
3. MySQL `ON DUPLICATE KEY UPDATE` 不支持指定冲突索引；替换 PostgreSQL 定向 `ON CONFLICT` 时要确认不会吞掉其他唯一键冲突。
4. MySQL upsert 的 `RowsAffected` 可能返回 1、2 或 0，与 PostgreSQL 不完全一致，不能直接沿用所有业务判断。
5. `CONVERT_TZ` 使用 IANA 时区名时依赖 MySQL 时区表；若部署环境未加载时区表，应改用可控偏移或 Go 端分桶。
6. 账户配额 JSON 更新不建议机械翻译长 SQL；事务锁定后在 Go 中计算更容易保持业务语义和可维护性。
7. 不要修改 `backend/migrations`。
8. 不要使用可能改变文件编码的整文件写入方式；优先 `apply_patch`，格式化只使用 `gofmt`。
