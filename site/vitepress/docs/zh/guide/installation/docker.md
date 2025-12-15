# Docker 安装

Docker 是部署 vStats 的推荐方式。它提供隔离性、便捷更新和一致的环境。

## 前提条件

- 服务器已安装 Docker
- 至少 512MB 内存
- 端口 3001 可用（或配置其他端口）

## 快速开始

```bash
# 创建数据目录
mkdir -p data && sudo chown -R 1000:1000 data

# 运行容器
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

## Docker Compose

生产环境部署推荐使用 Docker Compose：

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
      - TZ=Asia/Shanghai
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:3001/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

启动：

```bash
docker compose up -d
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `3001` | HTTP 服务端口 |
| `TZ` | `UTC` | 时区 |
| `LOG_LEVEL` | `info` | 日志级别 (debug, info, warn, error) |

使用环境变量的示例：

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

## 卷挂载

| 路径 | 用途 |
|------|------|
| `/app/data` | 数据库和配置文件 |

::: warning 重要
务必挂载 `/app/data` 以在容器重启后保留数据。
:::

## 获取管理员密码

首次启动后：

```bash
docker logs vstats-server 2>&1 | grep -i password
```

## 重置密码

如果忘记密码：

```bash
docker exec -it vstats-server /app/vstats-server --reset-password
```

## 查看日志

```bash
# 实时日志
docker logs -f vstats-server

# 最后 100 行
docker logs --tail 100 vstats-server
```

## 更新

```bash
# 拉取最新镜像
docker pull zsai001/vstats-server:latest

# 停止并删除旧容器
docker stop vstats-server
docker rm vstats-server

# 用新镜像启动
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

或使用 Docker Compose：

```bash
docker compose pull
docker compose up -d
```

## 可用标签

| 标签 | 说明 |
|------|------|
| `latest` | 最新稳定版 |
| `x.y.z` | 特定版本 |
| `dev` | 开发版（不稳定） |

## 资源限制

生产环境建议设置资源限制：

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

## 故障排除

### 容器无法启动

```bash
# 检查日志
docker logs vstats-server

# 检查容器状态
docker ps -a | grep vstats
```

### 权限问题

```bash
# 修复数据目录权限
sudo chown -R 1000:1000 data
```

### 端口已被占用

```bash
# 使用其他端口
docker run -d -p 8080:3001 ...
```

