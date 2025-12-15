---
title: vStats Cloud
---

# vStats Cloud

`vStats Cloud` is the SaaS backend for vStats. It provides multi-user server management, real-time metrics, and API/WebSocket access.

## Features

- **Multi-user**: OAuth login (GitHub / Google)
- **Server management**: per-user server list and agent keys
- **Real-time monitoring**: WebSocket metric push
- **API service**: REST API for servers, metrics, history
- **Storage**: PostgreSQL + Redis

## Environment Variables

```bash
# Service
PORT=3001
APP_ENV=production
APP_URL=https://vstats.example.com
LOG_LEVEL=info

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/vstats_cloud?sslmode=disable

# Redis
REDIS_URL=redis://:password@localhost:6379/0

# Auth
JWT_SECRET=your-jwt-secret
SESSION_SECRET=your-session-secret

# OAuth
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=xxx

# Other
CORS_ORIGINS=https://vstats.example.com
METRICS_RETENTION_DAYS=30
```

## Local Development

```bash
# Start PostgreSQL and Redis
docker compose -f site/deploy/docker-compose.yml up -d postgres redis

# Run cloud server
cd server-go
go run ./cmd/cloud
```

## Build

```bash
cd server-go
go build -o vstats-cloud ./cmd/cloud
```

## Deployment

See `site/deploy/` for Docker Compose based deployment.

## More

- Cloud README: `server-go/cmd/cloud/README.md`
- Releases: `https://github.com/zsai001/vstats/releases`


