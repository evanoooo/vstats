# Uninstall

This guide covers completely removing vStats from your system.

## Before Uninstalling

1. **Export any data** you want to keep
2. **Backup configuration** if you might reinstall later
3. **Remove agents first** from monitored servers

## Docker Uninstall

### Stop and Remove Container

```bash
# Stop container
docker stop vstats-server

# Remove container
docker rm vstats-server
```

### Remove Data (Optional)

```bash
# Remove data directory
rm -rf ./data

# Or backup first
mv ./data ./data.backup
```

### Remove Image

```bash
# Remove image
docker rmi zsai001/vstats-server:latest

# Remove all related images
docker rmi $(docker images zsai001/vstats-* -q)
```

### Docker Compose

```bash
# Stop and remove containers
docker compose down

# Also remove volumes
docker compose down -v

# Remove images
docker compose down --rmi all
```

## Script Uninstall

### Linux/macOS

```bash
# Uninstall server
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall

# With data removal
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall --purge
```

The uninstall script will:
1. Stop the service
2. Remove the systemd service file
3. Remove the binary and installation directory
4. Optionally remove data (with `--purge`)

## Manual Uninstall

### Stop Service

```bash
# Linux
sudo systemctl stop vstats
sudo systemctl disable vstats

# macOS
launchctl unload ~/Library/LaunchAgents/io.vstats.server.plist
```

### Remove Service Files

```bash
# Linux
sudo rm /etc/systemd/system/vstats.service
sudo systemctl daemon-reload

# macOS
rm ~/Library/LaunchAgents/io.vstats.server.plist
```

### Remove Files

```bash
# Linux
sudo rm -rf /opt/vstats

# macOS
rm -rf ~/.vstats
```

### Remove User (Linux)

```bash
sudo userdel vstats
sudo groupdel vstats
```

## Agent Uninstall

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --uninstall
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-uninstall.ps1 -OutFile agent-uninstall.ps1
.\agent-uninstall.ps1
```

### Manual Agent Uninstall

```bash
# Linux
sudo systemctl stop vstats-agent
sudo systemctl disable vstats-agent
sudo rm /etc/systemd/system/vstats-agent.service
sudo rm -rf /opt/vstats-agent
sudo systemctl daemon-reload
```

## Cleanup

### Remove Configuration

```bash
# Remove any remaining config files
rm -f /etc/vstats/*
rm -rf /etc/vstats
```

### Remove Logs

```bash
# Clear journal logs (Linux)
sudo journalctl --vacuum-time=0 -u vstats
sudo journalctl --vacuum-time=0 -u vstats-agent
```

### Remove Backups

```bash
# If you have automated backups
rm -rf /opt/vstats/backups
```

### Remove Cron Jobs

```bash
# Edit crontab
crontab -e

# Remove any vstats-related lines
```

## Verify Removal

```bash
# Check if service still exists
systemctl status vstats

# Check if files remain
ls -la /opt/vstats
ls -la ~/.vstats

# Check if processes are running
ps aux | grep vstats

# Check if port is free
lsof -i :3001
```

## Reinstall Fresh

If you want to reinstall after uninstalling:

1. Ensure all files are removed
2. Run the install script again
3. A new admin password will be generated
4. Reinstall agents on monitored servers

