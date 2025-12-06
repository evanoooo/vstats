# vStats Agent (Go Implementation)

这是 vStats 代理的 Go 语言实现。

## 构建

```bash
cd agent-go
go mod tidy
go build -o vstats-agent
```

## 使用

### 注册代理

```bash
./vstats-agent register --server http://dashboard:3001 --token <admin_token> [--name <server_name>]
```

### 运行代理

```bash
./vstats-agent run
# 或
./vstats-agent run --config /path/to/config.json
```

### 安装为服务

```bash
sudo ./vstats-agent install
```

### 卸载服务

```bash
sudo ./vstats-agent uninstall
```

### 显示配置

```bash
./vstats-agent show-config
```

## 🐳 Docker 部署

### 方式一：使用配置文件

```bash
# 创建配置目录和文件
mkdir -p /opt/vstats-agent
cat > /opt/vstats-agent/config.json << EOF
{
  "dashboard_url": "http://YOUR_DASHBOARD_IP:3001",
  "server_id": "YOUR_SERVER_ID",
  "agent_token": "YOUR_AGENT_TOKEN",
  "server_name": "my-server",
  "location": "US",
  "provider": "Docker",
  "interval_secs": 5
}
EOF

# 运行容器
docker run -d \
  --name vstats-agent \
  --restart unless-stopped \
  --net host \
  --pid host \
  -v /opt/vstats-agent:/opt/vstats-agent:ro \
  -v /:/host:ro \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  zsai001/vstats-agent:latest
```

### 方式二：使用环境变量

```bash
docker run -d \
  --name vstats-agent \
  --restart unless-stopped \
  --net host \
  --pid host \
  -e VSTATS_DASHBOARD_URL="http://YOUR_DASHBOARD_IP:3001" \
  -e VSTATS_SERVER_ID="YOUR_SERVER_ID" \
  -e VSTATS_AGENT_TOKEN="YOUR_AGENT_TOKEN" \
  -e VSTATS_SERVER_NAME="my-server" \
  -e VSTATS_LOCATION="US" \
  -e VSTATS_PROVIDER="Docker" \
  -e VSTATS_INTERVAL_SECS="5" \
  -v /:/host:ro \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  zsai001/vstats-agent:latest
```

### 方式三：使用 Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'
services:
  vstats-agent:
    image: zsai001/vstats-agent:latest
    container_name: vstats-agent
    restart: unless-stopped
    network_mode: host
    pid: host
    environment:
      - VSTATS_DASHBOARD_URL=http://YOUR_DASHBOARD_IP:3001
      - VSTATS_SERVER_ID=YOUR_SERVER_ID
      - VSTATS_AGENT_TOKEN=YOUR_AGENT_TOKEN
      - VSTATS_SERVER_NAME=my-server
      - VSTATS_LOCATION=US
      - VSTATS_PROVIDER=Docker
    volumes:
      - /:/host:ro
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
```

运行: `docker-compose up -d`

### 环境变量说明

| 变量 | 必填 | 说明 |
|------|------|------|
| `VSTATS_DASHBOARD_URL` | ✅ | Dashboard 服务器地址 |
| `VSTATS_SERVER_ID` | ✅ | 服务器 ID (从 Dashboard 获取) |
| `VSTATS_AGENT_TOKEN` | ✅ | Agent Token (从 Dashboard 获取) |
| `VSTATS_SERVER_NAME` | ❌ | 服务器显示名称 |
| `VSTATS_LOCATION` | ❌ | 服务器位置 |
| `VSTATS_PROVIDER` | ❌ | 服务器提供商 |
| `VSTATS_INTERVAL_SECS` | ❌ | 上报间隔(秒)，默认 5 |
| `VSTATS_CONFIG_PATH` | ❌ | 配置文件路径 |

> **注意**: 使用 `--net host` 和 `--pid host` 可以让容器获取宿主机的真实网络和进程信息。

## 配置文件

配置文件位置：`vstats-agent.json`

默认位置：
- Linux/macOS: `/etc/vstats-agent/vstats-agent.json` 或 `~/.config/vstats-agent/vstats-agent.json`
- Windows: `%PROGRAMDATA%\vstats-agent\vstats-agent.json` 或 `%APPDATA%\vstats-agent\vstats-agent.json`
- Docker: `/opt/vstats-agent/config.json`

## 功能

- 自动收集系统指标（CPU、内存、磁盘、网络）
- 通过 WebSocket 实时推送指标到服务器
- 支持自定义 ping 目标
- 自动重连
- 支持系统服务安装（systemd/launchd/Windows Service）
- 支持 Docker 部署
- 支持环境变量配置

