# MVP-015：实现审计与系统角色安全保护

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `1 个专注开发日`
- Estimate rationale: `将高风险不变量和审计事务集中交付，可独立用 Service 测试验证。`
- Dependencies: `MVP-004, MVP-005`

## 预期成果

admin/user 系统角色受到不可绕过的保护，所有角色和权限变更具有事务内审计记录。

## 背景

admin 不能删除、停用、改编码或失去 `*`；不能移除最后一个有效超级管理员。

## 范围内

- 角色和权限变更审计 Repository/Service。
- admin/user 系统角色保护。
- 自定义角色禁止获得 `*`。
- 最后超级管理员并发保护。
- 修改内置 user 权限的高影响标记和审计。

## 范围外

- HTTP 管理 API。
- 前端二次确认界面。

## 实现说明

- 业务变更、版本递增和审计写入同一事务。
- 最后 admin 检查使用数据库锁或等价并发安全机制。
- 审计失败应使高风险变更回滚。

## 验收标准

- [x] admin 不能删除、停用、改编码或移除 `*`。
- [x] 自定义角色不能绑定 `*`。
- [x] 并发请求不能移除最后一个有效 admin。
- [x] 角色、权限和用户角色变更均生成完整审计快照。

## 验证计划

- `cd backend && go test ./internal/rbac/... -run '(Audit|SystemRole|LastAdmin)'`
- `cd backend && go test ./internal/repository/... -run RBACAudit`

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 安全规则 | `backend/internal/rbac/mutation.go` | admin 必须是系统角色且保留 `*`；非 admin 角色禁止绑定 `*`。 |
| 事务实现 | `backend/internal/repository/rbac_mutation_repo.go` | 权限/用户角色替换、版本递增和 before/after 审计在同一事务；最后 admin 检查通过 `FOR UPDATE` 锁住有效 admin 分配。 |
| 测试 | `go test ./internal/rbac/... -run '(Audit|SystemRole|LastAdmin)' -count=1` | 通过。 |
| 测试 | `go test ./internal/repository/... -run 'RBAC(Audit|LastAdmin)' -count=1` | 通过；验证审计失败回滚和最后 admin 锁定拒绝。 |

## 执行记录

锁策略为事务内 `SELECT ... FOR UPDATE` 锁定目标分配和全部有效 admin 分配；审计插入失败不提交业务变更。审计仅保存角色/权限编码快照，不记录密钥、Token 等敏感值；内置 user 权限变更使用 `system_user.permissions.replace_high_impact` 动作标记。
