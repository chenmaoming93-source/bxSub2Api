# MVP-002：实现 Redis Key、Field 编码与 TTL

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `40min`
- Estimate rationale: 三类 Key、可逆编码和绝对过期时间可作为独立纯逻辑组件完成并测试。
- Dependencies: `MVP-001`

## 预期成果

提供统一组件生成每天三个 Redis Hash Key、无歧义 Field，并按 `Asia/Shanghai` 计算配置化绝对过期时间。

## 背景

Redis 客户端由 `backend/internal/repository/redis.go` 提供；统计维度定义位于 daily token quota 相关 service/repository。

## 范围内

- 三类每日 Key 生成。
- 模型、用户模型、分组候选 Field 编解码。
- 特殊字符、空值和非法 Field 处理。
- 业务日期及 `redis_retention_days` 绝对过期时间计算。

## 范围外

- Redis 网络读写和 MySQL 同步。

## 实现说明

- 不依赖未经转义的 `|` 拆分。
- 解码错误必须携带统计类型和失败原因，禁止只返回通用错误。

## 验收标准

- [x] 三类 Key 精确符合 Plan。
- [x] Field 编解码往返无损并覆盖特殊模型名。
- [x] 默认 TTL 为业务日期结束后完整保留 2 天。
- [x] 非法输入错误具体可诊断。

## 验证计划

- `cd backend; go test ./internal/repository -run "TokenStatistics.*(Key|Codec|TTL)"`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 实现 | `backend/internal/repository/token_statistics_codec.go` | 实现三类 Key、Base64URL+JSON 可逆 Field 编解码和固定绝对过期时间。 |
| 测试 | `cd backend; go test ./internal/repository -run "TokenStatistics.*(Key|Codec|TTL)" -count=1` | 通过：`ok github.com/Wei-Shaw/sub2api/internal/repository 5.289s`。 |
| 静态检查 | `git diff --check` | 通过，无空白错误。 |

## 执行记录

- 2026-07-14：采用不依赖裸 `|` 分隔的 URL-safe 编码；验证特殊字符、空值、非法 ID/Field 和上海业务日跨天边界。
- 首次测试进程在编译阶段触发 124 秒命令超时，无失败断言；随后使用同一聚焦范围重新运行并通过。
