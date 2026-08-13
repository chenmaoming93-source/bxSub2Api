# MVP-001：候选并发分配计算能力

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Dependencies: `none`

## 目标

实现账号并发与候选并发分配之间的纯计算规则，不接入请求限流执行链路。

## 范围

- 具体候选并发总和计算；
- `null`/缺失配置排除；
- 账号并发为 0 或负数时的不限语义；
- 正数之间按比例缩放；
- 0 与正数互转；
- 最大余数法处理整数舍入；
- 比例和展示状态计算。

## 非范围

- 不修改请求候选选择、账号槽位或候选槽位逻辑；
- 不修改数据库或 Redis。

## 验收标准

- [x] 正数账号并发下，计算结果总和不超过新账号并发；
- [x] `null` 候选不参与分配总和和比例；
- [x] 账号并发为 0/负数时返回不限状态且不计算比例；
- [x] 0→正数和正数→0符合 Plan 规则；
- [x] 正数→正数使用最大余数法且结果可复现；
- [x] 存在覆盖边界条件的自动化测试。

## 验证计划

- 运行领域/服务层相关 Go 测试；
- 检查舍入、空列表、全为 `null`、总和小于账号并发等边界。

## 证据

完成证据：

- 新增 `backend/internal/service/model_route_concurrency_allocation.go`，实现候选分配缩放和比例计算；
- 新增 `backend/internal/service/model_route_concurrency_allocation_test.go`，覆盖正数缩放、`null` 候选、0/正数切换和比例计算；
- `go test ./internal/service -run 'TestScaleModelRouteConcurrencyAllocations|TestCandidateConcurrencyShare'` 通过。
