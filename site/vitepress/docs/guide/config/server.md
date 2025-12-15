# Server Configuration

This document covers the configuration options for the vStats server.

## Configuration File

The server configuration file is located at:

| Installation | Path |
|--------------|------|
| Docker | `/app/vstats-config.json` |
| Script (Linux) | `/opt/vstats/vstats-config.json` |
| Script (macOS) | `~/.vstats/vstats-config.json` |
| Manual | Current working directory |

## Configuration Options

```json
{
  "port": 3001,
  "host": "0.0.0.0",
  "data_dir": "./data",
  "log_level": "info",
  "jwt_secret": "auto-generated",
  "cors_origins": ["*"],
  "ssl": {
    "enabled": false,
    "cert": "",
    "key": ""
  },
  "metrics": {
    "retention_days": 30,
    "aggregation_enabled": true
  },
  "auth": {
    "session_duration": "24h",
    "max_login_attempts": 5,
    "lockout_duration": "15m"
  }
}
```

### Core Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `port` | int | 3001 | HTTP server port |
| `host` | string | "0.0.0.0" | Listen address |
| `data_dir` | string | "./data" | Directory for database and files |
| `log_level` | string | "info" | Log level (debug, info, warn, error) |
| `jwt_secret` | string | auto | JWT signing secret (auto-generated if not set) |
| `cors_origins` | array | ["*"] | Allowed CORS origins |

### SSL Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ssl.enabled` | bool | false | Enable HTTPS |
| `ssl.cert` | string | "" | Path to SSL certificate |
| `ssl.key` | string | "" | Path to SSL private key |

::: tip
For production, we recommend using a reverse proxy (Nginx, Caddy) for SSL termination instead of configuring SSL directly.
:::

### Metrics Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `metrics.retention_days` | int | 30 | Days to keep historical metrics |
| `metrics.aggregation_enabled` | bool | true | Enable metric aggregation |

### Auth Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auth.session_duration` | string | "24h" | JWT token validity duration |
| `auth.max_login_attempts` | int | 5 | Max failed login attempts |
| `auth.lockout_duration` | string | "15m" | Account lockout duration |

## Example Configurations

### Production Configuration

```json
{
  "port": 3001,
  "host": "127.0.0.1",
  "data_dir": "/opt/vstats/data",
  "log_level": "warn",
  "jwt_secret": "your-secure-secret-here",
  "cors_origins": ["https://vstats.example.com"],
  "metrics": {
    "retention_days": 90,
    "aggregation_enabled": true
  },
  "auth": {
    "session_duration": "8h",
    "max_login_attempts": 3,
    "lockout_duration": "30m"
  }
}
```

### Development Configuration

```json
{
  "port": 3001,
  "host": "0.0.0.0",
  "data_dir": "./data",
  "log_level": "debug",
  "cors_origins": ["*"]
}
```

## Applying Configuration

After modifying the configuration file, restart the server:

```bash
# Docker
docker restart vstats-server

# Systemd (Linux)
sudo systemctl restart vstats

# macOS
launchctl stop io.vstats.server
launchctl start io.vstats.server
```

## Configuration Validation

The server validates configuration on startup. Check logs for any errors:

```bash
# Docker
docker logs vstats-server

# Systemd
journalctl -u vstats -n 50
```

