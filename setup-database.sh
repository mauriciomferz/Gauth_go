#!/bin/bash
# Database setup script for GAuth admin handlers
# This script starts PostgreSQL and runs migrations

set -e

echo "🗄️  GAuth Database Setup"
echo "======================="
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

echo -e "${BLUE}Step 1: Starting PostgreSQL container...${NC}"
docker-compose -f docker-compose.database.yml up -d postgres

echo ""
echo -e "${BLUE}Step 2: Waiting for PostgreSQL to be ready...${NC}"
for i in {1..30}; do
    if docker exec gauth-postgres pg_isready -U postgres -d gauth > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PostgreSQL is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
    if [ $i -eq 30 ]; then
        echo ""
        echo "❌ PostgreSQL failed to start within 30 seconds"
        docker-compose -f docker-compose.database.yml logs postgres
        exit 1
    fi
done

echo ""
echo -e "${BLUE}Step 3: Running database migrations...${NC}"
docker exec -i gauth-postgres psql -U postgres -d gauth < database/migrations/001_admin_handlers_schema.sql

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Migrations completed successfully!${NC}"
else
    echo "❌ Migration failed"
    exit 1
fi

echo ""
echo -e "${BLUE}Step 4: Verifying tables...${NC}"
TABLE_COUNT=$(docker exec gauth-postgres psql -U postgres -d gauth -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')
echo -e "   Tables created: ${GREEN}${TABLE_COUNT}${NC}"

if [ "$TABLE_COUNT" -ge "17" ]; then
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
echo "  Database: gauth"
echo "  User:     postgres"
echo "  Password: gauth_dev_password"
echo ""
echo "To connect with psql:"
echo "  docker exec -it gauth-postgres psql -U postgres -d gauth"
echo ""
echo "To view logs:"
echo "  docker-compose -f docker-compose.database.yml logs -f postgres"
echo ""
echo "To stop the database:"
echo "  docker-compose -f docker-compose.database.yml down"
echo ""
echo "To start with pgAdmin UI:"
echo "  docker-compose -f docker-compose.database.yml --profile with-ui up -d"
echo "  Then open: http://localhost:5050 (admin@gauth.local / admin)"
echo ""
