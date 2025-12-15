# 安装 Agent

vStats Agent 是一个轻量级守护进程，负责收集系统指标并发送到 vStats 服务端。

## 前提条件

- vStats 服务端已运行且可访问
- 从面板获取的管理员 Token
- 被监控服务器的 Root/管理员权限

## 获取管理员 Token

1. 登录 vStats 面板
2. 进入管理面板（齿轮图标）
3. 复制管理员 Token

## Linux / macOS

### 一键安装

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://你的面板IP:3001 \
  --token "你的管理员token" \
  --name "$(hostname)"
```

### 带附加选项

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://你的面板IP:3001 \
  --token "你的管理员token" \
  --name "我的服务器" \
  --location "上海" \
  --provider "阿里云"
```

### 参数说明

| 参数 | 必需 | 说明 |
|------|------|------|
| `--server` | 是 | vStats 服务端 URL |
| `--token` | 是 | 用于认证的管理员 Token |
| `--name` | 否 | 显示名称（默认为主机名） |
| `--location` | 否 | 服务器位置标签 |
| `--provider` | 否 | 服务器提供商标签 |

## Windows

### PowerShell

```powershell
# 下载脚本
irm https://vstats.zsoft.cc/agent.ps1 -OutFile agent.ps1

# 运行安装
.\agent.ps1 -Server "http://你的面板IP:3001" -Token "你的管理员token"
```

### 带选项

```powershell
.\agent.ps1 `
  -Server "http://你的面板IP:3001" `
  -Token "你的管理员token" `
  -Name "windows-server" `
  -Location "北京" `
  -Provider "腾讯云"
```

## Docker Agent

也可以在 Docker 中运行 Agent：

```bash
docker run -d \
  --name vstats-agent \
  --net=host \
  --pid=host \
  --privileged \
  -e VSTATS_SERVER="http://你的面板IP:3001" \
  -e VSTATS_TOKEN="你的管理员token" \
  -e VSTATS_NAME="docker-host" \
  -v /:/host:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  zsai001/vstats-agent:latest
```

::: warning 注意
Docker Agent 需要特权访问才能收集准确的系统指标。
:::

## 验证安装

安装后，服务器应该在几秒钟内出现在面板中。

### 检查服务状态

```bash
# Linux
systemctl status vstats-agent

# macOS
launchctl list | grep vstats

# Windows
Get-Service vstats-agent
```

### 查看日志

```bash
# Linux
journalctl -u vstats-agent -f

# macOS
tail -f ~/.vstats-agent/agent.log

# Windows
Get-EventLog -LogName Application -Source vstats-agent -Newest 50
```

## 升级 Agent

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --upgrade
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-upgrade.ps1 -OutFile agent-upgrade.ps1
.\agent-upgrade.ps1
```

## 卸载 Agent

### Linux/macOS

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --uninstall
```

### Windows

```powershell
irm https://vstats.zsoft.cc/agent-uninstall.ps1 -OutFile agent-uninstall.ps1
.\agent-uninstall.ps1
```

## 故障排除

### Agent 无法连接

1. 检查服务端 URL 是否正确且可访问：
   ```bash
   curl http://你的面板IP:3001/health
   ```

2. 验证 Token 是否有效

3. 检查防火墙规则是否允许到服务端的出站连接

### CPU 使用率高

Agent 设计为轻量级。如果发现 CPU 使用率高：

1. 检查 Agent 版本（如果过时则升级）
2. 查看配置中的采集间隔

### 指标不更新

1. 检查 Agent 服务状态
2. 查看 Agent 日志中的错误
3. 验证到服务端的网络连接

