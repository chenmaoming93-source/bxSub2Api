# 模型路由与分组路由查询使用指南

本文介绍以下功能的配置与使用方式：

- 在管理端配置模型路由时，先选择账号，再选择账号共同支持的上游模型。
- 通过 integrations 开放接口查询指定分组的路由别名和上游模型。
- `upstream_models` 按候选优先级返回。

## 1. 启用 integrations 接口

查询接口复用 `external_api_key_provisioning` 配置和固定 Bearer Token 鉴权，不使用管理员 JWT。

开发环境配置文件：`backend/config/config.yaml`

部署示例配置文件：`deploy/config.example.yaml`

```yaml
external_api_key_provisioning:
  enabled: true
  access_token: "请替换为至少 32 字节且不含空白字符的随机令牌"
  rate_limit_biz_per_minute: 60
  rate_limit_auth_per_minute: 10
```

注意：

- 不要把生产 Token 写入仓库。
- `enabled` 为 `false`，或 `access_token` 为空时，接口返回 `404 NOT_FOUND`，从外部隐藏接口。
- 请求必须使用 `Authorization: Bearer <Token>`，不支持查询参数或请求体传递 Token。
- 请求体必须使用 `Content-Type: application/json`。

## 2. 管理端配置模型路由

在分组的模型路由编辑器中：

1. 启用模型路由。
2. 填写路由别名。
3. 搜索并选择一个或多个账号。
4. 在“上游模型”下拉框中选择模型。
5. 设置候选优先级和每日 Token 限额。
6. 保存分组。

模型下拉框遵循以下规则：

- 未选择账号时不可用。
- 只选择一个账号时，显示该账号支持的模型。
- 选择多个账号时，只显示所有账号共同支持的模型交集。
- 模型以 ID 判断是否相同，展示文字优先使用 `display_name`。
- 模型加载失败时可以重试，加载失败或没有共同模型时不能保存无效候选。
- 历史模型仍被账号支持时会保留；已失效时必须重新选择。
- 空值只作为“请选择上游模型”的占位状态，不会作为可选模型出现。

## 3. 查询分组模型路由

### 3.1 请求

```http
POST /api/v1/integrations/model-routes/list
Authorization: Bearer <固定访问令牌>
Content-Type: application/json
```

请求体：

```json
{
  "group_name": "public"
}
```

`group_name` 会去除首尾空白，并按完整分组名精确查询。

### 3.2 cURL 示例

```bash
curl -X POST "http://localhost:8080/api/v1/integrations/model-routes/list" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"group_name":"public"}'
```

### 3.3 成功响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "group_name": "public",
    "routes": [
      {
        "route_alias": "coding-fast",
        "upstream_models": [
          "claude-sonnet-4",
          "claude-opus-4",
          "claude-haiku-3-5"
        ]
      }
    ]
  }
}
```

响应不会返回账号 ID、候选优先级、每日限额、API Key 或 Token。

### 3.4 返回顺序

- `routes` 当前按 `route_alias` 字典序稳定返回。
- 每个路由的 `upstream_models` 按候选 `priority` 从小到大返回，数值越小越靠前。
- 相同 `priority` 的候选保持配置中的原始顺序。
- 同一个模型配置多次时只返回一次，并保留其最高优先级位置。
- 查询不依赖 `model_routing_enabled` 是否启用；只要分组保存了路由配置，就会返回只读投影。

例如候选配置顺序为：

| 模型 | priority |
|---|---:|
| `model-z` | 0 |
| `model-m` | 10 |
| `model-a` | 20 |

接口返回：

```json
{
  "route_alias": "coding-fast",
  "upstream_models": ["model-z", "model-m", "model-a"]
}
```

## 4. 空路由响应

分组存在但没有模型路由配置时，返回非 `null` 空数组：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "group_name": "public",
    "routes": []
  }
}
```

## 5. 常见错误

### 5.1 请求参数无效

缺少 `group_name`、内容为空白或 JSON 非法时：

```http
HTTP/1.1 400 Bad Request
```

```json
{
  "code": 400,
  "message": "INVALID_REQUEST"
}
```

### 5.2 Token 缺失或错误

```http
HTTP/1.1 401 Unauthorized
```

```json
{
  "code": 401,
  "message": "INVALID_ACCESS_TOKEN"
}
```

### 5.3 接口未启用

```http
HTTP/1.1 404 Not Found
```

```json
{
  "code": 404,
  "message": "NOT_FOUND"
}
```

### 5.4 分组不存在

```http
HTTP/1.1 404 Not Found
```

```json
{
  "code": 404,
  "message": "GROUP_NOT_FOUND"
}
```

### 5.5 服务内部错误

```http
HTTP/1.1 500 Internal Server Error
```

```json
{
  "code": 500,
  "message": "failed to list group model routes"
}
```

内部错误响应不会泄漏数据库错误或路由配置细节。

## 6. JavaScript 调用示例

```js
async function listGroupModelRoutes(baseURL, accessToken, groupName) {
  const response = await fetch(`${baseURL}/api/v1/integrations/model-routes/list`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${accessToken}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ group_name: groupName })
  })

  const result = await response.json()
  if (!response.ok) {
    throw new Error(result.message || `HTTP ${response.status}`)
  }
  return result.data
}
```
