# 卸载

本指南介绍从系统中完全移除 vStats。

## 卸载前准备

1. **导出需要保留的数据**
2. **备份配置**以备日后重新安装
3. **先从被监控服务器移除 Agent**

## Docker 卸载

### 停止并删除容器

```bash
# 停止容器
docker stop vstats-server

# 删除容器
docker rm vstats-server
```

### 删除数据（可选）

```bash
# 删除数据目录
rm -rf ./data

# 或先备份
mv ./data ./data.backup
```

### 删除镜像

```bash
# 删除镜像
docker rmi zsai001/vstats-server:latest

# 删除所有相关镜像
docker rmi $(docker images zsai001/vstats-* -q)
```

### Docker Compose

```bash
# 停止并删除容器
docker compose down

# 同时删除卷
docker compose down -v

# 删除镜像
docker compose down --rmi all
```

## 脚本卸载

### Linux/macOS

```bash
# 卸载服务端
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall

# 同时删除数据
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall --purge
```

卸载脚本将：
1. 停止服务
2. 删除 systemd 服务文件
3. 删除二进制文件和安装目录
4. 可选删除数据（使用 `--purge`）

## 手动卸载

### 停止服务

```bash
# Linux
sudo systemctl stop vstats
sudo systemctl disable vstats

# macOS
launchctl unload ~/Library/LaunchAgents/io.vstats.server.plist
```

### 删除服务文件

```bash
# Linux
sudo rm /etc/systemd/system/vstats.service
sudo systemctl daemon-reload

# macOS
rm ~/Library/LaunchAgents/io.vstats.server.plist
```

### 删除文件

```bash
# Linux
sudo rm -rf /opt/vstats

# macOS
rm -rf ~/.vstats
```

### 删除用户（Linux）

```bash
sudo userdel vstats
sudo groupdel vstats
```

## Agent 卸载

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --uninstall
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-uninstall.ps1 -OutFile agent-uninstall.ps1
.\agent-uninstall.ps1
```

### 手动 Agent 卸载

```bash
# Linux
sudo systemctl stop vstats-agent
sudo systemctl disable vstats-agent
sudo rm /etc/systemd/system/vstats-agent.service
sudo rm -rf /opt/vstats-agent
sudo systemctl daemon-reload
```

## 清理

### 删除配置

```bash
# 删除残留的配置文件
rm -f /etc/vstats/*
rm -rf /etc/vstats
```

### 删除日志

```bash
# 清除日志（Linux）
sudo journalctl --vacuum-time=0 -u vstats
sudo journalctl --vacuum-time=0 -u vstats-agent
```

### 删除备份

```bash
# 如果有自动备份
rm -rf /opt/vstats/backups
```

### 删除 Cron 任务

```bash
# 编辑 crontab
crontab -e

# 删除 vstats 相关行
```

## 验证移除

```bash
# 检查服务是否还存在
systemctl status vstats

# 检查是否有残留文件
ls -la /opt/vstats
ls -la ~/.vstats

# 检查是否有进程运行
ps aux | grep vstats

# 检查端口是否释放
lsof -i :3001
```

## 重新安装

如果卸载后想重新安装：

1. 确保所有文件已删除
2. 重新运行安装脚本
3. 将生成新的管理员密码
4. 在被监控服务器上重新安装 Agent

