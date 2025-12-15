# Site 部署指南

本文档说明如何配置 GitHub Actions 自动部署 site 到 Linux 服务器。

## 前置要求

1. Linux 服务器（已配置 SSH 访问）
2. Web 服务器（Nginx、Apache 或其他）
3. GitHub 仓库的 Secrets 配置权限

## GitHub Secrets 配置

在 GitHub 仓库设置中添加以下 Secrets：

### 必需配置

- `DEPLOY_HOST`: 服务器 IP 地址或域名（例如：`192.168.1.100` 或 `docs.example.com`）
- `DEPLOY_USER`: SSH 用户名（例如：`deploy` 或 `root`）
- `DEPLOY_SSH_KEY`: SSH 私钥（完整的私钥内容，包括 `-----BEGIN OPENSSH PRIVATE KEY-----` 和 `-----END OPENSSH PRIVATE KEY-----`）

### 可选配置

- `DEPLOY_PORT`: SSH 端口（默认：`22`）
- `DEPLOY_DIR`: 部署目录（默认：`/var/www/vstats-docs`）

## SSH 密钥生成

在本地生成 SSH 密钥对：

```bash
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/vstats_deploy
```

将公钥添加到服务器的 `~/.ssh/authorized_keys`：

```bash
ssh-copy-id -i ~/.ssh/vstats_deploy.pub user@your-server
```

将私钥内容复制到 GitHub Secrets 的 `DEPLOY_SSH_KEY`。

## 服务器配置

### 1. 创建部署目录

```bash
sudo mkdir -p /var/www/vstats-docs
sudo chown -R $USER:$USER /var/www/vstats-docs
```

### 2. 配置 Nginx（示例）

创建 `/etc/nginx/sites-available/vstats-docs`：

```nginx
server {
    listen 80;
    server_name vstats.zsoft.cc;
    
    root /var/www/vstats-docs;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # 确保脚本文件可访问
    location ~ \.(sh|ps1)$ {
        add_header Content-Type text/plain;
        add_header Content-Disposition inline;
    }
    
    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

启用配置：

```bash
sudo ln -s /etc/nginx/sites-available/vstats-docs /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### 3. 配置 SSL（可选，推荐）

使用 Let's Encrypt：

```bash
sudo apt-get install certbot python3-certbot-nginx
sudo certbot --nginx -d vstats.zsoft.cc
```

## OAuth 配置

site 使用统一的 vStats OAuth 2.0 代理进行认证：

- **代理 URL**: `https://auth.vstats.zsoft.cc`
- **支持的提供商**: GitHub、Google
- **无需任何配置**：直接使用默认代理即可

## 部署流程

1. 推送代码到 `main` 分支
2. GitHub Actions 自动触发构建
3. 构建完成后通过 SSH 部署到服务器
4. 自动备份旧版本
5. 部署新文件
6. 重新加载 Web 服务器

## 手动触发部署

在 GitHub Actions 页面，点击 "Deploy Docs Site to Linux Server" workflow，然后点击 "Run workflow" 按钮。

## 故障排查

### 部署失败

1. 检查 SSH 连接：
   ```bash
   ssh -i ~/.ssh/vstats_deploy user@your-server
   ```

2. 检查服务器权限：
   ```bash
   ls -la /var/www/vstats-docs
   ```

3. 查看 GitHub Actions 日志

### OAuth 登录失败

1. 确认 OAuth 代理 URL 正确
2. 检查浏览器控制台错误
3. 验证 OAuth 代理服务是否正常运行

## 备份和回滚

每次部署会自动创建备份，备份目录格式：`/var/www/vstats-docs.backup.YYYYMMDD_HHMMSS`

手动回滚：

```bash
sudo rm -rf /var/www/vstats-docs
sudo cp -r /var/www/vstats-docs.backup.YYYYMMDD_HHMMSS /var/www/vstats-docs
sudo systemctl reload nginx
```
