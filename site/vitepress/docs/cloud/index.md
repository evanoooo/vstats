---
title: vStats Cloud
---

# vStats Cloud

`vStats Cloud` is the SaaS backend for vStats. It provides multi-user server management, real-time metrics, and API/WebSocket access.

## Quick Links

- **CLI**: `/cli/` (manage Cloud via terminal)
- **Cloud server source**: `server-go/cmd/cloud/README.md`

## Features

- **Multi-user**: OAuth login (GitHub / Google)
- **Server management**: per-user server list and agent keys
- **Real-time monitoring**: WebSocket metric push
- **API service**: REST API for servers, metrics, history
- **Storage**: PostgreSQL + Redis

## Local Development

```bash
docker compose -f site/deploy/docker-compose.yml up -d postgres redis
cd server-go
go run ./cmd/cloud
```


