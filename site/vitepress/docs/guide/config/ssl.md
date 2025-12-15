# SSL/HTTPS Configuration

Securing your vStats installation with SSL/HTTPS is essential for production deployments.

## Recommended: Reverse Proxy

The recommended approach is to use a reverse proxy for SSL termination:

- [Nginx Configuration](/guide/config/reverse-proxy#nginx)
- [Caddy Configuration](/guide/config/reverse-proxy#caddy)
- [Traefik Configuration](/guide/config/reverse-proxy#traefik)

## Let's Encrypt with Certbot

### Install Certbot

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

### Obtain Certificate

```bash
# Standalone mode (stop web server first)
sudo certbot certonly --standalone -d vstats.example.com

# Or webroot mode (if web server is running)
sudo certbot certonly --webroot -w /var/www/html -d vstats.example.com
```

### Certificate Files

Certificates are stored in:

```
/etc/letsencrypt/live/vstats.example.com/
├── fullchain.pem  # Certificate + intermediate
├── privkey.pem    # Private key
├── cert.pem       # Certificate only
└── chain.pem      # Intermediate certificates
```

### Auto-Renewal

Certbot sets up automatic renewal. Test it with:

```bash
sudo certbot renew --dry-run
```

## Direct SSL Configuration

If you prefer not to use a reverse proxy, vStats supports direct SSL:

### Configuration File

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

### Docker with SSL

```bash
docker run -d \
  --name vstats-server \
  -p 443:443 \
  -e PORT=443 \
  -v $(pwd)/data:/app/data \
  -v /etc/letsencrypt:/etc/letsencrypt:ro \
  zsai001/vstats-server:latest
```

With configuration:

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

## Self-Signed Certificates

For testing or internal use:

```bash
# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout vstats.key \
  -out vstats.crt \
  -subj "/CN=vstats.local"
```

::: warning
Self-signed certificates will show browser warnings. Agents connecting to servers with self-signed certificates need `insecure_skip_verify: true` in their configuration.
:::

## Agent Configuration for HTTPS

When your server uses HTTPS, update agent configuration:

```bash
# Agent installation with HTTPS server
curl -fsSL https://vstats.zsoft.cc/agent.sh | sudo bash -s -- \
  --server https://vstats.example.com \
  --token "your-token"
```

## Security Best Practices

1. **Use TLS 1.2+**: Disable older protocols
2. **Strong Ciphers**: Use modern cipher suites
3. **HSTS**: Enable HTTP Strict Transport Security
4. **Certificate Renewal**: Automate with certbot or similar

### Nginx SSL Best Practices

```nginx
# Strong SSL configuration
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;

# HSTS (2 years)
add_header Strict-Transport-Security "max-age=63072000" always;

# OCSP Stapling
ssl_stapling on;
ssl_stapling_verify on;
resolver 8.8.8.8 8.8.4.4 valid=300s;
resolver_timeout 5s;
```

## Testing SSL Configuration

Use SSL Labs to test your configuration:

```
https://www.ssllabs.com/ssltest/analyze.html?d=vstats.example.com
```

Aim for an A+ rating.

