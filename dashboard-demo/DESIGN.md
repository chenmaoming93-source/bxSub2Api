# 模型使用观测台 · 设计与实现方案

> 版本 v1.8 / 2026-11-11
> 配套 Demo：`dashboard-demo/index.html`（零依赖单文件，双击即可打开）
> 适用项目：bxSub2Api（Wei-Shaw/sub2api 企业定制 fork，Go + Gin + Ent / Vue 3 + Chart.js）
>
> v1.8 明确采用“一个模型账号严格对应一个上游模型”的产品口径。观测台为纯只读诊断页面，不提供流量切换、候选容量写入、回滚或审计处置；Demo 中的确定性模拟曲线只用于表达交互，不构成后端能力假设。

---

## 1. 定位与目标

一个面向**平台运维与管理员**的实时观测看板，回答四类问题：

| 问题 | 看板载体 | 决策 |
|---|---|---|
| 哪个模型账号快打满了？ | 模型压力排行、账号维度表 | 摘除候选 / 扩容 / 上调并发 |
| 哪个业务场景在挤占容量、什么时候？ | 场景 × 时段热力图、趋势图 | 分时并发配置 / 候选容量调整 |
| 异常 token 消耗来自谁、哪条路由？ | 用户 / 路由别名维度下钻 | 配额重置 / 路由优先级调整 |
| 哪个账号或路由需要人工关注？ | 压力排行、告警和完整路径 | 给出排查建议；线上配置由对应管理页面人工处理 |

设计目标：**5 秒内判断全局是否健康，30 秒内定位到具体维度对象，2 分钟内形成可执行的排查结论**。

---

## 2. 参考与取舍

| 参考 | 采纳 | 不采纳 |
|---|---|---|
| Grafana Four Golden Signals / inference-gateway Grafana dashboard | 5s 刷新、按 `model/provider` 打标、速率 + 延迟 + 错误率同屏 | 直接引入 Grafana：本项目已有 admin 控制台，双入口成本高 |
| agent-otel-gateway | User 变量贯穿所有面板、per-user 表格、按 agent 拆分 | OTLP 采集链路改造（超出本期范围） |
| GatewayZ Observability Stack | 跨信号联动（点指标 → 下钻对象 → 排查建议）、两层告警 | Prometheus/Mimir/Loki/Tempo 全家桶（重，先用现有 tokenstat 预聚合） |
| 本项目 `TokenStatisticsView.vue` / `DashboardView.vue` / `components/charts` | 视觉规范、指标卡结构、Chart.js 10 色板 | — |

结论：**不新增 Prometheus/Grafana/Loki 监控栈**，复用 `service/tokenstat` 的 Token 累计、现有 `ops_error_logs` / `usage_logs` 请求事实和 `concurrency_service` 的 Redis 槽位。实时 Token/配额使用 `tokenstat` 作为权威累计；请求数、耗时、TTFT、失败率必须来自请求事实表；分钟/小时趋势需要新增轻量聚合或采样。看板保持只读，所有线上配置变更继续由现有账号、分组和并发管理功能完成。

### GitHub 案例观察

| 案例 | 可借鉴的设计 | 本方案取舍 |
|---|---|---|
| [damokuye-ethanhuang/litellm-operation-dashboard](https://github.com/damokuye-ethanhuang/litellm-operation-dashboard) | 从 SpendLogs 拆分 token、请求、流式延迟，按模型与用户观察 | 保留“Token + 延迟 + 错误”同屏，数据源换成本项目 tokenstat，避免引入 Grafana |
| [lhchingit/agent-otel-gateway](https://github.com/lhchingit/agent-otel-gateway) | 统一 Claude Code、Codex、Gemini CLI 等 Agent 标签，提供 Grafana 看板 | 采用 agent / user / model 的统一标签思路；本期不新增 OTLP 采集链路 |
| [alebgl77/grafana-llmops-forge](https://github.com/alebgl77/grafana-llmops-forge) | FinOps、Agent、质量与治理拆分，配合查询和部署检查 | 保留容量、成本、SLO 的跨信号组织方式，但信息架构按本项目的分组/路由模型重排 |

三类案例的共同点不是图表样式，而是：统一标签、指标可下钻、阈值可解释、异常能够定位到具体对象。Demo 因而把“趋势 → 压力排行/热力图 → 维度下钻 → 排查建议”放在同一条只读分析路径上。

---

## 3. 信息架构

```
顶部：环境 / 时间范围(1h·6h·24h·7d) / 自动刷新 / 数据时间
├─ 全局状态条     整体负载、活跃模型账号、待关注告警、数据延迟
├─ 对象筛选层     四维多选（用户/分组/别名/模型账号）+ 已选 chips + 范围 Token/并发/流量摘要
├─ KPI 区 (6)     Token 速率 · 峰值并发/上限 · 活跃账号 · 最高场景压力 · 限流失败 · P95 首 Token
├─ 主图区（同一左列，上下相邻、x 轴对齐）
│   ① Token 用量与并发趋势   左：按四维堆叠 Top4 + 并发右轴   右：模型压力排行 TOP6
│   ② 请求耗时与失败率趋势   紧接 ① 下方，沿用 ① 的卡片/画布/图例/hover 样式，
│      指标分段（三指标对照 / 单指标）+ 四维拆分下拉                 右：分时段明细 TOP8（点击定位）
├─ 压力区          左：四维可切换 × 时段 热力图                  右：并发水位与容量 + 排队指标
├─ 下钻区          调用链路(逐级联动) ｜ 交叉明细(完整路径行) ｜ 单维排行(四维 Tab) + 详情抽屉
└─ 告警区          实时告警与排查建议（只读，不修改线上配置）
```

优先级：**压力与异常 > 趋势 > 明细**。所有图表可点击联动，点击后自动滚动并过滤下钻表。

---

## 4. 指标口径

> 当前代码事实：`tokenstat` 只注册了 `total_tokens`，周期只支持 D/W/M；`usage_logs` 有 `duration_ms`、`first_token_ms` 和成功请求 Token，但没有持久化 `route_alias`；`ops_error_logs` 有错误请求耗时/TTFT 与状态分类，但没有 `route_alias`。因此“任意时间桶 + 多维拆分 + P95/失败率”不能只靠现有 tokenstat 查询实现。

| 指标 | 定义 | 数据源 | 刷新 |
|---|---|---|---|
| Token 累计（配额权威值） | `input + output + cache_creation + cache_read`，按自然周期和维度投影累计 | `tokenstat` Redis 分片计数 + `token_stat_aggregates` | 实时读 Redis；MySQL 默认约 5 分钟同步 |
| Token 历史趋势 | 按请求完成时间落入分钟/小时桶的 Token 增量 | 新观测请求事实/分钟聚合；过渡期可用 `usage_logs` 有界回退 | 趋势 30-60s；实时 KPI 5s |
| Token 速率 | 选定短窗口内的真实增量 / 秒，而不是拿 24h 总量乘固定系数 | Redis 滑动计数或分钟聚合相邻桶差 | 5s |
| 请求成功率 / 失败率 | 明确拆分 `client_failure_rate`、`business_limited_rate`、`sla_error_rate`、`upstream_error_rate`；分母统一为最终客户端请求数 | `usage_logs` 成功请求 + `ops_error_logs` 最终失败请求，按 `request_id` 去重 | 30-60s |
| 并发 / 饱和度 | 瞬时 Redis 槽位占用；饱和度 = 占用 / 有效上限；分用户、账号、路由候选 | `concurrency_service` 的 `GetAccountsLoadBatch` / `GetRouteLoadsBatch` | 5s |
| 峰值/历史并发 | 抽样窗口内的并发最大值和时间序列 | 新增周期采样；当前 Redis key 只提供瞬时值，不提供历史 | 15-30s 采样，1 分钟聚合 |
| 配额使用率 | 当前自然周期累计 / 生效限额（D/W/M） | `token_stat_quota_rules` + `token_stat_period_states` + 当前投影 | 10-30s |
| 等待队列 | 当前账号等待数、排队总深度；平均等待需新增采样 | 现有 Redis account wait counter；历史等待时长需新增采样字段 | 5s 当前值 |
| P95 请求耗时 | 选定范围内成功与失败请求的 p95；按桶从直方图合并或从原始样本精确计算 | `usage_logs.duration_ms` + `ops_error_logs.duration_ms` | 30-60s |
| P95 首 Token | 只统计 `first_token_ms/time_to_first_token_ms` 非空样本；按样本数加权 | `usage_logs.first_token_ms` + `ops_error_logs.time_to_first_token_ms` | 30-60s |
| 平均请求耗时 | 请求进入到最终响应完成；必须按样本数加权，不能对维度均值再平均 | 同上 | 30-60s |
| 平均首 Token 耗时 | 非空 TTFT 样本的加权均值；与总耗时是包含关系，不可横向相加 | 同上 | 30-60s |

口径统一原则：

- **Token 与配额的当前累计值以 tokenstat 为准**；历史趋势由请求事实聚合，并在页面显示两者对账差值和同步延迟。
- 耗时与失败率是**比值型指标**。分子分母必须使用同一窗口、同一筛选条件；跨维度汇总不能把各维度均值直接平均。
- 分位数不能对分钟级 P95 再取平均。精确查询使用原始样本；预聚合使用固定延迟/TTFT 直方图桶，再按桶合并。
- 最终客户端失败按 `request_id` 归类；上游重试中的中间失败只进入 `upstream_error_rate`，不得重复计为多个客户端失败。
- 业务主动限流与平台 SLA 错误分开展示。默认“失败率”指最终客户端失败率；若产品只关心 SLA，必须明确标注并排除业务限制类错误。
- 所有接口都返回 `as_of`、`lag_seconds`、`source`、`consistency`、`complete` 和 `warnings`，前端不得把缺数静默显示成 0。

---

## 5. 维度模型

事实路径：成功请求取 `usage_logs`，最终失败请求取 `ops_error_logs`。完整调用链：

```
用户/部门 → API Key（绑定一个分组）→ 分组场景 → 请求模型/路由别名
         → 路由候选 → 模型账号（绑定唯一实际上游模型）
```

| 维度 | 关键字段 | 说明 |
|---|---|---|
| 模型账号 | `account.id / platform / type / concurrency / schedulable` | 并发与调度载体；产品口径严格要求一个模型账号只对应一个实际上游模型 |
| 分组场景 | `group.name / scene_name / model_routing(JSON)` | `scene_name` 是展示名且允许重复，所有写操作和唯一键必须使用 `group_id`，不能用场景名代替 |
| 路由入口 | `group.model_routing` 的对象键 | 键既可以精确匹配，也可以是以 `*` 结尾的前缀模式；每条请求最终上报的是被请求模型/实际命中的 `route_alias` |
| 路由关系投影 | `group_model_route_accounts(group_id, route_alias, account_id, max_concurrency)` | 从 JSON 同步出的规范化关系，适合关系查询、并发分配和影响面计算 |
| 绑定模型 | `account.credentials.model_mapping`（单键）/ `usage_logs.upstream_model` | 服务层派生为响应字段 `account.bound_model`，使用账号对象/缓存复用解析结果；不需要新增数据库列 |
| 用户/部门 | `user.id / department`（LDAP 同步） | 部门归属按查询时的 `users.department` 关联；`token_stat_aggregates.department` 仅作历史兼容，不用于新报表 |
| API Key | `api_key.user_id / api_key.group_id / api_key_id` | 一个 Key 最多绑定一个分组；用户与分组则通过 `user_allowed_groups` 形成 N:N 授权 |

关系基数：

```
User 1 ── N API Key
API Key N ── 1 Group（group_id 可空）
User N ── N Group（通过 user_allowed_groups）
Group 1 ── N Route Pattern/Alias
Route Alias 1 ── N Candidate Account
Model Account 1 ── 1 Upstream Model
```

关键原则：

- 路由别名 **不等于** 一个模型账号；一个请求最终只命中一个账号和一个实际上游模型。
- 路由配置中的通配键不是实际别名。例如配置键 `gpt-*` 可以接管请求 `gpt-5`，上报维度应保留 `gpt-5`，关系计算再用配置匹配规则解释。
- 同一账号可被多个分组/路由别名复用，但 Token 不能在这些复用关系上重复相加；事实用量归属到 `user_id + group_id + route_alias + account_id`，实际上游模型由账号唯一确定。
- 实现必须把“一账号一上游模型”作为全平台不变量；现有仅限制非 OAuth 的校验需要补齐，避免 OAuth 或空映射绕过。模型映射、路由写入和批量导入都应拒绝多模型配置。
- 当前 `usage_logs` 与 `ops_error_logs` 未持久化 `route_alias`，实现前必须新增字段和写入链路；历史数据只能回退到 `requested_model/model`，并标记为推断值。
- 账号并发水位是账号级共享压力，别名流量展示必须带上账户，才能解释“别名健康但底层账号过载”。
- 所有交叉聚合都保留完整维度键，不允许先按别名聚合、再按账号聚合后拼表，否则会丢失“哪个用户、哪个分组、哪条别名命中了哪个模型账号”的关系。

交叉筛选示例：`用户=zhang.wei` + `分组=编码助手-公共` → `coding-fast` → `claude-pro-01`（唯一模型 `claude-sonnet-4-5`）→ 查看 95% 账号水位和排查建议。

### 5.1 看板三层表达

| 视图 | 回答的问题 | 交互 |
|---|---|---|
| 调用链路 | 当前用户通过哪个分组、哪个别名，最终命中哪个模型账号？ | 四列逐级联动；当前路径始终可见 |
| 交叉明细 | 相同账号是否被多个用户/分组/别名复用，各路径消耗多少？ | 完整路径一行一条；搜索、状态筛选、排序、详情、CSV |
| 单维排行 | 某个单维度内谁的压力最高？ | 保留用户、分组、别名、模型账号的横向排行 |

链路视图解决关系理解，交叉明细解决资源复用与归因，单维排行解决快速定位；三者共用同一时间范围与统计口径。

两张趋势图都支持四个维度（用户 / 分组场景 / 路由别名 / 模型账号）切换序列：上图按 Token 堆叠，下图按耗时/失败率多折线拆分。两者共用同一份 `selectedPaths()` 与时间桶，放在同一列上下相邻，x 轴逐点对齐；热力图同样支持四维切换，并从任意压力格子回到完整调用链。

### 5.2 主数据与事实路径分离

分组配置在系统中**只有一份**，不能按用户复制出不同的别名集合。实现依据：

| 代码位置 | 结论 |
|---|---|
| `internal/domain/model_routing.go` | `ModelRoutingConfig map[string][]ModelRouteCandidate`：键可为精确路由名或尾部 `*` 前缀模式，每个入口对应有序候选列表（**Group 1:N Route Entry**） |
| `ModelRouteCandidate.AccountIDs / Priority` | 同一入口挂多个模型账号，按优先级分层调度（**Route Entry 1:N Account**）；候选中的 `Model` 字段当前不持久化，不能把它当作可靠的关系主数据 |
| `ent/schema/token_stat_aggregate.go` | Token 聚合已内置 `user_id / api_key_id / group_id / route_alias / account_id / upstream_model / department`；其中 `upstream_model` 只用于保存账号模型快照，不作为独立看板维度 |
| `backend/internal/service/account_model_mapping_validation.go` | 当前实现只对非 OAuth 强制单模型；本项目要求扩展到所有账号类型，保证账号 1:1 模型 |
| `ent/schema/usage_log.go` | `model / requested_model / upstream_model / model_mapping_chain` 区分客户端请求的别名与账号绑定的具体模型 |
| `ent/schema/user_allowed_group.go` | 用户与分组是授权关系，不持有分组的路由配置 |

因此看板把数据拆成两层：

- **主数据**：`Group → Route Entry → Account（携带唯一 Upstream Model）`。优先读取 `group_model_route_accounts` 投影；通配路由按实际 `requested_model` 解释。
- **事实路径**：`User × API Key × Group × Route Alias × Account`，Token 按最终请求路径分别归属；并发是账号级共享状态，耗时和错误则按请求样本归属。

单维排行的四个维度成员、趋势堆叠序列、热力图行全部由这份主数据派生，避免出现「某个别名在排行里存在、在链路里却找不到」或「同一分组在不同用户下别名数量不同」。

### 5.3 全链路多维切换

| 区域 | 可切换维度 | 联动行为 |
|---|---|---|
| Token 用量与并发趋势 | 用户 / 分组场景 / 路由别名 / 模型账号 | 取该维度 Token Top4 堆叠，切换时重置图例；hover 明细随维度变化 |
| 请求耗时与失败率趋势 | 同上（另有「全部对象」汇总档） | 指标分段选「全部指标」时画三条对照线；选单项指标时按维度 Top4 + 「其他 N 个」画多序列，附一条流量加权汇总虚线；图例可隐藏单个成员，明细表增加「最差成员」列 |
| 热力图 | 同上 | 行标签来自主数据；点击格子按维度键反查完整路径，自动定位到用户、分组、别名与模型账号 |
| 单维排行 | 同上 | 归属列直接写明上一层与下一层基数（如「3 个路由别名 · 5 个候选模型账号」） |
| 对象详情 | 四个维度 | 打开模型账号详情时同时显示其唯一上游模型；看板不提供线上配置修改 |

### 5.4 自定义观测对象（全局筛选上下文）

用户诉求是「指定某个分组 / 别名 / 模型账号，看它的使用情况、并发与 Token 流量」。**切换聚合维度 ≠ 筛选对象**：前者改变 GROUP BY 的列，后者改变 WHERE 的范围。因此新增一层独立的全局筛选上下文，作用于所有视图。

```
SCOPE_DIMS[user | group | alias | account]  (多选、搜索、全选、清空、chips)
        ↓ state.sel + state.onlyPressure
selectedPaths()  →  完整调用路径子集（唯一事实来源）
        ↓
  ┌──────────────┬───────────────┬──────────────┬─────────────┐
  KPI / 趋势     热力图 / 排行     链路 / 明细     告警 / CSV
  Token 求和     行成员取子集      路径行过滤      影响面按子集展开
  并发去重求和   格子值取账号饱和度 节点只显示范围内 导出仅范围内行
```

| 口径 | 聚合规则 | 说明 |
|---|---|---|
| Token 用量 | `Σ path.tokens` | 按完整路径归属，不重复计 |
| 当前并发 / 上限 | 按 `account` 去重后 `Σ conc` / `Σ cap` | 同一账号被多个用户复用时只算一次，避免并发虚高 |
| Token 流量速率 | `Σ 当前短窗口真实 Token 增量 / 窗口秒数` | 不允许用长窗口总量乘固定系数伪造实时速率 |
| Token 趋势 | 各时间桶的直接聚合值；TopN 外成员由后端合并为“其他” | 过滤条件相同则各维度汇总必须等于总曲线 |
| 热力图 | 行成员 = 范围内对象；格子值优先显示真实采样并发饱和度，缺采样时明确显示“无历史采样” | 不能用“饱和度 × 时段系数”生成伪历史值 |
| 配额 / P95 / 错误率 | 配额取生效规则；P95/错误率按原始样本或直方图合并 | 不能把各维度峰值或均值简单拼接 |

**联动范围**：KPI、趋势与图例、压力排行、并发水位、热力图、调用链路节点、交叉明细、单维排行、详情抽屉关联维度、告警列表和 CSV 导出。筛选变化时自动重置图例隐藏状态，并把链路选中项落到范围内第一条路径。

**空组合与降级**：维度间取交集可能为空（例如选了分组 A + 只属于分组 B 的别名）。此时链路图显示空态、表格显示「没有符合条件的数据」、趋势给出「无匹配对象」零值序列，所有除法以 `|| 1` 兜底，不产生 NaN / Infinity。成员列表标注「无交集」提示，帮助用户发现冲突组合。

**真实落地**：`state.sel` 对应接口参数 `user_ids / group_ids / route_aliases / account_ids`。Token 累计从 tokenstat 投影读取；请求数、耗时和失败率从观测事实表读取；并发当前值从 `concurrency_service` 批量读取，历史值从并发采样表读取。所有筛选必须下推到 SQL/Redis，前端不做全量取回后再过滤。实际上游模型通过账号属性返回，不参与筛选。

---

### 5.5 分时耗时与失败率视图

「Token 用量与并发趋势」回答的是**量**，「请求耗时与失败率趋势」回答的是**质**。后者**紧跟前者正下方**，沿用相同的卡片头（`select` + `seg`）、SVG 画布尺寸（780×252 / L48 R48 T14 B28）、图例开关与 hover tooltip；两者共用时间桶与 `selectedPaths()` 筛选上下文，x 轴逐点对齐。窄屏（<1280px）时两张趋势图仍作为同一列整体排在前，不会被右侧卡片插开。

**两个控件、两种查看模式**：

| 控件 | 选项 | 行为 |
|---|---|---|
| 指标分段 | 全部指标 / 请求耗时 / 首 Token / 失败率 | 默认「全部指标」＝三指标对照（双轴 + 两条 SLO 线）；选中单项指标后启用维度拆分 |
| 维度下拉 | 全部对象 / 按模型账号 / 按分组场景 / 按路由别名 / 按用户 | 单指标模式下按维度成员画多序列；「全部指标」模式下禁用（该模式本就是汇总对照）。首次切到单项指标自动落到「按模型账号」 |

| 项 | 规则 |
|---|---|
| 全部指标模式 | 左轴平均请求耗时、平均首 Token 耗时（s）两条独立折线；右轴失败请求率（%）红色虚线；首 Token 已包含在请求耗时内，均值不可堆叠相加 |
| 单指标拆分模式 | 左轴为所选指标单位（s 或 %），不再使用右轴；每个维度成员一条彩色实线 + 一条 `var(--text2)` 流量加权汇总虚线；成员超 SLO 的桶标点 |
| 序列数 | 按 Token 权重取 Top4，其余成员合并为「其他 N 个」，**不丢弃任何路径**；图例可逐个隐藏成员 |
| 时间桶 | 1h/5m、6h/15m、24h/1h、7d/6h，与 Token 趋势的 `ranges` 完全相同，x 轴逐点对齐 |
| 日高峰 | 只展示真实聚合结果与历史基线比较；不得用固定 15:00 系数或人为劣化系数生成生产数据。Demo 中的曲线仅作视觉占位 |
| SLO | 首 Token ≤ 2.0s（amber 虚线）、失败率 ≤ 2%（red 虚线）；逐桶判定达标 / 接近 / 超 SLO |
| 筛选联动 | 全局筛选变化时所有维度序列、汇总线、明细表同步重算；无匹配对象时给空态、不产生 NaN |
| 交互 | hover tooltip 在拆分模式按当前指标降序列出各成员与汇总值；右侧明细表按当前指标降序 TOP8，拆分时增加「最差成员」列；点击行在图上定位并常驻 tooltip |

**唯一事实来源与一致性**：生产实现不得按路径和“确定性种子”生成耗时/失败序列。成功请求来自 `usage_logs`，最终失败请求来自 `ops_error_logs`，两者按相同桶边界与筛选条件聚合。平均值按请求数加权；分位数从原始样本或固定直方图桶计算。Demo 只保留确定性模拟以表达交互。

**后端口径**：`GET /latency-timeseries` 返回按桶聚合的 `avg_duration_ms / avg_first_token_ms / failure_rate`。成功样本取自 `usage_logs`，失败样本取自 `ops_error_logs`；`duration_ms` 已存在，错误 TTFT 字段名为 `time_to_first_token_ms`。最终失败以客户端最终状态为准，按 `request_id` 去重，避免把上游重试中间失败重复计入。`route_alias` 必须先持久化到两张事实表，否则只能按 `requested_model/model` 推断并返回降级警告。

---

## 6. 接口契约（建议新增，只读）

所有接口挂在 `/api/v1/admin/observability` 且全部为 GET。建议新增唯一的 `observability.read` 权限；不能使用不存在的 `token_usage.view`，也不需要 `observability.manage`，因为看板不修改线上配置。Token/配额面板可额外要求 `token_usage.read`。

### 6.0 公共查询与响应契约

- 时间参数统一为 `window=1h|6h|24h|7d&bucket=5m|15m|1h|6h`，服务端校验窗口与桶的组合；禁止使用含义模糊的 `range`。
- 维度编码统一使用 tokenstat 的 canonical code：`user_id | group_id | route_alias | account_id`。`user`、`account`、`group` 只可作为前端展示别名，不进入接口。账号的唯一上游模型作为 `account.bound_model` 属性返回。
- 多选筛选使用重复查询参数或 JSON body，例如 `user_id=1&user_id=2&group_id=8`；显示名永远不能作为唯一键。
- 比率统一返回 `0..1` 小数，前端负责格式化为百分比；`failure_rate=0.0082` 表示 0.82%。
- 公共响应必须包含：

```json
{
  "code": 0,
  "data": {
    "meta": {
      "as_of": "2026-11-10T15:00:00+08:00",
      "window": "24h",
      "bucket": "1h",
      "timezone": "Asia/Shanghai",
      "sources": ["usage_logs", "ops_error_logs", "concurrency"],
      "consistency": "hybrid_eventual",
      "lag_seconds": 34,
      "complete": true,
      "warnings": []
    }
  }
}
```

- 参数上限：数组筛选每类最多 50 个 ID；分页 `page_size<=100`；原始样本查询最多 7 天；超限返回 422 而不是降级成全表扫描。
- GET 响应支持 `ETag`；前端携带 `If-None-Match`，无变化返回 304，避免 5 秒轮询反复重绘。

### 6.1 时序

```http
GET /timeseries?window=24h&bucket=1h&metric_code=total_tokens&group_by=route_alias&group_id=8
```

```json
{
  "code": 0,
  "data": {
    "bucket": "1h", "timezone": "Asia/Shanghai",
    "labels": ["00:00", "01:00"],
    "series": [{ "name": "claude-pro-01", "points": [421300, 388900] }],
    "concurrency": [86, 79], "concurrency_limit": 200
  }
}
```

耗时时序与上面共用 `window` / `bucket` / `labels`，前端可直接对齐 x 轴；`metric_code=all` 返回三条汇总线，单项时按 `group_by` 返回各维度成员序列：

```http
GET /latency-timeseries?window=24h&bucket=1h&metric_code=avg_duration_ms&group_by=account_id&group_id=8
```

```json
{
  "code": 0,
  "data": {
    "labels": ["00:00", "01:00"],
    "summary": {
      "avg_duration_ms": [2140, 1980],
      "avg_first_token_ms": [760, 710],
      "failure_rate": [0.0082, 0.0061]
    },
    "series": [
      { "name": "claude-pro-01", "avg_duration_ms": [2310, 2140] },
      { "name": "claude-pro-02", "avg_duration_ms": [1980, 1830] }
    ],
    "slo": { "first_token_ms": 2000, "failure_rate_pct": 2.0 }
  }
}
```

`metric_code` 枚举 `all | avg_duration_ms | avg_first_token_ms | failure_rate`；`group_by` 枚举 `user_id | group_id | route_alias | account_id`。模型账号严格 1:1 上游模型，因此不提供独立模型维度。`metric_code=all` 时忽略 `group_by`。`summary` 是筛选范围内全量请求的一次聚合，不受 `group_by` 影响；`series` 为各维度成员的按桶序列，TopN 外由后端合并为「其他 N 个」。

筛选参数与 6.0 一致，必须下推到 WHERE。三项均值与失败率均按桶内样本数计算；分位数使用原始样本或直方图合并。成功/失败数据来自两套事实表时，必须返回 `sources`、`lag_seconds` 和 `complete`。

### 6.2 维度聚合表

```http
GET /paths?window=24h&user_id=1024&group_id=8&route_alias=coding-fast&status=warn,bad&sort=tokens&order=desc&page_size=50
```

```json
{
  "code": 0,
  "data": {
    "window": "24h", "total": 25,
    "rows": [{
      "user": {"id": 1024, "name": "zhang.wei", "department": "研发一部"},
      "group": {"id": 8, "name": "coding-public", "scene_name": "编码助手-公共"},
      "route_alias": "coding-fast",
      "account": {"id": 31, "name": "claude-pro-01", "platform": "anthropic", "bound_model": "claude-sonnet-4-5"},
      "tokens": 8420000, "share": 0.324, "delta": 0.124,
      "request_count": 18240,
      "conc_current": 38, "conc_peak": 38, "conc_limit": 40, "quota_used": 91,
      "p95_first_token_ms": 1420, "error_rate": 0.0082,
      "spark": [12, 15, 14, 22], "status": "bad"
    }]
  }
}
```

`/dimensions` 继续用于单维排行；`/paths` 专门返回完整维度键，不能把多行路径先压成无关联的独立数组后由前端拼接。账号并发是账号级共享状态，多个路径可展示同一个值，但汇总时按 `account_id` 去重；Token/请求数按最终事实路径归属。`account.bound_model` 是账号唯一模型属性，不参与 GROUP BY。

`window=7d` 的用户明细只返回按当前主指标排序的 TopN（默认 50，最大 100），禁止深分页扫描完整高基数路径；响应用 `coverage=top_n` 和 `total_estimated` 标明覆盖范围。更长时间范围只允许查看聚合趋势或分组/账号维度明细。

### 6.3 压力与热力图

```http
GET /pressure?window=24h               → 当前账号/候选水位、全局并发槽、排队数
GET /heatmap?group_by=route_alias&window=24h&slots=12  → rows[{key,name,value_kind,cells}]
```

`group_by` 枚举 `user_id | group_id | route_alias | account_id`。`value_kind` 必须是 `sampled_concurrency | token_rate | error_rate` 等明确定义；如果所选窗口没有并发历史采样，返回 `value_kind=unavailable` 和警告，不使用合成时段系数。

### 6.4 关系图

```http
GET /relations?window=24h&user_id=1024&group_id=8
→ nodes: user / api_key / group / route_alias / account（携带唯一 bound_model）
→ edges: user→api_key、api_key→group、group→route_alias、route_alias→account
```

关系图接口返回的是已命中的关系边及聚合指标；配置中存在但窗口内没有流量的关系可以通过 `include_idle=true` 单独返回，不能混入用量合计。

---

## 7. 数据链路与性能约束

```text
请求完成
├─ usage_logs（成功：Token、耗时、TTFT、账号唯一模型快照、新增 route_alias）
├─ ops_error_logs（最终失败：状态、阶段、耗时、TTFT、账号唯一模型快照、新增 route_alias）
├─ tokenstat Redis（当前累计与限额权威值；MySQL 约 5 分钟同步）
└─ 并发采样器（账号/候选级周期快照）
          ↓
观测分钟聚合（新增：additive counters + 延迟/TTFT 直方图）
          ↓
快照/时序/路径/关系/压力 API → 前端页面
```

- Token 趋势可以在首个版本从 `usage_logs` 有界聚合，但 7 天查询必须走分钟/小时聚合；禁止每次 5 秒轮询直接扫描原始表全窗口。
- 新增 `observability_request_minute`，至少包含 `bucket_start,user_id,api_key_id,group_id,route_alias,account_id,request_count,success_count,error_count,sum_tokens,sum_duration_ms,ttft_sample_count,sum_ttft_ms,duration_hist,ttft_hist`；上游模型由 account 唯一推导，不形成额外维度；时间桶使用 UTC 存储、查询时转换展示时区。
- 新增 `observability_concurrency_minute`：采样器可以每 15-30 秒从 Redis 读取，但**只在内存聚合并每分钟为每个对象写一行** `current_peak/current_avg/limit/waiting_peak`。禁止把每次采样都持久化并保留 7 天，否则并发采样本身会成为主要写压力。
- 聚合和查询使用联合索引：`(bucket_start,group_id,route_alias)`、`(bucket_start,user_id)`、`(bucket_start,account_id)`；高基数结果服务端 TopN。
- 分钟表只写当前分钟实际有请求的完整路径，不写零流量笛卡尔组合；对单分钟活跃路径设置容量告警和熔断阈值，避免高基数用户路径把聚合表撑爆。
- 短缓存键为 `window+bucket+group_by+canonical_filters+schema_version`，实时 KPI 5 秒短缓存，趋势 30 秒，配额面板 10 秒。缓存内容必须包含 `meta`。
- 刷新分层：KPI/并发 5 秒；趋势 30-60 秒；关系/明细仅在窗口或筛选变化时查询；页面隐藏时暂停，失败按 5→15→60 秒指数退避。
- 性能目标区分场景：缓存命中 API P95 < 200ms；常规聚合 P95 < 500ms；大窗口冷查询 P95 < 1.5s。禁止给所有接口承诺 300ms。
- 历史保留建议：1 分钟原始聚合 24-48 小时，5 分钟聚合 7 天，小时聚合 90 天，日聚合 1 年；原始事实按现有数据保留策略，不承诺独立 1 年。
- CSV 导出仅返回当前筛选范围，服务端限制 10000 行；对以 `= + - @` 开头的单元格做公式注入转义。

---

## 8. 压力分级与告警规则

| 等级 | 饱和度 | 颜色 | 自动建议 |
|---|---|---|---|
| 健康 | < 60% | primary teal | 保持默认调度 |
| 关注 | 60–80% | blue | 日高峰前预留 20% 缓冲 |
| 预警 | 80–95% | amber | 排查高峰流量和上游耗时，提前规划扩容 |
| 过载 | ≥ 95% | red | 触发告警；到账号或分组管理页人工处理 |

附加规则：配额使用率 ≥ 90% 或 SLA 错误率 ≥ 2% 时等级上调一级；**连续 3 个 5s 窗口越限**才产生告警，避免抖动误报；同对象告警合并、抑制重复通知。业务主动限流默认不升级为平台故障等级，必须单独展示。

参考告警表达式（语义等价即可，不依赖 Prometheus）：

```
max by (account_id) (slot_used / effective_limit) > 0.95   for 15s       → 严重
sum by (group_id, route_alias) (rate(total_tokens[5m])) > 1.5 * baseline_7d → 预警
p95(first_token_ms) > 2000   for 5m                                    → 提示
```

告警应复用现有 `ops_alert_rules` / `ops_alert_events` 与通知配置，不另建告警引擎；新增指标只扩展规则输入。

---

## 9. 只读诊断 SOP

1. **告警触发** → 顶部状态条与告警列表高亮，展示数据新鲜度与是否为部分数据。
2. **定位** → 在热力图确认时段和对象，在趋势图确认是持续越限还是单点毛刺。
3. **下钻** → 打开完整路径，分别查看账号共享并发、路由候选、唯一绑定模型和错误阶段。
4. **形成结论** → 记录受影响用户、分组、别名、账号、当前水位和证据时间，给出人工处理建议。
5. **人工处理** → 如需调整账号状态、分组或并发配置，由管理员到现有对应管理页面操作；本观测台不发起写请求。
6. **复核** → 返回观测台观察至少两个采样周期，确认指标变化与数据延迟。

本页面不提供应急模式、切换比例、阈值应用、审计写入或回滚操作，避免观测功能和线上配置变更职责混杂。

---

## 10. 前端实现建议

对齐现有工程结构，落地时建议新增：

```
frontend/src/views/admin/ModelObservabilityView.vue        # 页面容器（AppLayout）
frontend/src/components/observability/
  ├─ KpiCard.vue                 # 复用 DashboardView 指标卡结构 + sparkline
  ├─ TokenConcurrencyTrend.vue   # Chart.js 堆叠面积 + 右轴并发（vue-chartjs）
  ├─ PressureRanking.vue         # 横向条 TOP N，点击联动
  ├─ PressureHeatmap.vue         # 明确展示采样并发或 Token 压力语义
  ├─ DimensionTable.vue          # 四维 Tab + 排序 + 服务端分页 + 抽屉
  ├─ AlertPanel.vue              # 只读告警与排查建议
  ├─ DataQualityBanner.vue       # 延迟、缺采样、部分失败、对账差异
  └─ useObservability.ts         # 分层轮询、筛选状态、维度联动（composable）
frontend/src/api/admin/observability.ts                    # 6.x 接口封装
```

- 图表沿用项目已有的 `chart.js` + `vue-chartjs` 与 `components/charts` 的 10 色板，不引入新图表库。
- 轮询用 `@vueuse/core` 的 `useIntervalFn`，页面不可见时暂停；卸载时清理定时器。
- 所有数值展示走统一 `formatNumber`，时间统一 `Asia/Shanghai`（后端配置 `timezone`）。
- 表格用虚拟滚动或分页（@tanstack/vue-virtual 已在依赖中）应对用户维度高基数。
- 深色模式由根 `dark` class 驱动，色值全部走 CSS 变量 / Tailwind 语义类。
- 图表必须有文字图例、键盘可达的筛选控件和非颜色状态标识；热力图不能只靠颜色表达严重度。
- 每个卡片独立显示加载、空、部分、过期和错误状态；一个接口失败不能把全页 KPI 清零。

**视觉规范**：主色 `primary.500 #14b8a6` / `primary.600 #0d9488`；背景 `#f1f5f9`；卡片白底 + `1px #e2e8f0` + `14px` 圆角 + `card` 阴影；标题区渐变 `primary-50 → white → indigo-50`；状态色 teal / blue / amber / red。

---

## 11. Demo 边界与验收

**Demo 已实现**：四维自定义观测对象多选（搜索 / 全选 / 清空 / chips / 仅高压）与全局筛选联动、调用链路逐级下钻、完整路径交叉明细、单维排行与排序筛选、四类维度的趋势堆叠切换、四类维度的热力图切换、四种时间范围、指标切换与图例开关、趋势 hover tooltip、请求耗时 / 首 Token / 失败率分时趋势、热力图点击联动、压力排行联动、详情抽屉、只读告警列表、自动刷新、深浅色切换、CSV 导出和设计说明页。全部为确定性模拟数据，零外部依赖。

其中多维区提供三种视图：`调用链路`按用户 → 分组 → 别名 → 模型账号逐级联动；`交叉明细`逐行保留完整关系键；`单维排行`用于某个维度的横向比较。分组、别名、候选模型账号由**同一份主数据**派生：同一分组对所有用户展示一致的别名集合，仅 Token 用量按用户 × 路径分别归属；模型账号同时展示其唯一上游模型。

**Demo 未实现且不得视为已确定能力**：真实后端调用、告警通知、权限校验、并发历史采样、路由别名持久化。Demo 明确不包含流量切换、候选容量写入、回滚、阈值应用或操作审计。

**验收标准（真实落地时）**：
- 实时 KPI 与并发可在 5s 级更新；趋势按 30-60s 更新；缺失的数据源显式标注，不伪装成 0。
- 缓存命中 API P95 < 200ms，常规聚合 P95 < 500ms，大窗口冷查询 P95 < 1.5s；不是所有接口统一 300ms。
- Token 当前累计与 tokenstat 限额判定误差为 0；趋势对账差异有阈值、可解释、可追踪。
- 成功率分母同时包含成功和最终失败请求，按 `request_id` 去重；业务限制与 SLA 错误分开。
- 任一压力对象可在 3 次点击内定位到完整维度行，并能看到候选账号和实际上游模型。
- 所有请求均为 GET；页面不存在修改账号、分组、路由或并发配置的入口。
- 深色/浅色模式文字对比度达标，1280px 以下布局不溢出，图表和筛选可键盘操作。

---

## 12. 分期路线

| 阶段 | 内容 | 产出 |
|---|---|---|
| 一期 · 事实基础 | 强制账号 1:1 模型，持久化 `route_alias`，统一成功/失败请求口径，增加有界原始查询与分钟聚合，补并发采样 | 数据迁移 + 只读 API + 压测基线 |
| 二期 · 看板 | 快照、趋势、压力、关系、路径明细、数据质量状态；接入现有告警展示 | Vue 页面 + 组件 + E2E |
| 三期 · 验证发布 | 只读权限、性能、可访问性、灰度与发布回退验证 | 发布报告 + 运维只读 SOP |

---

## 13. 设计修订基线（v1.8）

本节是实现时必须遵守的修订结论；若它与前面早期描述或 Demo 行为冲突，以本节为准。

### 13.1 已确认的问题

| 级别 | 原设计问题 | 当前事实 | 修订 |
|---|---|---|---|
| P0 | 使用 `token_usage.view` 权限 | RBAC 目录只有 `token_usage.read/manage` | 新增只读 `observability.read`；看板无写权限 |
| P0 | 声称 tokenstat 支持任意 1h/6h/24h/7d 时间桶 | tokenstat 只有 `total_tokens` 指标和 D/W/M 周期，MySQL 默认约 5 分钟同步 | Token 当前累计用 tokenstat；历史趋势改走请求事实和新增分钟聚合 |
| P0 | 用 `usage_logs.route_alias` 做耗时/失败率拆分 | `UsageLog.RouteAlias` 明确注释为不持久化 | 迁移两张事实表并补齐 gateway/error 写入；旧数据标记为推断 |
| P0 | 原实现没有全局强制账号 1:1 模型 | 产品已明确采用“一账号一上游模型”口径，但当前代码仅校验非 OAuth | 全账号类型强制单模型；模型名作为账号属性，不新增独立维度 |
| P0 | Demo 包含“源分组 → 目标分组”切换与回滚 | 本次明确不需要切流动作 | 删除切换控制台、二次确认、执行、阈值应用、审计日志和回滚 |
| P0 | 热力图/趋势可用时段系数生成历史 | Redis 只保存当前槽位计数，没有并发历史 | 新增采样器；无样本时显示不可用，不合成数据 |
| P1 | 失败率定义同时包含 429、又单列错误率 | 现有 Ops 已区分业务限制、SLA 错误和上游错误 | 拆分四个比率，所有分母一致，按最终 `request_id` 去重 |
| P1 | 场景名可作唯一键 | `scene_name` 允许空且允许重复 | 所有接口改用 `group_id`，场景名只展示 |
| P1 | 账号模型关系可只靠候选 `Model` 字段 | `ModelRouteCandidate.Model` 不参与 JSON 序列化，路由键还可能是通配模式 | 关系以 `group_model_route_accounts` + 账号唯一模型属性为准 |
| P1 | 5 秒刷新全部视图、所有接口 P95 < 300ms | 全量趋势、明细和关系查询成本不同 | 实时 5s、趋势 30-60s、明细按需；目标按缓存/聚合/冷查询分层 |
| P1 | 对分钟 P95 再求平均或加权平均 | 分位数不可由均值精确重建 | 原始样本或固定直方图桶合并 |
| P1 | 没有数据质量状态 | tokenstat 投影可能未配置、MySQL 可能滞后、部分查询可能超时 | 每个响应返回新鲜度、来源、完整性、警告和对账差值 |
| P1 | CSV 仅做 RFC 引号转义 | 用户名等外部文本存在公式注入风险 | 对 `= + - @` 开头单元格加前缀转义 |

### 13.2 统一数据源决策

| 视图 | 权威数据源 | 延迟/一致性 | 禁止事项 |
|---|---|---|---|
| 当前 Token 累计、配额 | tokenstat Redis + MySQL 封账 | 实时累计；历史最终一致 | 不从 `usage_logs` 重新定义限额口径 |
| Token 历史趋势 | `observability_request_minute`，首个版本允许 `usage_logs` 有界回退 | 分钟级最终一致 | 不扫描原始表 7 天全量；不乘固定系数 |
| 请求数/成功率/失败率 | `usage_logs` 成功 + `ops_error_logs` 最终失败 | 请求结束后异步可查 | 不把上游中间重试算成多个客户端失败 |
| 耗时/TTFT/P95 | 原始事实或观测直方图 | 原始查询 7 天内；聚合更长 | 不对维度均值做平均 |
| 当前并发/等待 | Redis slot / wait counter | 5 秒级 | 不把账号并发重复加到每条路径 |
| 历史并发 | `observability_concurrency_minute` | Redis 高频读取，分钟落库 | 不逐采样落库；不用饱和度乘时段系数伪造 |
| 分组/别名/账号关系 | `group_model_route_accounts` + 账号唯一模型属性 | 配置更新后最终一致 | 不从用户维度复制分组配置 |
| 告警 | 现有 `ops_alert_rules` / `ops_alert_events` | 与现有 Ops 一致 | 不另起一套通知引擎 |

**对账规则**：页面显示 Token 当前累计时，以 tokenstat 为主；若同一窗口的历史聚合与 tokenstat 差值超过 `max(1%, 1000 tokens)`，返回 `reconciliation_warning`，并展示最晚同步时间。不能因此把其中一个值静默覆盖。

### 13.3 交互与视觉优化

- 顶部增加数据健康条，至少显示 `实时 / 延迟 30s / 部分数据 / 投影未启用 / 查询失败`；每个卡片也能独立进入空态或错误态。
- KPI 卡显示分母和数据来源提示。例如失败率必须明确是最终客户端失败率，TTFT 显示样本数。
- 首屏减少同时请求数，先加载 `snapshot` 包含 KPI、TopN、压力摘要和数据质量；趋势与明细进入视口后再加载。
- 图表统一使用服务端返回的 `labels/bucket_starts`，不在前端自行补桶或平均；无数据桶显示断点或明确 0，不能混淆。
- 热力图增加指标切换，但选项只有后端真正提供的 `sampled_concurrency/token_rate/error_rate`；每个值有 tooltip 说明口径。
- 过滤 chips 使用受控选项和 ID；显示名重复或改名时不破坏筛选。筛选变化后取消旧请求，防止慢响应覆盖新结果。
- 详情抽屉固定展示“当前值、窗口值、与上一窗口差值、样本数、数据时间”；避免只有颜色和百分比。
- Demo 顶部“生产环境”选择器没有后端契约，真实页面删除它，或改为后端支持的 `platform` 过滤/实例标识。
- 图表和热力格提供键盘焦点、ARIA 标签和文字状态；移动端将宽表转为分页/卡片，不挤压关键信息。

### 13.4 只读与安全约束

- 观测 API 只注册 GET 路由；前端不提供提交、确认、回滚或阈值修改控件。
- 只读权限不能绕过 RBAC scope；无 `observability.read` 的用户隐藏导航并得到 403。
- 用户 ID、邮箱和部门按已确认决策可进入页面和 CSV；API Key secret、请求正文、上游响应头和凭据不得写入观测 API 响应或日志。
- 页面上的建议只描述问题和可能的人工处理方向，不承诺执行，也不直接跳到会触发写操作的动作。

### 13.5 已确认的产品决策

1. **失败率**：主 KPI 使用最终客户端失败率，包含 429、5xx 和超时；业务限制率、SLA 错误率和上游错误率单独展示。
2. **用户数据**：拥有 `observability.read` 的管理员/运维角色可以查看用户 ID、邮箱和部门，不做邮箱脱敏；API Key secret、请求正文、凭据和上游响应头仍禁止返回。
3. **7 天明细**：分钟数据保留 24-48 小时并滚动到 5 分钟聚合；5 分钟聚合保留 7 天；7 天用户明细只提供 TopN，不承诺全量用户下钻。
4. **历史多模型账号**：新增和修改时强制一账号一模型；历史异常只生成清单供人工修复，不自动删改，也不做全表 JSON 回填。
5. **告警**：复用现有 `ops_alert_rules` / `ops_alert_events` 和通知渠道，在现有规则引擎上增加账号水位、TTFT、失败率指标，不新建告警系统。
6. **授权**：首期仅超级管理员开放；验证一天后通过 `observability.read` 授予运维只读角色。Token/配额面板额外要求 `token_usage.read`。
7. **页面关系**：新增独立“模型使用观测台”页面，与现有 Dashboard/Ops 页面并存；稳定后再评估是否合并入口。

### 13.6 数据库改动与性能暴露

| 表 | 改动类型 | 具体变更 | 性能风险/控制 |
|---|---|---|---|
| `usage_logs` | 现有表加字段、加索引 | 新增可空 `route_alias VARCHAR(255)`；评估 `(group_id, route_alias, created_at)` 与 `(route_alias, created_at)` | 高写入追加表。先加可空字段、后在线建索引；不要同时建多个重复索引，避免每次请求落库写放大 |
| `ops_error_logs` | 现有表加字段、加索引 | 新增可空 `route_alias VARCHAR(255)`；按错误分布选择时间 + 分组/别名的复合索引 | 错误高峰时写入放大；索引必须用小范围真实数据验证命中率 |
| `observability_request_minute` | 新表 | 分钟桶、账号路径维度、请求/成功/失败、Token 求和、duration/TTFT 固定直方图、水位字段 | 核心风险是高基数 `dimension_hash` 行数；分钟数据保留 24-48 小时并滚动为 5 分钟/小时数据，使用 UTC 桶、幂等 upsert 和分批聚合 |
| `observability_concurrency_minute` | 新表 | 每分钟每个活跃账号/候选一行：并发峰值/均值、有效上限、等待峰值 | 不逐 15-30 秒采样落库；1000 账号 + 1000 候选按 15 秒直写 7 天约 8000 万行，分钟聚合是硬约束 |
| `accounts` | 不改表结构；仅做质量审计 | 从现有 `credentials.model_mapping` 在服务层派生账号模型；只扫描并报告多模型历史异常，不做全表回填 | 复用现有 Account 领域对象和短缓存；若后续压测证明 JSON 解析成为瓶颈，再单独评审 `bound_model` 规范化列 |
| `group_model_route_accounts` | 不改表结构，重建投影数据 | 从 `groups.model_routing` 重建 group/alias/account 关系并校验账号单模型 | 关系查询必须读投影表，禁止每次解析 JSON；重建按分组批处理并记录失败 |
| `rbac_permissions` | 写入种子/兼容数据 | 新增 `observability.read` 权限 | 小表低风险；必须同步版本并失效 RBAC 缓存 |
| `rbac_role_permissions` | 写入授权数据 | 授予运维只读角色，admin 继续使用通配权限 | 避免给普通用户批量授权；授权后 bump 用户/策略版本 |
| `rbac_user_versions`、`rbac_policy_state` | 更新版本号 | 权限目录或角色授权变更后推进版本 | 不 bump 会导致 Redis 权限缓存继续使用旧结果 |
| `token_stat_aggregates` | 不改表结构 | 仅读取当前 Token/配额投影 | 禁止为看板逐请求扫描；沿用 tokenstat 投影和 Redis 当前计数 |

**关键数据链路与性能预算**

1. Gateway 在选择账号时已有 `selection.RouteAlias`，新增写入应复用内存值，**禁止在请求热路径再查一次路由配置**。
2. `usage_logs` / `ops_error_logs` 仍是生产事实表，观测追加字段必须保持批量写入；如果每请求额外同步写观测表，会直接抬高模型请求延迟，因此分钟聚合采用异步批处理。
3. 分钟聚合以 `watermark` 读取已完成事实，30-60 秒运行，按 `(bucket_start, dimension_hash)` 幂等 upsert；单批有行数和执行时间上限，失败只滞后不阻塞业务。
4. 并发采样高频读 Redis、低频写 MySQL：内存累计一分钟，仅写峰值/均值；分钟数据 24 小时，之后滚到 5 分钟/小时表。
5. 所有趋势接口只读分钟聚合；短窗口允许有界查询原始事实；7 天查询禁止扫描 `usage_logs` / `ops_error_logs`。
6. 索引目标是覆盖“时间 + 主维度”，不是给每个维度都加单列索引。新增索引必须用生产规模副本验证插入 QPS、索引大小和查询命中率。
7. 前端分层刷新：KPI/并发 5 秒，趋势 30-60 秒，明细按需；使用 ETag、取消旧请求和页面隐藏暂停，避免 5 秒刷新造成查询风暴。
8. 性能闸门：缓存命中 P95 < 200ms；常规聚合 P95 < 500ms；7 天冷查询 P95 < 1.5s；任一闸门未通过就关闭页面入口，不用 UI 轮询拖垮数据库。

---

## 14. 本次追加需求

以下三项与观测看板主链路独立，均纳入本次实现和验收。

### 14.1 非 429 失败请求报文日志

- 触发条件：网关请求最终返回 `status >= 400` 且 `status != 429`。
- 记录位置：**仅写入应用日志文件**，必须使用独立的 file-only logger/writer；不得写 `ops_error_logs.request_body`，也不得通过 `logger.WriteSinkEvent` 进入 `ops_system_logs`。
- 请求体只捕获 UTF-8 前缀，上限 64 KiB；使用受限 tee 读取，禁止为日志重新读取最大 256 MiB 的完整 body。
- 递归脱敏鉴权、API Key、token、password、secret 等字段；不记录完整请求头。
- 429、成功请求、静态页面和管理端普通请求不记录请求体。
- 捕获或日志写入失败必须 fail-open，不能改变客户端响应和上游重试。
- 回归测试必须同时断言数据库 `ops_error_logs` 和 `ops_system_logs` 均没有新增请求体内容。
- 风险暴露：日志会包含用户提示词等业务内容；仅授权管理员可访问日志，并按现有日志保留策略管理。

### 14.2 超级管理员豁免用户限额

- “超级管理员”按系统 admin 角色或 RBAC `permissions=["*"]` 判定，不能硬编码 `admin@example.com`；该邮箱只作为验收样本。
- 豁免范围：用户级并发、用户 RPM、用户 × 平台配额、含该 `user_id` 的 Token 限额、用户余额/订阅类用户资格限制。
- 不豁免：账号并发、路由候选并发、分组共享容量、上游供应商配额；这些属于平台资源保护。
- 用户限额列表、选择和统计页面排除超级管理员，历史用量仍正常记录。
- 认证快照增加 `IsSuperAdmin` 布尔值，避免模型请求热路径按请求查询 RBAC 或执行角色 SQL。
- 角色/权限变化必须触发认证缓存失效，使豁免状态及时生效。

### 14.3 使用记录页场景用量降级

- `frontend/src/views/admin/UsageView.vue` 的 `场景 Token 用量` 区块移动到页面最低端。
- 首页进入和普通筛选刷新不再自动请求场景用量；展开或显式点击查询时才加载。
- 使用记录表、错误记录和主要筛选保持优先展示。
- 场景用量保留原有查询、日期/分组筛选、空态、错误态和同步提示。
