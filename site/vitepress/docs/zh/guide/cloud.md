---
title: vStats Cloud（云端版）
---

# vStats Cloud（云端版）

`vStats Cloud` 是 vStats 的云端后端服务（SaaS），提供多用户服务器管理、实时指标、以及 API/WebSocket 能力。

## 功能特性

- **多用户支持**：OAuth 登录（GitHub / Google）
- **服务器管理**：每个用户管理自己的服务器与 Agent Key
- **实时监控**：WebSocket 实时推送指标
- **API 服务**：服务器/指标/历史数据 REST API
- **数据存储**：PostgreSQL + Redis

## 环境变量

```bash
# 服务配置
PORT=3001
APP_ENV=production
APP_URL=https://vstats.example.com
LOG_LEVEL=info

# 数据库
DATABASE_URL=postgres://user:pass@localhost:5432/vstats_cloud?sslmode=disable

# Redis
REDIS_URL=redis://:password@localhost:6379/0

# 认证
JWT_SECRET=your-jwt-secret
SESSION_SECRET=your-session-secret

# OAuth
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=xxx

# 其他
CORS_ORIGINS=https://vstats.example.com
METRICS_RETENTION_DAYS=30
```

## 本地开发

```bash
# 启动 PostgreSQL + Redis
docker compose -f site/deploy/docker-compose.yml up -d postgres redis

# 运行 Cloud 服务
cd server-go
go run ./cmd/cloud
```

## 构建

```bash
cd server-go
go build -o vstats-cloud ./cmd/cloud
```

## 部署

建议直接参考 `site/deploy/` 里的 Docker Compose 配置进行部署。

## 更多

- Cloud 完整说明：`server-go/cmd/cloud/README.md`
- Releases：`https://github.com/zsai001/vstats/releases`


