---
title: vStats CLI
---

# vStats CLI

`vStats CLI` is the command-line tool for managing **vStats Cloud**: authentication, server lifecycle, SSH deployment (agent/web), and metrics viewing.

## Install

**Linux / macOS:**

```bash
curl -fsSL https://vstats.zsoft.cc/cli.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://vstats.zsoft.cc/cli.ps1 | iex
```

## Quick Start

```bash
vstats login
vstats server list
vstats server create my-server
```

## More

- Full CLI README: `server-go/cmd/cli/README.md`


