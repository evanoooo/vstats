# Agent Configuration

The vStats agent can be configured through command-line arguments, environment variables, or a configuration file.

## Configuration File

The agent configuration file is located at:

| OS | Path |
|----|------|
| Linux | `/etc/vstats-agent/config.json` |
| macOS | `~/.vstats-agent/config.json` |
| Windows | `C:\ProgramData\vstats-agent\config.json` |

### Example Configuration

```json
{
  "server": "http://your-server:3001",
  "token": "your-admin-token",
  "name": "my-server",
  "location": "US-West",
  "provider": "AWS",
  "interval": 1,
  "enable_gpu": true,
  "enable_docker": false,
  "log_level": "info"
}
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `server` | string | - | vStats server URL (required) |
| `token` | string | - | Admin token (required) |
| `name` | string | hostname | Display name |
| `location` | string | - | Location label |
| `provider` | string | - | Provider label |
| `interval` | int | 1 | Collection interval in seconds |
| `enable_gpu` | bool | true | Enable GPU metrics |
| `enable_docker` | bool | false | Enable Docker stats |
| `log_level` | string | info | Log level (debug, info, warn, error) |

## Environment Variables

All configuration options can also be set via environment variables:

| Variable | Description |
|----------|-------------|
| `VSTATS_SERVER` | Server URL |
| `VSTATS_TOKEN` | Admin token |
| `VSTATS_NAME` | Display name |
| `VSTATS_LOCATION` | Location label |
| `VSTATS_PROVIDER` | Provider label |
| `VSTATS_INTERVAL` | Collection interval |
| `VSTATS_ENABLE_GPU` | Enable GPU (true/false) |
| `VSTATS_LOG_LEVEL` | Log level |

Environment variables take precedence over the configuration file.

## GPU Metrics

GPU monitoring is enabled by default for NVIDIA GPUs. Requirements:

- NVIDIA GPU
- NVIDIA drivers installed
- `nvidia-smi` available in PATH

To disable GPU metrics:

```json
{
  "enable_gpu": false
}
```

## Collection Interval

The default collection interval is 1 second. You can adjust this based on your needs:

```json
{
  "interval": 5
}
```

::: tip
Lower intervals provide more real-time data but increase network traffic. For most use cases, 1-5 seconds is recommended.
:::

## Network Configuration

### Behind a Proxy

If your agent needs to connect through a proxy:

```bash
export HTTP_PROXY=http://proxy:8080
export HTTPS_PROXY=http://proxy:8080
```

### Custom TLS

If your server uses a self-signed certificate:

```json
{
  "insecure_skip_verify": true
}
```

::: warning
Only use `insecure_skip_verify` in trusted networks. For production, use proper certificates.
:::

## Modifying Configuration

After modifying the configuration file, restart the agent:

```bash
# Linux
sudo systemctl restart vstats-agent

# macOS
launchctl stop io.vstats.agent
launchctl start io.vstats.agent

# Windows
Restart-Service vstats-agent
```

