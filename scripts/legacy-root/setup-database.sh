#!/usr/bin/env bash
# Database setup script for AgentAuth admin handlers
# Starts PostgreSQL (via compose) and runs migrations.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deployments/docker/docker-compose.database.yml"
MIGRATION_FILE="$ROOT_DIR/database/migrations/001_admin_handlers_schema.sql"

echo "🗄️  AgentAuth Database Setup"
echo "==========================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
  echo "❌ Docker is not running. Please start Docker and try again."
  exit 1
fi

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "❌ Compose file not found: $COMPOSE_FILE" >&2
  exit 1
fi

if [[ ! -f "$MIGRATION_FILE" ]]; then
  echo "❌ Migration file not found: $MIGRATION_FILE" >&2
  exit 1
fi

echo -e "${BLUE}Step 1: Starting PostgreSQL container...${NC}"
docker compose -f "$COMPOSE_FILE" up -d postgres

echo ""
echo -e "${BLUE}Step 2: Waiting for PostgreSQL to be ready...${NC}"
for i in {1..30}; do
  if docker exec agentauth-postgres pg_isready -U postgres -d agentauth > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PostgreSQL is ready!${NC}"
    break
  fi
  echo -n "."
  sleep 1
  if [[ $i -eq 30 ]]; then
    echo ""
    echo "❌ PostgreSQL failed to start within 30 seconds" >&2
    docker compose -f "$COMPOSE_FILE" logs postgres
    exit 1
  fi
done

echo ""
echo -e "${BLUE}Step 3: Running database migrations...${NC}"
docker exec -i agentauth-postgres psql -U postgres -d agentauth < "$MIGRATION_FILE"

echo ""
echo -e "${BLUE}Step 4: Verifying tables...${NC}"
TABLE_COUNT=$(docker exec agentauth-postgres psql -U postgres -d agentauth -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')
echo -e "   Tables created: ${GREEN}${TABLE_COUNT}${NC}"

if [[ "$TABLE_COUNT" -ge "17" ]]; then
  echo -e "${GREEN}✅ All required tables created!${NC}"
else
  echo -e "${YELLOW}⚠️  Expected at least 17 tables, found ${TABLE_COUNT}${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Database setup complete!${NC}"
echo ""
echo "Database connection details:"
echo "  Host:     localhost"
echo "  Port:     5432"
echo "  Database: agentauth"
echo "  User:     postgres"
echo "  Password: agentauth_dev_password"
echo ""
echo "To connect with psql:"
echo "  docker exec -it agentauth-postgres psql -U postgres -d agentauth"
echo ""
echo "To view logs:"
echo "  docker compose -f $COMPOSE_FILE logs -f postgres"
echo ""
echo "To stop the database:"
echo "  docker compose -f $COMPOSE_FILE down"
echo ""
echo "To start with pgAdmin UI:"
echo "  docker compose -f $COMPOSE_FILE --profile with-ui up -d"
echo "  Then open: http://localhost:5050 (admin@agentauth.local / admin)"
echo ""
