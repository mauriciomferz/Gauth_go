---
title: Postgresql Setup
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# PostgreSQL Setup Guide

This guide covers setting up and using PostgreSQL persistence for AAP-001 Extended Tokens in AgentAuth.

## Overview

AgentAuth now supports persistent storage for AAP-001 extended tokens using PostgreSQL with JSONB storage for complex authorization structures. This enables:

- **Token Persistence**: Tokens survive server restarts
- **Distributed Deployments**: Multiple server instances share token state
- **Production Scale**: Efficient queries with proper indexing
- **Audit Trail**: Complete token lifecycle tracking

## Quick Start with Docker Compose

### 1. Start the Environment

```bash
# Start PostgreSQL, Redis, and Web Server
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f web-server
```

The services include:
- **PostgreSQL 16**: Port 5432, auto-migrations on startup
- **Redis 7**: Port 6379 for caching
- **Web Server**: Port 8080 with AAP-001 enabled

### 2. Verify Database Setup

```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U agentauth -d agentauth

# Check tables
\dt

# View extended_tokens schema
\d extended_tokens

# View subscriptions schema
\d subscriptions

# Exit
\q
```

### 3. Stop the Environment

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (clean slate)
docker-compose down -v
```

## Database Schema

### Extended Tokens Table

The `extended_tokens` table stores AAP-001 tokens with:

**OAuth 2.0 Fields:**
- `access_token` (VARCHAR 512, PRIMARY KEY)
- `token_type` (VARCHAR 50, default: 'AgentAuth-Extended-Token')
- `expires_in` (BIGINT)
- `refresh_token` (VARCHAR 512, nullable)
- `scope` (TEXT[], nullable)
- `issued_at` (TIMESTAMP WITH TIME ZONE)

**AAP-001 Extended Fields (JSONB):**
- `power_of_attorney` - Power of Attorney definition
- `authorization_chain` - Complete authorization hierarchy
- `legal_framework` - Legal compliance context
- `verification_proof` - Identity verification chain
- `audit_trail` - Token lifecycle events

**Metadata Fields:**
- `client_id` (VARCHAR 255) - Extracted from authorization chain
- `grant_id` (VARCHAR 255) - Grant identifier
- `resource_owner` (VARCHAR 255) - Resource owner ID
- `purpose` (TEXT) - Authorization purpose
- `created_at` (TIMESTAMP WITH TIME ZONE)
- `revoked_at` (TIMESTAMP WITH TIME ZONE, nullable)
- `last_used_at` (TIMESTAMP WITH TIME ZONE, nullable)
- `use_count` (INTEGER, default: 0)
- `expires_at` (COMPUTED) - Automatically calculated from `issued_at + expires_in`

**Indexes:**
1. `idx_extended_tokens_expires_at` - Efficient expired token cleanup
2. `idx_extended_tokens_revoked_at` - Filter revoked tokens
3. `idx_extended_tokens_client_id` - JSONB path extraction for client lookups
4. `idx_extended_tokens_grant_id` - Grant-based queries
5. `idx_extended_tokens_issued_at` - Time-based queries

### Subscriptions Table

The `subscriptions` table stores AAP-001 subscription flow state:

**Core Fields:**
- `id` (VARCHAR 255, PRIMARY KEY)
- `status` (VARCHAR 50) - Workflow status with CHECK constraint
- `created_at` (TIMESTAMP WITH TIME ZONE)
- `updated_at` (TIMESTAMP WITH TIME ZONE)

**Subscription Steps (JSONB):**
- `owners_authorizer_identity` - Step I
- `authorization_proof` - Step II
- `client_owner_identity` - Step III
- `client_authorization_grant` - Step IV
- `resource_owner_identity` - Step V
- `resource_server_auth` - Step VI
- `authorization_chain` - Step VII
- `identity_verification_chain` - Step VIII (optional)

**Query Fields:**
- `client_id` (VARCHAR 255) - Extracted for queries
- `resource_owner` (VARCHAR 255) - Extracted for queries
- `completed` (BOOLEAN) - Quick status check

**Valid Status Values:**
- `pending`, `awaiting_identity`, `awaiting_auth_proof`
- `awaiting_client_owner`, `awaiting_client`, `awaiting_resource`
- `completed`, `failed`

## Environment Variables

### Database Configuration

```bash
# PostgreSQL Connection
DB_HOST=localhost           # Database host (default: localhost)
DB_PORT=5432               # Database port (default: 5432)
DB_NAME=agentauth              # Database name (default: agentauth)
DB_USER=agentauth              # Database user (default: agentauth)
DB_PASSWORD=agentauth_password # Database password (required)

# Connection Pool Settings (optional)
DB_MAX_OPEN_CONNS=25      # Maximum open connections (default: 25)
DB_MAX_IDLE_CONNS=5       # Maximum idle connections (default: 5)
DB_CONN_MAX_LIFETIME=5m   # Connection max lifetime (default: 5m)
DB_CONN_MAX_IDLE_TIME=1m  # Connection max idle time (default: 1m)
```

### AAP-001 Configuration

```bash
# Enable AAP-001 Features
AGENTAUTH_AAP-001_ENABLED=1              # Enable AAP-001 endpoints
AGENTAUTH_AAP-001_USE_MOCKS=0            # Use real services (1 for mocks)

# Token Store Selection
AGENTAUTH_TOKEN_STORE=postgres           # Options: memory, postgres (default: memory)

# Subscription Store Selection
AGENTAUTH_SUBSCRIPTION_STORE=postgres    # Options: memory, postgres (default: memory)
```

### Redis Configuration (Optional)

```bash
REDIS_HOST=localhost                 # Redis host (default: localhost)
REDIS_PORT=6379                      # Redis port (default: 6379)
REDIS_PASSWORD=                      # Redis password (optional)
```

## Manual PostgreSQL Setup

If not using Docker Compose, set up PostgreSQL manually:

### 1. Install PostgreSQL 16

**macOS:**
```bash
brew install postgresql@16
brew services start postgresql@16
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql-16
sudo systemctl start postgresql
```

### 2. Create Database and User

```bash
# Connect as postgres user
psql postgres

# Create user and database
CREATE USER agentauth WITH PASSWORD 'agentauth_password';
CREATE DATABASE agentauth OWNER agentauth;

# Grant privileges
GRANT ALL PRIVILEGES ON DATABASE agentauth TO agentauth;

\q
```

### 3. Run Migrations

```bash
# Apply migrations in order
psql -h localhost -U agentauth -d agentauth -f schema/migrations/001_create_extended_tokens.sql
psql -h localhost -U agentauth -d agentauth -f schema/migrations/002_create_subscriptions.sql
```

### 4. Verify Setup

```bash
psql -h localhost -U agentauth -d agentauth

# Check tables
SELECT table_name FROM information_schema.tables 
WHERE table_schema = 'public';

# Check indexes
SELECT indexname, tablename FROM pg_indexes 
WHERE schemaname = 'public';
```

## Using PostgreSQL Token Store

### In Go Code

```go
import (
    "context"
    "github.com/agentauth/AAP-RFC-0150-Go-Implementation-of-AgentAuth-1.0/pkg/agentauth"
)

// Create PostgreSQL token store from DSN
dsn := "postgres://agentauth:agentauth_password@localhost:5432/agentauth?sslmode=disable"
tokenStore, err := agentauth.NewPostgresExtendedTokenStoreFromDSN(dsn)
if err != nil {
    log.Fatalf("Failed to create token store: %v", err)
}
defer tokenStore.Close()

// Or create from existing *sql.DB
// tokenStore := agentauth.NewPostgresExtendedTokenStore(db)

// Save a token
ctx := context.Background()
err = tokenStore.SaveToken(ctx, extendedToken)
if err != nil {
    log.Printf("Failed to save token: %v", err)
}

// Retrieve a token
token, metadata, err := tokenStore.GetToken(ctx, "access_token_xyz")
if err == agentauth.ErrTokenNotFound {
    log.Println("Token not found")
} else if err == agentauth.ErrTokenExpired {
    log.Println("Token has expired")
} else if err != nil {
    log.Printf("Error: %v", err)
}

// Check revocation
isRevoked, err := tokenStore.IsRevoked(ctx, "access_token_xyz")

// Revoke a token (RFC 7009 compliant - idempotent)
err = tokenStore.RevokeToken(ctx, "access_token_xyz")

// List tokens by client
tokens, err := tokenStore.ListTokensByClient(ctx, "client_123")

// Cleanup expired tokens
deletedCount, err := tokenStore.DeleteExpiredTokens(ctx)
log.Printf("Deleted %d expired tokens", deletedCount)
```

### Token Metadata

The `TokenMetadata` includes usage tracking:

```go
type TokenMetadata struct {
    CreatedAt   time.Time
    RevokedAt   *time.Time  // nil if not revoked
    LastUsedAt  *time.Time  // Updated on GetToken
    UseCount    int         // Incremented on GetToken
}
```

## Performance Considerations

### Connection Pooling

The PostgreSQL store uses connection pooling by default:
- **Max Open Connections**: 25
- **Max Idle Connections**: 5
- **Connection Lifetime**: 5 minutes
- **Idle Timeout**: 1 minute

Tune these based on your load:

```go
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(10 * time.Minute)
```

### Index Optimization

The schema includes indexes for common queries:

```sql
-- Fast expired token cleanup
WHERE expires_at < NOW()

-- Client token lookup
WHERE client_id = 'client_123' AND revoked_at IS NULL

-- Grant-based queries
WHERE grant_id = 'grant_456'
```

### JSONB Query Performance

JSONB fields support efficient queries:

```sql
-- Extract client ID from authorization chain
SELECT client_id FROM extended_tokens 
WHERE authorization_chain->>'client'->>'entity_id' = 'client_123';

-- Use the indexed client_id column instead (faster)
SELECT * FROM extended_tokens WHERE client_id = 'client_123';
```

### Cleanup Jobs

Run periodic cleanup of expired tokens:

```go
// In a background goroutine
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        count, err := tokenStore.DeleteExpiredTokens(context.Background())
        if err != nil {
            log.Printf("Cleanup error: %v", err)
        } else {
            log.Printf("Cleaned up %d expired tokens", count)
        }
    }
}()
```

## Testing

### Run Tests with PostgreSQL

```bash
# Start test database
docker-compose up -d postgres

# Wait for database to be ready
sleep 5

# Run tests with PostgreSQL
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=agentauth
export DB_USER=agentauth
export DB_PASSWORD=agentauth_password
export AGENTAUTH_AAP-001_ENABLED=1
export AGENTAUTH_TOKEN_STORE=postgres

go test -v ./pkg/agentauth/... -run TestPostgres
```

### Performance Benchmarks

```bash
# Run performance tests
./scripts/test_aap001_performance.sh

# Expected results with PostgreSQL:
# - Token issuance: ~5-10ms per token
# - Token retrieval: ~2-5ms per token
# - Concurrent requests: 100+ req/s
```

### Integration Tests

```bash
# Start full environment
docker-compose up -d

# Run integration tests
./scripts/test_aap001_subscription_flow.sh
./scripts/test_aap001_end_to_end.sh
```

## Monitoring

### Database Metrics

```sql
-- Connection stats
SELECT * FROM pg_stat_database WHERE datname = 'agentauth';

-- Table sizes
SELECT 
    pg_size_pretty(pg_total_relation_size('extended_tokens') as tokens_size,
    pg_size_pretty(pg_total_relation_size('subscriptions') as subscriptions_size;

-- Index usage
SELECT 
    schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes 
WHERE schemaname = 'public';

-- Active tokens
SELECT 
    COUNT(*) as total_tokens,
    COUNT(*) FILTER (WHERE revoked_at IS NULL) as active_tokens,
    COUNT(*) FILTER (WHERE expires_at < NOW() as expired_tokens
FROM extended_tokens;
```

### Query Performance

```sql
-- Slow queries (requires pg_stat_statements extension)
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
WHERE query LIKE '%extended_tokens%'
ORDER BY mean_exec_time DESC 
LIMIT 10;

-- Explain analyze for optimization
EXPLAIN ANALYZE 
SELECT * FROM extended_tokens 
WHERE client_id = 'client_123' AND revoked_at IS NULL;
```

## Troubleshooting

### Connection Issues

```bash
# Test connection
psql postgres://agentauth:agentauth_password@localhost:5432/agentauth -c "SELECT 1"

# Check PostgreSQL is running
pg_isready -h localhost -p 5432

# View connection errors
docker-compose logs postgres | grep ERROR
```

### Migration Issues

```bash
# Check migration status
psql -h localhost -U agentauth -d agentauth -c "\dt"

# Manually run migrations
psql -h localhost -U agentauth -d agentauth -f schema/migrations/001_create_extended_tokens.sql

# Rollback (drop tables)
psql -h localhost -U agentauth -d agentauth -c "DROP TABLE IF EXISTS extended_tokens CASCADE;"
```

### Performance Issues

```bash
# Check slow queries
docker-compose exec postgres psql -U agentauth -d agentauth -c "
SELECT pid, now() - query_start as duration, query 
FROM pg_stat_activity 
WHERE state = 'active' AND query NOT LIKE '%pg_stat_activity%'
ORDER BY duration DESC;
"

# Check locks
docker-compose exec postgres psql -U agentauth -d agentauth -c "
SELECT * FROM pg_locks WHERE NOT granted;
"

# Vacuum tables
docker-compose exec postgres psql -U agentauth -d agentauth -c "
VACUUM ANALYZE extended_tokens;
VACUUM ANALYZE subscriptions;
"
```

## Migration from Memory Store

To migrate from in-memory storage to PostgreSQL:

1. **Stop the server** to prevent new tokens
2. **Export existing tokens** (if persistence needed)
3. **Update environment variables**:
   ```bash
   AGENTAUTH_TOKEN_STORE=postgres
   DB_HOST=localhost
   DB_PASSWORD=your_password
   ```
4. **Start with PostgreSQL**: `docker-compose up -d`
5. **Re-issue tokens** (memory store tokens are not persisted)

## Production Deployment

### Security Checklist

- [ ] Use SSL/TLS for database connections (`sslmode=require`)
- [ ] Strong database password (avoid defaults)
- [ ] Restrict database network access (firewall rules)
- [ ] Regular backups of PostgreSQL data
- [ ] Monitor for SQL injection attempts
- [ ] Use read-only replicas for read-heavy workloads
- [ ] Enable audit logging for compliance

### High Availability

```yaml
# docker-compose.yml with replication
services:
  postgres-primary:
    image: postgres:16-alpine
    environment:
      POSTGRES_REPLICATION_MODE: master
      
  postgres-replica:
    image: postgres:16-alpine
    environment:
      POSTGRES_REPLICATION_MODE: slave
      POSTGRES_MASTER_HOST: postgres-primary
```

### Backup Strategy

```bash
# Automated backup
docker-compose exec postgres pg_dump -U agentauth agentauth > backup_$(date +%Y%m%d).sql

# Restore from backup
docker-compose exec -T postgres psql -U agentauth -d agentauth < backup_20231115.sql
```

## Additional Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/16/)
- [JSONB Performance Tips](https://www.postgresql.org/docs/16/datatype-json.html)
- [AAP-001 Specification](../AAP-001_README.md)
- [AgentAuth Architecture](../ARCHITECTURE.md)

## Support

For issues or questions:
- Open an issue on GitHub
- Check existing issues for solutions
- Review the [FAQ](../FAQ.md)
