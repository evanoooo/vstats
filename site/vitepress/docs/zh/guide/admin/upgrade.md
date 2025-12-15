# 升级

本指南介绍将 vStats 升级到最新版本。

## 升级前准备

1. **查看发布说明**了解破坏性变更
2. **备份数据**（见[备份与恢复](/zh/guide/admin/backup)）
3. 如果可能，**在测试环境测试**

## Docker 升级

### 拉取并重启

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

### Docker Compose

```bash
# 拉取最新镜像
docker compose pull

# 重启服务
docker compose up -d
```

### 指定版本

```bash
# 使用特定版本而不是 latest
docker pull zsai001/vstats-server:1.2.0

docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  zsai001/vstats-server:1.2.0
```

## 脚本升级

### Linux/macOS

```bash
# 一命令升级
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

或使用 wget：

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

脚本将：
1. 下载最新二进制文件
2. 停止服务
3. 替换二进制文件
4. 启动服务
5. 验证升级

## 手动升级

### 下载新二进制文件

```bash
# 检查当前版本
/opt/vstats/vstats-server --version

# 下载最新版本
wget https://github.com/zsai001/vstats/releases/latest/download/vstats-server-linux-amd64.tar.gz

# 解压
tar -xzf vstats-server-linux-amd64.tar.gz
```

### 替换二进制文件

```bash
# 停止服务
sudo systemctl stop vstats

# 备份旧二进制文件
sudo mv /opt/vstats/vstats-server /opt/vstats/vstats-server.bak

# 安装新二进制文件
sudo mv vstats-server /opt/vstats/
sudo chmod +x /opt/vstats/vstats-server
sudo chown vstats:vstats /opt/vstats/vstats-server

# 启动服务
sudo systemctl start vstats
```

### 如需回滚

```bash
# 停止服务
sudo systemctl stop vstats

# 恢复旧二进制文件
sudo mv /opt/vstats/vstats-server.bak /opt/vstats/vstats-server

# 启动服务
sudo systemctl start vstats
```

## Agent 升级

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --upgrade
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-upgrade.ps1 -OutFile agent-upgrade.ps1
.\agent-upgrade.ps1
```

## 验证升级

### 检查版本

```bash
# 服务端版本
curl http://localhost:3001/health

# 或直接
/opt/vstats/vstats-server --version
```

### 检查服务

```bash
# 状态
systemctl status vstats

# 检查日志是否有错误
journalctl -u vstats -n 50
```

### 验证面板

1. 在浏览器中打开面板
2. 检查所有功能是否正常
3. 验证 Agent 是否已连接

