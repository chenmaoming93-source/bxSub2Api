# MVP-006：动态 Token 限额 wildcard 后端语义

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `类型扩展、规则管理校验和运行时 lookup identity 属于一个独立后端切片，现有 quota 测试可直接扩展。`
- Dependencies: `none`

## 预期成果

限额规则可将任意维度配置为 `{type:"wildcard"}`；规则匹配任意实际值，但每次使用请求实际值构造临时查询组合并分别限额。

## 背景

`DimensionValue` 位于 `backend/internal/service/tokenstat/types.go`，限额检查位于 `backend/internal/service/tokenstat/quota.go`。当前 `ruleMatches` 和 identity 均依赖严格规则值。

## 范围内

- 新增 `ValueTypeWildcard`。
- 限额创建、加载和序列化支持 wildcard。
- 规则匹配时 wildcard 接受任意存在的请求值。
- 新建局部 `lookupValues`，以实际值构造查询 hash。
- 保持 Redis 失败时现有 fail-open 行为。
- 对相同 lookup identity 做请求内去重（若同次检查存在重复读取）。

## 范围外

- 修改规则自身的 `DimensionValues`。
- 将 wildcard 写入 Usage Event。
- 创建跨实际值聚合桶。
- 改变普通统计查询 filter 语义。

## 实现说明

- wildcard 维度缺少请求实际值时规则不适用。
- `validateDimensionValue` 需区分事件具体值校验与限额配置值校验，防止 wildcard 进入统计事件。
- 规则持久化 hash 不得误用于 wildcard 的运行时统计读取。

## 验收标准

- [x] wildcard 规则可创建、加载和展示。
- [x] 不同实际维度值生成不同查询 hash，统计值不合并。
- [x] 检查前后规则 Map 内容完全不变。
- [x] 缺失实际维度时规则不适用。
- [x] 固定维度仍严格等值匹配，Redis 错误仍 fail-open。

## 验证计划

- `cd backend && go test ./internal/service/tokenstat`
- `cd backend && go test ./internal/service -run 'DynamicTokenQuota|TokenQuotaRouting'`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 类型/校验 | `ValueTypeWildcard`、`WildcardValue`、`validateQuotaDimensionValue` | quota 配置接受 wildcard；UsageEvent 与普通 identity 继续拒绝 wildcard，避免写入统计事件。 |
| 持久化 | `quota_admin.go` | wildcard 以 `{type:"wildcard"}` 明确存储并可加载；配置 hash 独立计算，不用于运行时 counter 查询。 |
| 运行时 | `QuotaChecker.Check` | ruleMatches 仅判断适用性；局部 lookupValues 从 available 复制实际值，规则 map 不变；相同 key/field 请求内只读一次。 |
| 测试 | `cd backend && go test ./internal/service/tokenstat` | 通过。 |
| 服务回归 | `cd backend && go test ./internal/service -run 'DynamicTokenQuota|TokenQuotaRouting' -count=1` | 通过（当前无匹配测试时正常显示 no tests to run）。 |

## 执行记录

- 示例测试规则 `(group=3, route=*, account=18)` 每日 100 万：`claude-code` 查询其具体 hash 得到 60 万，`gpt-code` 查询另一具体 hash 得到 45 万，二者均不触发且不合并。
- wildcard 维度在请求中缺失时，规则不适用且 Redis 读取次数为 0。
- 两条规则命中相同具体 identity 时共享本次 Check 的读取结果，测试断言 Redis reader 仅调用 1 次。
- 原固定值比较和 Redis 错误 fail-open 分支保持不变。
