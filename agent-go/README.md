# vStats Agent (Go Implementation)

这是 vStats 代理的 Go 语言实现。

## 构建

```bash
cd agent-go
go mod tidy
go build -o vstats-agent
```

## 使用

### 注册代理

```bash
./vstats-agent register --server http://dashboard:3001 --token <admin_token> [--name <server_name>]
```

### 运行代理

```bash
./vstats-agent run
# 或
./vstats-agent run --config /path/to/config.json
```

### 安装为服务

```bash
sudo ./vstats-agent install
```

### 卸载服务

```bash
sudo ./vstats-agent uninstall
```

### 显示配置

```bash
./vstats-agent show-config
```

## 配置文件

配置文件位置：`vstats-agent.json`

默认位置：
- Linux/macOS: `/etc/vstats-agent/vstats-agent.json` 或 `~/.config/vstats-agent/vstats-agent.json`
- Windows: `%PROGRAMDATA%\vstats-agent\vstats-agent.json` 或 `%APPDATA%\vstats-agent\vstats-agent.json`

## 功能

- 自动收集系统指标（CPU、内存、磁盘、网络）
- 通过 WebSocket 实时推送指标到服务器
- 支持自定义 ping 目标
- 自动重连
- 支持系统服务安装（systemd/launchd/Windows Service）

