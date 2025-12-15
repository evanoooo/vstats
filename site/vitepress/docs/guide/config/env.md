# Environment Variables

vStats server can be configured using environment variables. These take precedence over the configuration file.

## Available Variables

| Variable | Config Key | Default | Description |
|----------|------------|---------|-------------|
| `PORT` | `port` | 3001 | Server port |
| `HOST` | `host` | 0.0.0.0 | Listen address |
| `DATA_DIR` | `data_dir` | ./data | Data directory |
| `LOG_LEVEL` | `log_level` | info | Log level |
| `JWT_SECRET` | `jwt_secret` | auto | JWT signing secret |
| `CORS_ORIGINS` | `cors_origins` | * | CORS origins (comma-separated) |
| `TZ` | - | UTC | Timezone |

## Usage

### Docker

```bash
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -e PORT=3001 \
  -e LOG_LEVEL=debug \
  -e TZ=Asia/Shanghai \
  -e JWT_SECRET=your-secret \
  -v $(pwd)/data:/app/data \
  zsai001/vstats-server:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  vstats:
    image: zsai001/vstats-server:latest
    ports:
      - "3001:3001"
    environment:
      - PORT=3001
      - LOG_LEVEL=info
      - TZ=UTC
      - JWT_SECRET=${JWT_SECRET}
      - CORS_ORIGINS=https://vstats.example.com
    volumes:
      - ./data:/app/data
```

### Systemd Service

Edit the service file to add environment variables:

```ini
[Service]
Environment="PORT=3001"
Environment="LOG_LEVEL=info"
Environment="TZ=UTC"
```

Or use an environment file:

```ini
[Service]
EnvironmentFile=/etc/vstats/env
```

`/etc/vstats/env`:
```
PORT=3001
LOG_LEVEL=info
TZ=UTC
JWT_SECRET=your-secret
```

### Shell Export

```bash
export PORT=3001
export LOG_LEVEL=debug
export JWT_SECRET=your-secret
./vstats-server
```

## Priority Order

Configuration is loaded in this order (later overrides earlier):

1. Default values
2. Configuration file (`vstats-config.json`)
3. Environment variables
4. Command-line flags (if any)

## Security Considerations

::: warning
Never commit sensitive environment variables (like `JWT_SECRET`) to version control.
:::

For production, use one of these approaches:

1. **Secret Management**: Use tools like HashiCorp Vault, AWS Secrets Manager
2. **Environment Files**: Use `.env` files that are gitignored
3. **Docker Secrets**: Use Docker Swarm secrets for containerized deployments

