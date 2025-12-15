# 手动安装

适用于需要完全控制安装过程的高级用户。

## 下载二进制文件

从 [GitHub Releases](https://github.com/zsai001/vstats/releases) 下载最新版本。

### Linux

```bash
# amd64
wget https://github.com/zsai001/vstats/releases/latest/download/vstats-server-linux-amd64.tar.gz
tar -xzf vstats-server-linux-amd64.tar.gz

# arm64
wget https://github.com/zsai001/vstats/releases/latest/download/vstats-server-linux-arm64.tar.gz
tar -xzf vstats-server-linux-arm64.tar.gz
```

### macOS

```bash
# Intel
wget https://github.com/zsai001/vstats/releases/latest/download/vstats-server-darwin-amd64.tar.gz
tar -xzf vstats-server-darwin-amd64.tar.gz

# Apple Silicon
wget https://github.com/zsai001/vstats/releases/latest/download/vstats-server-darwin-arm64.tar.gz
tar -xzf vstats-server-darwin-arm64.tar.gz
```

### Windows

从发布页面下载并解压 Windows 二进制文件。

## 目录设置

```bash
# 创建目录
sudo mkdir -p /opt/vstats/data

# 移动二进制文件
sudo mv vstats-server /opt/vstats/

# 设置权限
sudo chmod +x /opt/vstats/vstats-server
sudo chown -R vstats:vstats /opt/vstats
```

## 创建用户（推荐）

```bash
sudo useradd -r -s /bin/false vstats
sudo chown -R vstats:vstats /opt/vstats
```

## 配置

创建配置文件：

```bash
sudo tee /opt/vstats/vstats-config.json << 'EOF'
{
  "port": 3001,
  "data_dir": "/opt/vstats/data",
  "log_level": "info"
}
EOF
```

## Systemd 服务

创建 systemd 服务文件：

```bash
sudo tee /etc/systemd/system/vstats.service << 'EOF'
[Unit]
Description=vStats Server
After=network.target

[Service]
Type=simple
User=vstats
Group=vstats
WorkingDirectory=/opt/vstats
ExecStart=/opt/vstats/vstats-server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# 安全加固
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/vstats/data
PrivateTmp=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes

[Install]
WantedBy=multi-user.target
EOF
```

## 启动服务

```bash
# 重新加载 systemd
sudo systemctl daemon-reload

# 启用并启动
sudo systemctl enable vstats
sudo systemctl start vstats

# 检查状态
sudo systemctl status vstats
```

## 获取管理员密码

```bash
journalctl -u vstats | grep -i password
```

## 从源码构建

如果想从源码构建：

### 前提条件

- Go 1.21+
- Node.js 18+
- Make

### 构建

```bash
# 克隆仓库
git clone https://github.com/zsai001/vstats.git
cd vstats

# 构建前端
cd web
npm install
npm run build
cd ..

# 构建后端
cd server-go
go build -o vstats-server ./cmd/server
```

### 开发模式

```bash
# 终端 1：后端
cd server-go
go run ./cmd/server

# 终端 2：前端
cd web
npm run dev
```

## 防火墙配置

如果启用了防火墙：

```bash
# UFW (Ubuntu)
sudo ufw allow 3001/tcp

# firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=3001/tcp
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp --dport 3001 -j ACCEPT
```

## SELinux (RHEL/CentOS)

如果启用了 SELinux：

```bash
# 允许网络访问
sudo setsebool -P httpd_can_network_connect 1

# 或创建自定义策略
sudo semanage port -a -t http_port_t -p tcp 3001
```

