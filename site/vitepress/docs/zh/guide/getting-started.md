# 快速开始

本指南将帮助你在几分钟内启动运行 vStats。

## 使用 Docker 快速开始

使用 Docker 是部署 vStats 最快的方式：

```bash
# 创建数据目录
mkdir -p data && sudo chown -R 1000:1000 data

# 运行 vStats 服务器
docker run -d \
  --name vstats-server \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  --restart unless-stopped \
  zsai001/vstats-server:latest
```

就是这样！访问 `http://你的服务器IP:3001` 即可查看面板。

## 获取管理员密码

首次启动后，系统会自动生成管理员密码。查看方法：

```bash
# 查看容器日志
docker logs vstats-server 2>&1 | grep -i password
```

你应该会看到类似这样的输出：

```
Admin password: abc123xyz
```

::: tip 提示
请保存好这个密码！你需要它来访问管理功能。
:::

## 在被监控服务器上安装 Agent

要监控一台服务器，需要在其上安装 vStats Agent：

::: code-group

```bash [Linux/macOS]
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server http://你的面板IP:3001 \
  --token "你的管理员token" \
  --name "$(hostname)"
```

```powershell [Windows]
irm https://vstats.zsoft.cc/agent.ps1 -OutFile agent.ps1
.\agent.ps1 -Server "http://你的面板IP:3001" -Token "你的管理员token"
```

:::

::: info 信息
登录面板后，可以在管理面板中获取管理员 token。
:::

## 验证安装

1. 用浏览器打开 `http://你的服务器IP:3001`
2. 你应该能看到 vStats 面板
3. 使用用户名 `admin` 和你的密码登录
4. 你的被监控服务器应该出现在服务器列表中

## 下一步

<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-top: 1rem;">

<a href="/zh/guide/installation/docker" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>🐳 Docker 配置</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">高级 Docker 配置</p>
</a>

<a href="/zh/guide/agent/install" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>📡 Agent 配置</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">配置监控探针</p>
</a>

<a href="/zh/guide/config/reverse-proxy" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>🔒 SSL/HTTPS</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">使用反向代理加密</p>
</a>

<a href="/zh/reference/api" style="display: block; padding: 1rem; background: var(--vp-c-bg-soft); border-radius: 8px; text-decoration: none;">
  <strong>📚 API 参考</strong>
  <p style="margin: 0.5rem 0 0; opacity: 0.8; font-size: 0.9rem;">REST API 文档</p>
</a>

</div>

