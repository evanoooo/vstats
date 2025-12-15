# Backup & Restore

This guide covers backing up and restoring your vStats installation.

## What to Backup

| Item | Path | Priority |
|------|------|----------|
| Database | `data/vstats.db` | Critical |
| Configuration | `vstats-config.json` | Important |
| SSL certificates | `/etc/letsencrypt/` | Important |

## Quick Backup

### One-liner

```bash
# Create timestamped backup
tar -czvf vstats-backup-$(date +%Y%m%d-%H%M%S).tar.gz data/ vstats-config.json
```

### Docker

```bash
# Stop container (optional but recommended)
docker stop vstats-server

# Backup data directory
tar -czvf vstats-backup-$(date +%Y%m%d-%H%M%S).tar.gz data/

# Start container
docker start vstats-server
```

## Automated Backups

### Backup Script

Create `/opt/vstats/backup.sh`:

```bash
#!/bin/bash

# Configuration
BACKUP_DIR="/opt/vstats/backups"
DATA_DIR="/opt/vstats/data"
RETENTION_DAYS=30

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Create backup
BACKUP_FILE="$BACKUP_DIR/vstats-$(date +%Y%m%d-%H%M%S).tar.gz"
tar -czvf "$BACKUP_FILE" -C /opt/vstats data/ vstats-config.json

# Remove old backups
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup created: $BACKUP_FILE"
```

Make it executable:

```bash
chmod +x /opt/vstats/backup.sh
```

### Cron Job

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /opt/vstats/backup.sh >> /var/log/vstats-backup.log 2>&1
```

### Docker Backup Script

```bash
#!/bin/bash

BACKUP_DIR="/backups/vstats"
CONTAINER="vstats-server"
RETENTION_DAYS=30

mkdir -p "$BACKUP_DIR"

# Create backup while container is running
docker exec "$CONTAINER" sqlite3 /app/data/vstats.db ".backup /app/data/backup.db"
docker cp "$CONTAINER:/app/data/backup.db" "$BACKUP_DIR/vstats-$(date +%Y%m%d-%H%M%S).db"
docker exec "$CONTAINER" rm /app/data/backup.db

# Cleanup old backups
find "$BACKUP_DIR" -name "*.db" -mtime +$RETENTION_DAYS -delete
```

## Remote Backups

### rsync to Remote Server

```bash
rsync -avz /opt/vstats/backups/ user@backup-server:/backups/vstats/
```

### S3 Upload

```bash
# Install AWS CLI
# Configure: aws configure

# Upload backup
aws s3 cp /opt/vstats/backups/ s3://your-bucket/vstats-backups/ --recursive
```

### Rclone (Multiple Cloud Providers)

```bash
# Install rclone and configure

# Sync to cloud storage
rclone sync /opt/vstats/backups/ remote:vstats-backups/
```

## Restore

### Stop Service First

```bash
# Linux
sudo systemctl stop vstats

# Docker
docker stop vstats-server
```

### Restore from Backup

```bash
# Extract backup
tar -xzvf vstats-backup-20240101-020000.tar.gz

# Move to correct location
cp -r data/* /opt/vstats/data/
cp vstats-config.json /opt/vstats/
```

### Docker Restore

```bash
# Stop container
docker stop vstats-server

# Extract backup to data directory
tar -xzvf vstats-backup-20240101-020000.tar.gz -C /path/to/data/

# Start container
docker start vstats-server
```

### Start Service

```bash
# Linux
sudo systemctl start vstats

# Docker
docker start vstats-server
```

### Verify Restore

```bash
# Check service status
systemctl status vstats

# Test health endpoint
curl http://localhost:3001/health

# Check logs for errors
journalctl -u vstats -n 50
```

## Database-Only Backup

For minimal backups focusing on just the database:

### SQLite Backup

```bash
# Online backup (safe while running)
sqlite3 /opt/vstats/data/vstats.db ".backup /tmp/vstats-backup.db"

# Or use dump
sqlite3 /opt/vstats/data/vstats.db ".dump" > /tmp/vstats-backup.sql
```

### Restore from SQL Dump

```bash
sqlite3 /opt/vstats/data/vstats.db < /tmp/vstats-backup.sql
```

## Best Practices

1. **Regular backups**: Daily at minimum
2. **Test restores**: Periodically verify backups work
3. **Multiple locations**: Keep backups in different locations
4. **Encryption**: Encrypt sensitive backups
5. **Retention policy**: Keep backups for at least 30 days
6. **Monitor**: Alert if backups fail

