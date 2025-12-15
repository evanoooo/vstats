# 服务器配置

本文档介绍 vStats 服务端的配置选项。

## 配置文件

服务端配置文件位置：

| 安装方式 | 路径 |
|----------|------|
| Docker | `/app/vstats-config.json` |
| 脚本安装 (Linux) | `/opt/vstats/vstats-config.json` |
| 脚本安装 (macOS) | `~/.vstats/vstats-config.json` |
| 手动安装 | 当前工作目录 |

## 配置选项

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

### 核心选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `port` | int | 3001 | HTTP 服务端口 |
| `host` | string | "0.0.0.0" | 监听地址 |
| `data_dir` | string | "./data" | 数据库和文件目录 |
| `log_level` | string | "info" | 日志级别 (debug, info, warn, error) |
| `jwt_secret` | string | 自动 | JWT 签名密钥（未设置时自动生成） |
| `cors_origins` | array | ["*"] | 允许的 CORS 来源 |

### SSL 选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `ssl.enabled` | bool | false | 启用 HTTPS |
| `ssl.cert` | string | "" | SSL 证书路径 |
| `ssl.key` | string | "" | SSL 私钥路径 |

::: tip 提示
生产环境建议使用反向代理（Nginx、Caddy）进行 SSL 终止，而不是直接配置 SSL。
:::

### 指标选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `metrics.retention_days` | int | 30 | 历史指标保留天数 |
| `metrics.aggregation_enabled` | bool | true | 启用指标聚合 |

### 认证选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `auth.session_duration` | string | "24h" | JWT Token 有效期 |
| `auth.max_login_attempts` | int | 5 | 最大登录失败次数 |
| `auth.lockout_duration` | string | "15m" | 账户锁定时间 |

## 配置示例

### 生产环境配置

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

### 开发环境配置

```json
{
  "port": 3001,
  "host": "0.0.0.0",
  "data_dir": "./data",
  "log_level": "debug",
  "cors_origins": ["*"]
}
```

## 应用配置

修改配置文件后，重启服务：

```bash
# Docker
docker restart vstats-server

# Systemd (Linux)
sudo systemctl restart vstats

# macOS
launchctl stop io.vstats.server
launchctl start io.vstats.server
```

## 配置验证

服务端在启动时验证配置。检查日志中的错误：

```bash
# Docker
docker logs vstats-server

# Systemd
journalctl -u vstats -n 50
```

