---
layout: home

hero:
  name: vStats
  text: 服务器监控面板
  tagline: 极简美观。Go 驱动，毫秒级延迟，一键部署。
  image:
    src: /logo-server.svg
    alt: vStats
  actions:
    - theme: brand
      text: 自部署文档
      link: /zh/server/
    - theme: alt
      text: 云端文档
      link: /zh/cloud/
    - theme: alt
      text: CLI 文档
      link: /zh/cli/
    - theme: alt
      text: GitHub
      link: https://github.com/zsai001/vstats

features:
  - icon: ⚡
    title: 实时监控
    details: WebSocket 实时推送系统指标，毫秒级延迟。
  - icon: 🖥️
    title: 多服务器管理
    details: 单一面板管理多台服务器，统一监控。
  - icon: 📊
    title: 全面指标
    details: CPU、内存、磁盘、网络、GPU 等，全方位系统可见性。
  - icon: 🎨
    title: 现代 UI
    details: 玻璃拟态设计，流畅动画，精美主题。
  - icon: 🔐
    title: 安全认证
    details: 基于 JWT 的认证保护管理接口。
  - icon: 🚀
    title: 一键部署
    details: Docker 或脚本，一分钟内启动运行。
---

<style>
:root {
  --vp-home-hero-name-color: transparent;
  --vp-home-hero-name-background: linear-gradient(135deg, #0ea5e9 0%, #10b981 100%);
}
</style>

## 📸 预览

<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem; margin-top: 2rem;">
  <img src="https://vstats.zsoft.cc/theme/1.png" alt="预览图 1" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
  <img src="https://vstats.zsoft.cc/theme/2.png" alt="预览图 2" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
  <img src="https://vstats.zsoft.cc/theme/3.png" alt="预览图 3" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
  <img src="https://vstats.zsoft.cc/theme/4.png" alt="预览图 4" style="border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.1);" />
</div>

## 🚀 快速开始

::: code-group

```bash [Docker]
# 创建数据目录
mkdir -p data && sudo chown -R 1000:1000 data

# 运行容器
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

```bash [脚本安装]
# 一键安装
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

:::

安装完成后，访问 `http://your-server-ip:3001` 查看面板。

## 💝 赞助商

<div style="display: flex; gap: 2rem; justify-content: center; margin-top: 2rem; flex-wrap: wrap;">
  <a href="https://www.tohu.cloud" target="_blank" style="padding: 1rem 2rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none; font-weight: 600;">TOHU Cloud</a>
  <a href="https://debee.io/" target="_blank" style="padding: 1rem 2rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none; font-weight: 600;">Debee</a>
</div>

## 💬 社区

加入我们的 [Telegram 群组](https://t.me/zsai010_group/10) 获取支持和参与讨论。

