# Growth 当前用户积分排名接口

- 版本：v1.0
- 日期：2026-11-23
- 调用方：AIWork Main
- 协议：HTTPS
- 方法：`POST`
- 认证：SM2 数字信封，不使用 JWT、API Key 或身份 Header

## 1. 接口地址

```text
POST https://<sub2api-host>/api/v1/growth/rank
```

请求头：

```http
Content-Type: application/json
X-Growth-Client-Version: 1.0.0
```

`X-Growth-Client-Version` 为可选的客户端版本标识。

## 2. 请求 payload

业务请求内容需要先放入 payload，再按 Growth 现有 SM2 信封规范加密。

```json
{
  "path": "/growth/rank",
  "username": "user@example.com",
  "params": {},
  "request_id": "trace-rank-20261123-01",
  "issued_at": 1795422000,
  "nonce": "5e7ea36a-66ad-4f42-a104-85ac0cb8c55b"
}
```

字段说明：

| 字段 | 必填 | 说明 |
|---|---|---|
| `path` | 是 | 固定为 `/growth/rank` |
| `username` | 是 | 当前登录用户的业务标识，服务端映射到 `user

 
 
 
 
 
 
 
    合格vfzgbtarrshgte|
| `issued_at` | 是 | Unix 秒时间戳，默认允许与服务端相差 ±5 分钟 |
| `nonce` | 是 | 每次请求随机生成，不得复用 |

该接口是只读接口，不需要 `idempotency_key`。

## 3. HTTP 请求体

实际发送的 body 是加密信封：

```json
{
  "kid": "growth-sm2-v1",
  "encrypted_key": "<base64>",
  "iv": "<base64>",
  "ciphertext": "<base64>",
  "mac": "<base64>"
}
```

加密使用现有 Growth v4.0 信封规范：

- SM2：`C1C3C2`
- SM4：CBC + PKCS#7
- SM3-HMAC：使用 Base64 字段字符串拼接
- 每次请求重新生成 SM4 会话密钥、IV、nonce 和 `issued_at`

## 4. 响应

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 12345,
    "available_points": 260,
    "rank": 3,
    "total_users": 20,
    "rank_text": "3/20"
  }
}
```

字段说明：

| 字段 | 类型 | 说明 |
|---|---|---|
| `user_id` | integer | 当前用户 ID |
| `available_points` | integer | 当前可用积分 |
| `rank` | integer | 当前用户排名 |
| `total_users` | integer | 参与排名的总用户数 |
| `rank_text` | string | 可直接展示的 `xx/xxx` 排名文本 |

## 5. 排名规则

1. 按 `growth_wallets.available_points` 从高到低计算。
2. 没有成长钱包的用户按 `0` 分参与排名。
3. 相同积分排名相同。
4. 后续名次采用竞争排名。

例如：

```text
用户 A：100 分
用户 B：100 分
用户 C：80 分
用户 D：50 分
```

排名结果：

```text
用户 A：1/4
用户 B：1/4
用户 C：3/4
用户 D：4/4
```

排除规则：

```text
admin@example.com
```

该账号：

- 不参与积分排名
- 不计入 `total_users`

## 6. TypeScript 调用示例

以下示例假定 AIWork 已实现现有 `callGrowth` 和 `sealGrowthEnvelope`。

```ts
type PointsRank = {
  user_id: number
  available_points: number
  rank: number
  total_users: number
  rank_text: string
}

const ranking = await callGrowth<PointsRank>({
  pathTemplate: '/growth/rank',
  requestPath: '/growth/rank',
  username: 'user@example.com',
  params: {},
})

console.log(ranking.rank_text) // 例如：3/20
```

## 7. cURL 示例

业务参数在密文中，cURL 请求体必须由 AIWork Main 动态生成：

```bash
curl -X POST 'https://sub2api.example.com/api/v1/growth/rank' \
  -H 'Content-Type: application/json' \
  -H 'X-Growth-Client-Version: 1.0.0' \
  --data @growth-rank-envelope.json
```

`growth-rank-envelope.json`：

```json
{
  "kid": "growth-sm2-v1",
  "encrypted_key": "<base64>",
  "iv": "<base64>",
  "ciphertext": "<base64>",
  "mac": "<base64>"
}
```

## 8. 错误处理

| HTTP | code | 说明 | 处理方式 |
|---|---|---|---|
| 400 | `CRYPTO_ENVELOPE_INVALID` | 信封字段或 Base64 格式错误 | 不重试，检查客户端加密实现 |
| 400 | `CRYPTO_KEY_UNKNOWN` | `kid` 不是 `growth-sm2-v1` | 不重试，修正 `kid` |
| 400 | `CRYPTO_PATH_MISMATCH` | payload `path` 与实际 URL 不一致 | 不重试，修正 `path` |
| 401 | `CRYPTO_KEY_UNWRAP_FAILED` | SM2 公私钥不匹配 | 不重试，检查双方密钥 |
| 401 | `CRYPTO_MAC_INVALID` | SM3-HMAC 校验失败 | 不重试，检查 MAC 拼接 |
| 401 | `CRYPTO_DECRYPT_FAILED` | SM4 解密或 PKCS#7 校验失败 | 不重试，检查 SM4 参数 |
| 401 | `CRYPTO_REPLAY_REJECTED` | 时间超窗或 nonce 重复 | 重新生成 nonce 后重发 |
| 404 | `GROWTH_USER_NOT_FOUND` | `username` 未映射到 sub2api 用户 | 不重试，检查账号开通状态 |
| 404 | `POINTS_RANK_UNAVAILABLE` | 当前账号被排除，不能参与排名 | 不重试，当前账号为管理员排除账号 |

## 9. 联调检查

- `path` 必须是 `/growth/rank`
- `params` 必须是 `{}`
- `username` 必须映射到 sub2api `users.email`
- 每次请求重新生成 `nonce` 和 `issued_at`
- 每次请求重新生成 SM4 会话密钥和 IV
- 不传 `user_id`、`points` 等账务字段
- 前端直接展示 `data.rank_text` 即可

## 10. 相关文档

- 完整 AIWork 契约：[GROWTH_PLAN_AIWORK_API.md](GROWTH_PLAN_AIWORK_API.md)
- 完整调用示例：[GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md](GROWTH_PLAN_EXTERNAL_API_EXAMPLES.md)
