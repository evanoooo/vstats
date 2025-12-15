#!/bin/bash
# ===========================================
# VStats Cloud - Database Restore Script
# ===========================================
# 
# Usage: ./restore.sh <postgres_backup.sql.gz> [redis_backup.rdb]
# 
# WARNING: This will overwrite existing data!

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

if [ -z "$1" ]; then
    echo -e "${RED}Usage: $0 <postgres_backup.sql.gz> [redis_backup.rdb]${NC}"
    echo -e "Example: $0 backups/postgres_20240101_030000.sql.gz"
    exit 1
fi

PG_BACKUP_FILE="$1"
REDIS_BACKUP_FILE="$2"

POSTGRES_CONTAINER="vstats-postgres"
REDIS_CONTAINER="vstats-redis"

# Load environment variables
if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

POSTGRES_USER="${POSTGRES_USER:-vstats}"
POSTGRES_DB="${POSTGRES_DB:-vstats_cloud}"
REDIS_PASSWORD="${REDIS_PASSWORD:-vstats_redis_password}"

echo -e "${YELLOW}============================================${NC}"
echo -e "${YELLOW}VStats Cloud Database Restore${NC}"
echo -e "${YELLOW}============================================${NC}"
echo ""
echo -e "${RED}WARNING: This will overwrite existing data!${NC}"
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

# ==========================================
# PostgreSQL Restore
# ==========================================
echo -e "\n${YELLOW}[1/2] Restoring PostgreSQL...${NC}"

if [ ! -f "${PG_BACKUP_FILE}" ]; then
    echo -e "${RED}✗ Backup file not found: ${PG_BACKUP_FILE}${NC}"
    exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$"; then
    echo -e "${RED}✗ PostgreSQL container not running!${NC}"
    exit 1
fi

# Drop and recreate database
echo "Dropping existing database..."
docker exec ${POSTGRES_CONTAINER} psql -U ${POSTGRES_USER} -c "DROP DATABASE IF EXISTS ${POSTGRES_DB};" postgres
docker exec ${POSTGRES_CONTAINER} psql -U ${POSTGRES_USER} -c "CREATE DATABASE ${POSTGRES_DB};" postgres

# Restore backup
echo "Restoring from backup..."
gunzip -c "${PG_BACKUP_FILE}" | docker exec -i ${POSTGRES_CONTAINER} psql -U ${POSTGRES_USER} ${POSTGRES_DB}

echo -e "${GREEN}✓ PostgreSQL restore complete${NC}"

# ==========================================
# Redis Restore (Optional)
# ==========================================
if [ -n "${REDIS_BACKUP_FILE}" ]; then
    echo -e "\n${YELLOW}[2/2] Restoring Redis...${NC}"
    
    if [ ! -f "${REDIS_BACKUP_FILE}" ]; then
        echo -e "${RED}✗ Redis backup file not found: ${REDIS_BACKUP_FILE}${NC}"
        exit 1
    fi
    
    if ! docker ps --format '{{.Names}}' | grep -q "^${REDIS_CONTAINER}$"; then
        echo -e "${YELLOW}⚠ Redis container not running, skipping...${NC}"
    else
        # Stop Redis to replace data file
        docker stop ${REDIS_CONTAINER}
        
        # Copy backup file
        docker cp "${REDIS_BACKUP_FILE}" ${REDIS_CONTAINER}:/data/dump.rdb
        
        # Start Redis
        docker start ${REDIS_CONTAINER}
        
        echo -e "${GREEN}✓ Redis restore complete${NC}"
    fi
else
    echo -e "\n${YELLOW}[2/2] Skipping Redis restore (no backup file specified)${NC}"
fi

# ==========================================
# Summary
# ==========================================
echo -e "\n${GREEN}============================================${NC}"
echo -e "${GREEN}Restore completed successfully!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo "Please restart your application to use the restored data."
