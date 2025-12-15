# Service Management

This guide covers managing the vStats server and agent services.

## Linux (Systemd)

### Server Service

```bash
# Check status
systemctl status vstats

# Start service
sudo systemctl start vstats

# Stop service
sudo systemctl stop vstats

# Restart service
sudo systemctl restart vstats

# Reload configuration (without restart)
sudo systemctl reload vstats

# Enable auto-start on boot
sudo systemctl enable vstats

# Disable auto-start
sudo systemctl disable vstats
```

### Agent Service

```bash
# Check status
systemctl status vstats-agent

# Start service
sudo systemctl start vstats-agent

# Stop service
sudo systemctl stop vstats-agent

# Restart service
sudo systemctl restart vstats-agent

# Enable auto-start on boot
sudo systemctl enable vstats-agent

# Disable auto-start
sudo systemctl disable vstats-agent
```

### View Logs

```bash
# View all server logs
journalctl -u vstats

# Real-time log following
journalctl -u vstats -f

# Last 100 lines
journalctl -u vstats -n 100

# Logs from today
journalctl -u vstats --since today

# Logs for specific time range
journalctl -u vstats --since "2024-01-01 00:00:00" --until "2024-01-02 00:00:00"

# Agent logs
journalctl -u vstats-agent -f
```

## macOS (launchd)

### Server Service

```bash
# Check status
launchctl list | grep vstats

# Start service
launchctl start io.vstats.server

# Stop service
launchctl stop io.vstats.server

# Load service (enable)
launchctl load ~/Library/LaunchAgents/io.vstats.server.plist

# Unload service (disable)
launchctl unload ~/Library/LaunchAgents/io.vstats.server.plist
```

### View Logs

```bash
# Server logs
tail -f ~/.vstats/data/vstats.log

# Last 100 lines
tail -100 ~/.vstats/data/vstats.log
```

## Windows

### Server Service

```powershell
# Check status
Get-Service vstats

# Start service
Start-Service vstats

# Stop service
Stop-Service vstats

# Restart service
Restart-Service vstats

# Enable auto-start
Set-Service vstats -StartupType Automatic

# Disable auto-start
Set-Service vstats -StartupType Disabled
```

### Agent Service

```powershell
# Check status
Get-Service vstats-agent

# Start service
Start-Service vstats-agent

# Stop service
Stop-Service vstats-agent

# Restart service
Restart-Service vstats-agent
```

### View Logs

```powershell
# View event logs
Get-EventLog -LogName Application -Source vstats -Newest 50

# Filter by time
Get-EventLog -LogName Application -Source vstats -After (Get-Date).AddHours(-24)
```

## Docker

### Container Management

```bash
# Check status
docker ps | grep vstats

# Start container
docker start vstats-server

# Stop container
docker stop vstats-server

# Restart container
docker restart vstats-server

# View logs
docker logs vstats-server

# Follow logs
docker logs -f vstats-server

# Last 100 lines
docker logs --tail 100 vstats-server
```

### Docker Compose

```bash
# Status
docker compose ps

# Start
docker compose up -d

# Stop
docker compose down

# Restart
docker compose restart

# Logs
docker compose logs -f
```

## Health Checks

### Check if service is running

```bash
# Linux
systemctl is-active vstats

# HTTP health check
curl http://localhost:3001/health
```

Expected response:
```json
{"status":"ok"}
```

### Check service details

```bash
# Detailed status
systemctl status vstats -l

# Check if enabled
systemctl is-enabled vstats
```

## Common Tasks

### View resource usage

```bash
# Linux - current process stats
systemctl status vstats

# Docker
docker stats vstats-server
```

### Debug mode

Enable debug logging temporarily:

```bash
# Edit config to set log_level: "debug"
# Then restart
sudo systemctl restart vstats
```

### Service file locations

| OS | Path |
|----|------|
| Linux | `/etc/systemd/system/vstats.service` |
| macOS | `~/Library/LaunchAgents/io.vstats.server.plist` |
| Windows | Managed via `sc.exe` |

