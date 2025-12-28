#!/bin/bash
set -e

echo "================================================"
echo "GAuth+ Standalone Deployment Script"
echo "================================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
BACKEND_PORT=8080
FRONTEND_PORT=3000
DB_NAME=gauth
DB_USER=gauth
DB_PASSWORD=gauth_dev_password

echo -e "${YELLOW}Step 1: Generating JWT Signing Key${NC}"
JWT_SIGNING_KEY=$(openssl rand -hex 32)
echo "✓ Generated JWT signing key"
echo ""

echo -e "${YELLOW}Step 2: Checking Prerequisites${NC}"
# Check if PostgreSQL is running
if docker ps | grep -q gauth-postgres; then
    echo "✓ PostgreSQL container running"
else
    echo "✗ PostgreSQL not running, starting..."
    docker compose up -d postgres
    sleep 5
fi

# Check if Redis is running
if docker ps | grep -q gauth-redis; then
    echo "✓ Redis container running"
else
    echo "✗ Redis not running, starting..."
    docker compose up -d redis
    sleep 3
fi
echo ""

echo -e "${YELLOW}Step 3: Stopping old backend${NC}"
docker stop gauth-backend 2>/dev/null || true
docker stop gauth-backend-new 2>/dev/null || true
docker rm gauth-backend-new 2>/dev/null || true
echo "✓ Old backend containers stopped"
echo ""

echo -e "${YELLOW}Step 4: Building backend with latest code${NC}"
docker compose build backend
echo "✓ Backend image built"
echo ""

echo -e "${YELLOW}Step 5: Starting backend with proper environment${NC}"
docker run -d \
  --name gauth-backend-new \
  --network gauth_go_gauth-network \
  -p ${BACKEND_PORT}:8080 \
  -e GAUTH_JWT_SIGNING_KEY="${JWT_SIGNING_KEY}" \
  -e GAUTH_DB_HOST=postgres \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_NAME=${DB_NAME} \
  -e DB_USER=${DB_USER} \
  -e DB_PASSWORD=${DB_PASSWORD} \
  -e DB_SSLMODE=disable \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e PORT=8080 \
  -e GIN_MODE=release \
  -e GAUTH_RFC0111_ENABLED=1 \
  -e GAUTH_METRICS_ENABLED=true \
  -e GAUTH_LOG_LEVEL=info \
  -e AUDIT_EXPORT_DIR=/tmp/gauth-audit-exports \
  gauth_go-backend

echo "✓ Backend container started"
echo ""

echo -e "${YELLOW}Step 6: Waiting for backend to be ready${NC}"
for i in {1..30}; do
    if curl -s http://localhost:${BACKEND_PORT}/api/v1/beta/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Backend is healthy and responding${NC}"
        break
    fi
    echo -n "."
    sleep 1
done
echo ""

echo -e "${YELLOW}Step 7: Health Check${NC}"
HEALTH=$(curl -s http://localhost:${BACKEND_PORT}/api/v1/beta/health | jq -r '.status' 2>/dev/null || echo "unknown")
if [ "$HEALTH" = "healthy" ]; then
    echo -e "${GREEN}✓ Backend health check passed${NC}"
else
    echo -e "${RED}✗ Backend health check failed${NC}"
    echo "Logs:"
    docker logs gauth-backend-new --tail=20
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 8: Testing New Features${NC}"

# Test API Key endpoint
echo -n "Testing API key management... "
API_KEY_RESPONSE=$(curl -s -X POST http://localhost:${BACKEND_PORT}/api/v1/admin/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "deployment-test",
    "keyName": "Deployment Verification",
    "description": "Automated deployment test",
    "scopes": ["poa:read"],
    "rateLimitPerMinute": 60,
    "createdBy": "deployment-script"
  }')

if echo "$API_KEY_RESPONSE" | jq -e '.apiKey' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ API key management working${NC}"
else
    echo -e "${YELLOW}⚠ API key endpoint returned: $(echo $API_KEY_RESPONSE | jq -r '.error // "unexpected response"')${NC}"
fi

# Test Audit Export endpoint
echo -n "Testing audit export... "
EXPORT_RESPONSE=$(curl -s -X POST http://localhost:${BACKEND_PORT}/api/v1/admin/audit/export \
  -H "Content-Type: application/json" \
  -d '{
    "format": "json",
    "dateRange": "last-24h",
    "compressed": false
  }')

if echo "$EXPORT_RESPONSE" | jq -e '.jobId' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Audit export working${NC}"
else
    echo -e "${YELLOW}⚠ Audit export endpoint returned: $(echo $EXPORT_RESPONSE | jq -r '.error // "unexpected response"')${NC}"
fi

echo ""
echo "================================================"
echo -e "${GREEN}Deployment Complete!${NC}"
echo "================================================"
echo ""
echo "Backend URL: http://localhost:${BACKEND_PORT}"
echo "Health Check: http://localhost:${BACKEND_PORT}/api/v1/beta/health"
echo "API Docs: http://localhost:${BACKEND_PORT}/api/docs/swagger"
echo ""
echo "Container Name: gauth-backend-new"
echo "View logs: docker logs -f gauth-backend-new"
echo "Stop: docker stop gauth-backend-new"
echo ""
echo "JWT Signing Key (save this securely):"
echo "${JWT_SIGNING_KEY}"
echo ""
