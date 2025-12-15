# 备份与恢复

本指南介绍备份和恢复 vStats 安装。

## 需要备份的内容

| 项目 | 路径 | 优先级 |
|------|------|--------|
| 数据库 | `data/vstats.db` | 关键 |
| 配置文件 | `vstats-config.json` | 重要 |
| SSL 证书 | `/etc/letsencrypt/` | 重要 |

## 快速备份

### 一行命令

```bash
# 创建带时间戳的备份
tar -czvf vstats-backup-$(date +%Y%m%d-%H%M%S).tar.gz data/ vstats-config.json
```

### Docker

```bash
# 停止容器（可选但推荐）
docker stop vstats-server

# 备份数据目录
tar -czvf vstats-backup-$(date +%Y%m%d-%H%M%S).tar.gz data/

# 启动容器
docker start vstats-server
```

## 自动备份

### 备份脚本

创建 `/opt/vstats/backup.sh`：

```bash
#!/bin/bash

# 配置
BACKUP_DIR="/opt/vstats/backups"
DATA_DIR="/opt/vstats/data"
RETENTION_DAYS=30

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 创建备份
BACKUP_FILE="$BACKUP_DIR/vstats-$(date +%Y%m%d-%H%M%S).tar.gz"
tar -czvf "$BACKUP_FILE" -C /opt/vstats data/ vstats-config.json

# 删除旧备份
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete

echo "备份已创建: $BACKUP_FILE"
```

设置执行权限：

```bash
chmod +x /opt/vstats/backup.sh
```

### Cron 任务

```bash
# 编辑 crontab
crontab -e

# 添加每天凌晨 2 点备份
0 2 * * * /opt/vstats/backup.sh >> /var/log/vstats-backup.log 2>&1
```

## 远程备份

### rsync 到远程服务器

```bash
rsync -avz /opt/vstats/backups/ user@backup-server:/backups/vstats/
```

### 上传到 S3

```bash
# 安装 AWS CLI
# 配置: aws configure

# 上传备份
aws s3 cp /opt/vstats/backups/ s3://your-bucket/vstats-backups/ --recursive
```

## 恢复

### 先停止服务

```bash
# Linux
sudo systemctl stop vstats

# Docker
docker stop vstats-server
```

### 从备份恢复

```bash
# 解压备份
tar -xzvf vstats-backup-20240101-020000.tar.gz

# 移动到正确位置
cp -r data/* /opt/vstats/data/
cp vstats-config.json /opt/vstats/
```

### 启动服务

```bash
# Linux
sudo systemctl start vstats

# Docker
docker start vstats-server
```

### 验证恢复

```bash
# 检查服务状态
systemctl status vstats

# 测试健康端点
curl http://localhost:3001/health

# 检查日志是否有错误
journalctl -u vstats -n 50
```

## 最佳实践

1. **定期备份**：至少每天一次
2. **测试恢复**：定期验证备份可用
3. **多地点**：在不同位置保留备份
4. **加密**：加密敏感备份
5. **保留策略**：至少保留 30 天的备份
6. **监控**：备份失败时告警

