# sub2api 成长计划上线变更单

| 项目 | 内容 |
|---|---|
| 变更编号 | `CHG-GROWTH-20261118-001` |
| 变更名称 | sub2api 成长计划 MVP 及代码审查修复上线 |
| 变更类型 | 新增业务模块、数据库增量变更、前后端发布 |
| 申请日期 | 2026-11-19 |
| 计划窗口 | `<待填写>` |
| 实施人 | `<待填写>` |
| 复核人 | `<待填写>` |
| 审批人 | `<待填写>` |
| 变更状态 | **草稿，待审批；当前投产门禁未全部关闭** |

## 1. 变更目的

上线 sub2api 成长计划管理端、Web 用户端和 AIWork U 面接口，并关闭代码审查发现的账务、并发、幂等和数据一致性问题。

本次重点包括：

- 同一任务在同一业务周期内支持按 `daily_limit`、`per_user_limit` 多次发分。
- 修复预算核减、任务额度划拨和发分事务的并发与账务一致性问题。
- 人工调账扣减使用独立 `ADJUST` 账户，不再混入兑换消费口径。
- 补充调账幂等、跨用户幂等冲突、零积分筛选、错误码和前端兼容处理。
- 数据库列名和数据类型调整为 GoldenDB 兼容语法。
- 管理端限额用户选择器改为服务端用户搜索。
- Dashboard 聚合改用固定 UTC 偏移，不依赖 MySQL 时区表。
- 恢复 Growth 页面的统一侧边栏和 Header 布局。

## 2. 影响范围

| 范围 | 影响 |
|---|---|
| 管理端 | 新增 `/admin/growth`，包含预算、任务、商品、订单、用户积分管理 |
| Web 用户端 | 新增 `/growth`，支持积分、任务、商品及兑换记录查看 |
| AIWork | 新增 11 个 `/api/v1/growth/*` U 面接口；接口路径和字段不变 |
| 账务 | 新增任务额度、钱包、转账、双边账务、调账和预算流水 |
| RBAC | 新增 5 个 Growth 权限码 |
| 数据库 | 新增 17 张 `growth_*` 表；升级预算、调账和多次发放约束 |
| 配置 | 新增并确认 `growth.crypto.time_window`，默认 `5m` |
| Dashboard 聚合 | MySQL `CONVERT_TZ` 使用 `+08:00` 等固定偏移，不加载时区表 |
| 现有业务 | 不修改现有账号、支付、订阅等主流程；不删除现有业务表 |

AIWork 协同事项：

- 每次真实任务完成使用新的 `completion_ref` 和新的幂等键。
- 网络超时或结果未知时，重试必须复用原 `completion_ref` 和原幂等键。
- 达到每日或每人上限后，服务端返回 `NOT_ELIGIBLE`。
- 同一 `completion_ref` 重放返回 `ALREADY_COMPLETED`，不会重复发分。

## 3. 数据库变更

### 3.1 变更清单

| 场景/顺序 | 脚本 | 变更内容 | 风险 |
|---|---|---|---|
| 新环境 1 | `backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql` | 创建 17 张 Growth 表、索引和唯一键，不包含数据库校验约束 | 中 |
| 存量环境 1（按需） | `backend/sqlArchiving/growth/ddl/187_growth_drop_legacy_constraints.sql` | 删除旧版 Growth 表遗留的校验约束 | 中 |
| 存量环境 2（按需） | `backend/sqlArchiving/growth/ddl/188_growth_drop_sequence_constraint.sql` | 删除旧版 185 留下的领取序号约束 | 低 |
| 存量环境 3（按需） | `backend/sqlArchiving/growth/ddl/185_growth_multi_reward_claims.sql` | 旧环境增加领取 `sequence` 和多发分唯一键 | 中 |
| 存量环境 4（按需） | `backend/sqlArchiving/growth/ddl/186_growth_goldendb_schema_compat.sql` | 旧环境重命名保留字列 | 中 |
| 存量环境 5（按需） | `backend/sqlArchiving/growth/ddl/189_growth_restore_json_columns.sql` | 将 5 个 JSON 字段恢复为原生 `JSON NULL` | 中 |
| 新/存量 DML 1 | `backend/sqlArchiving/163_seed_rbac_compatibility.sql` | 同步 RBAC 兼容 seed | 低 |
| 新/存量 DML 2 | `backend/sqlArchiving/growth/dml/183_add_growth_rbac_permissions.sql` | 新增 Growth 权限和普通用户授权 | 中 |

核心结构变化：

```text
growth_reward_claims.sequence INT NOT NULL DEFAULT 1
UNIQUE (user_id, task_id, cycle_key, sequence)

growth_budgets:
  distributed_amount <= total_amount

growth_transfers / growth_account_ledger:
  account_type 支持 ADJUST

GoldenDB-compatible columns:
  transfer_status / task_status / progress_status / claim_status
  product_status / order_status / job_status / action_type
  description -> remark

JSON fields:
  conditions / benefit_detail / product_snapshot
  before_json / after_json
  -> JSON NULL
```

### 3.2 执行前置条件

- 已完成数据库全量备份，并验证备份可恢复。
- 已获取数据库变更审批。
- 已停止成长计划写入及兑换履约流量。
- 已确认 MySQL 8 或 GoldenDB 兼容环境支持基础 `ALTER TABLE`。
- 已确认现有 `growth_*` 表状态，但不手工重建已有业务表。

备份示例：

```bash
mysqldump --single-transaction --routines --triggers \
  -h <db-host> -P <db-port> -u <db-user> -p \
  <database-name> > sub2api-before-growth-$(date +%Y%m%d%H%M%S).sql
```

### 3.3 执行命令

全新环境只执行完整建表和 RBAC 脚本：

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/growth/ddl/182_create_growth_tables.sql

mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/163_seed_rbac_compatibility.sql

mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/growth/dml/183_add_growth_rbac_permissions.sql
```

已经执行过旧版 182 的存量环境，再按以下顺序升级：

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/growth/ddl/187_growth_drop_legacy_constraints.sql

mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/growth/ddl/185_growth_multi_reward_claims.sql

mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/growth/ddl/186_growth_goldendb_schema_compat.sql

mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/growth/ddl/189_growth_restore_json_columns.sql
```

如果该环境曾经执行过旧版 `185_growth_multi_reward_claims.sql`，再执行：

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p <database-name> \
  < backend/sqlArchiving/growth/ddl/188_growth_drop_sequence_constraint.sql
```

说明：

- 最新版 182 只用于全新环境，不能在字段已经升级的环境重复执行增量字段升级。
- 185/186/187/188/189 只用于旧版 Growth 表，不使用 `PREPARE`、`EXECUTE` 或动态 SQL。
- 188 仅在旧版 185 已产生对应约束时执行。
- 185 直接替换旧三列唯一键为包含 `sequence` 的新唯一键。
- 187 先删除旧版校验约束，再执行列改名。
- 186 直接重命名列，不创建任何数据库校验约束。
- 189 将 5 个 JSON 字段恢复为 `JSON NULL`，已有合法 JSON 数据保持不变。
- 不在生产执行 Ent 重新生成命令。

## 4. 配置与权限

### 4.1 配置项

```yaml
growth:
  crypto:
    time_window: 5m
```

确认项：

- `server.host`、`server.port`、`frontend_url` 保持生产现有配置。
- 数据库、Redis、JWT、TOTP 等沿用现有生产配置。
- Redis 必须可用，用于 SM2 nonce 防重放。
- 服务端与 AIWork 主机 NTP 正常，时钟偏差小于 5 分钟。
- MySQL 连接不再要求加载 `mysql.time_zone_name`；Dashboard 聚合使用固定 `±HH:MM` 偏移。

### 4.2 RBAC 权限

| 权限码 | 目标角色 |
|---|---|
| `growth.read` | 管理员、运营只读 |
| `growth.manage` | 成长计划运营 |
| `growth.adjust` | 超级管理员或经审批财务管理员 |
| `growth.self.read` | 普通用户 |
| `growth.self.exchange` | 普通用户 |

严禁在变更单、工单正文、日志、截图或聊天记录中填写以下敏感值：

- SM2 私钥
- 数据库密码
- Redis 密码
- JWT Secret
- TOTP encryption key

## 5. 发布步骤

### 5.1 构建

```bash
cd frontend
npm ci
npm run typecheck
npm run test:run
npm run build

cd ../backend
go build -trimpath -o bin/server ./cmd/server
```

必须在前端构建成功后构建后端，使前端静态资源嵌入后端制品。

### 5.2 数据库

1. 执行变更前检查并创建备份。
2. 按第 3.3 节顺序执行 SQL。
3. 执行数据库验证 SQL，确认表、权限、索引和约束。

### 5.3 应用

1. 摘除旧实例流量。
2. 停止旧后端实例。
3. 替换后端二进制及嵌入的前端静态资源。
4. 使用生产配置启动新后端。
5. 确认 `/health` 返回 HTTP `200`。
6. 恢复流量。

### 5.4 AIWork

1. 将 SM2 公钥交付 AIWork Main。
2. 双方使用固定向量完成 SM2/SM4/SM3 互解测试。
3. 更新 AIWork 任务完成逻辑：每次真实完成生成新 `completion_ref` 和新幂等键。
4. 完成 U 面 12 个接口冒烟测试。

## 6. 数据库验证

```sql
SELECT COUNT(*) AS growth_table_count
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name LIKE 'growth_%';

SELECT COUNT(*) AS growth_validation_constraint_count
FROM information_schema.table_constraints
WHERE constraint_schema = DATABASE()
  AND constraint_name LIKE 'chk_growth_%';

SHOW INDEX FROM growth_reward_claims
WHERE Key_name = 'uq_growth_reward_claims_cycle';

SELECT table_name, column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name LIKE 'growth_%'
  AND (
    column_name IN ('status', 'description', 'action')
    OR data_type = 'json'
  );

SELECT code, name, module, status
FROM rbac_permissions
WHERE code LIKE 'growth.%'
ORDER BY code;
```

通过标准：

- Growth 表数量为 `17`。
- `growth_reward_claims.sequence` 存在。
- `uq_growth_reward_claims_cycle` 包含 `user_id, task_id, cycle_key, sequence`。
- Growth 表中不存在 `status/description/action` 列，也不存在 JSON 类型列。
- `conditions/benefit_detail/product_snapshot/before_json/after_json` 均为可空原生 JSON。
- Growth 表数据库校验约束数量为 `0`。
- Growth 权限数量为 `5`。
- 普通用户拥有 `growth.self.read`、`growth.self.exchange`。

## 7. 上线验证

| 场景 | 预期结果 |
|---|---|
| `/health` | HTTP 200 |
| 管理端 `/admin/growth` | 5 个 Tab 正常加载 |
| Web `/growth` | 积分总览、任务、商品、流水正常 |
| 权限不足页面 | 返回明确的 403 提示 |
| Growth 页面布局 | 桌面端侧边栏和 Header 正常；移动端可通过菜单按钮打开抽屉 |
| 任务每日上限配置为 3 | 当天前 3 次可返回 `AWARDED`，第 4 次返回 `NOT_ELIGIBLE` |
| 每人累计上限达到 | 后续完成返回 `NOT_ELIGIBLE` |
| 相同 `completion_ref` 重试 | 返回 `ALREADY_COMPLETED`，不重复发分 |
| 并发两个不同完成实例 | 额度与上限允许时可分别 `AWARDED`，序号连续 |
| 任务额度不足 | 返回 `POOL_EXHAUSTED`，不产生半笔账 |
| 人工调账扣减 | 进入 `ADJUST` 账户，不增加 `spent_total` |
| 预算核减低于已分配/已发放 | 返回 `BUDGET_BELOW_LOCKED` |
| 财务对账 | `data.ok = true`，`issues = []` |
| AIWork 重复 nonce | 拒绝，不重复发分 |

## 8. 回滚方案

### 8.1 触发条件

出现以下任一情况时启动回滚：

- 后端无法启动或 `/health` 持续失败。
- 发分、兑换、退款产生重复扣款或账务不平。
- AIWork 无法建立 SM2 会话，且短时间无法修复。
- RBAC 权限错误导致普通用户或管理员不可用。
- 数据库出现大范围锁等待、死锁或唯一键异常。

### 8.2 应用回滚

1. 摘除新实例流量。
2. 停止新后端实例。
3. 恢复上一版后端及嵌入前端制品。
4. 启动旧版本并验证 `/health`。
5. 保留 Growth 表和数据，禁止直接 `DROP TABLE`。
6. 恢复流量后检查现有订单和履约任务。

### 8.3 功能停用

无法立即回滚数据库时，可先撤销 Growth 权限：

- 撤销普通用户 `growth.self.read`、`growth.self.exchange`。
- 撤销管理员 `growth.read`、`growth.manage`、`growth.adjust`。
- 下线 AIWork Growth 入口。
- 保留全部账务数据用于退款、对账和审计。

### 8.4 数据库回滚

默认不删除 Growth 数据。若必须恢复数据库，只能使用发布前备份，并同步回滚应用版本。恢复操作须由数据库负责人审批。

如果 `185_growth_multi_reward_claims.sql` 已上线且已经产生同一周期多条奖励记录，不可只回退到仅支持单次领取的中间版本应用；应保持 Growth 功能停用，或使用数据库快照整体恢复，避免旧逻辑读取多条领取记录时异常。

## 9. 验证证据

当前代码已完成：

- 全仓后端串行单元测试通过。
- Growth MySQL 集成测试通过。
- `go vet` 通过。
- 大额/并发发分、每日上限、累计上限、序号连续性测试通过。
- 新库只执行最新版 182，连续执行两次通过。
- 旧库执行 `184 → 185 → 186` 升级测试通过。
- 旧版奖励唯一键成功升级为包含 `sequence` 的新唯一键。
- 旧版 JSON 字段保留为可空原生 JSON，数据内容保持一致。
- Growth 管理端和用户端已恢复 `AppLayout`，实际桌面窗口验证侧边栏和 Header 正常。
- MySQL 固定偏移聚合测试通过，Dashboard 聚合不再出现 `bucket_start cannot be null`。
- 限额用户选择器使用服务端搜索并通过类型检查、ESLint 和测试。
- 前端 15 个 Growth 测试、类型检查、ESLint、生产构建通过。
- 后端生产制品构建通过。

## 10. 投产门禁

| 门禁 | 状态 | 关闭条件 |
|---|---|---|
| 代码审查 P0/P1/P2 修复 | 已关闭 | 已通过自动化验证 |
| 同周期多次发分 | 已关闭 | 唯一键、事务和并发测试通过 |
| 生产 Token/订阅履约适配器 | **未关闭** | 接入现有 quota/subscription service，并完成成功/失败/超时/退款测试 |
| 真实加密 U→M→W 全链路 | **未关闭** | AIWork 使用生产公钥完成固定向量和全流程联调 |
| SM2 私钥有效性 | **未关闭** | 重新生成并按安全流程交付有效私钥；不得使用已暴露或格式无效内容 |
| 全仓 lint 门禁 | **未关闭** | 清理既有非 Growth findings，或取得明确豁免审批 |
| 预生产验收 | **待确认** | 完成数据库、管理端、Web、AIWork 全链路回归 |

## 11. 审批

| 角色 | 姓名 | 结论 | 时间 |
|---|---|---|---|
| 业务负责人 | `<待填写>` | `<同意/拒绝>` | `<待填写>` |
| 技术负责人 | `<待填写>` | `<同意/拒绝>` | `<待填写>` |
| DBA | `<待填写>` | `<同意/拒绝>` | `<待填写>` |
| 安全负责人 | `<待填写>` | `<同意/拒绝>` | `<待填写>` |
| 发布负责人 | `<待填写>` | `<同意/拒绝>` | `<待填写>` |

## 12. 关联材料

- [投产变更表](GROWTH_PLAN_RELEASE_CHANGES.md)
- [投产运行手册](GROWTH_PLAN_RELEASE_RUNBOOK.md)
- [AIWork 接口契约](GROWTH_PLAN_AIWORK_API.md)
- [对外接口调用示例](GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md)
- [代码审查报告](GROWTH_PLAN_CODE_REVIEW.md)
