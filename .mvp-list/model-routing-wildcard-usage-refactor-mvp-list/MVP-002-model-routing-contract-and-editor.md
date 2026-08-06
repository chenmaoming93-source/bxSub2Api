# MVP-002：模型路由候选契约与编辑器去除 model

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `领域 JSON 兼容和前端编辑器属于同一可观察契约切片，现有测试文件集中，可独立完成验证。`
- Dependencies: `none`

## 预期成果

新建和编辑的 `groups.model_routing` 候选只包含 `account_ids/priority`；旧候选中的 `model` 仍可读取但不再输出或参与新配置。

## 背景

领域解析位于 `backend/internal/domain/model_routing.go`，前端编辑器位于 `frontend/src/components/admin/group/GroupModelRoutingEditor.vue`，现有实现要求候选提供 model。

## 范围内

- 调整候选 JSON 的解析、校验和序列化。
- 兼容旧候选对象和旧账号 ID 数组。
- 前端移除模型选择、模型加载和交集计算交互。
- 保留账号多选与 priority。
- 更新前后端契约测试。

## 范围外

- 从账号解析运行时上游模型。
- 改变候选 priority 排序。
- 迁移或批量重写已有数据库 JSON。

## 实现说明

- 若保留旧 `model` 接收字段，应确保新序列化不输出它。
- 保留稳定的候选 priority 排序和 `account_ids` 校验。
- 删除前端不再需要的账号模型请求、竞态处理和缓存代码。

## 验收标准

- [x] 不含 `model` 的候选可成功解析和保存。
- [x] 含旧 `model` 的候选可读取，运行契约中忽略该字段。
- [x] 新输出 JSON 不包含 `model`。
- [x] 前端只展示路由别名、账号和 priority。
- [x] 原 priority 和账号顺序兼容测试通过。

## 验证计划

- `cd backend && go test ./internal/domain ./internal/handler/admin`
- `cd frontend && pnpm test:run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/views/admin/__tests__/groupsModelRouting.spec.ts`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 后端契约 | `backend/internal/domain/model_routing.go` | 候选可不含 model；历史 model 被忽略且候选序列化不输出；旧 ID 数组、账号顺序和 priority 稳定排序保留。 |
| 前端编辑器 | `frontend/src/components/admin/group/GroupModelRoutingEditor.vue` | 移除模型选择、模型接口请求、交集/缓存/竞态状态，只保留别名、账号多选和 priority。 |
| 后端测试 | `cd backend && go test ./internal/domain ./internal/handler/admin` | 通过。 |
| 前端测试 | `cd frontend && pnpm test:run src/components/admin/group/__tests__/GroupModelRoutingEditor.spec.ts src/views/admin/__tests__/groupsModelRouting.spec.ts` | 2 个文件、6 个测试通过。 |
| 类型检查 | `cd frontend && pnpm typecheck` | 通过。 |

## 执行记录

- Go 候选结构暂留 `Model` 作为 `json:"-"` 的进程内过渡字段，确保后续运行时 MVP 可独立迁移；JSON 读写契约已不再包含它。
- 历史对象中的未知 `model` 由标准 JSON 解码忽略；前端 normalize 同样只复制 `account_ids/priority`，再次保存时自然删除旧字段。
- 删除 `getAvailableModels` 调用、账号模型交集、请求缓存、加载失败重试、竞态版本和模型失效校验。
