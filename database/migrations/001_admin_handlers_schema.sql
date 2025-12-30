-- AgentAuth Admin Handlers Database Schema
-- PostgreSQL 14+ with Row-Level Security (RLS) for multi-tenant isolation
-- Created: 2025-11-22
-- Tables: 17 (for 5 admin handlers: PoA, Resilience, Events, Authz, Config)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- CORE: SUBSCRIBERS (Tenant Management) - Required by all handlers
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscribers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_name VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    tier VARCHAR(50) NOT NULL DEFAULT 'standard',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'suspended', 'pending', 'disabled')),
    CONSTRAINT valid_tier CHECK (tier IN ('free', 'standard', 'premium', 'enterprise'))
);

CREATE INDEX IF NOT EXISTS idx_subscribers_tenant_id ON subscribers(tenant_id);

-- ============================================================================
-- HANDLER 1: POWER OF ATTORNEY (2 tables)
-- ============================================================================

-- Power of Attorney records
CREATE TABLE IF NOT EXISTS power_of_attorney (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    grantor_id VARCHAR(255) NOT NULL,
    grantor_name VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    agent_name VARCHAR(255) NOT NULL,
    scope TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'revoked', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_poa_tenant_id ON power_of_attorney(tenant_id);
CREATE INDEX IF NOT EXISTS idx_poa_grantor_id ON power_of_attorney(grantor_id);
CREATE INDEX IF NOT EXISTS idx_poa_agent_id ON power_of_attorney(agent_id);
CREATE INDEX IF NOT EXISTS idx_poa_status ON power_of_attorney(status);

-- PoA delegation chains
CREATE TABLE IF NOT EXISTS delegation_chains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    chain_level INTEGER NOT NULL,
    delegator_id VARCHAR(255) NOT NULL,
    delegatee_id VARCHAR(255) NOT NULL,
    delegation_scope TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(poa_id, chain_level)
);

CREATE INDEX IF NOT EXISTS idx_delegation_chains_poa_id ON delegation_chains(poa_id);
CREATE INDEX IF NOT EXISTS idx_delegation_chains_tenant_id ON delegation_chains(tenant_id);

-- ============================================================================
-- HANDLER 2: RESILIENCE PATTERNS (4 tables)
-- ============================================================================

-- Circuit breakers
CREATE TABLE IF NOT EXISTS circuit_breakers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL DEFAULT 'closed',
    failure_threshold INTEGER NOT NULL DEFAULT 5,
    success_threshold INTEGER NOT NULL DEFAULT 2,
    timeout_seconds INTEGER NOT NULL DEFAULT 60,
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    last_failure_time TIMESTAMP WITH TIME ZONE,
    last_success_time TIMESTAMP WITH TIME ZONE,
    last_state_change TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_cb_state CHECK (state IN ('closed', 'open', 'half-open')),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_circuit_breakers_tenant_id ON circuit_breakers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_circuit_breakers_state ON circuit_breakers(state);

-- Rate limiters
CREATE TABLE IF NOT EXISTS rate_limiters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    max_requests INTEGER NOT NULL,
    window_seconds INTEGER NOT NULL,
    burst_size INTEGER,
    total_requests BIGINT DEFAULT 0,
    allowed_requests BIGINT DEFAULT 0,
    rejected_requests BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_algorithm CHECK (algorithm IN ('token_bucket', 'sliding_window', 'fixed_window', 'leaky_bucket')),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_rate_limiters_tenant_id ON rate_limiters(tenant_id);

-- Retry policies
CREATE TABLE IF NOT EXISTS retry_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    backoff_type VARCHAR(50) NOT NULL DEFAULT 'exponential',
    initial_delay_ms INTEGER NOT NULL DEFAULT 100,
    max_delay_ms INTEGER NOT NULL DEFAULT 10000,
    multiplier NUMERIC(5,2) DEFAULT 2.0,
    jitter_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_backoff_type CHECK (backoff_type IN ('exponential', 'linear', 'constant', 'fibonacci')),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_retry_policies_tenant_id ON retry_policies(tenant_id);

-- Bulkheads
CREATE TABLE IF NOT EXISTS bulkheads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    max_concurrent INTEGER NOT NULL DEFAULT 10,
    max_queue INTEGER NOT NULL DEFAULT 100,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    current_active INTEGER DEFAULT 0,
    current_queued INTEGER DEFAULT 0,
    total_executed BIGINT DEFAULT 0,
    total_rejected BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_bulkheads_tenant_id ON bulkheads(tenant_id);

-- ============================================================================
-- HANDLER 3: EVENT SYSTEM (3 tables)
-- ============================================================================

-- Events
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    source VARCHAR(255),
    user_id VARCHAR(255),
    resource VARCHAR(255),
    action VARCHAR(255),
    status VARCHAR(50),
    message TEXT,
    payload JSONB DEFAULT '{}'::jsonb,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_event_severity CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info'))
);

CREATE INDEX IF NOT EXISTS idx_events_tenant_id ON events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);

-- Event subscriptions
CREATE TABLE IF NOT EXISTS event_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    subscription_name VARCHAR(255) NOT NULL,
    event_types TEXT[] NOT NULL,
    endpoint_url TEXT NOT NULL,
    http_method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    retry_config JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_sub_status CHECK (status IN ('active', 'paused', 'disabled')),
    UNIQUE(tenant_id, subscription_name)
);

CREATE INDEX IF NOT EXISTS idx_event_subscriptions_tenant_id ON event_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_event_subscriptions_status ON event_subscriptions(status);

-- Event deliveries (tracking)
CREATE TABLE IF NOT EXISTS event_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES event_subscriptions(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_delivery_status CHECK (status IN ('pending', 'delivered', 'failed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_event_deliveries_subscription_id ON event_deliveries(subscription_id);
CREATE INDEX IF NOT EXISTS idx_event_deliveries_status ON event_deliveries(status);

-- ============================================================================
-- HANDLER 4: AUTHORIZATION ENGINE (3 tables)
-- ============================================================================

-- Authorization policies
CREATE TABLE IF NOT EXISTS authorization_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    policy_name VARCHAR(255) NOT NULL,
    policy_type VARCHAR(50) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL,
    effect VARCHAR(20) NOT NULL,
    resources TEXT[],
    actions TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    priority INTEGER DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_policy_type CHECK (policy_type IN ('rbac', 'abac', 'pbac', 'custom')),
    CONSTRAINT valid_effect CHECK (effect IN ('allow', 'deny')),
    CONSTRAINT valid_policy_status CHECK (status IN ('active', 'draft', 'archived')),
    UNIQUE(tenant_id, policy_name)
);

CREATE INDEX IF NOT EXISTS idx_authorization_policies_tenant_id ON authorization_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_authorization_policies_status ON authorization_policies(status);
CREATE INDEX IF NOT EXISTS idx_authorization_policies_priority ON authorization_policies(priority DESC);

-- Policy roles
CREATE TABLE IF NOT EXISTS policy_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    role_name VARCHAR(255) NOT NULL,
    description TEXT,
    policy_ids UUID[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, role_name)
);

CREATE INDEX IF NOT EXISTS idx_policy_roles_tenant_id ON policy_roles(tenant_id);

-- Role permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES policy_roles(id) ON DELETE CASCADE,
    resource VARCHAR(255) NOT NULL,
    actions TEXT[] NOT NULL,
    conditions JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_tenant_id ON role_permissions(tenant_id);

-- ============================================================================
-- HANDLER 5: CONFIGURATION MANAGEMENT (4 tables)
-- ============================================================================

-- Configuration variables
CREATE TABLE IF NOT EXISTS config_variables (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    variable_key VARCHAR(255) NOT NULL,
    variable_value TEXT NOT NULL,
    variable_type VARCHAR(50) NOT NULL,
    scope VARCHAR(50) NOT NULL DEFAULT 'global',
    description TEXT,
    is_sensitive BOOLEAN DEFAULT false,
    is_encrypted BOOLEAN DEFAULT false,
    category VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    
    CONSTRAINT valid_variable_type CHECK (variable_type IN ('string', 'number', 'boolean', 'json')),
    CONSTRAINT valid_scope CHECK (scope IN ('global', 'tenant', 'environment')),
    UNIQUE(tenant_id, variable_key, scope)
);

CREATE INDEX IF NOT EXISTS idx_config_variables_tenant_id ON config_variables(tenant_id);
CREATE INDEX IF NOT EXISTS idx_config_variables_scope ON config_variables(scope);

-- Configuration files
CREATE TABLE IF NOT EXISTS config_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_format VARCHAR(50) NOT NULL,
    file_content TEXT NOT NULL,
    description TEXT,
    checksum VARCHAR(64),
    size_bytes INTEGER,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    
    CONSTRAINT valid_format CHECK (file_format IN ('yaml', 'json', 'toml', 'properties')),
    UNIQUE(tenant_id, file_name, version)
);

CREATE INDEX IF NOT EXISTS idx_config_files_tenant_id ON config_files(tenant_id);
CREATE INDEX IF NOT EXISTS idx_config_files_name ON config_files(file_name);

-- Service configurations
CREATE TABLE IF NOT EXISTS service_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    config_version VARCHAR(50) NOT NULL,
    config_data JSONB NOT NULL,
    environment VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    deployed_at TIMESTAMP WITH TIME ZONE,
    last_reload_at TIMESTAMP WITH TIME ZONE,
    requires_restart BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_service_status CHECK (status IN ('pending', 'deployed', 'active', 'failed')),
    UNIQUE(tenant_id, service_name, config_version)
);

CREATE INDEX IF NOT EXISTS idx_service_configs_tenant_id ON service_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_service_configs_service ON service_configs(service_name);

-- Feature flags
CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    flag_key VARCHAR(255) NOT NULL,
    flag_name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT false,
    rollout_percentage INTEGER DEFAULT 0,
    targeting_rules JSONB,
    category VARCHAR(100),
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    
    CONSTRAINT valid_rollout CHECK (rollout_percentage >= 0 AND rollout_percentage <= 100),
    UNIQUE(tenant_id, flag_key)
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_tenant_id ON feature_flags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_feature_flags_enabled ON feature_flags(enabled);

-- ============================================================================
-- ROW-LEVEL SECURITY (RLS) - Multi-tenant isolation
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE subscribers ENABLE ROW LEVEL SECURITY;
ALTER TABLE power_of_attorney ENABLE ROW LEVEL SECURITY;
ALTER TABLE delegation_chains ENABLE ROW LEVEL SECURITY;
ALTER TABLE circuit_breakers ENABLE ROW LEVEL SECURITY;
ALTER TABLE rate_limiters ENABLE ROW LEVEL SECURITY;
ALTER TABLE retry_policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE bulkheads ENABLE ROW LEVEL SECURITY;
ALTER TABLE events ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE authorization_policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_variables ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flags ENABLE ROW LEVEL SECURITY;

-- Create function to get current tenant from session context
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS VARCHAR AS $$
BEGIN
    RETURN current_setting('app.current_tenant_id', true);
END;
$$ LANGUAGE plpgsql STABLE;

-- Create RLS policies for each table (tenant isolation)
CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON power_of_attorney
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON delegation_chains
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON circuit_breakers
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON rate_limiters
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON retry_policies
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON bulkheads
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON events
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON event_subscriptions
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_deliveries
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON authorization_policies
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON policy_roles
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON role_permissions
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON config_variables
    FOR ALL
    USING (tenant_id IS NULL OR tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON config_files
    FOR ALL
    USING (tenant_id IS NULL OR tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON service_configs
    FOR ALL
    USING (tenant_id IS NULL OR tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON feature_flags
    FOR ALL
    USING (tenant_id IS NULL OR tenant_id = current_tenant_id());

-- ============================================================================
-- TRIGGERS - Automatic updated_at timestamps
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER IF NOT EXISTS update_subscribers_updated_at BEFORE UPDATE ON subscribers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_poa_updated_at BEFORE UPDATE ON power_of_attorney
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_circuit_breakers_updated_at BEFORE UPDATE ON circuit_breakers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_bulkheads_updated_at BEFORE UPDATE ON bulkheads
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_authorization_policies_updated_at BEFORE UPDATE ON authorization_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_config_variables_updated_at BEFORE UPDATE ON config_variables
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_config_files_updated_at BEFORE UPDATE ON config_files
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_feature_flags_updated_at BEFORE UPDATE ON feature_flags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SEED DATA - Create a test tenant
-- ============================================================================

INSERT INTO subscribers (tenant_id, tenant_name, status, tier)
VALUES ('test-tenant-1', 'Test Tenant', 'active', 'standard')
ON CONFLICT (tenant_id) DO NOTHING;

-- ============================================================================
-- GRANTS - Database permissions
-- ============================================================================

-- Note: Adjust role name and password for your environment
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'gauth_app') THEN
        CREATE ROLE gauth_app WITH LOGIN PASSWORD 'change_me_in_production';
    END IF;
END
$$;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO gauth_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO gauth_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO gauth_app;

-- ============================================================================
-- SCHEMA COMPLETE
-- ============================================================================

-- Summary:
-- ✅ 17 tables created for 5 admin handlers
-- ✅ Row-Level Security (RLS) enabled for multi-tenant isolation
-- ✅ Indexes created for performance
-- ✅ Triggers for automatic updated_at timestamps
-- ✅ Test tenant seeded
-- ✅ Application role created with permissions

COMMENT ON TABLE subscribers IS 'Core tenant management (required by all handlers)';
COMMENT ON TABLE power_of_attorney IS 'Power of Attorney records (Handler 1)';
COMMENT ON TABLE delegation_chains IS 'PoA delegation tracking (Handler 1)';
COMMENT ON TABLE circuit_breakers IS 'Circuit breaker state (Handler 2)';
COMMENT ON TABLE rate_limiters IS 'Rate limiter configuration (Handler 2)';
COMMENT ON TABLE retry_policies IS 'Retry policy configuration (Handler 2)';
COMMENT ON TABLE bulkheads IS 'Bulkhead configuration (Handler 2)';
COMMENT ON TABLE events IS 'Event stream storage (Handler 3)';
COMMENT ON TABLE event_subscriptions IS 'Event subscription configuration (Handler 3)';
COMMENT ON TABLE event_deliveries IS 'Event delivery tracking (Handler 3)';
COMMENT ON TABLE authorization_policies IS 'Authorization policies (Handler 4)';
COMMENT ON TABLE policy_roles IS 'Policy roles (Handler 4)';
COMMENT ON TABLE role_permissions IS 'Role permissions (Handler 4)';
COMMENT ON TABLE config_variables IS 'Configuration variables (Handler 5)';
COMMENT ON TABLE config_files IS 'Configuration files (Handler 5)';
COMMENT ON TABLE service_configs IS 'Service configurations (Handler 5)';
COMMENT ON TABLE feature_flags IS 'Feature flags (Handler 5)';
