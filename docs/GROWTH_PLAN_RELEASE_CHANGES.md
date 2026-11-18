# 成长计划本次投产变更表

- 投产批次：sub2api 成长计划 MVP
- 日期：2026-11-18
- 范围：管理端、Web 用户端、AIWork U 面、共享账务与履约
- 当前状态：业务功能已完成；`MVP-019` 发布总验收仍为 `BLOCKED`

> 本文把自动生成的 138 个 Ent 文件和同类实现文件按模块汇总。上线时仍应以 Git 提交中的完整文件列表为准。

## 1. 数据库与权限

| 类型 | 对象 | 变更内容 | 投产动作 |
|---|---|---|---|
| DDL | `backend/sqlArchiving/182_create_growth_tables.sql` | 新增 17 张成长计划表：总预算、预算流水、钱包、转账、双边账务、任务、任务规则、用户任务进度、完成请求、奖励领取、商品、商品版本、库存、兑换订单、履约任务、库存流水、管理员审计 | 上线前执行 |
| DML | `backend/sqlArchiving/183_add_growth_rbac_permissions.sql` | 新增 5 个权限码；为普通用户角色授权 `growth.self.read`、`growth.self.exchange` | 在 182 后执行，可重复执行 |
| DML | `backend/sqlArchiving/163_seed_rbac_compatibility.sql` | 兼容 seed 同步加入 5 个 growth 权限及普通用户授权 | 在 182 后执行 |
| Ent 元数据 | `backend/ent/schema/user.go` | 用户增加成长钱包关联关系 | 随代码发布 |
| ORM 生成物 | `backend/ent/growth*`、`backend/ent/growth*/`、17 个 `backend/ent/schema/growth_*.go` | 17 个 Growth Entity、查询条件与 CRUD 生成代码，共 138 个文件 | 随代码发布，不单独执行 |

新增权限：

| 权限码 | 风险级别 | 用途 |
|---|---|---|
| `growth.read` | low | 查看管理端成长计划 |
| `growth.manage` | high | 管理预算、任务、商品、订单履约 |
| `growth.adjust` | critical | 用户积分手工调账 |
| `growth.self.read` | low | Web/AIWork 查看个人积分 |
| `growth.self.exchange` | high | Web/AIWork 创建商品兑换 |

投产前必须备份数据库，并在预生产验证 17 张表的索引、唯一键、外键约束和外键级联策略。

## 2. 后端运行时变更

| 模块 | 主要文件/目录 | 变更内容 |
|---|---|---|
| 服务与账务 | `backend/internal/service/growth/` | 总预算、任务额度、规则版本、钱包、双边转账、签到、任务完成、商品版本、库存、订单、Outbox 履约、退款、调账、审计、对账 |
| SM2 网关 | `backend/internal/service/growth/crypto/` | SM2 C1C3C2、SM4-CBC、SM3-HMAC、时间窗、nonce 防重放、payload/path 校验、固定向量测试 |
| U 面 Handler | `backend/internal/handler/growth/` | AIWork 11 个操作的参数解析、业务调用、幂等处理 |
| M/W 面 Handler | `backend/internal/handler/growth/admin_points_handler.go` 等 | 管理端数据/写操作、Web 用户端读接口和兑换 |
| Redis nonce | `backend/internal/service/growth/nonce_cache.go` | 防重放 nonce 窗口缓存 |
| 履约 Worker | `backend/internal/service/growth/fulfillment_service.go` | 订单预占、Outbox、自动重试、人工完成、失败退款、库存恢复 |
| 对账 | `backend/internal/service/growth/admin_service.go` | 六类账务不变量检查；用户积分列表支持零钱包用户 |
| 路由 | `backend/internal/server/routes/growth.go` | AIWork U 面 `/api/v1/growth/*` |
| 路由 | `backend/internal/server/routes/user_growth.go` | Web W 面 `/api/v1/user/growth/*` |
| 路由 | `backend/internal/server/routes/admin.go` | 管理 M 面 `/api/v1/admin/growth/*` |
| 中间件 | `backend/internal/server/middleware/growth_payload.go` | SM2 信封解密、用户邮箱映射、nonce 和错误 envelope |
| 配置 | `backend/internal/config/config.go` | `growth.crypto.time_window`，默认 `5m` |
| RBAC | `backend/internal/rbac/permissions.go`、`coverage.go` | 5 个权限码；U 面登记为 SM2 信任边界 |
| 依赖装配 | `backend/cmd/server/wire.go`、`wire_gen.go`、`backend/internal/handler/wire.go`、`backend/internal/server/http.go`、`router.go` | Growth Service、Handler、Decryptor、Fulfillment Worker 接入 |
| 依赖 | `backend/go.mod`、`backend/go.sum` | 国密、Redis、UUID、Ent 等依赖 |
| CI/Lint | `backend/.golangci.yml` | 为 Growth Redis nonce 及既有 tokenstat Redis 实现补充 depguard 路径例外 |

## 3. AIWork 对外接口

全部接口为 `POST /api/v1/growth/*`，使用 SM2 数字信封，无 JWT/API Key。

| # | URL | 用途 | 幂等 |
|---|---|---|---|
| 1 | `/growth/me` | 个人积分概览、签到状态、任务摘要 | 否 |
| 2 | `/growth/tasks` | 任务列表与当前进度 | 否 |
| 3 | `/growth/tasks/{task_id}` | 任务详情与完成引导 | 否 |
| 4 | `/growth/ledger` | 积分流水，游标分页 | 否 |
| 5 | `/growth/products` | 可兑换商品列表 | 否 |
| 6 | `/growth/products/{product_id}` | 商品详情与兑换资格 | 否 |
| 7 | `/growth/exchanges` | 我的兑换订单列表 | 否 |
| 8 | `/growth/checkins` | AIWork 判定签到后提交发分 | 是 |
| 9 | `/growth/tasks/{task_id}/completion-requests` | AIWork 判定任务完成后提交发分 | 是 |
| 10 | `/growth/exchanges` | 创建兑换订单 | 是 |
| 11 | `/growth/exchanges/{order_id}` | 兑换履约状态查询 | 否 |

调用说明：[GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md](GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md)  
冻结契约：[GROWTH_PLAN_AIWORK_API.md](GROWTH_PLAN_AIWORK_API.md)

## 4. 前端变更

### 4.1 管理端

| 文件/目录 | 变更内容 |
|---|---|
| `frontend/src/views/growth/AdminGrowthView.vue` | `/admin/growth` 页面外壳、权限 403、URL-backed 5 Tab、全局刷新 |
| `frontend/src/views/growth/admin/BudgetTab.vue` | 总预算 KPI、任务额度、增减预算、追加/回收、预算流水 |
| `frontend/src/views/growth/admin/TasksTab.vue` | 任务筛选、创建/编辑、规则发布、下线/上线、额度、规则历史 |
| `frontend/src/views/growth/admin/ProductsTab.vue` | 商品表格/卡片、版本、发布、下架、库存、Token 商品校验 |
| `frontend/src/views/growth/admin/OrdersTab.vue` | 订单筛选、失败提示、详情时间线、重试、退款、人工完成、CSV |
| `frontend/src/views/growth/admin/UserPointsTab.vue` | 用户搜索/区间/排序、三合一抽屉、调账、CSV |

### 4.2 Web 用户端

| 文件/目录 | 变更内容 |
|---|---|
| `frontend/src/views/growth/UserGrowthView.vue` | `/growth` 页面、权限 403、兑换确认、有限轮询、余额局部刷新 |
| `frontend/src/views/growth/user/OverviewTab.vue` | 任务、商品预览、最近流水 |
| `frontend/src/views/growth/user/StoreTab.vue` | 商品筛选/排序、兑换记录 |
| `frontend/src/views/growth/user/LedgerTab.vue` | 类型/时间筛选、游标加载、移动端卡片 |
| `frontend/src/views/growth/pollExchange.ts` | 最多 5 次、2 秒间隔、页面隐藏/终态停止 |

### 4.3 共享组件、API 与入口

| 文件/目录 | 变更内容 |
|---|---|
| `frontend/src/components/growth/` | Tabs、积分金额、额度条、状态徽标、提示条、积分 Hero、签到卡、商品卡、页面规则与测试 |
| `frontend/src/api/growth/` | M/W API 客户端、TypeScript 类型、Growth API barrel |
| `frontend/src/i18n/locales/growth.zh.ts`、`growth.en.ts` | Growth 完整中英文命名空间 |
| `frontend/src/i18n/locales/zh.ts`、`en.ts` | 挂载 Growth 命名空间 |
| `frontend/src/router/index.ts` | 追加 `/admin/growth`、`/growth` 懒加载路由和权限页降级 |
| `frontend/src/rbac/permissionMatrix.ts` | 两个页面权限映射 |
| `frontend/src/components/layout/AppSidebar.vue` | 管理端和用户端菜单入口 |

## 5. 测试与验证资产

| 类型 | 文件/目录 | 覆盖 |
|---|---|---|
| Ent Schema | `backend/ent/schema/growth_schema_test.go` | 17 个 Schema 与约束 |
| 配置 | `backend/internal/config/growth_config_test.go` | 时间窗配置 |
| Handler | `backend/internal/handler/growth/*_test.go` | 路由、字段契约、参数 |
| Middleware | `backend/internal/server/middleware/growth_payload_test.go` | 信封错误分类、生产公私钥配对 |
| Service Unit | `backend/internal/service/growth/*_unit_test.go` | 纯业务规则 |
| Service Integration | `backend/internal/service/growth/*_integration_test.go` | MySQL 事务、并发、幂等、账务、履约、退款、对账 |
| MySQL Schema | `backend/internal/repository/growthmysqltest/` | 182 DDL 实际执行 |
| Route/RBAC | `backend/internal/server/routes/*growth*_test.go` | M/U/W 路由和权限注册 |
| SM2 Contract | `backend/internal/service/growth/crypto/` | 固定信封向量 |
| 前端 | `frontend/src/components/growth/__tests__/`、`frontend/src/views/growth/__tests__/` | 状态矩阵、轮询、结构回归 |
| 测试兼容 | `backend/cmd/server/wire_gen_test.go`、`backend/internal/service/account_repository_test_compat_test.go`、`backend/migrations/stop_reason_hotfix_test.go` | 依赖注入参数、Unit build tag、已有迁移断言适配 |
| 截图 | `mvp-list/evidence/screenshots/` | 管理端/用户端、375/768/1440、浅色/深色 |
| 验收报告 | `mvp-list/evidence/growth-release-checklist.md` | 命令、浏览器流程、限制 |

## 6. MVP 与设计文档

新增：

- `mvp-list/growth-plan-mvp-list/`：MVP-001～MVP-019 与进度表。
- `mvp-list/evidence/growth-release-checklist.md`
- `docs/GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md`
- `docs/GROWTH_PLAN_DESIGN.md`
- `docs/GROWTH_PLAN_AIWORK_API.md`
- `docs/GROWTH_PLAN_AIWORK_DEV.md`
- `docs/GROWTH_PLAN_SUB2API_DEV.md`
- `docs/GROWTH_PLAN_PAGES_DESIGN.md`
- `docs/prototype/growth_pages_preview.html`
- `RBAC_ENDPOINT_INVENTORY.md` 中追加 Growth M/W/U 路由记录。

## 7. 刻意未包含

| 项目 | 原因 |
|---|---|
| `backend/internal/server/middleware/provisioning_hardening.go` 的格式化差异 | 与 Growth 无关的既有工作树变更，不纳入本次投产说明。 |
| `backend/config/config.yaml` | 本机敏感配置，不作为投产制品提交；生产只需确认 `growth.crypto.time_window`，默认 `5m`。 |
| Docker Compose | 本次使用本机 MySQL 8.4 验证；Docker Desktop 引擎不可用，未新增或修改 Compose 配置。 |
| AIWork Main/Renderer 源码 | 由其他项目组实现；sub2api 只交付 U 面、固定公钥、契约和调用示例。 |

## 8. 建议投产顺序

1. 备份数据库，确认 MySQL 8 / GoldenDB 兼容性。
2. 执行 `182_create_growth_tables.sql`。
3. 执行 `163_seed_rbac_compatibility.sql`。
4. 执行 `183_add_growth_rbac_permissions.sql`。
5. 发布后端，确认 `/health` 正常。
6. 发布前端，检查 `/admin/growth`、`/growth` 和权限菜单。
7. 将固定 SM2 公钥交付 AIWork Main，完成固定向量互操作测试。
8. 配置并验证 Token/订阅生产履约适配器。
9. 验证自动兑换、失败退款、库存恢复、对账和告警。
10. 完成 `MVP-019` 剩余发布门禁后再对外投产。

## 9. 投产阻塞

| 阻塞项 | 影响 | 当前处置 |
|---|---|---|
| 生产 Token/订阅履约适配器未接入 | 自动兑换会安全失败并退款，无法实际发放权益 | 投产前接入现有 quota/subscription service |
| 全仓 `golangci-lint run ./...` 仍失败 | 98 个既有、非 Growth 问题 | 本次变更集 lint 为 0；全仓门禁需另行清理 |
| 真实加密 U→M→W HTTP 全链路未跑通 | 已有字段契约和固定向量，但缺少真实跨平面联调证据 | 与 AIWork 联调时完成 |
