-- Database Performance Optimization - Index Creation
-- Execute these DDL statements with CONCURRENTLY to avoid locking

-- ============================================================================
-- 1. PoA (Proof of Authority) Table Indexes
-- ============================================================================

-- Index for listing PoAs by user with date sorting
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poas_user_id_created_at 
    ON poas(user_id, created_at DESC) 
    WHERE deleted_at IS NULL;

-- Index for filtering by status
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poas_status_created_at 
    ON poas(status, created_at DESC) 
    WHERE deleted_at IS NULL;

-- Index for external ID lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poas_external_id 
    ON poas(external_id) 
    WHERE external_id IS NOT NULL;

-- Covering index to avoid table lookups (INDEX INCLUDE)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poas_list_covering 
    ON poas(user_id, created_at DESC) 
    INCLUDE (id, external_id, status, poa_type, metadata);

-- Index for filtering by type
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poas_type_created_at 
    ON poas(poa_type, created_at DESC) 
    WHERE deleted_at IS NULL;

-- Composite index for multi-column queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poas_user_status_created 
    ON poas(user_id, status, created_at DESC) 
    WHERE deleted_at IS NULL;

-- ============================================================================
-- 2. Audit Logs Table Indexes
-- ============================================================================

-- Index for user audit history
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_user_id_timestamp 
    ON audit_logs(user_id, timestamp DESC);

-- Index for action-based queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_action_timestamp 
    ON audit_logs(action, timestamp DESC);

-- Partial index for high-severity events
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_severity_timestamp 
    ON audit_logs(severity, timestamp DESC) 
    WHERE severity IN ('HIGH', 'CRITICAL');

-- Index for resource tracking
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_resource_id 
    ON audit_logs(resource_id) 
    WHERE resource_id IS NOT NULL;

-- Covering index for export queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_export_covering 
    ON audit_logs(timestamp DESC) 
    INCLUDE (user_id, action, severity, resource_type, resource_id, metadata);

-- ============================================================================
-- 3. API Keys Table Indexes
-- ============================================================================

-- Partial index for active API keys
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_active 
    ON api_keys(user_id, created_at DESC) 
    WHERE revoked_at IS NULL;

-- Index for API key lookups (hash for security)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_hash 
    ON api_keys(key_hash);

-- Index for expiry monitoring
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_expires_at 
    ON api_keys(expires_at) 
    WHERE revoked_at IS NULL AND expires_at IS NOT NULL;

-- ============================================================================
-- 4. Webhooks Table Indexes
-- ============================================================================

-- Partial index for active webhooks
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_webhooks_active 
    ON webhooks(user_id, created_at DESC) 
    WHERE active = true AND deleted_at IS NULL;

-- Index for event type filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_webhooks_event_type 
    ON webhooks(event_type) 
    WHERE active = true;

-- ============================================================================
-- 5. Webhook Deliveries Table Indexes
-- ============================================================================

-- Index for delivery status tracking
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_webhook_deliveries_webhook_status 
    ON webhook_deliveries(webhook_id, status, created_at DESC);

-- Index for retry queue
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_webhook_deliveries_retry_queue 
    ON webhook_deliveries(next_retry_at) 
    WHERE status = 'pending' AND retry_count < max_retries;

-- ============================================================================
-- 6. Users Table Indexes
-- ============================================================================

-- Index for email lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email 
    ON users(email) 
    WHERE deleted_at IS NULL;

-- Index for role-based queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_role 
    ON users(role) 
    WHERE deleted_at IS NULL;

-- Index for active users
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_last_login 
    ON users(last_login_at DESC) 
    WHERE deleted_at IS NULL;

-- ============================================================================
-- 7. Cache Statistics (if table exists)
-- ============================================================================

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cache_stats_key_timestamp 
    ON cache_statistics(cache_key, timestamp DESC);

-- ============================================================================
-- Index Maintenance Queries
-- ============================================================================

-- Check index usage
-- Run periodically to identify unused indexes
CREATE OR REPLACE VIEW v_index_usage AS
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid) as index_size
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;

-- Find duplicate indexes
CREATE OR REPLACE VIEW v_duplicate_indexes AS
SELECT 
    pg_size_pretty(SUM(pg_relation_size(idx))::BIGINT) AS size,
    (array_agg(idx))[1] AS idx1,
    (array_agg(idx))[2] AS idx2,
    (array_agg(idx))[3] AS idx3,
    (array_agg(idx))[4] AS idx4
FROM (
    SELECT 
        indexrelid::regclass AS idx,
        (indrelid::text ||E'\n'|| indclass::text ||E'\n'|| indkey::text ||E'\n'||
         COALESCE(indexprs::text,'')||E'\n' || COALESCE(indpred::text,'') AS key
    FROM pg_index
) sub
GROUP BY key 
HAVING COUNT(*) > 1
ORDER BY SUM(pg_relation_size(idx) DESC;

-- ============================================================================
-- Analyze Tables (Update Statistics)
-- ============================================================================

-- Run ANALYZE to update query planner statistics
ANALYZE poas;
ANALYZE audit_logs;
ANALYZE api_keys;
ANALYZE webhooks;
ANALYZE webhook_deliveries;
ANALYZE users;

-- Enable auto-analyze for critical tables
ALTER TABLE poas SET (autovacuum_analyze_scale_factor = 0.05);
ALTER TABLE audit_logs SET (autovacuum_analyze_scale_factor = 0.05);

-- ============================================================================
-- Query Performance Monitoring
-- ============================================================================

-- Enable pg_stat_statements extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- View slow queries
CREATE OR REPLACE VIEW v_slow_queries AS
SELECT 
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    max_exec_time,
    stddev_exec_time,
    rows
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- Queries slower than 100ms
ORDER BY mean_exec_time DESC
LIMIT 50;

-- Reset pg_stat_statements (use after analysis)
-- SELECT pg_stat_statements_reset();
