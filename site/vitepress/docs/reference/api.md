# REST API Reference

vStats provides a RESTful API for interacting with the server programmatically.

## Base URL

```
http://your-server:3001/api
```

## Authentication

Most endpoints require authentication via JWT token. Include the token in the `Authorization` header:

```http
Authorization: Bearer <your-jwt-token>
```

Get a token by logging in via the `/api/auth/login` endpoint.

## Public Endpoints

These endpoints don't require authentication:

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "ok"
}
```

### Get Current Metrics

```http
GET /api/metrics
```

Returns the latest metrics for all connected servers.

**Response:**
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

### Get Server List

```http
GET /api/servers
```

Returns list of all registered servers.

**Response:**
```json
{
  "servers": [
    {
      "id": "abc123",
      "name": "my-server",
      "location": "US-West",
      "provider": "AWS",
      "online": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Login

```http
POST /api/auth/login
```

**Request:**
```json
{
  "username": "admin",
  "password": "your-password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-02T00:00:00Z"
}
```

### Verify Token

```http
GET /api/auth/verify
```

**Headers:**
```http
Authorization: Bearer <token>
```

**Response:**
```json
{
  "valid": true,
  "user": "admin",
  "expires_at": "2024-01-02T00:00:00Z"
}
```

## Authenticated Endpoints

These endpoints require a valid JWT token.

### Add Server

```http
POST /api/servers
```

**Request:**
```json
{
  "name": "new-server",
  "location": "US-East",
  "provider": "DigitalOcean"
}
```

**Response:**
```json
{
  "id": "xyz789",
  "name": "new-server",
  "token": "server-connection-token",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Update Server

```http
PUT /api/servers/{id}
```

**Request:**
```json
{
  "name": "updated-name",
  "location": "EU-West",
  "provider": "Hetzner"
}
```

**Response:**
```json
{
  "id": "xyz789",
  "name": "updated-name",
  "location": "EU-West",
  "provider": "Hetzner",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

### Delete Server

```http
DELETE /api/servers/{id}
```

**Response:**
```json
{
  "success": true,
  "message": "Server deleted"
}
```

### Get Server Details

```http
GET /api/servers/{id}
```

**Response:**
```json
{
  "id": "abc123",
  "name": "my-server",
  "location": "US-West",
  "provider": "AWS",
  "online": true,
  "metrics": { ... },
  "history": [ ... ],
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Change Password

```http
POST /api/auth/password
```

**Request:**
```json
{
  "current_password": "old-password",
  "new_password": "new-password"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Password changed"
}
```

### Get Settings

```http
GET /api/settings
```

**Response:**
```json
{
  "site_name": "My vStats",
  "theme": "default",
  "language": "en",
  "data_retention_days": 30
}
```

### Update Settings

```http
PUT /api/settings
```

**Request:**
```json
{
  "site_name": "Updated Name",
  "theme": "dark",
  "language": "zh"
}
```

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Missing or invalid token |
| `FORBIDDEN` | 403 | Token valid but lacks permission |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid request data |
| `INTERNAL_ERROR` | 500 | Server error |

## Rate Limiting

The API implements rate limiting:

- **Public endpoints**: 100 requests/minute
- **Authenticated endpoints**: 1000 requests/minute

Rate limit headers are included in responses:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067200
```

## Examples

### cURL

```bash
# Login
curl -X POST http://localhost:3001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# Get servers with token
curl http://localhost:3001/api/servers \
  -H "Authorization: Bearer eyJhbG..."
```

### JavaScript

```javascript
// Login and get token
const response = await fetch('http://localhost:3001/api/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: 'password' })
});
const { token } = await response.json();

// Use token for authenticated requests
const servers = await fetch('http://localhost:3001/api/servers', {
  headers: { 'Authorization': `Bearer ${token}` }
}).then(r => r.json());
```

### Python

```python
import requests

# Login
response = requests.post('http://localhost:3001/api/auth/login', json={
    'username': 'admin',
    'password': 'password'
})
token = response.json()['token']

# Get servers
servers = requests.get('http://localhost:3001/api/servers', headers={
    'Authorization': f'Bearer {token}'
}).json()
```

