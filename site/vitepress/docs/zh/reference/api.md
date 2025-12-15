# REST API 参考

vStats 提供 RESTful API 用于程序化交互。

## 基础 URL

```
http://your-server:3001/api
```

## 认证

大多数端点需要通过 JWT token 认证。在 `Authorization` 头中包含 token：

```http
Authorization: Bearer <your-jwt-token>
```

通过 `/api/auth/login` 端点登录获取 token。

## 公开端点

这些端点不需要认证：

### 健康检查

```http
GET /health
```

**响应:**
```json
{
  "status": "ok"
}
```

### 获取当前指标

```http
GET /api/metrics
```

返回所有已连接服务器的最新指标。

**响应:**
```json
{
  "servers": [
    {
      "id": "abc123",
      "name": "my-server",
      "metrics": {
        "cpu": { "usage": 45.2 },
        "memory": { "usage_percent": 67.8 },
        ...
      },
      "online": true,
      "last_seen": "2024-01-01T12:00:00Z"
    }
  ]
}
```

### 获取服务器列表

```http
GET /api/servers
```

返回所有注册服务器的列表。

**响应:**
```json
{
  "servers": [
    {
      "id": "abc123",
      "name": "my-server",
      "location": "上海",
      "provider": "阿里云",
      "online": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### 登录

```http
POST /api/auth/login
```

**请求:**
```json
{
  "username": "admin",
  "password": "your-password"
}
```

**响应:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-02T00:00:00Z"
}
```

### 验证 Token

```http
GET /api/auth/verify
```

**请求头:**
```http
Authorization: Bearer <token>
```

**响应:**
```json
{
  "valid": true,
  "user": "admin",
  "expires_at": "2024-01-02T00:00:00Z"
}
```

## 需认证端点

这些端点需要有效的 JWT token。

### 添加服务器

```http
POST /api/servers
```

**请求:**
```json
{
  "name": "new-server",
  "location": "北京",
  "provider": "腾讯云"
}
```

**响应:**
```json
{
  "id": "xyz789",
  "name": "new-server",
  "token": "server-connection-token",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### 更新服务器

```http
PUT /api/servers/{id}
```

**请求:**
```json
{
  "name": "updated-name",
  "location": "深圳",
  "provider": "华为云"
}
```

**响应:**
```json
{
  "id": "xyz789",
  "name": "updated-name",
  "location": "深圳",
  "provider": "华为云",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

### 删除服务器

```http
DELETE /api/servers/{id}
```

**响应:**
```json
{
  "success": true,
  "message": "Server deleted"
}
```

### 修改密码

```http
POST /api/auth/password
```

**请求:**
```json
{
  "current_password": "旧密码",
  "new_password": "新密码"
}
```

**响应:**
```json
{
  "success": true,
  "message": "Password changed"
}
```

## 错误响应

所有错误遵循此格式：

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "人类可读的消息"
  }
}
```

### 错误代码

| 代码 | HTTP 状态 | 说明 |
|------|-----------|------|
| `UNAUTHORIZED` | 401 | 缺少或无效的 token |
| `FORBIDDEN` | 403 | Token 有效但权限不足 |
| `NOT_FOUND` | 404 | 资源未找到 |
| `VALIDATION_ERROR` | 400 | 无效的请求数据 |
| `INTERNAL_ERROR` | 500 | 服务器错误 |

## 速率限制

API 实现了速率限制：

- **公开端点**: 100 请求/分钟
- **需认证端点**: 1000 请求/分钟

响应中包含速率限制头：

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067200
```

## 示例

### cURL

```bash
# 登录
curl -X POST http://localhost:3001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# 使用 token 获取服务器
curl http://localhost:3001/api/servers \
  -H "Authorization: Bearer eyJhbG..."
```

### JavaScript

```javascript
// 登录获取 token
const response = await fetch('http://localhost:3001/api/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: 'password' })
});
const { token } = await response.json();

// 使用 token 进行认证请求
const servers = await fetch('http://localhost:3001/api/servers', {
  headers: { 'Authorization': `Bearer ${token}` }
}).then(r => r.json());
```

### Python

```python
import requests

# 登录
response = requests.post('http://localhost:3001/api/auth/login', json={
    'username': 'admin',
    'password': 'password'
})
token = response.json()['token']

# 获取服务器
servers = requests.get('http://localhost:3001/api/servers', headers={
    'Authorization': f'Bearer {token}'
}).json()
```

