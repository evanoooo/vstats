# SSL/HTTPS 配置

使用 SSL/HTTPS 保护你的 vStats 安装对于生产环境部署至关重要。

## 推荐：反向代理

推荐使用反向代理进行 SSL 终止：

- [Nginx 配置](/zh/guide/config/reverse-proxy#nginx)
- [Caddy 配置](/zh/guide/config/reverse-proxy#caddy)
- [Traefik 配置](/zh/guide/config/reverse-proxy#traefik)

## 使用 Certbot 获取 Let's Encrypt 证书

### 安装 Certbot

::: code-group

```bash [Ubuntu/Debian]
sudo apt update
sudo apt install certbot
```

```bash [CentOS/RHEL]
sudo dnf install certbot
```

```bash [macOS]
brew install certbot
```

:::

### 获取证书

```bash
# 独立模式（先停止 web 服务器）
sudo certbot certonly --standalone -d vstats.example.com

# 或 webroot 模式（web 服务器运行中）
sudo certbot certonly --webroot -w /var/www/html -d vstats.example.com
```

### 证书文件

证书存储在：

```
/etc/letsencrypt/live/vstats.example.com/
├── fullchain.pem  # 证书 + 中间证书
├── privkey.pem    # 私钥
├── cert.pem       # 仅证书
└── chain.pem      # 中间证书
```

### 自动续期

Certbot 会设置自动续期。测试续期：

```bash
sudo certbot renew --dry-run
```

## 直接 SSL 配置

如果你不想使用反向代理，vStats 支持直接 SSL：

### 配置文件

```json
{
  "port": 443,
  "ssl": {
    "enabled": true,
    "cert": "/path/to/fullchain.pem",
    "key": "/path/to/privkey.pem"
  }
}
```

### Docker 使用 SSL

```bash
docker run -d \
  --name vstats-server \
  -p 443:443 \
  -e PORT=443 \
  -v $(pwd)/data:/app/data \
  -v /etc/letsencrypt:/etc/letsencrypt:ro \
  zsai001/vstats-server:latest
```

配置：

```json
{
  "port": 443,
  "ssl": {
    "enabled": true,
    "cert": "/etc/letsencrypt/live/vstats.example.com/fullchain.pem",
    "key": "/etc/letsencrypt/live/vstats.example.com/privkey.pem"
  }
}
```

## 自签名证书

用于测试或内部使用：

```bash
# 生成自签名证书
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout vstats.key \
  -out vstats.crt \
  -subj "/CN=vstats.local"
```

::: warning 警告
自签名证书会显示浏览器警告。连接到使用自签名证书服务器的 Agent 需要在配置中设置 `insecure_skip_verify: true`。
:::

## Agent 配置 HTTPS

当服务端使用 HTTPS 时，更新 Agent 配置：

```bash
# 使用 HTTPS 服务器安装 Agent
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server https://vstats.example.com \
  --token "你的token"
```

## 安全最佳实践

1. **使用 TLS 1.2+**：禁用旧协议
2. **强密码套件**：使用现代密码套件
3. **HSTS**：启用 HTTP 严格传输安全
4. **证书续期**：使用 certbot 或类似工具自动化

### Nginx SSL 最佳实践

```nginx
# 强 SSL 配置
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;

# HSTS (2 年)
add_header Strict-Transport-Security "max-age=63072000" always;

# OCSP Stapling
ssl_stapling on;
ssl_stapling_verify on;
resolver 8.8.8.8 8.8.4.4 valid=300s;
resolver_timeout 5s;
```

## 测试 SSL 配置

使用 SSL Labs 测试你的配置：

```
https://www.ssllabs.com/ssltest/analyze.html?d=vstats.example.com
```

目标是 A+ 评级。

