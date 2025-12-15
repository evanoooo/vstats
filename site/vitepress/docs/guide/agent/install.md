# Install Agent

The vStats Agent is a lightweight daemon that collects system metrics and sends them to the vStats server.

## Prerequisites

- vStats server running and accessible
- Admin token from the dashboard
- Root/Administrator access on the monitored server

## Get Admin Token

1. Log in to your vStats dashboard
2. Go to Admin Panel (gear icon)
3. Copy your admin token

## Linux / macOS

### One-Line Installation

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://YOUR_DASHBOARD_IP:3001 \
  --token "your-admin-token" \
  --name "$(hostname)"
```

### With Additional Options

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://YOUR_DASHBOARD_IP:3001 \
  --token "your-admin-token" \
  --name "my-server" \
  --location "US-West" \
  --provider "AWS"
```

### Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `--server` | Yes | vStats server URL |
| `--token` | Yes | Admin token for authentication |
| `--name` | No | Display name (defaults to hostname) |
| `--location` | No | Server location label |
| `--provider` | No | Server provider label |

## Windows

### PowerShell

```powershell
# Download script
irm https://vstats.zsoft.cc/agent.ps1 -OutFile agent.ps1

# Run installation
.\agent.ps1 -Server "http://YOUR_DASHBOARD_IP:3001" -Token "your-admin-token"
```

### With Options

```powershell
.\agent.ps1 `
  -Server "http://YOUR_DASHBOARD_IP:3001" `
  -Token "your-admin-token" `
  -Name "windows-server" `
  -Location "EU-Central" `
  -Provider "Azure"
```

## Docker Agent

You can also run the agent in Docker:

```bash
docker run -d \
  --name vstats-agent \
  --net=host \
  --pid=host \
  --privileged \
  -e VSTATS_SERVER="http://YOUR_DASHBOARD_IP:3001" \
  -e VSTATS_TOKEN="your-admin-token" \
  -e VSTATS_NAME="docker-host" \
  -v /:/host:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  zsai001/vstats-agent:latest
```

::: warning
The Docker agent needs privileged access to collect accurate system metrics.
:::

## Verify Installation

After installation, the server should appear in your dashboard within a few seconds.

### Check Service Status

```bash
# Linux
systemctl status vstats-agent

# macOS
launchctl list | grep vstats

# Windows
Get-Service vstats-agent
```

### View Logs

```bash
# Linux
journalctl -u vstats-agent -f

# macOS
tail -f ~/.vstats-agent/agent.log

# Windows
Get-EventLog -LogName Application -Source vstats-agent -Newest 50
```

## Upgrade Agent

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --upgrade
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-upgrade.ps1 -OutFile agent-upgrade.ps1
.\agent-upgrade.ps1
```

## Uninstall Agent

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --uninstall
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-uninstall.ps1 -OutFile agent-uninstall.ps1
.\agent-uninstall.ps1
```

## Troubleshooting

### Agent not connecting

1. Check if the server URL is correct and accessible:
   ```bash
   curl http://YOUR_DASHBOARD_IP:3001/health
   ```

2. Verify the token is valid

3. Check firewall rules allow outbound connections to the server

### High CPU usage

The agent is designed to be lightweight. If you notice high CPU usage:

1. Check the agent version (upgrade if outdated)
2. Review the collection interval in configuration

### Metrics not updating

1. Check agent service status
2. Review agent logs for errors
3. Verify network connectivity to server

