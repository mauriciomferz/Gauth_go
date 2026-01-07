---
title: Database Setup Guide
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Database Setup Guide - AgentAuth Admin Handlers

## Option 1: Docker Setup (Recommended)

### Prerequisites
- Docker Desktop installed and running
- Docker Compose installed

### Quick Start
```bash
# Start Docker Desktop first, then run:
./setup-database.sh
```bash
# 1. Start PostgreSQL (canonical compose)
docker compose -f deployments/docker/docker-compose.yml up -d postgres

# 2. Wait for PostgreSQL to be ready
docker compose -f deployments/docker/docker-compose.yml exec postgres pg_isready -U postgres -d agentauth

# 3. Run migrations
docker compose -f deployments/docker/docker-compose.yml exec -T postgres psql -U postgres -d agentauth < database/migrations/001_admin_handlers_schema.sql

# 4. Verify tables
docker compose -f deployments/docker/docker-compose.yml exec postgres psql -U postgres -d agentauth -c "\dt"
```

### Connection Details
```
Host:     localhost
Port:     5432
Database: agentauth
User:     postgres
Password: agentauth_dev_password
```

### Useful Commands
```bash
# Connect to database
docker compose -f deployments/docker/docker-compose.yml exec -it postgres psql -U postgres -d agentauth

# View logs
docker compose -f deployments/docker/docker-compose.yml logs -f postgres

# Stop database
docker compose -f deployments/docker/docker-compose.yml down

# Stop and remove data
docker compose -f deployments/docker/docker-compose.yml down -v
```

## Option 2: Local PostgreSQL Installation

### Prerequisites
- PostgreSQL 14+ installed locally
- psql command-line tool available

### Setup Steps

1. **Create database:**
```bash
createdb agentauth
```

2. **Run migrations:**
```bash
psql -U your_username -d agentauth -f database/migrations/001_admin_handlers_schema.sql
```

3. **Verify tables:**
```bash
psql -U your_username -d agentauth -c "\dt"
```

Expected: 17 tables created.

### Connection Details
Update these environment variables for your local setup:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=your_username
export DB_PASSWORD=your_password
export DB_NAME=agentauth
export DB_SSLMODE=disable
```

## Option 3: Cloud PostgreSQL (AWS RDS, Azure, GCP, etc.)

1. **Create PostgreSQL instance** in your cloud provider
2. **Configure security groups** to allow connections
3. **Get connection details** from cloud console
4. **Run migrations:**
```bash
psql -h your-host.region.rds.amazonaws.com -U admin -d agentauth -f database/migrations/001_admin_handlers_schema.sql
```

5. **Update environment variables:**
```bash
export DB_HOST=your-host.region.rds.amazonaws.com
export DB_PORT=5432
export DB_USER=admin
export DB_PASSWORD=your_secure_password
export DB_NAME=agentauth
export DB_SSLMODE=require  # Required for cloud databases
```

## Database Schema

The migration creates **17 tables** for **5 admin handlers**:

### Core (1 table)
- `subscribers` - Tenant management

### Handler 1: Proof of Authorization (2 tables)
- `power_of_attorney` - PoA records
- `delegation_chains` - Delegation tracking

### Handler 2: Resilience Patterns (4 tables)
- `circuit_breakers` - Circuit breaker state
- `rate_limiters` - Rate limiter configuration
- `retry_policies` - Retry policy configuration
- `bulkheads` - Bulkhead configuration

### Handler 3: Event System (3 tables)
- `events` - Event stream
- `event_subscriptions` - Subscription configuration
- `event_deliveries` - Delivery tracking

### Handler 4: Authorization Engine (3 tables)
- `authorization_policies` - Authorization policies
- `policy_roles` - Policy roles
- `role_permissions` - Role permissions

### Handler 5: Configuration Management (4 tables)
- `config_variables` - Configuration variables
- `config_files` - Configuration files
- `service_configs` - Service configurations
- `feature_flags` - Feature flags

## Security Features

### Row-Level Security (RLS)
All tables have RLS enabled for multi-tenant isolation. The `current_tenant_id()` function controls access.

### Test Tenant
A test tenant is automatically created:
- Tenant ID: `test-tenant-1`
- Tenant Name: `Test Tenant`
- Status: `active`

## Verification

### Check Tables
```sql
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;
```

Expected: 17 tables

### Check RLS Policies
```sql
SELECT schemaname, tablename, policyname 
FROM pg_policies 
WHERE schemaname = 'public';
```

### Check Test Data
```sql
SELECT tenant_id, tenant_name, status 
FROM subscribers;
```

Expected: 1 row with `test-tenant-1`

## Troubleshooting

### Docker Container Won't Start
```bash
# Check if port 5432 is already in use
lsof -i:5432

# Kill existing process if needed
kill -9 <PID>

# Remove old container and try again
docker compose -f deployments/docker/docker-compose.yml down
docker compose -f deployments/docker/docker-compose.yml up -d postgres
```

### Migration Fails
```bash
# Check PostgreSQL logs
docker compose -f deployments/docker/docker-compose.yml logs postgres

# Try running migration manually
docker compose -f deployments/docker/docker-compose.yml exec -it postgres psql -U postgres -d agentauth

# Then in psql:
\i /docker-entrypoint-initdb.d/001_admin_handlers_schema.sql
```

### Connection Refused
- Ensure PostgreSQL is running: `docker compose -f deployments/docker/docker-compose.yml ps postgres`
- Check port mapping: `docker compose -f deployments/docker/docker-compose.yml port postgres 5432`
- Verify network: `docker compose -f deployments/docker/docker-compose.yml ps`

### Permission Denied
```bash
# If using local PostgreSQL, grant permissions:
psql -U postgres -c "CREATE ROLE agentauth_app WITH LOGIN PASSWORD 'change_me';"
psql -U postgres -d agentauth -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO agentauth_app;"
```

## Next Steps

After database setup:

1. **Start AgentAuth server:**
```bash
export AGENTAUTH_JWT_SIGNING_KEY="your-secret-key"
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=agentauth_dev_password
export DB_NAME=agentauth
export DB_SSLMODE=disable

go run ./cmd/web-server
```

2. **Verify admin handlers:**
Look for these logs:
```
[database] PostgreSQL connection established
[admin] handlers registered: poa, resilience, events, authz, config (5 total)
```

3. **Test endpoints:**
```bash
curl http://localhost:8080/api/admin/poa
curl http://localhost:8080/api/admin/resilience/circuit-breakers
curl http://localhost:8080/api/admin/events
curl http://localhost:8080/api/admin/authz/policies
curl http://localhost:8080/api/admin/config/variables
```

## Support

- Database schema: `database/migrations/001_admin_handlers_schema.sql`
- Integration docs: `FINAL_COMPLETION_REPORT.md`
- Quick reference: `QUICKSTART_ADMIN_HANDLERS.md`
