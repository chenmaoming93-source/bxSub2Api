# sub2api 成长计划投产步骤

> 适用版本：Growth Contract v4.0  
> 投产范围：sub2api 管理端、Web 用户端、AIWork U 面、共享账务与履约  
> 当前结论：代码审查发现的 P0/P1/P2 项已完成修复并通过自动化验证；正式投产仍需关闭文末的履约、全链路联调和密钥交付门禁。

## 0. 投产准入检查

以下条件全部满足后才能开始生产变更：

| 检查项 | 通过标准 |
|---|---|
| 数据库 | MySQL 8 / GoldenDB 可连接；已完成全量备份和恢复演练 |
| Redis | sub2api 可连接；生产 nonce 防重放缓存正常 |
| 业务适配器 | Token/订阅履约适配器已接入并完成成功、失败、超时、重试测试 |
| SM2 密钥 | 服务端持有有效私钥；AIWork Main 持有配对公钥；固定向量双方互解成功 |
| 时间同步 | sub2api 与 AIWork 主机 NTP 正常，时钟偏差小于 5 分钟 |
| 网络 | AIWork 到 sub2api 走内网/VPN；HTTPS；源 IP 白名单已配置 |
| RBAC | 已确认管理员、运营只读、普通用户和调账权限角色映射 |
| 验收环境 | 预生产已通过数据库、前后端、U/M/W 全链路回归 |
| 回滚 | 已准备上一版后端制品、前端制品和数据库回滚方案 |

数据库变更前必须执行：

```bash
mysqldump --single-transaction --routines --triggers \
  -h <db-host> -P <db-port> -u <db-user> -p \
  <database-name> > sub2api-before-growth-$(date +%Y%m%d%H%M%S).sql
```

Windows PowerShell 可使用等效的 MySQL 备份命令，不要将数据库密码写入命令行历史。

## 1. 准备发布制品

### 1.1 构建前端

```bash
cd frontend
npm ci
npm run typecheck
npm run test:run
npm run build
```

`npm run build` 会把生产前端产物写入 `backend/internal/web/dist`。必须在前端构建成功后构建后端，确保静态资源被嵌入后端制品。

### 1.2 构建后端

```bash
cd backend
go build -trimpath -o bin/server ./cmd/server
```

当前 Windows 本机可执行：

```powershell
cd backend
go build -trimpath -o server.exe ./cmd/server
```

不要在生产重新生成 Ent 文件；使用已提交并验证过的生成物。

## 2. 配置生产环境

必须确认的配置项：

```yaml
server:
  host: 0.0.0.0
  port: 8080
  frontend_url: https://<sub2api-host>

growth:
  crypto:
    time_window: 5m
```

数据库、Redis、JWT、TOTP 等沿用现有生产配置。不要提交或输出以下敏感值：

- 数据库密码
- Redis 密码
- JWT Secret
- TOTP encryption key
- SM2 私钥

### 2.1 SM2 密钥交付

当前服务端代码期望 **32 字节 SM2 私钥标量的 64 位十六进制字符串**，不是带换行符的 PKCS#8 Base64。

交付步骤：

1. 在受控环境生成或导入 SM2 密钥对。
2. 校验私钥标量范围为 `1 <= d < n`。
3. 由私钥推导公钥，并验证公钥在 SM2 曲线上。
4. 服务端只保存私钥标量；AIWork Main 只保存公钥。
5. 用 `backend/internal/service/growth/crypto/testdata/valid_envelope.json` 同样的固定向量做双方互解测试。
6. 私钥不得进入 Git、日志、截图、IM、工单正文或前端包。

AIWork Main 使用的公钥格式：

```text
04 || X(32 bytes) || Y(32 bytes)
```

### 2.2 权限配置

权限码：

| 权限码 | 建议角色 |
|---|---|
| `growth.read` | 管理员、运营只读 |
| `growth.manage` | 成长计划运营 |
| `growth.adjust` | 仅超级管理员或经审批的财务管理员 |
| `growth.self.read` | 普通用户 |
| `growth.self.exchange` | 普通用户 |

普通用户默认获得：

- `growth.self.read`
- `growth.self.exchange`

管理员和运营角色需在 RBAC 管理页确认授权。

## 3. 执行数据库变更

先进入维护/低峰窗口，停止成长计划相关写入。执行顺序固定如下：

### 3.1 创建成长计划表

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p \
  <database-name> < backend/sqlArchiving/182_create_growth_tables.sql
```

创建 17 张表：

```text
growth_budgets
growth_budget_entries
growth_wallets
growth_transfers
growth_account_ledger
growth_tasks
growth_task_rules
growth_user_tasks
growth_completion_requests
growth_reward_claims
growth_products
growth_product_versions
growth_product_inventory
growth_exchange_orders
growth_fulfillment_jobs
growth_inventory_ledger
growth_admin_audit
```

### 3.2 存量环境升级账务约束与 ADJUST 账户

如果环境此前已经执行过旧版 `182_create_growth_tables.sql`，必须继续执行：

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p \
  <database-name> < backend/sqlArchiving/184_growth_adjustment_account.sql
```

该脚本完成两项必要变更：

- 允许人工调账扣减写入独立的 `ADJUST` 账户，不再混入 `REDEEMED` 消费口径。
- 将预算检查修正为 `distributed_amount <= total_amount`，保证“任务发分 + 人工调账增加”不超过总预算；任务分配仍满足 `allocated_amount <= total_amount`。

全新环境执行最新版 182 后，也应执行一次 184，保持所有环境结构一致。

### 3.3 启用同周期多次任务奖励

任务奖励领取表已增加周期内 `sequence`，唯一键调整为 `(user_id, task_id, cycle_key, sequence)`。如果环境此前使用过只允许单次领取的版本，继续执行：

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p \
  <database-name> < backend/sqlArchiving/185_growth_multi_reward_claims.sql
```

脚本会兼容新旧表结构：缺列时补列，旧三列唯一键存在时自动重建，并补充 `sequence > 0` 检查约束。

### 3.4 更新 RBAC 兼容 seed

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p \
  <database-name> < backend/sqlArchiving/163_seed_rbac_compatibility.sql
```

该脚本必须幂等，可重复执行。

### 3.5 添加 Growth 权限

```bash
mysql -h <db-host> -P <db-port> -u <db-user> -p \
  <database-name> < backend/sqlArchiving/183_add_growth_rbac_permissions.sql
```

### 3.6 数据库验证

```sql
SELECT COUNT(*) AS growth_table_count
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name LIKE 'growth_%';

SELECT constraint_name, check_clause
FROM information_schema.check_constraints
WHERE constraint_schema = DATABASE()
  AND constraint_name IN (
    'chk_growth_budgets_nonnegative',
    'chk_growth_transfers_accounts',
    'chk_growth_account_ledger_values',
    'chk_growth_reward_claims_sequence'
  );

SHOW INDEX FROM growth_reward_claims
WHERE Key_name = 'uq_growth_reward_claims_cycle';

SELECT code, name, module, status
FROM rbac_permissions
WHERE code LIKE 'growth.%'
ORDER BY code;

SELECT p.code
FROM rbac_role_permissions rp
JOIN rbac_roles r ON r.id = rp.role_id
JOIN rbac_permissions p ON p.id = rp.permission_id
WHERE r.code = 'user'
  AND p.code IN ('growth.self.read', 'growth.self.exchange');
```

预期：

- Growth 表数量：`17`
- Growth 权限数量：`5`
- 普通用户 Growth 权限：`2`
- 预算、账户与奖励序号相关检查约束均存在，转账和账务账户枚举均包含 `ADJUST`
- `growth_reward_claims.sequence` 存在，且唯一键包含 `user_id, task_id, cycle_key, sequence`

## 4. 发布后端和前端

### 4.1 停止旧实例

建议通过现有发布平台/服务管理器停止旧实例，先摘除流量，再停止进程，避免中断已有兑换履约。

发布前确认：

```sql
SELECT status, COUNT(*)
FROM growth_exchange_orders
GROUP BY status;

SELECT status, COUNT(*)
FROM growth_fulfillment_jobs
GROUP BY status;
```

如果存在大量 `CREATED` / `FULFILLING` 订单，应先等待处理和退款完成，避免新旧版本切换时订单长期悬挂。

### 4.2 替换制品

发布内容至少包括：

- 后端二进制
- 嵌入的前端静态资源
- 数据库脚本已执行
- 生产配置

### 4.3 启动服务

按现有生产进程管理方式启动后端，禁止使用临时前台窗口作为长期生产运行方式。

确认监听端口、工作目录、配置路径和日志目录正确。

## 5. 上线后冒烟验证

### 5.1 基础可用性

```bash
curl -fsS https://<sub2api-host>/health
```

预期 HTTP `200`。

检查：

- 登录页正常
- 管理员可以登录
- `/admin/growth` 可以打开
- `/growth` 可以打开
- 侧边栏权限显示正确

### 5.2 管理端

按顺序检查：

1. 总预算与任务额度：KPI、预算流水、额度表加载正常。
2. 任务管理：中文任务类型、任务分类、奖励周期显示正常；每日上限和每人上限可配置为大于 `1`；规则历史可打开。
3. 兑换商品：表格/卡片切换、版本历史、库存显示正常。
4. 兑换订单：筛选、详情、CSV 可用。
5. 用户积分：零积分用户可见；无 `growth.adjust` 权限时调账按钮隐藏。

### 5.3 Web 用户端

检查：

- 积分总览与签到只读卡
- 可做任务跳转 AIWork 文案
- 兑换中心商品资格状态
- 兑换确认与余额刷新
- 积分明细筛选和游标加载

确认浏览器 Network 中没有：

```text
POST /api/v1/user/growth/checkins
POST /api/v1/user/growth/completion-requests
```

### 5.4 AIWork U 面

AIWork Main 使用生产公钥依次调用：

1. `POST /api/v1/growth/me`
2. `POST /api/v1/growth/tasks`
3. `POST /api/v1/growth/ledger`
4. `POST /api/v1/growth/products`
5. `POST /api/v1/growth/checkins`
6. `POST /api/v1/growth/tasks/{task_id}/completion-requests`
7. `POST /api/v1/growth/exchanges`
8. `POST /api/v1/growth/exchanges/{order_id}`

接口调用示例见：

[GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md](GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md)

重点验证：

- AIWork 公钥与服务端私钥配对
- C1C3C2 密文格式一致
- SM3-HMAC 拼接顺序一致
- `path` 与路由模板一致
- 重试复用幂等键，不复用 nonce
- 重复请求不重复发分
- 兑换失败自动退款
- 库存预占和退款恢复正确

### 5.5 财务对账

管理员调用：

```text
GET /api/v1/admin/growth/reconciliation
```

预期 `data.ok = true`，且 `issues = []`。

检查：

- 总预算 = TOPUP − REDUCE
- 任务已分配 = 总预算已分配
- 任务已发放 + 调账增加 = 总预算已发放
- 用户钱包余额 = 钱包双边流水净额
- 每笔转账恰好两条流水且金额相抵

## 6. 日志与监控

上线后重点观察：

| 指标 | 异常信号 |
|---|---|
| `CRYPTO_*` | 公钥不配对、时钟偏移、nonce 重放、MAC 顺序错误 |
| `GROWTH_USER_NOT_FOUND` | AIWork 登录邮箱未映射到 sub2api 用户 |
| `POOL_EXHAUSTED` | 任务额度或总预算不足 |
| `FULFILLMENT_UNAVAILABLE` | 履约适配器不可用 |
| Outbox backlog | `PENDING`、`FAILED` 履约任务持续增加 |
| reconciliation | 任意不变量异常 |

日志禁止出现：

- SM2 私钥
- SM4 会话密钥
- 明文 payload
- JWT、数据库密码、Redis 密码

## 7. 回滚方案

### 7.1 应用回滚

如果新版本启动失败：

1. 摘除新实例流量。
2. 恢复上一版后端二进制和嵌入前端资源。
3. 启动旧版并确认 `/health`。
4. 保留 Growth 数据表，不要立即删除。

### 7.2 功能停用

如需快速停用 Growth：

1. 移除普通用户 `growth.self.read`、`growth.self.exchange`。
2. 移除管理员 `growth.read`、`growth.manage`、`growth.adjust`。
3. 下线 AIWork 成长入口。
4. 保留现有数据供退款、对账和审计。

### 7.3 数据库回滚

Growth DDL 是增量新增。生产上默认**不执行 DROP TABLE**：

- 旧版本会忽略新增表。
- 数据删除不可恢复，必须经过负责人审批。
- 如必须恢复数据库，使用发布前备份，并同时回滚应用版本。

### 7.4 履约回滚

回滚前检查 `growth_fulfillment_jobs`：

- `PENDING`：保留新版本 Worker 处理或安排人工处理。
- `PROCESSING`：等待超时释放锁。
- `FAILED`：确认是否已退款；未退款订单进入人工退款流程。
- `SUCCEEDED`：不得再次履约。

## 8. 正式投产门禁

目前以下事项未完成，不能直接转为正式投产：

| 阻塞项 | 关闭条件 |
|---|---|
| 生产 Token/订阅履约适配器未接入 | 接入现有 quota/subscription service，并通过成功/超时/失败/重试/退款测试 |
| 全仓 lint 未通过 | 清理既有非 Growth findings，或先取得明确豁免审批 |
| 真实加密 U→M→W 全链路未验收 | AIWork 项目组使用生产公钥完成固定向量和全流程联调 |
| 已提供的 SM2 私钥格式无效 | 重新生成并按规定安全交付有效私钥；已暴露内容视为泄露 |

以上门禁全部关闭后，才可以执行正式生产发布。
