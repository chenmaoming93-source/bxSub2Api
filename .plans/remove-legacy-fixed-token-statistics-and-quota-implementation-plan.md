# 旧固定 Token 统计与限额体系移除实施 Plan

**状态：Draft — 待用户审核**  
**版本：v0.1**  
**日期：2026-07-30**  
**目标项目：bxSub2Api**  
**前置条件：** 新的“可配置多维 Token 统计与限额系统”已经开发完成、通过验收并投入使用。

## 1. 引言

### 1.1 目标

完整移除以下三套旧固定功能：

1. 模型 Token 统计与限额；
2. 用户＋模型 Token 统计与限额；
3. 分组＋路由别名＋上游模型 Token 统计与限额。

移除范围包括：

- 后端统计累计；
- 后端限额检查；
- Redis 读取、写入、同步和修复；
- MySQL 旧表及对应 Ent Schema；
- Service、Repository、Handler、DTO 和依赖注入；
- 后端 API；
- 前端页面、组件、API、路由、菜单和文案；
- 旧配置、测试和文档。

### 1.2 最高优先级安全原则

本 Plan 只允许删除上述三套旧统计限额的专属逻辑。

必须保证不影响：

- 新的可配置多维 Token 统计与限额系统；
- `usage_logs` 请求用量明细；
- 用户余额和计费；
- API Key 额度；
- 订阅额度；
- 模型路由；
- 账号调度；
- 模型映射；
- 并发控制；
- 账号负载；
- 渠道定价；
- 其他非旧 Token 限额；
- 模型调用协议和上游转发；
- 公共图表、表格、表单、分页和权限组件。

任何同时被旧体系和其他功能使用的代码都属于共享能力，不允许直接删除。

### 1.3 新体系保护边界

不得删除、修改语义或清理新的可配置统计限额体系，包括：

```text
维度注册表
指标注册表
统一用量事件
异步统计队列和 Worker
统计投影管理
通用限额
通用查询
周期封账
MySQL 同步
开发指引和操作手册
```

不得触碰新 Redis 命名空间：

```text
sub2api:dynamic_token_stats:v1:*
sub2api:dynamic_token_stats_ver:v1:*
sub2api:dynamic_token_stats_dirty:v1:*
sub2api:dynamic_token_stats_config:v1:*
```

不得删除新 MySQL 表：

```text
token_stat_projections
token_stat_projection_metrics
token_stat_aggregates
token_stat_quota_rules
token_stat_period_states
```

不得删除新 API 和前端入口：

```text
/admin/token-statistics/*
/admin/token-statistics
```

不得删除或破坏：

```text
token_usage.manage
token_usage.read
token_quota.read
token_quota.update
```

后三个权限仍被新体系复用，只允许更新旧文案和删除旧路由绑定。

### 1.4 不迁移原则

- 不迁移旧历史统计；
- 不迁移旧限额配置；
- 不将旧数据转换成新投影；
- 不保留旧 API 兼容层；
- 不让新体系读取旧表；
- 不让旧体系作为新体系故障兜底。

管理员将在新体系自行建立所需投影和限额。

### 1.5 数据删除执行原则

MySQL 生产表结构删除必须人工执行：

- 本 Plan 的执行者必须生成正式删除 SQL；
- SQL 是本 Plan 的工程交付物并纳入版本控制和代码评审；
- SQL 放入项目现有 SQL 迁移/归档目录，文件编号在执行时按项目规范确定；
- 应用启动、Ent 自动迁移、部署脚本和后台任务不得执行生产 DROP；
- 生产环境由用户人工审核并执行 SQL。

Redis 不主动批量清理：

- 停止旧写入、读取、修复和同步逻辑；
- 核查旧 Key 的 TTL；
- TTL 正常的 Key 等待自然过期；
- 发现 `TTL=-1` 的异常旧 Key 时，只交付精确清单和人工处理建议；
- 不实现或运行宽泛的 Redis 批量删除程序。

## 2. 前置条件和停止条件

### 2.1 执行前置条件

必须全部满足：

- 新可配置体系通过正式验收；
- 新 Redis 累计、MySQL 同步、封账和清理正常；
- 新通用查询页面可用；
- 新限额能够产生限制效果；
- 新体系故障时符合 fail-open 约束；
- 管理员已创建必要投影和限额；
- 新体系监控与告警已部署；
- 已确认旧历史和旧限额无需保留；
- 已安排生产数据库人工 SQL 执行窗口。

### 2.2 强制停止条件

出现以下任一情况必须停止：

- 无法证明候选代码只服务旧三套逻辑；
- 删除后新体系测试失败；
- 删除后模型调用、计费、余额或路由测试失败；
- 外部系统仍调用旧 API；
- 公共前端组件仍有其他调用者；
- 新旧 Redis 前缀存在匹配歧义；
- 旧表仍被非旧功能查询；
- 生产 SQL 未经过人工审核；
- 管理员尚未完成新投影或新限额配置。

## 3. 删除对象分类

### 3.1 旧体系专属代码

同时满足以下条件才允许直接删除：

- 只实现旧三套统计或限额；
- 没有其他生产调用者；
- 新体系没有引用；
- 删除后不影响计费、调用、调度和公共能力。

候选标识包括：

```text
TokenStatisticsModel
TokenStatisticsUserModel
TokenStatisticsGroupCandidate
```

以及对应的：

- 固定字段编解码器；
- Redis 累计器；
- 当前用量读取器；
- Redis 读修复器；
- MySQL 同步器；
- 固定报表 Service；
- 固定限额 Service；
- 固定限额缓存；
- 固定 DTO、Handler 和 Repository。

### 3.2 共享代码

共享代码禁止直接删除，例如：

- `UsageLog` 和 `UsageLogRepository`；
- Token 用量解析和总 Token 计算；
- 账号选择和模型路由；
- 时间工具；
- Redis、MySQL 客户端；
- 通用分页、响应和 RBAC 中间件；
- 公共图表、表格和筛选组件。

处理方式：

- 只删除旧调用点；
- 保留共享实现；
- 必要时移动到中性命名文件；
- 移动时保持行为不变并补充回归测试。

### 3.3 新体系代码

新可配置统计限额体系一律不属于本 Plan 的删除对象。

不能仅凭文件名或文本搜索删除，必须通过以下信息确认身份：

- 包路径；
- Redis 前缀；
- MySQL 表；
- API 路径；
- 构造依赖；
- 生产调用者。

## 4. 后端移除范围

### 4.1 旧统计累计

移除旧三类固定统计累计调用。

重点审查：

```text
incrementDailyTokenQuotasForUsage
accumulateTokenStatisticsForUsage
```

如果函数混合了共享计费或新体系调用，必须先拆分：

```text
旧固定统计调用 → 删除
共享计费调用   → 保留
新动态统计调用 → 保留
```

禁止为了删除旧函数而连带删除整个用量结算流程。

### 4.2 旧限额检查

移除：

- 模型每日 Token 限额；
- 用户模型每日 Token 限额；
- 分组候选每日 Token 限额；
- 旧限额缓存；
- 旧限额导致的候选账号过滤；
- 旧默认限额；
- 旧限额管理和批处理。

必须保留：

- 新通用限额两阶段检查；
- 非 Token 原因的候选排除；
- 模型支持性检查；
- 账号健康、并发和负载；
- 会话粘滞；
- 失败账号排除；
- 渠道限制；
- 模型映射。

### 4.3 旧同步和读修复

移除：

- 旧 `TokenStatisticsSyncEngine`；
- 旧三类 Redis Hash 扫描；
- 旧固定表绝对值 UPSERT；
- 旧当前日 Redis/MySQL 混合读取；
- 旧读修复；
- 旧同步调度；
- 旧统计同步配置。

必须保留：

- 新通用同步任务；
- 新脏集合；
- 新版本 UPSERT；
- 新周期状态机；
- 新 Redis 清理；
- 其他业务的 Redis/MySQL 修复逻辑。

### 4.4 旧管理 Service 和 Repository

候选包括：

```text
TokenUsageReportService
ModelTokenQuotaAdminService
UserModelTokenQuotaAdminService
```

以及只被它们使用的：

- Repository 接口和实现；
- 查询合同；
- 合并函数；
- DTO；
- Handler；
- Wire Provider；
- 单元和合约测试。

删除前必须完成调用关系证明。

### 4.5 旧后端 API

移除：

```text
/admin/token-usage/models
/admin/token-usage/routes
/admin/token-usage/users
/admin/token-usage/options/models
/admin/token-usage/options/groups
/admin/token-usage/options/groups/:group_id/routes
/admin/token-usage/options/groups/:group_id/routes/:route_alias/models
/admin/token-usage/options/users
/admin/token-usage/options/users/:user_id/models
/admin/token-usage/default-target

/admin/model-token-quotas
/admin/users/:id/model-token-quotas
/admin/users/model-token-quotas/batch
/admin/settings/default-model-token-quotas
```

对应路由、Handler、DTO、测试和 API 文档一并移除。

新 `/admin/token-statistics/*` API 不得修改或转发到旧接口。

### 4.6 Wire 和启动依赖

- 删除旧 Provider 和构造参数；
- 更新 Wire 声明；
- 重新生成 Wire 产物；
- 删除 Handler 容器旧字段；
- 删除旧后台任务初始化；
- 增加服务启动测试。

必须验证新统计 Worker、同步任务和通用限额仍正常注入。

## 5. MySQL 清理交付

### 5.1 候选旧表

旧统计表：

```text
model_token_daily_usages
user_model_token_daily_usages
group_candidate_token_daily_usages
```

旧限额表：

```text
model_token_daily_limit_configs
user_model_token_daily_limit_configs
group_candidate_token_daily_limit_configs
```

### 5.2 Ent Schema

移除相应：

- Ent Schema；
- 生成代码；
- 只服务旧表的边；
- Repository 和查询代码。

不得删除 `User`、`Group` 等公共实体或被其他功能使用的边。

### 5.3 SQL 工程交付物

执行阶段必须在项目现有 SQL 迁移/归档目录生成一个独立 SQL 文件，例如：

```text
backend/sqlArchiving/<实际下一个序号>_remove_legacy_fixed_token_statistics.sql
```

最终编号必须在执行时检查目录后确定，不得预先覆盖已有文件。

SQL 文件必须包含：

1. 执行前只读检查；
2. 表存在性检查；
3. 外键、索引和引用关系检查；
4. 明确的保护对象说明；
5. 需要人工确认的删除清单；
6. 显式删除旧外键；
7. 显式删除六张旧表；
8. 执行后检查。

文件注释必须明确禁止删除：

```text
token_stat_* 新体系表
usage_logs
users
api_keys
groups
accounts
```

### 5.4 SQL 验证

Plan 执行者必须：

- 从真实 Schema 获取外键和索引名称；
- 验证 MySQL 版本语法；
- 在测试数据库执行；
- 验证删除前后业务；
- 提交 SQL 供代码评审；
- 提供生产人工执行顺序；
- 提供执行后验证 SQL。

### 5.5 生产执行边界

执行者不得：

- 从应用启动执行 DROP；
- 将 DROP 接入 Ent 自动迁移；
- 将 DROP 接入部署脚本；
- 自动连接生产数据库执行；
- 声称生产表已经删除。

生产 SQL 由用户人工审核和执行。

代码应先发布为“不再访问旧表”的版本。SQL 只有在稳定观察后才允许人工执行。

## 6. Redis 处理

### 6.1 旧统计 Key

旧三类统计 Key：

```text
sub2api:token_stats:model:{日期}
sub2api:token_stats:user_model:{日期}
sub2api:token_stats:group_candidate:{日期}
```

当前代码每次累计都会设置 `EXPIREAT`。

过期配置：

```text
gateway.token_statistics.redis_retention_days
```

默认值为 2，表示业务日期结束后再保留两个完整自然日。

### 6.2 旧限额缓存

旧限额缓存前缀：

```text
quota:daily_token:model:...
quota:daily_token:user_model:...
quota:daily_token:group_candidate:...
```

TTL 为对应自然日结束时间加 5 分钟缓冲。

旧同步锁：

```text
sub2api:token_stats:sync_lock
```

同样具有有限 TTL，并在正常同步结束时主动释放。

### 6.3 默认处理策略

本 Plan 不实现 Redis 批量删除程序。

执行步骤：

1. 删除旧统计写入；
2. 删除旧读修复；
3. 删除旧限额缓存创建和更新；
4. 停止旧同步任务；
5. 对实际旧 Key 执行只读 TTL 审计；
6. TTL 正常的 Key 等待自然过期；
7. 观察旧 Key 数量归零。

### 6.4 TTL 审计

交付 Redis TTL 审计记录，至少包含：

```text
旧前缀
样本 Key
TTL
预期过期时间
是否仍有写入
```

TTL 结果：

- `TTL > 0`：等待自然过期；
- `TTL = -2`：Key 已不存在；
- `TTL = -1`：异常无过期时间，需要单独记录精确 Key。

对于 `TTL=-1`：

- 不自动删除；
- 提供精确 Key 清单；
- 说明来源和风险；
- 提供人工处理建议；
- 明确排除新体系 Key。

### 6.5 新 Redis 保护

TTL 审计和任何人工建议均不得匹配：

```text
sub2api:dynamic_token_stats:v1:
sub2api:dynamic_token_stats_ver:v1:
sub2api:dynamic_token_stats_dirty:v1:
sub2api:dynamic_token_stats_config:v1:
```

## 7. 配置与后台任务

移除只服务旧体系的：

- 固定统计开关；
- 旧 HSCAN 参数；
- 旧日统计保留参数；
- 旧同步批量和重试参数；
- 旧限额缓存参数；
- 旧调度器。

保留：

```text
gateway.dynamic_token_statistics.*
```

如果名称存在重叠，必须先证明新体系使用独立配置，再删除旧配置读取、默认值和校验。

## 8. 前端移除范围

### 8.1 旧统计页面

移除：

```text
/admin/token-usage/models
/admin/token-usage/routes
/admin/token-usage/users
```

候选页面：

```text
ModelTokenUsageView.vue
RouteTokenUsageView.vue
UserModelTokenUsageView.vue
```

### 8.2 旧限额页面和组件

移除：

- 全局模型 Token 限额入口；
- 用户模型 Token 限额弹窗；
- `UserModelTokenQuotaModal.vue`；
- 默认模型 Token 限额设置；
- 用户模型限额批处理；
- 旧分组候选限额控件；
- 旧限额状态展示。

### 8.3 旧前端 API

移除：

```text
frontend/src/api/admin/tokenUsage.ts
frontend/src/api/admin/modelTokenQuotas.ts
```

同时删除：

- `admin/index.ts` 对应导出；
- 专属 TypeScript 类型；
- Mock；
- API 单元测试；
- 所有调用者。

### 8.4 路由、导航和文案

移除：

- 旧 Vue Router 记录；
- 旧导航项和面包屑；
- 旧页面标题；
- 旧 RBAC 路由矩阵项；
- 旧路由守卫测试；
- 旧 i18n 文案。

必须保留新入口：

```text
/admin/token-statistics
```

### 8.5 其他页面中的旧入口

如果用户、分组等页面存在旧限额按钮：

- 删除旧按钮和弹窗；
- 如需快捷入口，可改为跳转新限额页面；
- 跳转不得继续调用旧 API。

### 8.6 公共组件保护

删除前检查所有导入者。仍被其他页面使用的以下组件必须保留：

- 通用趋势图；
- 日期选择器；
- 分页；
- 表格；
- 表单验证；
- 权限组件；
- 搜索选择器。

## 9. RBAC

保留并更新语义：

```text
token_usage.read
token_usage.manage
token_quota.read
token_quota.update
```

只删除旧路由绑定，不删除权限。

权限文案更新为通用体系语义，并使用幂等权限种子迁移。

## 10. 文档

删除或修订：

- 旧统计 API 文档；
- 旧日 Token 限额说明；
- 旧页面截图；
- 旧配置项说明；
- 旧同步任务说明。

保护：

```text
docs/token-statistics-development-guide.md
docs/token-statistics-operation-guide.md
```

SQL 文件是 Plan 的工程交付物，不放入普通 `docs` 目录。

## 11. 实施顺序

### 阶段一：资产分类

产出：

- 文件、符号、API、表、Redis Key、页面清单；
- 每项资产的“旧专属/共享/新体系”分类；
- 新体系保护清单。

未完成分类的资产不得删除。

### 阶段二：删除旧运行时逻辑

- 删除旧统计累计；
- 删除旧限额检查；
- 删除旧同步和修复任务；
- 保留新统计和新限额；
- 执行模型调用、计费、路由和新体系回归测试。

### 阶段三：删除旧后端管理面

- 删除旧 Handler、Service、Repository、DTO；
- 删除旧 API；
- 删除旧 Wire 依赖；
- 更新测试。

### 阶段四：删除旧前端

- 删除旧页面、组件和 API；
- 删除旧路由、菜单和文案；
- 更新 RBAC 矩阵和测试；
- 验证新页面不受影响。

### 阶段五：发布无旧依赖版本

验证：

- 旧 Redis 不再增长；
- 旧表不再被访问；
- 旧 API 返回 404；
- 旧页面不存在；
- 新统计、限额和查询正常；
- 模型调用和计费正常。

### 阶段六：生成和验证 SQL

执行者：

1. 核对真实表、外键和索引；
2. 生成 SQL 工程交付物；
3. 在测试数据库验证；
4. 提交代码评审；
5. 提供生产人工执行说明。

本阶段不执行生产 SQL。

### 阶段七：Redis TTL 审计

- 抽样并记录旧 Key TTL；
- 确认无旧写入；
- 等待正常 Key 自然过期；
- 对异常无 TTL Key 提供人工建议；
- 验证新 Redis Key 不受影响。

### 阶段八：生产人工 SQL 执行后的验证

用户执行 SQL 后，开发者只进行只读验证：

- 旧表不存在；
- 新表存在且数据正常；
- 服务运行正常；
- 新统计、限额和查询正常；
- 计费和模型调用正常。

### 阶段九：残留审计

全仓库扫描旧标识。生产代码、前端代码和运行配置不得残留旧逻辑。

## 12. 关键伪代码

### 12.1 删除候选分类

```text
function classify(candidate):
  callers = find production callers
  storage = find storage dependencies

  if candidate belongs to new dynamic system:
    return PROTECTED_NEW

  if used by any non-legacy feature:
    return SHARED

  if used only by three legacy token systems:
    return LEGACY_EXCLUSIVE

  return UNKNOWN
```

只有 `LEGACY_EXCLUSIVE` 可以直接删除。

### 12.2 SQL 交付门禁

```text
function prepareDropSQL():
  inspect real MySQL schema
  verify application no longer accesses old tables
  verify new tables are excluded
  generate pre-check, drop and post-check SQL
  execute in test database
  submit for review
  never execute against production
```

### 12.3 Redis TTL 审计

```text
function auditLegacyRedisTTL():
  for each exact legacy prefix:
    scan bounded samples
    record TTL and last write evidence

    if TTL > 0:
      wait for natural expiry
    if TTL == -1:
      record exact key and manual recommendation

  assert no new dynamic prefix was scanned or modified
```

## 13. 验证策略

### 13.1 静态检查

搜索旧候选标识并确认生产代码无残留：

```text
TokenStatisticsModel
TokenStatisticsUserModel
TokenStatisticsGroupCandidate
ModelTokenDailyUsage
UserModelTokenDailyUsage
GroupCandidateTokenDailyUsage
ModelTokenDailyLimitConfig
UserModelTokenDailyLimitConfig
GroupCandidateTokenDailyLimitConfig
/admin/token-usage/
/admin/model-token-quotas
```

同时确认新体系标识仍存在：

```text
dynamic_token_statistics
DynamicTokenStat
token_stat_projections
token_stat_aggregates
token_stat_quota_rules
/admin/token-statistics
```

### 13.2 后端回归

必须覆盖：

- 所有模型协议；
- 流式和非流式；
- 重试与账号切换；
- 计费、余额、API Key额度和订阅；
- 新统计累计；
- 新同步和封账；
- 新限额；
- 新查询；
- 服务启动和依赖注入。

### 13.3 前端回归

必须覆盖：

- 新 Token 统计入口；
- 投影管理；
- 新限额；
- 多维查询；
- RBAC；
- 用户、分组和账号管理；
- 公共组件；
- 构建、类型检查和测试。

### 13.4 数据保护

- Redis TTL 审计不修改新 Key；
- 新 MySQL 表不出现在 DROP SQL；
- `usage_logs` 不受影响；
- 计费结果不受影响；
- 测试数据库执行 SQL 后新体系正常。

### 13.5 端到端

1. 在新页面创建投影和限额；
2. 发起模型调用；
3. 验证新 Redis、MySQL和限额；
4. 删除旧运行逻辑；
5. 再次验证新体系；
6. 删除旧前端；
7. 验证旧 Key 自然过期；
8. 在测试库执行删除 SQL；
9. 再次执行完整业务流程。

## 14. 验收标准

| 编号 | 验收条件 |
|---|---|
| AC-01 | 旧模型统计不再累计 |
| AC-02 | 旧用户＋模型统计不再累计 |
| AC-03 | 旧分组＋路由＋上游模型统计不再累计 |
| AC-04 | 三套旧限额均不再参与请求或调度 |
| AC-05 | 旧同步和读修复任务不再启动 |
| AC-06 | 旧后端 API 全部移除 |
| AC-07 | 旧前端页面、组件、路由和入口全部移除 |
| AC-08 | 旧 Redis 写入停止且正常 TTL Key 自然过期 |
| AC-09 | 异常无 TTL Key 形成精确人工处理清单 |
| AC-10 | 交付经过测试和评审的 MySQL 删除 SQL |
| AC-11 | 应用、Ent 和部署流程均不会自动执行生产 DROP |
| AC-12 | `usage_logs`、计费、余额和订阅不受影响 |
| AC-13 | 模型路由、账号调度和模型映射不受影响 |
| AC-14 | 新统计投影、异步写入和同步不受影响 |
| AC-15 | 新限额检查和规则管理不受影响 |
| AC-16 | 新通用查询页面和 API 不受影响 |
| AC-17 | 新 Redis 命名空间没有被修改或清理 |
| AC-18 | 新 MySQL 表未出现在删除 SQL |
| AC-19 | 新 RBAC 权限和页面入口正常 |
| AC-20 | 共享后端和前端组件没有被误删 |
| AC-21 | 全仓库旧逻辑残留扫描通过 |
| AC-22 | 前后端完整测试和端到端测试通过 |

## 15. 回滚

### 15.1 代码阶段

旧表仍存在时可以回滚代码，用于处理：

- 误删共享逻辑；
- 新体系受影响；
- 模型调用回归；
- 前端关键功能回归。

### 15.2 数据阶段

生产 SQL 由用户人工执行。

执行前必须：

- 代码已经不访问旧表；
- 经过稳定观察；
- SQL通过测试和评审；
- 用户完成必要备份。

旧表删除后不以恢复旧业务功能为目标。

## 16. 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| 误删共享计费代码 | 高 | 三分类和端到端测试 |
| 误删新统计逻辑 | 高 | 新体系保护清单和架构测试 |
| SQL误删新表 | 高 | 显式保护清单、测试库验证、人工执行 |
| 自动迁移执行生产DROP | 高 | 禁止接入应用和部署流程 |
| 旧Key无TTL | 中 | TTL审计和精确人工清单 |
| 新限额未配置便删除旧限额 | 高 | 前置门禁和管理员确认 |
| 某协议遗漏旧调用 | 中 | 全协议测试 |
| 前端残留旧入口 | 中 | 路由、菜单和全文扫描 |
| 权限被一并删除 | 高 | 保留通用权限并更新语义 |

## 17. 追踪矩阵

| 需求 | 实施模块 | 验收 |
|---|---|---|
| 只删除旧三套逻辑 | 资产分类、调用链清理 | AC-01～05 |
| 删除旧后端 | Service、Repository、API | AC-05～06 |
| 删除旧前端 | 页面、组件、路由、API | AC-07 |
| Redis自然清理 | TTL审计 | AC-08～09 |
| 手工MySQL删除 | SQL工程交付 | AC-10～11、18 |
| 保护公共业务 | 回归测试 | AC-12～13、20 |
| 保护新体系 | 保护清单、隔离测试 | AC-14～19 |
| 清理残留 | 静态扫描 | AC-21 |
| 完整验证 | 前后端E2E | AC-22 |

## 18. 评审记录

| 版本 | 状态 | 内容 |
|---|---|---|
| v0.1 | Draft — 待用户审核 | 独立移除旧三套固定统计限额；强化共享代码和新体系保护；MySQL删除SQL作为工程交付物并由用户在生产人工执行；Redis采用TTL审计和自然过期 |

请审核本 Plan。若内容符合预期，请明确回复“审核通过”；若需要修改，请列出调整项。明确批准后，将更新为最终版，供后续 MVP 拆分和执行使用。
