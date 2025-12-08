#!/bin/bash
# ===========================================
# VStats Cloud - API Container Diagnostic Script
# ===========================================

set -e

API_CONTAINER="vstats-api"
POSTGRES_CONTAINER="vstats-postgres"
REDIS_CONTAINER="vstats-redis"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}VStats Cloud API Container Diagnostic${NC}"
echo -e "${BLUE}============================================${NC}"

# ==========================================
# 1. Check Container Status
# ==========================================
echo -e "\n${YELLOW}[1] Container Status${NC}"
echo "----------------------------------------"

if docker ps --format '{{.Names}}' | grep -q "^${API_CONTAINER}$"; then
    echo -e "${GREEN}✓${NC} Container is running"
    
    # Get container status
    STATUS=$(docker inspect --format='{{.State.Status}}' ${API_CONTAINER} 2>/dev/null)
    HEALTH=$(docker inspect --format='{{.State.Health.Status}}' ${API_CONTAINER} 2>/dev/null || echo "no-healthcheck")
    RESTART_COUNT=$(docker inspect --format='{{.RestartCount}}' ${API_CONTAINER} 2>/dev/null)
    
    echo "  Status: $STATUS"
    echo "  Health: $HEALTH"
    echo "  Restarts: $RESTART_COUNT"
    
    if [ "$HEALTH" = "unhealthy" ]; then
        echo -e "  ${RED}⚠ Container is unhealthy!${NC}"
    fi
else
    echo -e "${RED}✗${NC} Container is not running"
    echo "  Run: docker compose up -d"
    exit 1
fi

# ==========================================
# 2. Check Container Logs
# ==========================================
echo -e "\n${YELLOW}[2] Recent Container Logs (last 30 lines)${NC}"
echo "----------------------------------------"
docker logs --tail 30 ${API_CONTAINER} 2>&1 | head -30

# ==========================================
# 3. Check Health Endpoint
# ==========================================
echo -e "\n${YELLOW}[3] Health Check Endpoint${NC}"
echo "----------------------------------------"

if docker exec ${API_CONTAINER} wget -q -O- http://localhost:3001/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} /health endpoint is responding"
    RESPONSE=$(docker exec ${API_CONTAINER} wget -q -O- http://localhost:3001/health 2>/dev/null || echo "")
    echo "  Response: $RESPONSE"
else
    echo -e "${RED}✗${NC} /health endpoint is not responding"
    echo "  This usually means the application failed to start"
fi

# ==========================================
# 4. Check Detailed Health
# ==========================================
echo -e "\n${YELLOW}[4] Detailed Health Status${NC}"
echo "----------------------------------------"

DETAILED=$(docker exec ${API_CONTAINER} wget -q -O- http://localhost:3001/health/detailed 2>/dev/null || echo "")
if [ -n "$DETAILED" ]; then
    echo "$DETAILED" | python3 -m json.tool 2>/dev/null || echo "$DETAILED"
    
    # Extract status
    OVERALL_STATUS=$(echo "$DETAILED" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "unknown")
    DB_STATUS=$(echo "$DETAILED" | grep -o '"database":{[^}]*}' | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
    REDIS_STATUS=$(echo "$DETAILED" | grep -o '"redis":{[^]}"*}' | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
    
    echo ""
    echo "Summary:"
    echo "  Overall: $OVERALL_STATUS"
    echo "  Database: $DB_STATUS"
    echo "  Redis: $REDIS_STATUS"
else
    echo -e "${RED}✗${NC} Cannot get detailed health status"
fi

# ==========================================
# 5. Check Environment Variables
# ==========================================
echo -e "\n${YELLOW}[5] Environment Variables${NC}"
echo "----------------------------------------"

ENV_VARS=$(docker exec ${API_CONTAINER} env 2>/dev/null | grep -E "DATABASE_URL|REDIS_URL|PORT|JWT_SECRET|SESSION_SECRET" || echo "")
if [ -n "$ENV_VARS" ]; then
    echo "$ENV_VARS" | while IFS= read -r line; do
        KEY=$(echo "$line" | cut -d'=' -f1)
        VALUE=$(echo "$line" | cut -d'=' -f2-)
        
        # Mask sensitive values
        if [[ "$KEY" == *"PASSWORD"* ]] || [[ "$KEY" == *"SECRET"* ]] || [[ "$KEY" == *"TOKEN"* ]]; then
            VALUE="***hidden***"
        fi
        
        echo "  $KEY=$VALUE"
    done
else
    echo -e "${RED}✗${NC} Cannot read environment variables"
fi

# ==========================================
# 6. Check Database Connection
# ==========================================
echo -e "\n${YELLOW}[6] Database Connection Test${NC}"
echo "----------------------------------------"

if docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$"; then
    if docker exec ${POSTGRES_CONTAINER} pg_isready -U vstats -d vstats_cloud > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} PostgreSQL is ready"
        
        # Test connection from API container
        if docker exec ${API_CONTAINER} wget -q -O- http://localhost:3001/health/detailed 2>/dev/null | grep -q '"database".*"status":"up"'; then
            echo -e "${GREEN}✓${NC} API can connect to database"
        else
            echo -e "${RED}✗${NC} API cannot connect to database"
            echo "  Check DATABASE_URL environment variable"
        fi
    else
        echo -e "${RED}✗${NC} PostgreSQL is not ready"
    fi
else
    echo -e "${RED}✗${NC} PostgreSQL container is not running"
fi

# ==========================================
# 7. Check Redis Connection
# ==========================================
echo -e "\n${YELLOW}[7] Redis Connection Test${NC}"
echo "----------------------------------------"

if docker ps --format '{{.Names}}' | grep -q "^${REDIS_CONTAINER}$"; then
    if docker exec ${REDIS_CONTAINER} redis-cli -a vstats_redis_password ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Redis is responding"
        
        # Test connection from API container
        if docker exec ${API_CONTAINER} wget -q -O- http://localhost:3001/health/detailed 2>/dev/null | grep -q '"redis".*"status":"up"'; then
            echo -e "${GREEN}✓${NC} API can connect to Redis"
        else
            echo -e "${RED}✗${NC} API cannot connect to Redis"
            echo "  Check REDIS_URL environment variable and password"
        fi
    else
        echo -e "${RED}✗${NC} Redis is not responding"
        echo "  Check Redis password in .env file"
    fi
else
    echo -e "${RED}✗${NC} Redis container is not running"
fi

# ==========================================
# 8. Check Port Binding
# ==========================================
echo -e "\n${YELLOW}[8] Port and Network${NC}"
echo "----------------------------------------"

# Check if port is listening
if docker exec ${API_CONTAINER} netstat -tuln 2>/dev/null | grep -q ":3001" || \
   docker exec ${API_CONTAINER} ss -tuln 2>/dev/null | grep -q ":3001"; then
    echo -e "${GREEN}✓${NC} Port 3001 is listening"
else
    echo -e "${RED}✗${NC} Port 3001 is not listening"
    echo "  Application may not have started"
fi

# Check network connectivity
if docker exec ${API_CONTAINER} ping -c 1 postgres > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Can reach PostgreSQL container"
else
    echo -e "${RED}✗${NC} Cannot reach PostgreSQL container"
fi

if docker exec ${API_CONTAINER} ping -c 1 redis > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Can reach Redis container"
else
    echo -e "${RED}✗${NC} Cannot reach Redis container"
fi

# ==========================================
# 9. Check Health Check History
# ==========================================
echo -e "\n${YELLOW}[9] Health Check History${NC}"
echo "----------------------------------------"

HEALTH_LOGS=$(docker inspect ${API_CONTAINER} --format='{{range .State.Health.Log}}{{.Output}}{{end}}' 2>/dev/null || echo "")
if [ -n "$HEALTH_LOGS" ]; then
    echo "$HEALTH_LOGS" | tail -5
else
    echo "No health check logs available"
fi

# ==========================================
# Summary and Recommendations
# ==========================================
echo -e "\n${BLUE}============================================${NC}"
echo -e "${BLUE}Diagnostic Summary${NC}"
echo -e "${BLUE}============================================${NC}"

if [ "$HEALTH" = "unhealthy" ]; then
    echo -e "\n${RED}Container is unhealthy. Common fixes:${NC}"
    echo ""
    echo "1. Check application logs:"
    echo "   docker logs ${API_CONTAINER} --tail 50"
    echo ""
    echo "2. Verify environment variables in .env file:"
    echo "   - POSTGRES_PASSWORD"
    echo "   - REDIS_PASSWORD"
    echo "   - JWT_SECRET"
    echo "   - SESSION_SECRET"
    echo ""
    echo "3. Ensure database and Redis are healthy:"
    echo "   docker compose ps"
    echo ""
    echo "4. Restart the API container:"
    echo "   docker compose restart api"
    echo ""
    echo "5. If issues persist, check full logs:"
    echo "   docker compose logs api"
else
    echo -e "\n${GREEN}Container appears to be healthy!${NC}"
fi

echo ""
