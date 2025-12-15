---
title: vStats CLI（命令行）
---

# vStats CLI（命令行）

`vStats CLI` 是用于管理 **vStats Cloud** 的命令行工具：登录、服务器管理、通过 SSH 部署 Agent / Web、查看指标等。

## 安装

### 一键安装（推荐）

**Linux / macOS：**

```bash
curl -fsSL https://vstats.zsoft.cc/cli.sh | sh
```

**Windows（PowerShell）：**

```powershell
irm https://vstats.zsoft.cc/cli.ps1 | iex
```

### Homebrew（macOS / Linux）

```bash
brew install zsai001/vstats/vstats
```

### Scoop（Windows）

```powershell
scoop bucket add vstats https://github.com/zsai001/scoop-vstats
scoop install vstats
```

## 快速开始

```bash
# 登录 vStats Cloud
vstats login

# 列出服务器
vstats server list

# 创建服务器
vstats server create my-server

# 通过 SSH 部署 Agent
vstats ssh agent root@192.168.1.1
```

## 常用命令

### 认证

```bash
vstats login
vstats login --token <your-token>
vstats whoami
vstats logout
```

### 服务器管理

```bash
vstats server list
vstats server create <name>
vstats server show <name-or-id>
vstats server update <name-or-id> --name <new-name>
vstats server delete <name-or-id> --force
```

### 指标

```bash
vstats server metrics <name-or-id>
vstats server history <name-or-id> --range 24h
```

## 配置

CLI 默认配置文件：`~/.vstats/config.yaml`

```yaml
cloud_url: https://api.vstats.zsoft.cc
token: <your-jwt-token>
username: your-username
expires_at: 1234567890
```

## 更多

- 完整说明：`server-go/cmd/cli/README.md`
- Releases：`https://github.com/zsai001/vstats/releases`


