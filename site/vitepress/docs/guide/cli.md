---
title: vStats CLI
---

# vStats CLI

`vStats CLI` is the command-line tool for managing **vStats Cloud**: authentication, server lifecycle, SSH deployment (agent/web), and metrics viewing.

## Installation

### Quick Install (Recommended)

**Linux / macOS:**

```bash
curl -fsSL https://vstats.zsoft.cc/cli.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://vstats.zsoft.cc/cli.ps1 | iex
```

### Homebrew (macOS / Linux)

```bash
brew install zsai001/vstats/vstats
```

### Scoop (Windows)

```powershell
scoop bucket add vstats https://github.com/zsai001/scoop-vstats
scoop install vstats
```

## Quick Start

```bash
# Login to vStats Cloud
vstats login

# List servers
vstats server list

# Create a server
vstats server create my-server

# Deploy agent via SSH
vstats ssh agent root@192.168.1.1
```

## Common Commands

### Authentication

```bash
vstats login
vstats login --token <your-token>
vstats whoami
vstats logout
```

### Server Management

```bash
vstats server list
vstats server create <name>
vstats server show <name-or-id>
vstats server update <name-or-id> --name <new-name>
vstats server delete <name-or-id> --force
```

### Metrics

```bash
vstats server metrics <name-or-id>
vstats server history <name-or-id> --range 24h
```

## Configuration

Default config file: `~/.vstats/config.yaml`

```yaml
cloud_url: https://api.vstats.zsoft.cc
token: <your-jwt-token>
username: your-username
expires_at: 1234567890
```

## More

- Full CLI README: `server-go/cmd/cli/README.md`
- Releases: `https://github.com/zsai001/vstats/releases`


