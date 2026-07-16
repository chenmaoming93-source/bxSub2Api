# 集成 API Key 指定分组与授权校验实施 Plan

状态：Final — 用户已批准  
版本：v1.0  
日期：2026-07-16  
变更摘要：为 `POST /api/v1/integrations/api-keys/getOrCreate` 增加必填分组名，按分组类型和用户授权校验创建资格，并将平台 Key 的幂等范围调整为 `(user_id, platform, group_id)`。

## 1. 引言

### 1.1 背景

当前外部供应接口根据用户和 `platform` 获取或创建平台专用 API Key。新建 Key 时自动绑定系统默认分组，且不会校验用户是否具有该分组的访问权限。当前幂等范围为 `(user_id, platform)`，因此同一用户和同一调用系统只能拥有一个平台 Key。

本次改造要求调用方明确指定 Key 使用的分组，并把现有专属分组授权规则纳入创建流程。同时，允许同一用户、同一调用系统针对不同分组分别创建 Key。

### 1.2 术语

- `platform`：调用或使用 API Key 的外部系统标识，例如 `github`、`gitlab`、`internal_portal`。它不表示大模型所属平台。
- `groups.platform`：分组关联的大模型平台。它与请求中的 `platform` 是不同概念，本功能不要求二者匹配。
- 公开标准分组：`subscription_type = standard` 且 `is_exclusive = false`。
- 专属标准分组：`subscription_type = standard` 且 `is_exclusive = true`。
- 订阅分组：`subscription_type = subscription`，本功能明确不支持。
- 平台 Key：`api_keys.purpose = platform` 的 API Key。

### 1.3 目标

- `FR-01`：调用方必须通过 `group_name` 指定 Key 绑定的分组。
- `FR-02`：仅允许使用存在、启用且类型为标准类型的分组。
- `FR-03`：公开标准分组允许有效用户直接使用。
- `FR-04`：专属标准分组仅允许已获得该分组授权的用户使用。
- `FR-05`：以 `(user_id, platform, group_id)` 为范围幂等获取或创建平台 Key。
- `FR-06`：相同用户和 `platform` 可以针对不同分组分别创建 Key。
- `FR-07`：请求中的 `platform` 不与 `groups.platform` 做一致性校验。
- `FR-08`：取消本接口创建 Key 时自动绑定系统默认分组的行为。

### 1.4 成功标准

- 未提供有效 `group_name` 时不能创建 Key。
- 无权限用户不能为专属分组创建 Key。
- 相同三元组重复调用始终返回同一条未软删除 Key。
- 相同用户和 `platform` 指定不同分组时可以获得不同 Key。
- 并发调用相同三元组时最多创建一条未软删除 Key。
- 请求 `platform` 与 `groups.platform` 不同时仍可正常创建。

### 1.5 范围

本次范围包括：

- 外部供应接口请求与响应契约；
- 用户解析后的分组查询和授权校验；
- 平台 Key 服务及仓储查询维度；
- 并发幂等处理；
- 必要的数据库查询索引；
- 单元测试、仓储测试、接口契约测试和集成测试。

### 1.6 非目标

- 不修改分组管理、用户管理或订阅管理页面；
- 不实现订阅分组的有效订阅校验；
- 不要求请求 `platform` 与 `groups.platform` 一致；
- 不迁移、解绑或修改现有平台 Key；
- 不改变普通用户创建 API Key 的流程；
- 不扩展通用 RBAC 或新的用户分组授权模型；
- 不修改外部供应接口现有 Bearer Token 认证方式。

## 2. 假设与已确认决策

### 2.1 已确认约束

- 请求参数新增必填字段 `group_name`。
- 分组名称在现有模型中全局唯一，按名称可解析为唯一分组。
- 订阅分组本次直接拒绝。
- 专属标准分组使用现有 `user_allowed_groups` 关系判断用户授权。
- 公开标准分组不要求显式用户授权。
- 平台 Key 唯一业务范围调整为 `(user_id, platform, group_id)`。
- 请求 `platform` 表示调用系统，不代表大模型平台。
- 不比较请求 `platform` 与 `groups.platform`。
- 已有 Key 不因新请求指定其他分组而迁移；不同分组创建不同 Key。

### 2.2 保守默认

- `group_name` 去除首尾空格后进行精确名称查询，不引入模糊匹配。
- 分组必须满足 `status = active`。
- 用户必须满足现有 `User.IsActive()` 判断。
- 已存在但状态为 `disabled`、`expired` 或 `quota_exhausted` 的未软删除平台 Key，继续按现有行为返回，不自动新建。
- 已软删除 Key 不参与幂等匹配，允许重新创建。
- 保留现有 `platform` 格式规则：小写字母和下划线、以字母开头、最长 50 个字符。
- 返回结果增加 `group_id` 和 `group_name`，便于调用方确认实际绑定分组。

### 2.3 无阻塞待定项

无。错误响应的中文或英文 `detail` 可沿用项目统一错误响应机制，但稳定错误码须按本 Plan 定义。

## 3. 概念功能设计

### 3.1 请求分组解析

参与者：持有外部供应 Bearer Token 的可信调用系统。

输入：用户标识、调用系统 `platform`、分组名 `group_name`。

行为：

1. 验证请求字段；
2. 解析本地用户，必要时通过 LDAP 创建本地用户；
3. 按名称查询分组；
4. 验证分组状态和类型；
5. 对专属分组校验用户授权；
6. 按用户、调用系统和分组获取或创建 Key。

输出：API Key、用户信息、调用系统标识、分组信息以及用户/Key 是否新建。

### 3.2 分组准入规则

| 分组条件 | 结果 |
|---|---|
| 分组不存在 | 拒绝 |
| 分组不是 active | 拒绝 |
| `subscription_type = subscription` | 拒绝 |
| 标准公开分组 | 允许 |
| 标准专属分组且存在用户授权 | 允许 |
| 标准专属分组但不存在用户授权 | 拒绝 |

专属授权判断复用现有领域逻辑 `User.CanBindGroup(groupID, isExclusive)`，避免产生第二套准入规则。

### 3.3 多分组 Key

同一用户、同一 `platform` 可在不同分组下分别持有平台 Key：

| user_id | platform | group_id | 结果 |
|---:|---|---:|---|
| 100 | github | 12 | Key-A |
| 100 | github | 18 | Key-B |
| 100 | gitlab | 12 | Key-C |

重复请求 `(100, github, 12)` 返回 Key-A，不创建新记录。

### 3.4 错误与边界行为

- 请求缺字段或格式错误：返回 400；
- 用户在本地及 LDAP 均不存在：返回 404；
- 用户状态不可用：返回业务错误，不创建 Key；
- 分组不存在：返回 404；
- 分组未启用：返回 409；
- 订阅分组：返回 400；
- 专属分组未授权：返回 403；
- 分组查询、LDAP、用户供应或数据库异常：返回 500，并写失败审计；
- 任一校验失败时不得创建或修改 API Key。

## 4. 详细技术设计

### 4.1 组件职责

| 组件 | 职责 |
|---|---|
| `ExternalProvisioningHandler` | 解析新增请求字段、映射稳定错误、返回分组信息 |
| `ExternalProvisioningService` | 编排用户解析、分组解析、授权校验和 Key 获取/创建 |
| `GroupRepository` | 按全局唯一名称查询分组 |
| `PlatformAPIKeyService` | 按三元组实现平台 Key 幂等创建 |
| `APIKeyRepository` | 按三元组查询未软删除 Key，并在创建前执行并发安全重复检查 |
| `ExternalProvisioningAuth` | 保持现有 Bearer Token 认证 |
| `ProvisioningHardening` | 保持 Content-Type、4 KB 请求体、限流和审计保护 |

### 4.2 API 契约

#### `API-01` 获取或创建指定分组的平台 Key

```http
POST /api/v1/integrations/api-keys/getOrCreate
Authorization: Bearer <external provisioning token>
Content-Type: application/json
```

请求：

```json
{
  "user": "user@example.com",
  "platform": "github",
  "group_name": "企业专属组"
}
```

字段：

| 字段 | 类型 | 必填 | 规则 | 语义 |
|---|---|---:|---|---|
| `user` | string | 是 | 去除首尾空格，按现有邮箱解析规则处理 | 本地或 LDAP 用户标识 |
| `platform` | string | 是 | `^[a-z][a-z_]*$`，最长 50 | 使用 Key 的调用系统 |
| `group_name` | string | 是 | 去除首尾空格后不能为空 | Key 绑定的分组名称 |

成功响应：

```json
{
  "api_key": "sk-...",
  "user_id": 100,
  "user": "user@example.com",
  "username": "测试用户",
  "platform": "github",
  "group_id": 12,
  "group_name": "企业专属组",
  "user_created": false,
  "key_created": true
}
```

- 创建用户或 Key 时返回 HTTP 201；
- 用户和目标三元组 Key 均已存在时返回 HTTP 200；
- 保留 `Cache-Control: no-store` 和 `Pragma: no-cache`。

错误契约：

| HTTP | 错误码 | 场景 |
|---:|---|---|
| 400 | `INVALID_REQUEST` | 缺少字段或字段格式错误 |
| 400 | `SUBSCRIPTION_GROUP_NOT_SUPPORTED` | 指定订阅分组 |
| 401 | `INVALID_ACCESS_TOKEN` | Bearer Token 无效 |
| 403 | `GROUP_NOT_ALLOWED` | 用户无专属分组权限 |
| 404 | `NOT_FOUND` | 供应接口未启用 |
| 404 | `USER_NOT_FOUND` | 本地和 LDAP 均不存在用户 |
| 404 | `GROUP_NOT_FOUND` | 分组名称不存在 |
| 409 | `GROUP_INACTIVE` | 分组未启用 |
| 429 | `RATE_LIMITED` | 超过接口限流 |
| 500 | `INTERNAL_ERROR` | 内部依赖或持久化失败 |

### 4.3 服务接口调整

平台 Key 服务调整为：

```go
GetOrCreatePlatformKey(ctx, userID, platform, groupID)
```

平台 Key 仓储查询调整为：

```go
GetByUserIDPlatformAndGroup(ctx, userID, platform, groupID)
```

旧的仅按 `(user_id, platform)` 查询不再用于外部供应流程；如无其他调用方，可删除或重命名，避免误用旧语义。

### 4.4 数据模型

#### `groups`

本次不新增字段。

| 字段 | 类型 | Null | 默认值 | 约束/索引 | 本功能含义 |
|---|---|---:|---|---|---|
| `id` | BIGINT | 否 | — | 主键 | Key 绑定分组 ID |
| `name` | VARCHAR | 否 | — | 全局唯一 | 请求 `group_name` 的解析目标 |
| `status` | VARCHAR | 否 | active | 索引 | 必须为 active |
| `subscription_type` | VARCHAR | 否 | standard | 索引 | 必须为 standard |
| `is_exclusive` | BOOLEAN | 否 | false | 索引 | 决定是否校验用户授权 |
| `platform` | VARCHAR | 否 | — | 索引 | 不与请求 `platform` 比较 |

#### `user_allowed_groups`

本次不新增字段。

| 字段 | 类型 | Null | 默认值 | 约束/索引 | 本功能含义 |
|---|---|---:|---|---|---|
| `user_id` | BIGINT | 否 | — | 联合主键、外键 | 被授权用户 |
| `group_id` | BIGINT | 否 | — | 联合主键、外键、索引 | 专属分组 |
| `created_at` | DATETIME(6) | 否 | 当前时间 | — | 授权建立时间 |

#### `api_keys`

不新增业务字段，继续使用现有 `user_id`、`platform`、`group_id`、`purpose` 和软删除字段。

| 字段 | 类型 | Null | 默认值 | 约束/索引 | 本功能含义 |
|---|---|---:|---|---|---|
| `id` | BIGINT | 否 | — | 主键 | Key ID |
| `user_id` | BIGINT | 否 | — | 外键、索引 | Key 所属用户 |
| `platform` | VARCHAR(50) | 是 | NULL | 新增组合查询索引 | 调用系统标识 |
| `group_id` | BIGINT | 是 | NULL | 外键、索引 | 本接口新建时必须非空 |
| `purpose` | VARCHAR(20) | 否 | user_created | — | 本接口写入 platform |
| `deleted_at` | DATETIME(6) | 是 | NULL | 索引 | 仅未软删除记录参与幂等匹配 |

新增普通组合索引：

```text
(user_id, platform, group_id, deleted_at)
```

数据库层暂不增加简单 UNIQUE 索引，因为软删除后需要允许相同三元组重新创建，且 MySQL 对含 NULL 的唯一键语义不能直接表达“仅未删除记录唯一”。并发唯一性继续采用当前用户行锁和事务内重复查询方案。

### 4.5 事务与并发

平台 Key 创建沿用现有并发策略并把查询维度扩展为三元组：

1. 查询未软删除的目标三元组 Key；
2. 未找到时开启创建事务；
3. 锁定对应用户行；
4. 在锁内重新查询 `(user_id, platform, group_id, deleted_at IS NULL)`；
5. 若已存在则返回冲突，由服务重新读取该三元组；
6. 否则创建新 Key 并提交。

分组权限校验发生在创建尝试前。并发期间管理员撤销授权属于低概率竞态；本次以调用开始时读取到的授权状态为准，不额外锁定授权关系。若后续要求撤权与 Key 创建严格串行，应另行引入授权版本或统一授权事务边界。

### 4.6 数据迁移与兼容性

- 新增组合查询索引，不回填或修改现有数据；
- 现有带 `group_id` 的平台 Key 可被新三元组查询命中；
- 现有 `group_id IS NULL` 的平台 Key 不会被新请求命中；调用方指定分组后创建新 Key；
- 旧调用方未传 `group_name` 时返回 400，这是已批准的破坏性接口变更；
- 不再读取默认分组配置来创建平台 Key；
- 回滚代码时新增普通索引可以保留，不影响旧逻辑；如需完整回滚，可单独删除索引。

### 4.7 安全与隐私

- 保持独立 Bearer Token 认证和常量时间比较；
- 保持接口关闭时返回 404；
- 保持 4 KB 请求体限制和 `application/json` 限制；
- 保持按来源 IP 限流；
- 明文 API Key 响应继续禁止缓存；
- 审计日志增加 `group_id` 或安全的 `group_name`，但不得记录明文 API Key 或 Bearer Token；
- 专属分组授权失败日志记录用户 ID、分组 ID、调用系统和结果，不记录敏感凭据。

### 4.8 可靠性与运维

- LDAP、用户供应、分组查询或数据库失败时不降级为跳过权限检查；
- 分组查询失败必须 fail closed；
- 保持现有成功/失败审计事件，并增加分组维度；
- 建议监控 `GROUP_NOT_FOUND`、`GROUP_NOT_ALLOWED`、`SUBSCRIPTION_GROUP_NOT_SUPPORTED` 和创建冲突重读失败次数；
- 不新增外部服务、后台任务或消息队列；
- 容量影响主要为按三元组查询，组合索引用于避免用户 Key 数量增长后的扫描。

## 5. 伪代码与运行逻辑

### 5.1 外部供应主流程

```text
function ensurePlatformKey(input):
  validate input.user
  validate input.platform
  validate trim(input.group_name) is not empty

  user = findLocalUser(normalize(input.user))
  userCreated = false
  if user not found:
    ldapUser = findLDAPUser(input.user)
    if ldapUser not found:
      return USER_NOT_FOUND
    user, userCreated = provisionLocalUser(ldapUser)
  else if lookup failed:
    return INTERNAL_ERROR

  if user is not active:
    reject

  group = findGroupByExactName(trim(input.group_name))
  if group not found:
    return GROUP_NOT_FOUND
  if group.status != active:
    return GROUP_INACTIVE
  if group.subscription_type != standard:
    return SUBSCRIPTION_GROUP_NOT_SUPPORTED
  if not user.CanBindGroup(group.id, group.is_exclusive):
    return GROUP_NOT_ALLOWED

  key, keyCreated = getOrCreatePlatformKey(
    user.id,
    input.platform,
    group.id
  )
  if failed:
    return INTERNAL_ERROR

  audit success without secrets
  if userCreated or keyCreated:
    return HTTP 201
  return HTTP 200
```

### 5.2 三元组幂等创建

```text
function getOrCreatePlatformKey(userID, platform, groupID):
  validate userID > 0
  validate platform
  validate groupID > 0

  existing = findUndeleted(userID, platform, groupID)
  if existing:
    return existing, false
  if lookup failed other than not-found:
    return error

  generatedKey = generateKey()
  candidate = APIKey(
    user_id = userID,
    platform = platform,
    group_id = groupID,
    purpose = platform,
    status = active
  )

  begin transaction
  lock user row
  if findUndeleted(userID, platform, groupID) exists:
    rollback
    return existing, false
  persist candidate
  commit
  return candidate, true
```

## 6. 验证策略

### 6.1 单元测试

- 请求缺少 `group_name`；
- `group_name` 只有空格；
- 公开标准分组允许；
- 专属标准分组有授权允许；
- 专属标准分组无授权拒绝；
- 订阅分组拒绝；
- inactive 分组拒绝；
- 分组不存在拒绝；
- 请求 `platform` 与 `groups.platform` 不同仍允许；
- LDAP 新用户使用公开分组成功；
- LDAP 新用户使用未授权专属分组失败；
- 服务错误映射到稳定 HTTP 错误码。

### 6.2 仓储与并发测试

- 相同三元组返回同一 Key；
- 相同用户和 `platform`、不同 `group_id` 创建不同 Key；
- 相同用户和 `group_id`、不同 `platform` 创建不同 Key；
- 不同用户相同 `platform` 和 `group_id` 创建不同 Key；
- 已软删除三元组允许重新创建；
- 未软删除但 disabled/expired/quota_exhausted 的 Key 被返回；
- 并发相同三元组最终只有一条未软删除记录；
- 组合索引存在且查询路径可使用。

### 6.3 契约与集成测试

- 请求、响应包含正确的 `group_name`、`group_id`；
- 创建返回 201，重复获取返回 200；
- 响应包含 `Cache-Control: no-store` 和 `Pragma: no-cache`；
- Bearer Token、Content-Type、请求体限制和限流保持有效；
- 失败场景不会新增 API Key；
- 老请求不传 `group_name` 时稳定返回 400。

### 6.4 验收标准

- `AC-01`：有效用户指定公开标准分组时成功获得绑定该分组的 Key。
- `AC-02`：用户指定已授权专属标准分组时成功获得 Key。
- `AC-03`：用户指定未授权专属标准分组时返回 403，数据库无新增 Key。
- `AC-04`：指定订阅分组时返回 400，数据库无新增 Key。
- `AC-05`：相同三元组连续调用返回相同 Key ID 和 Key 值。
- `AC-06`：相同用户和 `platform` 指定两个分组时产生两个不同 Key。
- `AC-07`：请求 `platform` 和分组平台不同时仍成功。
- `AC-08`：并发相同三元组调用后最多存在一条未软删除 Key。
- `AC-09`：新创建 Key 的 `purpose = platform` 且 `group_id` 为请求分组 ID。
- `AC-10`：接口不再将新 Key 绑定到系统默认分组。

## 7. 实施顺序与后续拆分边界

### 阶段一：契约与领域错误

- 增加 `group_name` 请求字段和分组响应字段；
- 定义稳定错误类型及 HTTP 映射；
- 更新接口契约测试。

产物：确定的 API 请求、响应和错误契约。

### 阶段二：分组解析与授权

- 为供应服务注入所需的分组查询能力；
- 查询并校验分组；
- 复用 `CanBindGroup` 校验专属授权；
- 明确拒绝订阅分组。

产物：创建前完整的分组准入链路。

### 阶段三：三元组幂等

- 修改平台 Key 服务和仓储接口；
- 修改锁内重复检查；
- 移除默认分组解析依赖；
- 增加组合索引。

产物：支持同一调用系统多分组 Key 的持久化能力。

### 阶段四：验证与兼容检查

- 完成单元、仓储、并发和集成测试；
- 验证旧数据行为；
- 验证安全中间件和审计未退化。

产物：测试证据和可发布实现。

推荐后续按上述四个阶段拆成 MVP；阶段二依赖阶段一，阶段三可在阶段一契约确定后与阶段二并行开发，阶段四依赖前述全部阶段。

## 8. 风险与开放项

| 风险 | 可能性/影响 | 缓解措施 | 触发与处置 |
|---|---|---|---|
| 老调用方未传 `group_name` | 高/高 | 发布前同步调用方并更新契约 | 出现 400 时升级调用方请求 |
| 现有无分组平台 Key 不再命中 | 中/中 | 明确保留旧 Key并创建指定分组的新 Key | 通过审计确认新旧 Key 数量 |
| 平台字段被误解为大模型平台 | 中/中 | 文档和代码注释明确“调用系统标识” | 禁止增加与 `groups.platform` 的比较 |
| 并发重复创建 | 低/高 | 用户行锁、锁内复查、并发测试 | 冲突后按三元组重读 |
| 权限校验后立即撤权的竞态 | 低/中 | 本次按请求读取时授权为准 | 若业务要求严格串行，后续引入授权版本 |
| 订阅分组被调用方误用 | 低/低 | 返回稳定、可识别错误码 | 调用方改用标准分组 |

无阻塞开放项。

## 9. 追踪矩阵

| 需求 | 功能模块 | 技术组件 | 验收标准 |
|---|---|---|---|
| `FR-01` | 请求分组解析 | Handler、API 契约 | `AC-01`、`AC-03` |
| `FR-02` | 分组准入 | Provisioning Service、Group Repository | `AC-01`、`AC-04` |
| `FR-03` | 公开分组访问 | `User.CanBindGroup` | `AC-01` |
| `FR-04` | 专属分组授权 | `user_allowed_groups`、`User.CanBindGroup` | `AC-02`、`AC-03` |
| `FR-05` | 三元组幂等 | PlatformAPIKeyService、APIKeyRepository | `AC-05`、`AC-08` |
| `FR-06` | 多分组 Key | APIKeyRepository | `AC-06` |
| `FR-07` | platform 语义分离 | Provisioning Service | `AC-07` |
| `FR-08` | 移除默认分组绑定 | PlatformAPIKeyService | `AC-09`、`AC-10` |

## 10. 评审记录

| 版本 | 状态 | 评审结果 |
|---|---|---|
| v0.1 | Draft | 初步梳理现有用户—专属分组授权及平台 Key 供应关系 |
| v0.2 | Draft | 确认新增 `group_name`、拒绝订阅分组、三元组唯一范围，以及 `platform` 为调用系统标识 |
| v1.0 | Final — 用户已批准 | 用户确认方案符合预期并要求直接形成文档，不执行其他变更 |
