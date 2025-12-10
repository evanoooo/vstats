# vStats - Server Monitoring Dashboard

[![GitHub Release](https://img.shields.io/github/v/release/zsai001/vstats?style=flat-square)](https://github.com/zsai001/vstats/releases)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-20232A?style=flat-square&logo=react&logoColor=61DAFB)
![TypeScript](https://img.shields.io/badge/TypeScript-007ACC?style=flat-square&logo=typescript&logoColor=white)

极简美观的服务器探针监控系统。Go 驱动，毫秒级延迟，一键部署。

## 💝 赞助商

<div align="center">

感谢以下赞助商对本项目的支持！

[TOHU Cloud](https://www.tohu.cloud) | [Debee](https://debee.io/)

</div>

## 📸 预览

<table>
  <tr>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/1.png" alt="预览图 1" width="100%"/>
    </td>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/2.png" alt="预览图 2" width="100%"/>
    </td>
  </tr>
  <tr>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/3.png" alt="预览图 3" width="100%"/>
    </td>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/4.png" alt="预览图 4" width="100%"/>
    </td>
  </tr>
</table>

## ✨ 特性

- 🚀 **实时监控** - WebSocket 实时推送系统指标
- 🖥️ **多服务器管理** - 支持监控多台服务器
- 💻 **CPU / 内存 / 磁盘 / 网络** - 全方位监控
- 🎨 **现代 UI** - 玻璃拟态设计，流畅动画
- 🔐 **安全认证** - JWT 认证保护管理接口
- ⚡ **一键部署** - Docker / 脚本一键安装

## 📚 文档与资源

| 资源 | 链接 |
|------|------|
| 📖 **完整文档** | [vstats.zsoft.cc](https://vstats.zsoft.cc) |
| 🎯 **在线演示** | [vps.zsoft.cc](https://vps.zsoft.cc/) |
| 🐳 **Docker Hub** | [zsai001/vstats-server](https://hub.docker.com/r/zsai001/vstats-server) |
| 📦 **GitHub Releases** | [下载页面](https://github.com/zsai001/vstats/releases) |

## 🚀 快速开始

```bash
# Docker 一键部署
docker run -d --name vstats-server -p 3001:3001 \
  -v $(pwd)/data:/app/data zsai001/vstats-server:latest
```

更多安装方式请访问 **[文档站点](https://vstats.zsoft.cc/docs)**

## 📦 脚本安装

### Server 安装

使用官方安装脚本一键安装 Server：

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

或使用 wget：

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash
```

安装完成后，访问 `http://your-server-ip:3001` 查看控制面板。

**获取管理员密码：**

```bash
# Linux
journalctl -u vstats | grep -i password

# macOS
tail -20 ~/.vstats/data/vstats.log | grep -i password

# 或重置密码
/opt/vstats/vstats-server --reset-password  # Linux
~/.vstats/vstats-server --reset-password     # macOS
```

### Agent 安装

在被监控的服务器上运行以下命令安装 Agent：

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://your-server-ip:3001 \
  --name "$(hostname)" \
  --token "your-admin-token"
```

参数说明：
- `--server`: Server 的访问地址
- `--name`: 服务器显示名称（可选，默认为主机名）
- `--token`: 管理员 Token（在 Server 控制面板中获取）

### 升级

**Server 升级：**

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

**Agent 升级：**

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --upgrade
```

### 卸载

**Server 卸载：**

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall
```

**Agent 卸载：**

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --uninstall
```

### 服务管理

安装完成后，Server 和 Agent 会注册为 systemd 服务，可以使用 `systemctl` 命令进行管理。

#### Server 服务管理

**查看服务状态：**

```bash
systemctl status vstats
```

**启动服务：**

```bash
systemctl start vstats
```

**停止服务：**

```bash
systemctl stop vstats
```

**重启服务：**

```bash
systemctl restart vstats
```

**重新加载配置（无需重启）：**

```bash
systemctl reload vstats
```

**设置开机自启：**

```bash
systemctl enable vstats
```

**取消开机自启：**

```bash
systemctl disable vstats
```

**查看服务日志：**

```bash
# 查看所有日志
journalctl -u vstats

# 实时查看日志（类似 tail -f）
journalctl -u vstats -f

# 查看最近 100 行日志
journalctl -u vstats -n 100

# 查看指定时间段的日志
journalctl -u vstats --since "2024-01-01 00:00:00" --until "2024-01-02 00:00:00"
```

#### Agent 服务管理

**查看服务状态：**

```bash
systemctl status vstats-agent
```

**启动服务：**

```bash
systemctl start vstats-agent
```

**停止服务：**

```bash
systemctl stop vstats-agent
```

**重启服务：**

```bash
systemctl restart vstats-agent
```

**设置开机自启：**

```bash
systemctl enable vstats-agent
```

**取消开机自启：**

```bash
systemctl disable vstats-agent
```

**查看服务日志：**

```bash
# 查看所有日志
journalctl -u vstats-agent

# 实时查看日志
journalctl -u vstats-agent -f

# 查看最近 100 行日志
journalctl -u vstats-agent -n 100
```

#### 常用 systemctl 命令

```bash
# 查看所有已启用的服务
systemctl list-units --type=service --state=running

# 查看服务是否运行
systemctl is-active vstats
systemctl is-active vstats-agent

# 查看服务是否已启用开机自启
systemctl is-enabled vstats
systemctl is-enabled vstats-agent

# 重新加载 systemd 配置（修改服务文件后需要执行）
systemctl daemon-reload
```

## 💬 问题反馈

遇到问题或有建议？欢迎加入 Telegram 群组进行实时反馈：

👉 [vStats 问题反馈群](https://t.me/zsai010_group/10)

## ⭐ Star History

<a href="https://star-history.com/#zsai001/vstats&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=zsai001/vstats&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=zsai001/vstats&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=zsai001/vstats&type=Date" />
 </picture>
</a>

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
