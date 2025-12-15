# Agent 故障排除

vStats Agent 常见问题及解决方案。

## 连接问题

### Agent 无法连接到服务端

**症状**：服务器不出现在面板中，Agent 日志显示连接错误。

**解决方案**：

1. **验证服务端 URL**
   ```bash
   curl http://你的服务端:3001/health
   ```
   应该返回：`{"status":"ok"}`

2. **检查网络连通性**
   ```bash
   ping 你的服务端
   telnet 你的服务端 3001
   ```

3. **检查防火墙规则**
   ```bash
   # 服务端
   sudo ufw status
   sudo firewall-cmd --list-ports
   ```

4. **验证 Token 正确**
   - 从面板管理界面获取新 Token
   - 检查 Token 中是否有多余的空格或换行

### 连接频繁断开

**解决方案**：

1. 检查网络稳定性
2. 查看服务端日志中的错误
3. 如有需要，增加 WebSocket 超时时间

## 指标问题

### 缺少 CPU 指标

**解决方案**：

1. **Linux**：检查 `/proc/stat` 是否可读
   ```bash
   cat /proc/stat
   ```

2. **权限问题**：Agent 可能需要提升权限

### 缺少内存指标

**解决方案**：

1. **Linux**：检查 `/proc/meminfo`
   ```bash
   cat /proc/meminfo
   ```

2. **macOS**：确保 Agent 有系统权限

### 缺少磁盘指标

**解决方案**：

1. 检查磁盘是否正确挂载
   ```bash
   df -h
   lsblk
   ```

2. 某些虚拟文件系统可能无法正确报告

### 缺少网络指标

**解决方案**：

1. **Linux**：检查 `/proc/net/dev`
   ```bash
   cat /proc/net/dev
   ```

2. 检查网络接口是否启用
   ```bash
   ip link show
   ```

### GPU 指标不显示

**要求**：
- NVIDIA GPU
- 已安装 NVIDIA 驱动
- `nvidia-smi` 在 PATH 中

**检查**：
```bash
nvidia-smi
which nvidia-smi
```

## 服务问题

### Agent 无法启动

**Linux**：
```bash
# 检查状态
systemctl status vstats-agent

# 检查日志
journalctl -u vstats-agent -n 100

# 尝试手动运行
/opt/vstats-agent/vstats-agent --config /etc/vstats-agent/config.json
```

**macOS**：
```bash
# 检查是否加载
launchctl list | grep vstats

# 检查日志
tail -100 ~/.vstats-agent/agent.log

# 尝试手动运行
~/.vstats-agent/vstats-agent
```

**Windows**：
```powershell
# 检查服务状态
Get-Service vstats-agent

# 检查事件日志
Get-EventLog -LogName Application -Source vstats-agent -Newest 50

# 尝试手动运行
& "C:\Program Files\vstats-agent\vstats-agent.exe"
```

### 资源占用高

**CPU 高**：
1. 检查 Agent 版本（如果过时则升级）
2. 增加采集间隔
3. 禁用未使用的指标（GPU、Docker）

**内存高**：
1. 通常表示有泄漏 - 重启 Agent
2. 如果持续存在，请报告问题

### Agent 崩溃

1. 检查系统日志获取崩溃详情
2. 在前台运行 Agent 查看错误输出
3. 检查权限问题
4. 验证所有必需的系统调用是否被允许（容器环境）

## 日志分析

### 启用调试日志

```json
{
  "log_level": "debug"
}
```

更改日志级别后重启 Agent。

### 日志位置

| 操作系统 | 位置 |
|----------|------|
| Linux | `journalctl -u vstats-agent` |
| macOS | `~/.vstats-agent/agent.log` |
| Windows | 事件查看器 → 应用程序 |

### 常见日志消息

| 消息 | 含义 |
|------|------|
| `connected to server` | 成功连接 |
| `connection refused` | 服务端不可达 |
| `authentication failed` | Token 无效 |
| `websocket closed` | 连接断开 |
| `metrics sent` | 数据发送成功 |

## 获取帮助

如果无法解决问题：

1. 收集信息：
   - 操作系统和版本
   - Agent 版本（`vstats-agent --version`）
   - 错误日志
   - 网络配置

2. 加入我们的 [Telegram 群组](https://t.me/zsai010_group/10)

3. 在 [GitHub](https://github.com/zsai001/vstats/issues) 上提交 Issue

