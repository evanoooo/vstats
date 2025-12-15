# Getting Started

This guide will help you get vStats up and running in just a few minutes.

## Quick Start with Docker

The fastest way to deploy vStats is using Docker:

```bash
# Create data directory
mkdir -p data && sudo chown -R 1000:1000 data

# Run vStats server
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

That's it! Visit `http://your-server-ip:3001` to access the dashboard.

## Get Admin Password

After first startup, an admin password is automatically generated. To find it:

```bash
# View container logs
docker logs vstats-server 2>&1 | grep -i password
```

You should see something like:

```
Admin password: abc123xyz
```

::: tip
Save this password! You'll need it to access admin features.
:::

## Install Agent on Monitored Servers

To monitor a server, install the vStats agent on it:

::: code-group

```bash [Linux/macOS]
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://YOUR_DASHBOARD_IP:3001 \
  --token "your-admin-token" \
  --name "$(hostname)"
```

```powershell [Windows]
irm https://vstats.zsoft.cc/agent.ps1 -OutFile agent.ps1
.\agent.ps1 -Server "http://YOUR_DASHBOARD_IP:3001" -Token "your-admin-token"
```

:::

::: info
Get the admin token from the dashboard's admin panel after logging in.
:::

## Verify Installation

1. Open your browser to `http://your-server-ip:3001`
2. You should see the vStats dashboard
3. Log in with username `admin` and your password
4. Your monitored server should appear in the server list

## What's Next?

<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-top: 1rem;">

<a href="/guide/installation/docker" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>🐳 Docker Setup</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">Advanced Docker configuration</p>
</a>

<a href="/guide/agent/install" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>📡 Agent Setup</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">Configure monitoring agents</p>
</a>

<a href="/guide/config/reverse-proxy" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>🔒 SSL/HTTPS</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">Secure with reverse proxy</p>
</a>

<a href="/reference/api" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>📚 API Reference</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">REST API documentation</p>
</a>

</div>

