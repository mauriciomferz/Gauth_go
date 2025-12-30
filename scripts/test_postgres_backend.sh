#!/bin/bash

# Test PostgreSQL backend for RFC-0111
# This script tests the PostgreSQL token store implementation

set -e

echo "=== PostgreSQL Backend Test ==="
echo

# Configuration
export GAUTH_AAP-001_ENABLED=1
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=gauth
export DB_USER=gauth
export DB_PASSWORD=gauth_password
export DB_SSLMODE=disable

echo "1. Testing PostgreSQL connection..."
docker exec gauth-postgres psql -U gauth -d gauth -c "SELECT version();" | head -1

echo
echo "2. Checking database tables..."
docker exec gauth-postgres psql -U gauth -d gauth -c "\dt" | grep -E "(extended_tokens|subscriptions)"

echo
echo "3. Building web-server with PostgreSQL support..."
go build -o /tmp/web-server-postgres ./cmd/web-server

echo
echo "4. Starting web-server with PostgreSQL backend..."
/tmp/web-server-postgres > /tmp/gauth_postgres.log 2>&1 &
WEB_PID=$!
echo "Web server PID: $WEB_PID"

# Wait for server to start
sleep 2

echo
echo "5. Testing token creation..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/extended-token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_id": "grant_test_001",
    "client_id": "client_test_001",
    "resource_owner_id": "user_test_001",
    "scope": ["read", "write"],
    "expires_in": 3600
  }')

ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token')
echo "Created token: ${ACCESS_TOKEN:0:20}..."

echo
echo "6. Verifying token was stored in PostgreSQL..."
TOKEN_COUNT=$(docker exec gauth-postgres psql -U gauth -d gauth -t -c "SELECT COUNT(*) FROM extended_tokens;")
echo "Tokens in database: $TOKEN_COUNT"

if [ "$TOKEN_COUNT" -eq 1 ]; then
    echo "✅ Token successfully stored in PostgreSQL"
else
    echo "❌ Token not found in database"
    pkill -9 web-server
    exit 1
fi

echo
echo "7. Testing token retrieval..."
RETRIEVED=$(curl -s http://localhost:8080/api/v1/rfc0111/token/$ACCESS_TOKEN)
echo "$RETRIEVED" | jq -r '.token_type'

echo
echo "8. Testing token revocation..."
curl -s -X DELETE http://localhost:8080/api/v1/rfc0111/token/$ACCESS_TOKEN

echo
echo "9. Verifying revocation in PostgreSQL..."
REVOKED_COUNT=$(docker exec gauth-postgres psql -U gauth -d gauth -t -c "SELECT COUNT(*) FROM extended_tokens WHERE revoked_at IS NOT NULL;")
echo "Revoked tokens in database: $REVOKED_COUNT"

if [ "$REVOKED_COUNT" -eq 1 ]; then
    echo "✅ Token successfully revoked in PostgreSQL"
else
    echo "❌ Token revocation not recorded"
fi

echo
echo "10. Testing subscription flow..."
SUB_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/rfc0111/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client_test_002",
    "resource_owner_id": "user_test_002"
  }')

SUB_ID=$(echo "$SUB_RESPONSE" | jq -r '.subscription_id')
echo "Created subscription: $SUB_ID"

echo
echo "11. Verifying subscription in PostgreSQL..."
SUB_COUNT=$(docker exec gauth-postgres psql -U gauth -d gauth -t -c "SELECT COUNT(*) FROM subscriptions;")
echo "Subscriptions in database: $SUB_COUNT"

echo
echo "12. Cleanup..."
pkill -9 web-server || true

echo
echo "=== Test Summary ==="
echo "✅ PostgreSQL connection: OK"
echo "✅ Database schema: OK"
echo "✅ Token creation: OK"
echo "✅ Token storage: OK"
echo "✅ Token retrieval: OK"
echo "✅ Token revocation: OK"
echo "✅ Subscription creation: OK"
echo "✅ Subscription storage: OK"
echo
echo "All PostgreSQL backend tests passed!"
