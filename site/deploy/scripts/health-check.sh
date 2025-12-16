#!/bin/bash
# ===========================================
# VStats Cloud - Health Check Script
# ===========================================

set -e

# Configuration
POSTGRES_CONTAINER="vstats-postgres"
REDIS_CONTAINER="vstats-redis"
API_CONTAINER="vstats-api"

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
echo -e "${GREEN}VStats Cloud Health Check${NC}"
echo -e "${GREEN}============================================${NC}"

STATUS=0

# ==========================================
# Check PostgreSQL
# ==========================================
echo -e "\n${YELLOW}[PostgreSQL]${NC}"

if docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$"; then
    echo -e "  Container: ${GREEN}Running${NC}"
    
    # Check connection
    if docker exec ${POSTGRES_CONTAINER} pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB} > /dev/null 2>&1; then
        echo -e "  Connection: ${GREEN}OK${NC}"
        
        # Get some stats
        USER_COUNT=$(docker exec ${POSTGRES_CONTAINER} psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -t -c "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;" 2>/dev/null | tr -d ' ')
        SERVER_COUNT=$(docker exec ${POSTGRES_CONTAINER} psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -t -c "SELECT COUNT(*) FROM servers WHERE deleted_at IS NULL;" 2>/dev/null | tr -d ' ')
        
        echo -e "  Users: ${USER_COUNT:-0}"
        echo -e "  Servers: ${SERVER_COUNT:-0}"
    else
        echo -e "  Connection: ${RED}Failed${NC}"
        STATUS=1
    fi
else
    echo -e "  Container: ${RED}Not Running${NC}"
    STATUS=1
fi

# ==========================================
# Check Redis
# ==========================================
echo -e "\n${YELLOW}[Redis]${NC}"

if docker ps --format '{{.Names}}' | grep -q "^${REDIS_CONTAINER}$"; then
    echo -e "  Container: ${GREEN}Running${NC}"
    
    # Check connection
    if docker exec ${REDIS_CONTAINER} redis-cli -a ${REDIS_PASSWORD} ping > /dev/null 2>&1; then
        echo -e "  Connection: ${GREEN}OK${NC}"
        
        # Get memory usage
        MEM_USED=$(docker exec ${REDIS_CONTAINER} redis-cli -a ${REDIS_PASSWORD} INFO memory 2>/dev/null | grep "used_memory_human" | cut -d: -f2 | tr -d '\r')
        KEYS=$(docker exec ${REDIS_CONTAINER} redis-cli -a ${REDIS_PASSWORD} DBSIZE 2>/dev/null | awk '{print $2}')
        
        echo -e "  Memory: ${MEM_USED:-N/A}"
        echo -e "  Keys: ${KEYS:-0}"
    else
        echo -e "  Connection: ${RED}Failed${NC}"
        STATUS=1
    fi
else
    echo -e "  Container: ${RED}Not Running${NC}"
    STATUS=1
fi

# ==========================================
# Check API Server
# ==========================================
echo -e "\n${YELLOW}[API Server]${NC}"

if docker ps --format '{{.Names}}' | grep -q "^${API_CONTAINER}$"; then
    echo -e "  Container: ${GREEN}Running${NC}"
    
    # Check container health status
    HEALTH=$(docker inspect --format='{{.State.Health.Status}}' ${API_CONTAINER} 2>/dev/null || echo "unknown")
    if [ "$HEALTH" = "healthy" ]; then
        echo -e "  Health: ${GREEN}Healthy${NC}"
    elif [ "$HEALTH" = "unhealthy" ]; then
        echo -e "  Health: ${RED}Unhealthy${NC}"
        STATUS=1
    else
        echo -e "  Health: ${YELLOW}${HEALTH}${NC}"
    fi
    
    # Check if API is responding
    if docker exec ${API_CONTAINER} wget -q -O- http://localhost:3100/health > /dev/null 2>&1; then
        echo -e "  Endpoint: ${GREEN}OK${NC}"
        
        # Get detailed health
        DETAILED=$(docker exec ${API_CONTAINER} wget -q -O- http://localhost:3100/health/detailed 2>/dev/null || echo "")
        if [ -n "$DETAILED" ]; then
            DB_STATUS=$(echo "$DETAILED" | grep -o '"database":{[^}]*}' | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
            REDIS_STATUS=$(echo "$DETAILED" | grep -o '"redis":{[^}]*}' | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
            
            if [ "$DB_STATUS" = "up" ]; then
                echo -e "  Database Connection: ${GREEN}OK${NC}"
            else
                echo -e "  Database Connection: ${RED}Failed${NC}"
                STATUS=1
            fi
            
            if [ "$REDIS_STATUS" = "up" ]; then
                echo -e "  Redis Connection: ${GREEN}OK${NC}"
            else
                echo -e "  Redis Connection: ${RED}Failed${NC}"
                STATUS=1
            fi
        fi
    else
        echo -e "  Endpoint: ${RED}Not Responding${NC}"
        STATUS=1
    fi
else
    echo -e "  Container: ${RED}Not Running${NC}"
    STATUS=1
fi

# ==========================================
# Check Docker Volumes
# ==========================================
echo -e "\n${YELLOW}[Docker Volumes]${NC}"

PG_VOLUME=$(docker volume inspect vstats_postgres_data 2>/dev/null | grep -o '"Mountpoint": "[^"]*"' | cut -d'"' -f4)
REDIS_VOLUME=$(docker volume inspect vstats_redis_data 2>/dev/null | grep -o '"Mountpoint": "[^"]*"' | cut -d'"' -f4)

if [ -n "$PG_VOLUME" ]; then
    echo -e "  PostgreSQL Volume: ${GREEN}OK${NC}"
else
    echo -e "  PostgreSQL Volume: ${YELLOW}Not Found${NC}"
fi

if [ -n "$REDIS_VOLUME" ]; then
    echo -e "  Redis Volume: ${GREEN}OK${NC}"
else
    echo -e "  Redis Volume: ${YELLOW}Not Found${NC}"
fi

# ==========================================
# Summary
# ==========================================
echo -e "\n${GREEN}============================================${NC}"
if [ $STATUS -eq 0 ]; then
    echo -e "${GREEN}All services are healthy!${NC}"
else
    echo -e "${RED}Some services have issues!${NC}"
fi
echo -e "${GREEN}============================================${NC}"

exit $STATUS
