# 服务管理

本指南介绍管理 vStats 服务端和 Agent 服务。

## Linux (Systemd)

### 服务端服务

```bash
# 查看状态
systemctl status vstats

# 启动服务
sudo systemctl start vstats

# 停止服务
sudo systemctl stop vstats

# 重启服务
sudo systemctl restart vstats

# 重新加载配置（无需重启）
sudo systemctl reload vstats

# 设置开机自启
sudo systemctl enable vstats

# 取消开机自启
sudo systemctl disable vstats
```

### Agent 服务

```bash
# 查看状态
systemctl status vstats-agent

# 启动服务
sudo systemctl start vstats-agent

# 停止服务
sudo systemctl stop vstats-agent

# 重启服务
sudo systemctl restart vstats-agent

# 设置开机自启
sudo systemctl enable vstats-agent

# 取消开机自启
sudo systemctl disable vstats-agent
```

### 查看日志

```bash
# 查看所有服务端日志
journalctl -u vstats

# 实时日志
journalctl -u vstats -f

# 最后 100 行
journalctl -u vstats -n 100

# 今天的日志
journalctl -u vstats --since today

# 指定时间范围的日志
journalctl -u vstats --since "2024-01-01 00:00:00" --until "2024-01-02 00:00:00"

# Agent 日志
journalctl -u vstats-agent -f
```

## macOS (launchd)

### 服务端服务

```bash
# 查看状态
launchctl list | grep vstats

# 启动服务
launchctl start io.vstats.server

# 停止服务
launchctl stop io.vstats.server

# 加载服务（启用）
launchctl load ~/Library/LaunchAgents/io.vstats.server.plist

# 卸载服务（禁用）
launchctl unload ~/Library/LaunchAgents/io.vstats.server.plist
```

### 查看日志

```bash
# 服务端日志
tail -f ~/.vstats/data/vstats.log

# 最后 100 行
tail -100 ~/.vstats/data/vstats.log
```

## Windows

### 服务端服务

```powershell
# 查看状态
Get-Service vstats

# 启动服务
Start-Service vstats

# 停止服务
Stop-Service vstats

# 重启服务
Restart-Service vstats

# 设置开机自启
Set-Service vstats -StartupType Automatic

# 取消开机自启
Set-Service vstats -StartupType Disabled
```

### Agent 服务

```powershell
# 查看状态
Get-Service vstats-agent

# 启动服务
Start-Service vstats-agent

# 停止服务
Stop-Service vstats-agent

# 重启服务
Restart-Service vstats-agent
```

### 查看日志

```powershell
# 查看事件日志
Get-EventLog -LogName Application -Source vstats -Newest 50

# 按时间筛选
Get-EventLog -LogName Application -Source vstats -After (Get-Date).AddHours(-24)
```

## Docker

### 容器管理

```bash
# 查看状态
docker ps | grep vstats

# 启动容器
docker start vstats-server

# 停止容器
docker stop vstats-server

# 重启容器
docker restart vstats-server

# 查看日志
docker logs vstats-server

# 实时日志
docker logs -f vstats-server

# 最后 100 行
docker logs --tail 100 vstats-server
```

### Docker Compose

```bash
# 状态
docker compose ps

# 启动
docker compose up -d

# 停止
docker compose down

# 重启
docker compose restart

# 日志
docker compose logs -f
```

## 健康检查

### 检查服务是否运行

```bash
# Linux
systemctl is-active vstats

# HTTP 健康检查
curl http://localhost:3001/health
```

预期响应：
```json
{"status":"ok"}
```

### 服务文件位置

| 操作系统 | 路径 |
|----------|------|
| Linux | `/etc/systemd/system/vstats.service` |
| macOS | `~/Library/LaunchAgents/io.vstats.server.plist` |
| Windows | 通过 `sc.exe` 管理 |

