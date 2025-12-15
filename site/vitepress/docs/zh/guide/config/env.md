# 环境变量

vStats 服务端可以使用环境变量配置。环境变量优先级高于配置文件。

## 可用变量

| 变量 | 配置键 | 默认值 | 说明 |
|------|--------|--------|------|
| `PORT` | `port` | 3001 | 服务端口 |
| `HOST` | `host` | 0.0.0.0 | 监听地址 |
| `DATA_DIR` | `data_dir` | ./data | 数据目录 |
| `LOG_LEVEL` | `log_level` | info | 日志级别 |
| `JWT_SECRET` | `jwt_secret` | 自动 | JWT 签名密钥 |
| `CORS_ORIGINS` | `cors_origins` | * | CORS 来源（逗号分隔） |
| `TZ` | - | UTC | 时区 |

## 使用方式

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
      - TZ=Asia/Shanghai
      - JWT_SECRET=${JWT_SECRET}
      - CORS_ORIGINS=https://vstats.example.com
    volumes:
      - ./data:/app/data
```

### Systemd 服务

编辑服务文件添加环境变量：

```ini
[Service]
Environment="PORT=3001"
Environment="LOG_LEVEL=info"
Environment="TZ=Asia/Shanghai"
```

或使用环境文件：

```ini
[Service]
EnvironmentFile=/etc/vstats/env
```

`/etc/vstats/env`:
```
PORT=3001
LOG_LEVEL=info
TZ=Asia/Shanghai
JWT_SECRET=your-secret
```

### Shell 导出

```bash
export PORT=3001
export LOG_LEVEL=debug
export JWT_SECRET=your-secret
./vstats-server
```

## 优先级顺序

配置按以下顺序加载（后面覆盖前面）：

1. 默认值
2. 配置文件（`vstats-config.json`）
3. 环境变量
4. 命令行参数（如有）

## 安全注意事项

::: warning 警告
切勿将敏感环境变量（如 `JWT_SECRET`）提交到版本控制。
:::

生产环境使用以下方式之一：

1. **密钥管理**：使用 HashiCorp Vault、AWS Secrets Manager 等工具
2. **环境文件**：使用被 .gitignore 忽略的 `.env` 文件
3. **Docker Secrets**：容器化部署使用 Docker Swarm secrets

