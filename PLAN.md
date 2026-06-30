# 分组模型路由与每日 Token 限额实现计划

## Summary

实现“用户在请求体 `model` 里传分组模型名，后端按分组配置选择具体模型和账号”的能力，并按每日总 token 维度做两类限额：

- 全局模型每日限额：`model -> daily_token_limit`
- 用户模型每日限额：`user_id + model -> daily_token_limit`
- 分组内候选模型按优先级排序；高优先级模型失败或达到任一每日 token 限额时自动降级到下一候选。

总 token 口径使用现有 `UsageLog.TotalTokens()`：`input + output + cache_creation + cache_read`。

## Key Changes

- 扩展现有 `groups.model_routing` JSON，兼容旧格式：
  - 旧格式继续支持：`{ "claude-opus-*": [accountId...] }`
  - 新格式建议为：
    ```json
    {
      "fast-code": [
        { "model": "claude-sonnet-4-5", "account_ids": [1,2], "priority": 1, "daily_token_limit": 1000000 },
        { "model": "gpt-5.2-codex", "account_ids": [9], "priority": 2, "daily_token_limit": 500000 }
      ]
    }
    ```
  - `fast-code` 是用户请求的分组模型名；`model` 是实际上游模型。
  - 分组候选自己的 `daily_token_limit` 作为该分组模型内的模型日额度；未配置表示不限。

- 新增 DB 持久化表：
  - `model_token_daily_usages(model, usage_date, used_tokens, daily_limit_tokens, ...)`
  - `user_model_token_daily_usages(user_id, model, usage_date, used_tokens, daily_limit_tokens, ...)`
  - 使用唯一索引保证每日一行，日期按项目现有全局时区的 `StartOfDay` 口径。
  - 额度配置接口写入 `daily_limit_tokens`，用量记录成功后原子累加 `used_tokens`。

- 新增 service/repository/cache 层：
  - `ModelTokenQuotaRepository`：查询限额、校验当日是否耗尽、成功后累加 token。
  - Redis 缓存参考现有 `UserPlatformQuota` 模式：cache hit 直接判断，cache miss 回源 DB，写后标脏或同步 DB。
  - 限额检查失败返回可识别错误，例如 `ErrModelDailyTokenQuotaExhausted`、`ErrUserModelDailyTokenQuotaExhausted`，用于路由降级而不是直接返回客户端。

- 调整路由选择：
  - 在 `GatewayService.SelectAccountWithLoadAwareness` 中解析新格式 `model_routing`。
  - 如果请求模型命中新格式分组模型名，按候选 `priority` 升序选择。
  - 对每个候选先检查：
    - 分组候选每日 token 限额
    - 全局模型每日 token 限额
    - 当前用户该模型每日 token 限额
    - 账号可调度、平台、模型映射、模型限流、quota、window cost、RPM、并发
  - 候选通过后，将请求的实际上游模型改写为候选 `model`，并仅在该候选的 `account_ids` 内选择账号。
  - failover 时如果当前候选账号全部失败或候选模型达到限额，继续下一候选；全部耗尽后返回 429/503 中合适的现有错误响应。

- 调整转发与 usage：
  - `ForwardResult.Model` 保留客户端请求的分组模型名用于兼容展示时，新增或复用 `UpstreamModel` 记录实际候选模型。
  - 计费与 usage 中的 `requested_model` 写入分组模型名，`upstream_model` 写入实际模型。
  - 每次成功 `RecordUsage` 后，用实际模型和总 token 更新：
    - 分组候选模型用量
    - 全局模型用量
    - 用户模型用量

- 管理后台 UI：
  - 在 `GroupsView.vue` 现有“模型路由”区域升级为“分组模型路由”编辑器。
  - 每条分组模型规则包含：分组模型名、候选列表、候选实际模型、账号列表、优先级、候选每日 token 限额。
  - 在用户管理页新增用户模型每日 token 限额配置入口，复用现有用户平台额度弹窗的交互风格。
  - 新增全局模型每日 token 限额管理入口，可放在管理员设置或分组模型路由旁的独立弹窗中。
  - 更新 `frontend/src/types/index.ts`、admin groups API 类型与中英文文案。

## Test Plan

- 后端单元测试：
  - 旧 `model_routing` 格式仍可解析并按账号优先列表工作。
  - 新分组模型名命中后按 `priority` 选择候选模型。
  - 高优先级候选账号 failover 后降级到下一账号/下一模型。
  - 模型每日额度耗尽时跳过该候选。
  - 用户模型每日额度耗尽时只影响该用户，不影响其他用户。
  - usage 成功后按总 token 原子累加，失败请求不累加。

- 后端集成测试：
  - 新 migration 创建索引和默认字段正确。
  - 并发请求下 `used_tokens` 不丢增量。
  - 跨日窗口自动切换到新日期。
  - `requested_model` 和 `upstream_model` 写入符合预期。

- 前端测试：
  - 新旧 `model_routing` 数据加载、编辑、保存正常。
  - 候选模型增删、优先级、账号选择、token 限额输入校验正常。
  - 用户模型限额弹窗保存与回显正常。
  - i18n key 覆盖中英文。

## Assumptions

- 每日窗口按项目现有 `timezone.StartOfDay` 口径，不单独为每个用户设置时区。
- “模型维度限额”的模型名使用实际上游模型名，不使用用户传入的分组模型名。
- 限额值 `0` 或 `null` 表示不限额。
- `CountTokens` 不计入每日 token 限额。
- 旧 `model_routing` 配置保持兼容，但新 UI 保存后会写入新格式。