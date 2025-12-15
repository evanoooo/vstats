# Reverse Proxy

Setting up a reverse proxy is recommended for production deployments. It provides SSL termination, caching, and additional security.

## Nginx

### Basic Configuration

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

    # SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # Security headers
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

    # WebSocket support
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

### With Caching

```nginx
# Add caching for static assets
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
    proxy_pass http://127.0.0.1:3001;
    proxy_cache_valid 200 7d;
    add_header Cache-Control "public, immutable";
    expires 7d;
}
```

## Caddy

Caddy is the easiest option with automatic HTTPS.

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

### With Docker

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

## Apache

```apache
<VirtualHost *:443>
    ServerName vstats.example.com
    
    SSLEngine on
    SSLCertificateFile /etc/letsencrypt/live/vstats.example.com/fullchain.pem
    SSLCertificateKeyFile /etc/letsencrypt/live/vstats.example.com/privkey.pem
    
    ProxyPreserveHost On
    ProxyPass / http://127.0.0.1:3001/
    ProxyPassReverse / http://127.0.0.1:3001/
    
    # WebSocket
    RewriteEngine On
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/ws$ ws://127.0.0.1:3001/ws [P,L]
</VirtualHost>
```

Enable required modules:

```bash
sudo a2enmod proxy proxy_http proxy_wstunnel rewrite ssl
sudo systemctl restart apache2
```

## Cloudflare

If using Cloudflare:

1. **SSL Mode**: Set to "Full (strict)" in SSL/TLS settings
2. **WebSocket**: Enable WebSocket support in Network settings
3. **Proxy Status**: Ensure the DNS record is proxied (orange cloud)

::: warning
Cloudflare has a default WebSocket timeout of 100 seconds. This may affect long-idle connections.
:::

