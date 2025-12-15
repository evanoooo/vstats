# Script Installation

The installation script provides an easy way to install vStats directly on your system without Docker.

## Prerequisites

- Linux (Debian, Ubuntu, CentOS, RHEL, Fedora, Arch, etc.) or macOS
- Root/sudo access
- curl or wget

## Install Server

### Using curl

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

### Using wget

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash
```

## What the Script Does

1. Detects your OS and architecture
2. Downloads the appropriate binary
3. Creates installation directory (`/opt/vstats` on Linux, `~/.vstats` on macOS)
4. Sets up systemd service (Linux) or launchd (macOS)
5. Starts the service
6. Generates admin password

## Installation Paths

| OS | Binary Location | Data Directory |
|----|-----------------|----------------|
| Linux | `/opt/vstats/vstats-server` | `/opt/vstats/data` |
| macOS | `~/.vstats/vstats-server` | `~/.vstats/data` |

## Get Admin Password

After installation:

```bash
# Linux
journalctl -u vstats | grep -i password

# macOS
tail -20 ~/.vstats/data/vstats.log | grep -i password
```

## Reset Password

```bash
# Linux
/opt/vstats/vstats-server --reset-password

# macOS
~/.vstats/vstats-server --reset-password
```

## Upgrade

To upgrade to the latest version:

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

Or:

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

## Uninstall

To completely remove vStats:

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall
```

::: warning
This will remove the binary and service, but preserves data by default. Add `--purge` to remove data as well.
:::

## Custom Installation

You can customize the installation with environment variables:

```bash
# Custom port
PORT=8080 curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash

# Custom installation directory (Linux only)
INSTALL_DIR=/usr/local/vstats curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

## Verify Installation

```bash
# Check service status
systemctl status vstats  # Linux
launchctl list | grep vstats  # macOS

# Check if server is responding
curl http://localhost:3001/health
```

## Troubleshooting

### Installation fails

Check the error message. Common issues:

- **Permission denied**: Run with `sudo`
- **Architecture not supported**: Only amd64 and arm64 are supported
- **Network error**: Check your internet connection

### Service won't start

```bash
# Check logs
journalctl -u vstats -f  # Linux
tail -f ~/.vstats/data/vstats.log  # macOS

# Check service status
systemctl status vstats
```

### Port already in use

Edit the configuration file and change the port:

```bash
# Linux
vim /opt/vstats/vstats-config.json

# macOS
vim ~/.vstats/vstats-config.json
```

Then restart the service:

```bash
systemctl restart vstats  # Linux
```

