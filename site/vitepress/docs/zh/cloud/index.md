---
title: vStats Cloud（云端版）
---

# vStats Cloud（云端版）

`vStats Cloud` 是 vStats 的云端后端服务（SaaS），提供多用户服务器管理、实时指标、以及 API/WebSocket 能力。

## 快速入口

- **CLI**：`/zh/cli/`（用命令行管理 Cloud）
- **Cloud 服务源码说明**：`server-go/cmd/cloud/README.md`

## 功能特性

- **多用户支持**：OAuth 登录（GitHub / Google）
- **服务器管理**：每个用户管理自己的服务器与 Agent Key
- **实时监控**：WebSocket 实时推送指标
- **API 服务**：服务器/指标/历史数据 REST API
- **数据存储**：PostgreSQL + Redis

## 本地开发

```bash
docker compose -f site/deploy/docker-compose.yml up -d postgres redis
cd server-go
go run ./cmd/cloud
```


