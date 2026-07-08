# MVP-010：新增 LDAP 连接与目录查询组件

- Protocol: `mvp-list/v1`
- State: `VERIFIED`
- Estimate: `约 30 分钟`
- Estimate rationale: `范围限定为一个可独立验证的结果；预计在 15–45 分钟窗口内完成实现与定向测试。`
- Dependencies: `none`

## 预期成果

新版 `LDAPDirectory.LookupUser` 可通过服务账户无密码精确查询目录用户。

## 背景

旧版 `backend/internal/pkg/ldapauth/client.go` 源码必须保留；新版逻辑写入独立文件/类型，不在此 MVP切换登录入口。

## 范围内

- 新增可注入的新版 LDAP连接、TLS/StartTLS和服务 Bind能力。
- 实现 `LookupUser`、属性映射、size limit=2及错误分类。
- 转义用户输入，区分未命中、重复命中和不可用。
- 使用假 LDAP连接或协议层 stub 完成单元测试。

## 范围外

- LDAP密码认证。
- 业务用户落库。
- 删除或调用旧版实现。

## 实现说明

- 旧版文件不删除、不重命名也可，优先最小 Git噪声。
- 没有服务账户时返回明确配置/不可用错误。

## 验收标准

- [x] Lookup不需要目标用户密码，并准确区分三类结果。
- [x] 旧版源码仍存在，新目录查询测试通过。

## 验证计划

- `cd backend; go test ./internal/pkg/ldapauth/... -run 'Directory|LookupUser'`

## 完成证据

> 在实际完成工作前保持本节为空。

| 类型 | 命令或路径 | 结果 |
|---|---|---|
| 定向测试 | `cd backend; go test ./internal/pkg/ldapauth/... -run 'Directory|LookupUser' -count=1` | 通过；覆盖命中、未命中、重复命中、不可用及过滤器转义。 |
| 完整回归 | `cd backend; go test ./internal/pkg/ldapauth/... -count=1` | 通过，退出码 0。 |
| 兼容确认 | `backend/internal/pkg/ldapauth/client.go` | 文件仍存在且未删除；完整包测试包含旧版测试并通过。 |
| 新版实现 | `backend/internal/pkg/ldapauth/directory.go` | 支持可注入连接、LDAP/LDAPS、StartTLS、服务账户 Bind、`SizeLimit=2` 和稳定错误分类。 |

## 执行记录

实现与验证完成。新版 `LDAPDirectory` 独立于旧版认证客户端，查询过程仅执行服务账户 Bind，不执行目标用户密码 Bind；没有服务凭据时返回 `ErrDirectoryUnavailable`。
