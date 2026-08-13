# MVP-002：账号并发变化同步候选分配

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Dependencies: `MVP-001`

## 目标

账号并发更新时，在事务内重算并保存相关候选分配，并同步 Redis 配置缓存。

## 范围

- 账号更新流程读取旧并发和候选关系；
- 调用 MVP-001 的计算能力；
- 事务内更新账号和候选配置；
- 账号 0/正数切换；
- 数据库提交后同步候选 Redis 配置；
- 更新失败时保持数据库与缓存的一致性策略。

## 非范围

- 不改请求执行时的并发槽位算法；
- 不增加全局限流逻辑。

## 验收标准

- [x] 账号并发修改会同步调整具体候选分配；
- [x] `null` 候选不会被改写；
- [x] 数据库事务失败时账号和候选配置不会部分提交；
- [x] 正数总和始终不超过账号新并发；
- [x] 成功提交后相关 Redis 配置得到更新；
- [x] 存在账号并发更新的自动化测试。

## 验证计划

- 运行 repository/service/admin handler 相关测试；
- 覆盖正数变更、0→正数、正数→0和事务失败场景。

## 证据

完成证据：

- `backend/internal/repository/account_repo.go` 将账号更新与候选分配重算放入同一事务；
- `backend/internal/repository/group_model_route_account_repo.go` 新增账号候选分配重算逻辑；
- `backend/internal/service/admin_service.go` 在账号并发修改后刷新相关 Redis 配置；
- 批量账号并发更新也会按账号逐个重算候选分配；
- `go test ./internal/service ./internal/repository ./internal/handler/admin` 通过。
