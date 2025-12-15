#!/bin/bash
# ===========================================
# VStats Cloud - Database Backup Script
# ===========================================
# 
# Usage: ./backup.sh [backup_dir]
# 
# This script creates a backup of PostgreSQL and Redis data.
# Run this script regularly via cron for automated backups.

set -e

# Configuration
BACKUP_DIR="${1:-./backups}"
DATE=$(date +%Y%m%d_%H%M%S)
POSTGRES_CONTAINER="vstats-postgres"
REDIS_CONTAINER="vstats-redis"

# Load environment variables
if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

POSTGRES_USER="${POSTGRES_USER:-vstats}"
POSTGRES_DB="${POSTGRES_DB:-vstats_cloud}"
REDIS_PASSWORD="${REDIS_PASSWORD:-vstats_redis_password}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}VStats Cloud Backup - ${DATE}${NC}"
echo -e "${GREEN}============================================${NC}"

# Create backup directory
mkdir -p "${BACKUP_DIR}"

# ==========================================
# PostgreSQL Backup
# ==========================================
echo -e "\n${YELLOW}[1/3] Backing up PostgreSQL...${NC}"

PG_BACKUP_FILE="${BACKUP_DIR}/postgres_${DATE}.sql.gz"

if docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$"; then
    docker exec ${POSTGRES_CONTAINER} pg_dump -U ${POSTGRES_USER} ${POSTGRES_DB} | gzip > "${PG_BACKUP_FILE}"
    
    if [ -f "${PG_BACKUP_FILE}" ]; then
        PG_SIZE=$(du -h "${PG_BACKUP_FILE}" | cut -f1)
        echo -e "${GREEN}✓ PostgreSQL backup complete: ${PG_BACKUP_FILE} (${PG_SIZE})${NC}"
    else
        echo -e "${RED}✗ PostgreSQL backup failed!${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ PostgreSQL container not running!${NC}"
    exit 1
fi

# ==========================================
# Redis Backup
# ==========================================
echo -e "\n${YELLOW}[2/3] Backing up Redis...${NC}"

REDIS_BACKUP_FILE="${BACKUP_DIR}/redis_${DATE}.rdb"

if docker ps --format '{{.Names}}' | grep -q "^${REDIS_CONTAINER}$"; then
    # Trigger RDB save
    docker exec ${REDIS_CONTAINER} redis-cli -a ${REDIS_PASSWORD} BGSAVE 2>/dev/null
    sleep 2
    
    # Copy dump file
    docker cp ${REDIS_CONTAINER}:/data/dump.rdb "${REDIS_BACKUP_FILE}" 2>/dev/null || true
    
    if [ -f "${REDIS_BACKUP_FILE}" ]; then
        REDIS_SIZE=$(du -h "${REDIS_BACKUP_FILE}" | cut -f1)
        echo -e "${GREEN}✓ Redis backup complete: ${REDIS_BACKUP_FILE} (${REDIS_SIZE})${NC}"
    else
        echo -e "${YELLOW}⚠ Redis backup skipped (no data or AOF only)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Redis container not running, skipping...${NC}"
fi

# ==========================================
# Cleanup old backups (keep last 7 days)
# ==========================================
echo -e "\n${YELLOW}[3/3] Cleaning up old backups...${NC}"

RETENTION_DAYS=7
find "${BACKUP_DIR}" -name "postgres_*.sql.gz" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true
find "${BACKUP_DIR}" -name "redis_*.rdb" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true

BACKUP_COUNT=$(ls -1 "${BACKUP_DIR}"/*.sql.gz 2>/dev/null | wc -l || echo "0")
echo -e "${GREEN}✓ Cleanup complete. Total backups: ${BACKUP_COUNT}${NC}"

# ==========================================
# Summary
# ==========================================
echo -e "\n${GREEN}============================================${NC}"
echo -e "${GREEN}Backup completed successfully!${NC}"
echo -e "${GREEN}============================================${NC}"
echo -e "Backup directory: ${BACKUP_DIR}"
echo -e "PostgreSQL: ${PG_BACKUP_FILE}"
if [ -f "${REDIS_BACKUP_FILE}" ]; then
    echo -e "Redis: ${REDIS_BACKUP_FILE}"
fi
echo ""

# Cron hint
echo -e "${YELLOW}To schedule daily backups, add to crontab:${NC}"
echo -e "0 3 * * * cd $(pwd) && ./scripts/backup.sh >> /var/log/vstats-backup.log 2>&1"
