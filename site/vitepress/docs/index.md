---
layout: home

hero:
  name: vStats
  text: Server Monitoring Dashboard
  tagline: Minimalist & Beautiful. Go-powered, millisecond latency, one-click deployment.
  image:
    src: /logo-server.svg
    alt: vStats
  actions:
    - theme: brand
      text: Server Docs
      link: /server/
    - theme: alt
      text: Cloud Docs
      link: /cloud/
    - theme: alt
      text: CLI Docs
      link: /cli/
    - theme: alt
      text: GitHub
      link: https://github.com/zsai001/vstats

features:
  - icon: ⚡
    title: Real-time Monitoring
    details: WebSocket-powered real-time push of system metrics with millisecond-level latency.
  - icon: 🖥️
    title: Multi-Server Management
    details: Monitor multiple servers from a single dashboard with unified management.
  - icon: 📊
    title: Comprehensive Metrics
    details: CPU, Memory, Disk, Network, GPU and more - full system visibility.
  - icon: 🎨
    title: Modern UI
    details: Glassmorphism design with smooth animations and beautiful themes.
  - icon: 🔐
    title: Secure Authentication
    details: JWT-based authentication protects your admin interfaces.
  - icon: 🚀
    title: One-Click Deployment
    details: Docker or script - get up and running in under a minute.
---

<style>
:root {
  --vp-home-hero-name-color: transparent;
  --vp-home-hero-name-background: linear-gradient(135deg, #0ea5e9 0%, #10b981 100%);
}
</style>

## 📸 Preview

<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem; margin-top: 2rem;">
  <img src="https://vstats.zsoft.cc/theme/1.png" alt="Preview 1" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
  <img src="https://vstats.zsoft.cc/theme/2.png" alt="Preview 2" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
  <img src="https://vstats.zsoft.cc/theme/3.png" alt="Preview 3" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
  <img src="https://vstats.zsoft.cc/theme/4.png" alt="Preview 4" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
</div>

## 🚀 Quick Start

::: code-group

```bash [Docker]
# Create data directory
mkdir -p data && sudo chown -R 1000:1000 data

# Run container
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

```bash [Script]
# One-click installation
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

:::

After installation, visit `http://your-server-ip:3001` to access the dashboard.

## 💝 Sponsors

<div style="display: flex; gap: 2rem; justify-content: center; margin-top: 2rem; flex-wrap: wrap;">
  <a href="https://www.tohu.cloud" target="_blank" style="padding: 1rem 2rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none; font-weight: 600;">TOHU Cloud</a>
  <a href="https://debee.io/" target="_blank" style="padding: 1rem 2rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none; font-weight: 600;">Debee</a>
</div>

## 💬 Community

Join our [Telegram Group](https://t.me/zsai010_group/10) for support and discussions.

