# Upgrade

This guide covers upgrading vStats to the latest version.

## Before Upgrading

1. **Check release notes** for breaking changes
2. **Backup your data** (see [Backup & Restore](/guide/admin/backup))
3. **Test in staging** if possible

## Docker Upgrade

### Pull and Restart

```bash
# Pull latest image
docker pull zsai001/vstats-server:latest

# Stop and remove old container
docker stop vstats-server
docker rm vstats-server

# Start with new image
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

### Docker Compose

```bash
# Pull latest image
docker compose pull

# Restart services
docker compose up -d
```

### Specific Version

```bash
# Use specific version instead of latest
docker pull zsai001/vstats-server:1.2.0

docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  zsai001/vstats-server:1.2.0
```

## Script Upgrade

### Linux/macOS

```bash
# One-command upgrade
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

Or with wget:

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

The script will:
1. Download the latest binary
2. Stop the service
3. Replace the binary
4. Start the service
5. Verify the upgrade

## Manual Upgrade

### Download New Binary

```bash
# Check current version
/opt/vstats/vstats-server --version

# Download latest release
wget https://github.com/zsai001/vstats/releases/latest/download/vstats-server-linux-amd64.tar.gz

# Extract
tar -xzf vstats-server-linux-amd64.tar.gz
```

### Replace Binary

```bash
# Stop service
sudo systemctl stop vstats

# Backup old binary
sudo mv /opt/vstats/vstats-server /opt/vstats/vstats-server.bak

# Install new binary
sudo mv vstats-server /opt/vstats/
sudo chmod +x /opt/vstats/vstats-server
sudo chown vstats:vstats /opt/vstats/vstats-server

# Start service
sudo systemctl start vstats
```

### Rollback if Needed

```bash
# Stop service
sudo systemctl stop vstats

# Restore old binary
sudo mv /opt/vstats/vstats-server.bak /opt/vstats/vstats-server

# Start service
sudo systemctl start vstats
```

## Agent Upgrade

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --upgrade
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-upgrade.ps1 -OutFile agent-upgrade.ps1
.\agent-upgrade.ps1
```

### Upgrade All Agents

If you have many agents, you can use SSH to upgrade them:

```bash
# List of servers
servers=("server1.example.com" "server2.example.com" "server3.example.com")

# Upgrade each
for server in "${servers[@]}"; do
    ssh root@$server "curl -fsSL https://vstats.zsoft.cc/agent.sh | bash -s -- --upgrade"
done
```

## Verify Upgrade

### Check Version

```bash
# Server version
curl http://localhost:3001/health

# Or directly
/opt/vstats/vstats-server --version
```

### Check Service

```bash
# Status
systemctl status vstats

# Logs for any errors
journalctl -u vstats -n 50
```

### Verify Dashboard

1. Open your browser to the dashboard
2. Check that all features work
3. Verify agents are connected

## Troubleshooting

### Service won't start after upgrade

```bash
# Check logs
journalctl -u vstats -n 100

# Common fixes:
# - Check config file for deprecated options
# - Verify file permissions
# - Check if port is in use
```

### Database migration errors

Upgrades may include database migrations. If they fail:

```bash
# Restore from backup
sudo systemctl stop vstats
cp /path/to/backup/vstats.db /opt/vstats/data/
sudo systemctl start vstats
```

### Rollback

If the upgrade causes issues:

```bash
# Docker
docker stop vstats-server
docker rm vstats-server
docker run -d --name vstats-server ... zsai001/vstats-server:previous-version

# Script install - restore backed up binary
sudo systemctl stop vstats
sudo mv /opt/vstats/vstats-server.bak /opt/vstats/vstats-server
sudo systemctl start vstats
```

