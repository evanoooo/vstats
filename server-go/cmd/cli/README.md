# vStats CLI

A command-line interface for managing vStats Cloud servers.

## Installation

### Quick Install (Recommended)

**Linux/macOS:**

```bash
curl -fsSL https://vstats.zsoft.cc/cli.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://vstats.zsoft.cc/cli.ps1 | iex
```

### Homebrew (macOS/Linux)

```bash
# Quick install
brew install zsai001/vstats/vstats

# Or add tap first
brew tap zsai001/vstats
brew install vstats
```

### Linux Package Repositories

#### Debian/Ubuntu (APT)

```bash
# Add GPG key
curl -fsSL https://vstats.zsoft.cc/gpg | sudo gpg --dearmor -o /usr/share/keyrings/vstats-archive-keyring.gpg

# Add repository
echo "deb [signed-by=/usr/share/keyrings/vstats-archive-keyring.gpg] https://repo.vstats.zsoft.cc/apt stable main" | \
  sudo tee /etc/apt/sources.list.d/vstats.list

# Install
sudo apt update
sudo apt install vstats
```

#### Fedora/RHEL/CentOS (DNF/YUM)

```bash
# Add repository
sudo tee /etc/yum.repos.d/vstats.repo << 'EOF'
[vstats]
name=vStats Repository
baseurl=https://repo.vstats.zsoft.cc/yum/$basearch
enabled=1
gpgcheck=1
gpgkey=https://vstats.zsoft.cc/gpg
EOF

# Install
sudo dnf install vstats  # or: sudo yum install vstats
```

#### Arch Linux (AUR)

```bash
# Using yay
yay -S vstats-bin

# Using paru
paru -S vstats-bin
```

### Windows Package Managers

#### Scoop

```powershell
scoop bucket add vstats https://github.com/zsai001/scoop-vstats
scoop install vstats
```

#### Chocolatey

```powershell
choco install vstats
```

### Manual Binary Download

Download the appropriate binary for your platform from the [releases page](https://github.com/zsai001/vstats/releases).

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | x64 | `vstats-cli-linux-amd64` |
| Linux | ARM64 | `vstats-cli-linux-arm64` |
| macOS | Intel | `vstats-cli-darwin-amd64` |
| macOS | Apple Silicon | `vstats-cli-darwin-arm64` |
| Windows | x64 | `vstats-cli-windows-amd64.exe` |
| Windows | ARM64 | `vstats-cli-windows-arm64.exe` |

### Build From Source

```bash
cd server-go
go build -o vstats ./cmd/cli

# Move to PATH
sudo mv vstats /usr/local/bin/
```

## Quick Start

```bash
# Login to vStats Cloud
vstats login

# List servers
vstats server list

# Create a new server
vstats server create my-server

# Deploy agent to a server via SSH
vstats ssh agent root@192.168.1.1

# Deploy web dashboard via SSH
vstats ssh web root@192.168.1.2

# View server metrics
vstats server metrics my-server
```

## Commands

### Authentication

```bash
# Login with interactive prompt
vstats login

# Login with token directly
vstats login --token <your-token>

# Show current user
vstats whoami

# Logout
vstats logout
```

### Server Management

```bash
# List all servers
vstats server list
vstats server ls

# Create a new server
vstats server create <name>

# Show server details
vstats server show <name-or-id>

# Update server name
vstats server update <name-or-id> --name <new-name>

# Delete a server
vstats server delete <name-or-id>
vstats server delete <name-or-id> --force
```

### Metrics

```bash
# View current metrics
vstats server metrics <name-or-id>

# View metrics history
vstats server history <name-or-id>
vstats server history <name-or-id> --range 24h
vstats server history <name-or-id> --range 7d
vstats server history <name-or-id> --range 30d
```

### SSH Deployment

Deploy agents and web dashboards to remote servers via SSH.
Uses your system SSH configuration (`~/.ssh/config`) for host management.

```bash
# Deploy agent via SSH (creates server + installs agent)
vstats ssh agent root@192.168.1.1
vstats ssh agent myserver                      # Use SSH config host alias
vstats ssh agent server.com -u admin
vstats ssh agent server.com --name "Prod-01"
vstats ssh agent server.com --server existing-server-id

# Deploy web dashboard via SSH
vstats ssh web root@192.168.1.1
vstats ssh web myserver --name "Home Dashboard"
vstats ssh web server.com --web-port 8080 --ssl --domain dash.example.com
```

Configure your hosts in `~/.ssh/config` for easier access:

```
Host myserver
    HostName 192.168.1.100
    User root
    IdentityFile ~/.ssh/myserver_key

Host prod-*
    User deploy
    IdentityFile ~/.ssh/prod_key
```

### Agent Management

```bash
# Get agent installation command
vstats server install <name-or-id>

# Show agent key
vstats server key <name-or-id>

# Regenerate agent key
vstats server key <name-or-id> --regenerate

# Deploy agent remotely via SSH
vstats ssh agent root@server.com --name "My Server"
```

### Web Dashboard Management

Manage web dashboard instances that connect to vStats Cloud.

**Free users:** 1 web instance  
**Pro users:** Unlimited web instances

```bash
# List all web instances
vstats web list

# Check plan and limits
vstats web status

# Check web instance health
vstats web check <instance-id>

# Remove a web instance
vstats web remove <instance-id>
vstats web remove <instance-id> --force
```

To deploy a web dashboard, use `vstats ssh web`. See SSH Deployment section.

### Configuration

```bash
# Show current configuration
vstats config show

# Set configuration value
vstats config set cloud_url https://api.vstats.example.com

# Show config file path
vstats config path
```

## Output Formats

The CLI supports multiple output formats:

```bash
# Table format (default)
vstats server list

# JSON format
vstats server list -o json

# YAML format
vstats server list -o yaml
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Config file path (default: `~/.vstats/config.yaml`) |
| `-o, --output` | Output format: `table`, `json`, `yaml` |
| `--cloud-url` | Override vStats Cloud URL |
| `--no-color` | Disable colored output |

## Configuration File

The CLI stores configuration in `~/.vstats/config.yaml`:

```yaml
cloud_url: https://api.vstats.zsoft.cc
token: <your-jwt-token>
username: your-username
expires_at: 1234567890
```

## Examples

### Deploy agent to multiple servers

```bash
# Create servers and deploy agents using SSH config hosts
for host in server1 server2 server3; do
  vstats ssh agent $host --name "$host"
done

# Or with explicit user@host
for host in server1 server2 server3; do
  vstats ssh agent root@$host.example.com --name "$host"
done
```

### Create and monitor a server

```bash
# Create a new server
vstats server create web-prod-01

# Deploy agent via SSH (use existing server)
vstats ssh agent root@web-prod-01.example.com --server web-prod-01

# Check server status
vstats server list

# View detailed metrics
vstats server metrics web-prod-01
```

### Deploy web dashboard with SSL

```bash
# Deploy with custom domain and SSL
vstats ssh web root@dashboard.example.com \
  --name "Main Dashboard" \
  --domain dashboard.example.com \
  --ssl

# Check the deployment
vstats web check <instance-id>
```

### Export server list to JSON

```bash
vstats server list -o json > servers.json
```

### Automation with shell scripts

```bash
#!/bin/bash

# Get all offline servers
vstats server list -o json | jq '.[] | select(.status == "offline") | .name'

# Check CPU usage for all servers
for server in $(vstats server list -o json | jq -r '.[].name'); do
    echo "Server: $server"
    vstats server metrics $server -o json | jq '.cpu_usage'
done
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VSTATS_CLOUD_URL` | Override default cloud URL |
| `VSTATS_TOKEN` | Authentication token |
| `NO_COLOR` | Disable colored output |

## Subscription Plans

| Feature | Free | Pro |
|---------|------|-----|
| Servers | 5 | Unlimited |
| Web Instances | 1 | Unlimited |
| Data Retention | 7 days | 365 days |
| SSH Deploy | ✓ | ✓ |
| API Access | ✓ | ✓ |

Upgrade at: https://vstats.zsoft.cc/pricing

## Upgrade

**Linux/macOS (via script):**
```bash
curl -fsSL https://vstats.zsoft.cc/cli.sh | sh
```

**Homebrew:**
```bash
brew upgrade vstats
```

**APT:**
```bash
sudo apt update && sudo apt upgrade vstats
```

**DNF/YUM:**
```bash
sudo dnf upgrade vstats
```

**Scoop:**
```bash
scoop update vstats
```

## Uninstall

**Homebrew:**
```bash
brew uninstall vstats
```

**APT:**
```bash
sudo apt remove vstats
```

**DNF/YUM:**
```bash
sudo dnf remove vstats
```

**Scoop:**
```bash
scoop uninstall vstats
```

**Manual (binary):**
```bash
sudo rm /usr/local/bin/vstats
rm -rf ~/.vstats
```

## License

MIT License - see the main repository for details.
