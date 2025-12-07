-- Migration: Create Admin Portal Tables
-- Covers: Resilience, Authorization (Authz), and Configuration (Config) handlers
-- Version: 003
-- Created: 2025-12-07

-- Create pg_trgm extension for text search if needed, and pgcrypto for UUIDs
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- RESILIENCE TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS circuit_breakers (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    breaker_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL DEFAULT 'closed',
    failure_threshold INTEGER NOT NULL,
    success_threshold INTEGER NOT NULL,
    timeout_seconds INTEGER NOT NULL,
    half_open_max_requests INTEGER,
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    consecutive_successes INTEGER DEFAULT 0,
    last_failure_time TIMESTAMP WITH TIME ZONE,
    last_success_time TIMESTAMP WITH TIME ZONE,
    last_state_change TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rate_limiters (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    limiter_name VARCHAR(255) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    max_requests INTEGER NOT NULL,
    window_seconds INTEGER NOT NULL,
    burst_size INTEGER,
    total_requests BIGINT DEFAULT 0,
    allowed_requests BIGINT DEFAULT 0,
    rejected_requests BIGINT DEFAULT 0,
    last_request_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS retry_policies (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    policy_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    max_attempts INTEGER NOT NULL,
    backoff_type VARCHAR(50) NOT NULL,
    initial_delay_ms INTEGER NOT NULL,
    max_delay_ms INTEGER NOT NULL,
    multiplier FLOAT NOT NULL DEFAULT 2.0,
    jitter_enabled BOOLEAN DEFAULT FALSE,
    retryable_errors TEXT[],
    total_retries BIGINT DEFAULT 0,
    successful_retries BIGINT DEFAULT 0,
    failed_retries BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bulkheads (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    bulkhead_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    max_concurrent INTEGER NOT NULL,
    max_queue INTEGER NOT NULL,
    timeout_seconds INTEGER NOT NULL,
    current_active INTEGER DEFAULT 0,
    current_queued INTEGER DEFAULT 0,
    total_executed BIGINT DEFAULT 0,
    total_rejected BIGINT DEFAULT 0,
    total_timeout BIGINT DEFAULT 0,
    peak_concurrent INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for Resilience
CREATE INDEX idx_circuit_breakers_tenant ON circuit_breakers(tenant_id);
CREATE INDEX idx_rate_limiters_tenant ON rate_limiters(tenant_id);
CREATE INDEX idx_retry_policies_tenant ON retry_policies(tenant_id);
CREATE INDEX idx_bulkheads_tenant ON bulkheads(tenant_id);


-- ============================================================================
-- AUTHORIZATION (AUTHZ) TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS authorization_policies (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    policy_name VARCHAR(255) NOT NULL,
    policy_type VARCHAR(50) NOT NULL,
    version INTEGER DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    description TEXT,
    rules JSONB,
    conditions JSONB,
    actions TEXT[],
    resources TEXT[],
    effect VARCHAR(50) NOT NULL,
    priority INTEGER DEFAULT 100,
    created_by VARCHAR(255),
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_until TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS policy_attributes (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    attribute_name VARCHAR(255) NOT NULL,
    attribute_type VARCHAR(50) NOT NULL,
    source VARCHAR(255),
    value_type VARCHAR(50),
    description TEXT,
    sample_value TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS authorization_logs (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    user_id VARCHAR(255),
    action VARCHAR(255),
    resource VARCHAR(255),
    decision VARCHAR(50),
    policy_id VARCHAR(255),
    policy_name VARCHAR(255),
    ip_address VARCHAR(255),
    user_agent VARCHAR(255),
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    context JSONB,
    evaluation_time_ms INTEGER
);

-- Indexes for Authz
CREATE INDEX idx_authz_policies_tenant ON authorization_policies(tenant_id);
CREATE INDEX idx_authz_logs_tenant ON authorization_logs(tenant_id);
CREATE INDEX idx_authz_logs_timestamp ON authorization_logs(timestamp DESC);


-- ============================================================================
-- CONFIGURATION (CONFIG) TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS config_variables (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255), -- NULL for global/system
    variable_key VARCHAR(255) NOT NULL,
    variable_value TEXT,
    variable_type VARCHAR(50) DEFAULT 'string',
    scope VARCHAR(50) DEFAULT 'tenant',
    description TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,
    is_encrypted BOOLEAN DEFAULT FALSE,
    category VARCHAR(255),
    updated_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT config_vars_unique UNIQUE (tenant_id, variable_key)
);

CREATE TABLE IF NOT EXISTS config_files (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255),
    file_name VARCHAR(255) NOT NULL,
    file_format VARCHAR(50) NOT NULL,
    file_content TEXT,
    description TEXT,
    checksum VARCHAR(255),
    size_bytes INTEGER,
    version INTEGER NOT NULL DEFAULT 1,
    updated_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS service_configs (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255),
    service_name VARCHAR(255) NOT NULL,
    config_version VARCHAR(255),
    status VARCHAR(50),
    config_data JSONB,
    environment VARCHAR(50),
    deployed_at TIMESTAMP WITH TIME ZONE,
    last_reload_at TIMESTAMP WITH TIME ZONE,
    requires_restart BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_config_overrides (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    config_key VARCHAR(255) NOT NULL,
    override_value TEXT,
    override_type VARCHAR(50),
    enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS feature_flags (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255),
    flag_key VARCHAR(255) NOT NULL,
    flag_name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT FALSE,
    rollout_percentage INTEGER DEFAULT 0,
    targeting_rules JSONB,
    category VARCHAR(255),
    tags TEXT[],
    updated_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for Config
CREATE INDEX idx_config_vars_tenant ON config_variables(tenant_id);
CREATE INDEX idx_config_files_tenant ON config_files(tenant_id);
CREATE INDEX idx_feature_flags_tenant ON feature_flags(tenant_id);
