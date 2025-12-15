# Docker Installation

Docker is the recommended way to deploy vStats. It provides isolation, easy updates, and consistent environments.

## Prerequisites

- Docker installed on your server
- At least 512MB RAM
- Port 3001 available (or configure a different port)

## Quick Start

```bash
# Create data directory
mkdir -p data && sudo chown -R 1000:1000 data

# Run container
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

## Docker Compose

For production deployments, we recommend using Docker Compose:

```yaml
# docker-compose.yml
version: '3.8'

services:
  vstats:
    image: zsai001/vstats-server:latest
    container_name: vstats-server
    ports:
      - "3001:3001"
    volumes:
      - ./data:/app/data
    environment:
      - TZ=UTC
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:3001/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Start with:

```bash
docker compose up -d
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3001` | HTTP server port |
| `TZ` | `UTC` | Timezone |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

Example with environment variables:

```bash
docker run -d \
  --name vstats-server \
  -p 8080:8080 \
  -e PORT=8080 \
  -e TZ=Asia/Shanghai \
  -e LOG_LEVEL=debug \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

## Volume Mounts

| Path | Purpose |
|------|---------|
| `/app/data` | Database and configuration files |

::: warning Important
Always mount `/app/data` to persist your data across container restarts.
:::

## Get Admin Password

After first startup:

```bash
docker logs vstats-server 2>&1 | grep -i password
```

## Reset Password

If you forget your password:

```bash
docker exec -it vstats-server /app/vstats-server --reset-password
```

## View Logs

```bash
# Follow logs
docker logs -f vstats-server

# Last 100 lines
docker logs --tail 100 vstats-server
```

## Update

```bash
# Pull latest image
docker pull zsai001/vstats-server:latest

# Stop and remove old container
docker stop vstats-server
docker rm vstats-server

# Start with new image
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

Or with Docker Compose:

```bash
docker compose pull
docker compose up -d
```

## Available Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `x.y.z` | Specific version |
| `dev` | Development build (unstable) |

## Resource Limits

For production, consider setting resource limits:

```yaml
services:
  vstats:
    image: zsai001/vstats-server:latest
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 128M
```

## Troubleshooting

### Container won't start

```bash
# Check logs for errors
docker logs vstats-server

# Check container status
docker ps -a | grep vstats
```

### Permission issues

```bash
# Fix data directory ownership
sudo chown -R 1000:1000 data
```

### Port already in use

```bash
# Use a different port
docker run -d -p 8080:3001 ...
```

