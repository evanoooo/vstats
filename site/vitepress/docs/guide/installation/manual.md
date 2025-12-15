# Manual Installation

For advanced users who want full control over the installation process.

## Download Binary

Download the latest release from [GitHub Releases](https://github.com/zsai001/vstats/releases).

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

Download and extract the Windows binary from the releases page.

## Directory Setup

```bash
# Create directories
sudo mkdir -p /opt/vstats/data

# Move binary
sudo mv vstats-server /opt/vstats/

# Set permissions
sudo chmod +x /opt/vstats/vstats-server
sudo chown -R vstats:vstats /opt/vstats
```

## Create User (Recommended)

```bash
sudo useradd -r -s /bin/false vstats
sudo chown -R vstats:vstats /opt/vstats
```

## Configuration

Create a configuration file:

```bash
sudo tee /opt/vstats/vstats-config.json << 'EOF'
{
  "port": 3001,
  "data_dir": "/opt/vstats/data",
  "log_level": "info"
}
EOF
```

## Systemd Service

Create a systemd service file:

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

# Security hardening
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

## Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable and start
sudo systemctl enable vstats
sudo systemctl start vstats

# Check status
sudo systemctl status vstats
```

## Get Admin Password

```bash
journalctl -u vstats | grep -i password
```

## Build from Source

If you want to build from source:

### Prerequisites

- Go 1.21+
- Node.js 18+
- Make

### Build

```bash
# Clone repository
git clone https://github.com/zsai001/vstats.git
cd vstats

# Build frontend
cd web
npm install
npm run build
cd ..

# Build backend
cd server-go
go build -o vstats-server ./cmd/server
```

### Development Mode

```bash
# Terminal 1: Backend
cd server-go
go run ./cmd/server

# Terminal 2: Frontend
cd web
npm run dev
```

## Firewall Configuration

If you have a firewall enabled:

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

If SELinux is enabled:

```bash
# Allow network access
sudo setsebool -P httpd_can_network_connect 1

# Or create a custom policy
sudo semanage port -a -t http_port_t -p tcp 3001
```

