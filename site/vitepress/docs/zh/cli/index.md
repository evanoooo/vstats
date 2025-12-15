---
title: vStats CLI（命令行）
---

# vStats CLI（命令行）

`vStats CLI` 是用于管理 **vStats Cloud** 的命令行工具：登录、服务器管理、通过 SSH 部署 Agent / Web、查看指标等。

## 安装

**Linux / macOS：**

```bash
curl -fsSL https://vstats.zsoft.cc/cli.sh | sh
```

**Windows（PowerShell）：**

```powershell
irm https://vstats.zsoft.cc/cli.ps1 | iex
```

## 快速开始

```bash
vstats login
vstats server list
vstats server create my-server
```

## 更多

- 完整说明：`server-go/cmd/cli/README.md`


