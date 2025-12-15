# 反向代理

生产环境部署建议设置反向代理。它提供 SSL 终止、缓存和额外的安全性。

## Nginx

### 基本配置

```nginx
server {
    listen 80;
    server_name vstats.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name vstats.example.com;

    ssl_certificate /etc/letsencrypt/live/vstats.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/vstats.example.com/privkey.pem;

    # SSL 设置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        proxy_pass http://127.0.0.1:3001;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持
    location /ws {
        proxy_pass http://127.0.0.1:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }
}
```

### 带缓存

```nginx
# 为静态资源添加缓存
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
    proxy_pass http://127.0.0.1:3001;
    proxy_cache_valid 200 7d;
    add_header Cache-Control "public, immutable";
    expires 7d;
}
```

## Caddy

Caddy 是最简单的选项，带自动 HTTPS。

### Caddyfile

```caddyfile
vstats.example.com {
    reverse_proxy localhost:3001

    # WebSocket
    @websockets {
        header Connection *Upgrade*
        header Upgrade websocket
    }
    reverse_proxy @websockets localhost:3001
}
```

### 配合 Docker

```yaml
version: '3.8'

services:
  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config

  vstats:
    image: zsai001/vstats-server:latest
    volumes:
      - ./data:/app/data

volumes:
  caddy_data:
  caddy_config:
```

## Traefik

### docker-compose.yml

```yaml
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt

  vstats:
    image: zsai001/vstats-server:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.vstats.rule=Host(`vstats.example.com`)"
      - "traefik.http.routers.vstats.entrypoints=websecure"
      - "traefik.http.routers.vstats.tls.certresolver=letsencrypt"
    volumes:
      - ./data:/app/data
```

## Cloudflare

如果使用 Cloudflare：

1. **SSL 模式**：在 SSL/TLS 设置中设置为"Full (strict)"
2. **WebSocket**：在网络设置中启用 WebSocket 支持
3. **代理状态**：确保 DNS 记录已代理（橙色云）

::: warning 注意
Cloudflare 默认的 WebSocket 超时时间为 100 秒。这可能影响长时间空闲的连接。
:::

