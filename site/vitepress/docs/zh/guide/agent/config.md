# Agent 配置

vStats Agent 可以通过命令行参数、环境变量或配置文件进行配置。

## 配置文件

Agent 配置文件位置：

| 操作系统 | 路径 |
|----------|------|
| Linux | `/etc/vstats-agent/config.json` |
| macOS | `~/.vstats-agent/config.json` |
| Windows | `C:\ProgramData\vstats-agent\config.json` |

### 配置示例

```json
{
  "server": "http://your-server:3001",
  "token": "你的管理员token",
  "name": "我的服务器",
  "location": "上海",
  "provider": "阿里云",
  "interval": 1,
  "enable_gpu": true,
  "enable_docker": false,
  "log_level": "info"
}
```

## 配置选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `server` | string | - | vStats 服务端 URL（必需） |
| `token` | string | - | 管理员 Token（必需） |
| `name` | string | 主机名 | 显示名称 |
| `location` | string | - | 位置标签 |
| `provider` | string | - | 提供商标签 |
| `interval` | int | 1 | 采集间隔（秒） |
| `enable_gpu` | bool | true | 启用 GPU 指标 |
| `enable_docker` | bool | false | 启用 Docker 统计 |
| `log_level` | string | info | 日志级别 (debug, info, warn, error) |

## 环境变量

所有配置选项也可以通过环境变量设置：

| 变量 | 说明 |
|------|------|
| `VSTATS_SERVER` | 服务端 URL |
| `VSTATS_TOKEN` | 管理员 Token |
| `VSTATS_NAME` | 显示名称 |
| `VSTATS_LOCATION` | 位置标签 |
| `VSTATS_PROVIDER` | 提供商标签 |
| `VSTATS_INTERVAL` | 采集间隔 |
| `VSTATS_ENABLE_GPU` | 启用 GPU (true/false) |
| `VSTATS_LOG_LEVEL` | 日志级别 |

环境变量优先级高于配置文件。

## GPU 指标

NVIDIA GPU 的 GPU 监控默认启用。要求：

- NVIDIA GPU
- 已安装 NVIDIA 驱动
- `nvidia-smi` 在 PATH 中可用

禁用 GPU 指标：

```json
{
  "enable_gpu": false
}
```

## 采集间隔

默认采集间隔为 1 秒。你可以根据需要调整：

```json
{
  "interval": 5
}
```

::: tip 提示
较短的间隔提供更实时的数据，但会增加网络流量。大多数场景下，1-5 秒是推荐值。
:::

## 网络配置

### 通过代理

如果 Agent 需要通过代理连接：

```bash
export HTTP_PROXY=http://proxy:8080
export HTTPS_PROXY=http://proxy:8080
```

### 自定义 TLS

如果服务端使用自签名证书：

```json
{
  "insecure_skip_verify": true
}
```

::: warning 警告
仅在受信任的网络中使用 `insecure_skip_verify`。生产环境请使用正规证书。
:::

## 修改配置

修改配置文件后，重启 Agent：

```bash
# Linux
sudo systemctl restart vstats-agent

# macOS
launchctl stop io.vstats.agent
launchctl start io.vstats.agent

# Windows
Restart-Service vstats-agent
```

