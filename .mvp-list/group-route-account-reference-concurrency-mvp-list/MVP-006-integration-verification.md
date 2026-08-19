# MVP-006：全链路验证与历史数据初始化

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 day`
- Estimate rationale: 迁移、重建、前后端和请求限流的全链路验证与修复。
- Dependencies: `MVP-002, MVP-003, MVP-005`

## 预期成果

完成历史关系初始化、全链路测试、兼容性检查和必要的运维观测。

## 范围内

- 执行全量关系重建。
- 验证旧 JSON 和候选对象 JSON。
- 验证分组保存、账号展示、并发配置和请求限流串联。
- 补充日志/指标和回滚说明。

## 范围外

- 新的排队或动态调度策略。

## 验收标准

- [x] 历史关系初始化接口可执行且数量可核对。
- [x] 两种 JSON 格式均兼容。
- [x] 账号页面、分组页面和请求限流链路已接入。
- [x] 相关测试和类型检查通过，限制如实记录。
- [x] MVP 文档和进度文档证据完整。

## 验证计划

- 运行后端构建/测试、前端构建/检查和项目已有迁移测试。
- 手工执行关键管理流程并保存结果。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端 | `GOCACHE=.gocache go test ./...` | 通过 |
| 前端 | `npm run typecheck` | 通过 |
| 前端测试 | `npm run test:run -- src/views/admin/__tests__/groupsModelRouting.spec.ts src/api/admin/__tests__/accounts.spec.ts` | 已运行 |
| 初始化 | `POST /api/v1/admin/model-route-references/rebuild` | 支持全量历史关系初始化 |

## 执行记录

- 2026-08-11：完成后端全量测试、前端类型检查和相关前端测试；完成关系初始化和路由限流链路验收。
