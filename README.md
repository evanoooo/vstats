# vStats - Server Monitoring Dashboard

**English** | [中文](README.zh.md)

[![GitHub Release](https://img.shields.io/github/v/release/zsai001/vstats?style=flat-square)](https://github.com/zsai001/vstats/releases)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-20232A?style=flat-square&logo=react&logoColor=61DAFB)
![TypeScript](https://img.shields.io/badge/TypeScript-007ACC?style=flat-square&logo=typescript&logoColor=white)

A minimalist and beautiful server monitoring dashboard. Powered by Go, millisecond-level latency, one-click deployment.

## 💝 Sponsors

<div align="center">

Thanks to the following sponsors for supporting this project!

[TOHU Cloud](https://www.tohu.cloud) | [Debee](https://debee.io/)

</div>

## 📸 Preview

<table>
  <tr>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/1.png" alt="Preview 1" width="100%"/>
    </td>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/2.png" alt="Preview 2" width="100%"/>
    </td>
  </tr>
  <tr>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/3.png" alt="Preview 3" width="100%"/>
    </td>
    <td align="center">
      <img src="https://vstats.zsoft.cc/theme/4.png" alt="Preview 4" width="100%"/>
    </td>
  </tr>
</table>

## ✨ Features

- 🚀 **Real-time Monitoring** - WebSocket real-time push of system metrics
- 🖥️ **Multi-Server Management** - Support monitoring multiple servers
- 💻 **CPU / Memory / Disk / Network** - Comprehensive monitoring
- 🎨 **Modern UI** - Glassmorphism design with smooth animations
- 🔐 **Secure Authentication** - JWT authentication protects admin interfaces
- ⚡ **One-Click Deployment** - Docker / script one-click installation

## 📋 TODO

- [ ] Data aggregation and optimization
- [ ] Billing and cycle management
- [ ] Global map view
- [ ] Performance testing (speedtest / latency)
- [ ] Media unlock detection
- [ ] Custom banner / slogan
- [ ] Referral & affiliate settings
- [ ] Alert notifications (Email / Telegram / Discord / Webhook)
- [ ] Custom dashboard layout
- [ ] Mobile responsive optimization
- [ ] Server grouping and tagging
- [ ] Dark / Light theme switching
- [ ] Custom theme
- [ ] Improve vstats-cli
- [ ] Improve vstats-cloud

## 📚 Documentation & Resources

| Resource | Link |
|----------|------|
| 📖 **Full Documentation** | [vstats.zsoft.cc](https://vstats.zsoft.cc) |
| 🎯 **Online Demo** | [vps.zsoft.cc](https://vps.zsoft.cc/) |
| 🐳 **Docker Hub** | [zsai001/vstats-server](https://hub.docker.com/r/zsai001/vstats-server) |
| 📦 **GitHub Releases** | [Download Page](https://github.com/zsai001/vstats/releases) |

## 🚀 Quick Start

```bash
# Docker one-click deployment
mkdir -p data
sudo chown -R 1000:1000 data
docker run -d --name vstats-server -p 3001:3001 \
  -v $(pwd)/data:/app/data zsai001/vstats-server:latest
```

For more installation methods, please visit **[Documentation Site](https://vstats.zsoft.cc/docs)**

## 📦 Script Installation

### Server Installation

Use the official installation script to install Server with one command:

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

Or use wget:

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash
```

After installation, visit `http://your-server-ip:3001` to access the control panel.

**Get Admin Password:**

```bash
# Linux
journalctl -u vstats | grep -i password

# macOS
tail -20 ~/.vstats/data/vstats.log | grep -i password

# Or reset password
/opt/vstats/vstats-server --reset-password  # Linux
~/.vstats/vstats-server --reset-password     # macOS
```

### Agent Installation

Run the following command on the monitored server to install Agent:

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://your-server-ip:3001 \
  --name "$(hostname)" \
  --token "your-admin-token"
```

Parameter Description:
- `--server`: Server access address
- `--name`: Server display name (optional, defaults to hostname)
- `--token`: Admin Token (obtained from Server control panel)

### Upgrade

**Server Upgrade:**

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

**Agent Upgrade:**

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --upgrade
```

### Uninstall

**Server Uninstall:**

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall
```

**Agent Uninstall:**

```bash
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- --uninstall
```

### Service Management

After installation, Server and Agent will be registered as systemd services and can be managed using `systemctl` commands.

#### Server Service Management

**Check Service Status:**

```bash
systemctl status vstats
```

**Start Service:**

```bash
systemctl start vstats
```

**Stop Service:**

```bash
systemctl stop vstats
```

**Restart Service:**

```bash
systemctl restart vstats
```

**Reload Configuration (without restart):**

```bash
systemctl reload vstats
```

**Enable Auto-start on Boot:**

```bash
systemctl enable vstats
```

**Disable Auto-start on Boot:**

```bash
systemctl disable vstats
```

**View Service Logs:**

```bash
# View all logs
journalctl -u vstats

# View logs in real-time (similar to tail -f)
journalctl -u vstats -f

# View last 100 lines of logs
journalctl -u vstats -n 100

# View logs for a specific time period
journalctl -u vstats --since "2024-01-01 00:00:00" --until "2024-01-02 00:00:00"
```

#### Agent Service Management

**Check Service Status:**

```bash
systemctl status vstats-agent
```

**Start Service:**

```bash
systemctl start vstats-agent
```

**Stop Service:**

```bash
systemctl stop vstats-agent
```

**Restart Service:**

```bash
systemctl restart vstats-agent
```

**Enable Auto-start on Boot:**

```bash
systemctl enable vstats-agent
```

**Disable Auto-start on Boot:**

```bash
systemctl disable vstats-agent
```

**View Service Logs:**

```bash
# View all logs
journalctl -u vstats-agent

# View logs in real-time
journalctl -u vstats-agent -f

# View last 100 lines of logs
journalctl -u vstats-agent -n 100
```

#### Common systemctl Commands

```bash
# View all enabled services
systemctl list-units --type=service --state=running

# Check if service is running
systemctl is-active vstats
systemctl is-active vstats-agent

# Check if service is enabled for auto-start
systemctl is-enabled vstats
systemctl is-enabled vstats-agent

# Reload systemd configuration (required after modifying service files)
systemctl daemon-reload
```

## 💬 Feedback

Encountered issues or have suggestions? Welcome to join the Telegram group for real-time feedback:

👉 [vStats Feedback Group](https://t.me/zsai010_group/10)

## Thanks 

[yeah.cx](https://yeah.cx) with ipv6 vps for testing

## ⭐ Star History

<a href="https://star-history.com/#zsai001/vstats&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=zsai001/vstats&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=zsai001/vstats&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=zsai001/vstats&type=Date" />
 </picture>
</a>

## 📄 License

MIT License

## 🤝 Contributing

Welcome to submit Issues and Pull Requests!
