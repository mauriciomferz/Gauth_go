-- GAuth Admin Portal - Initial Database Schema
-- PostgreSQL 14+ with Row-Level Security (RLS) for multi-tenant isolation
-- Version: 1.0.0
-- Created: 2025-11-22

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- TENANTS & SUBSCRIBERS
-- ============================================================================

-- Subscribers table: Core tenant information
CREATE TABLE subscribers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_name VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    tier VARCHAR(50) NOT NULL DEFAULT 'standard',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    
    -- OIDC Configuration
    oidc_provider VARCHAR(100),
    oidc_issuer TEXT,
    oidc_client_id VARCHAR(255),
    oidc_client_secret TEXT,
    oidc_scopes TEXT[],
    oidc_discovery_url TEXT,
    
    -- Key Configuration
    key_type VARCHAR(50),
    public_key TEXT,
    private_key_id VARCHAR(255),
    key_generated_at TIMESTAMP WITH TIME ZONE,
    key_expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Policy Configuration
    policy_template VARCHAR(100),
    legal_framework VARCHAR(100),
    
    -- Notification Configuration
    notification_channels TEXT[], -- email, sms, webhook, slack
    notification_email VARCHAR(255),
    notification_webhook_url TEXT,
    
    -- Metadata
    contact_email VARCHAR(255),
    contact_name VARCHAR(255),
    domain VARCHAR(255),
    max_users INTEGER DEFAULT 100,
    max_tokens INTEGER DEFAULT 1000,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'suspended', 'pending', 'disabled')),
    CONSTRAINT valid_tier CHECK (tier IN ('free', 'standard', 'premium', 'enterprise'))
);

-- Indexes for subscribers
CREATE INDEX idx_subscribers_tenant_id ON subscribers(tenant_id);
CREATE INDEX idx_subscribers_status ON subscribers(status);
CREATE INDEX idx_subscribers_created_at ON subscribers(created_at DESC);

-- ============================================================================
-- TOKENS & BLACKLIST
-- ============================================================================

-- Tokens table: Long-term token metadata (active tokens in Redis)
CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_id VARCHAR(255) UNIQUE NOT NULL,
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    token_type VARCHAR(50) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    audience TEXT[],
    issuer VARCHAR(255),
    scope TEXT[],
    
    -- Lifecycle
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    
    -- Metadata
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    usage_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    CONSTRAINT valid_token_type CHECK (token_type IN ('access', 'refresh', 'id', 'api_key', 'service'))
);

-- Indexes for tokens
CREATE INDEX idx_tokens_token_id ON tokens(token_id);
CREATE INDEX idx_tokens_tenant_id ON tokens(tenant_id);
CREATE INDEX idx_tokens_subject ON tokens(subject);
CREATE INDEX idx_tokens_expires_at ON tokens(expires_at);
CREATE INDEX idx_tokens_revoked_at ON tokens(revoked_at) WHERE revoked_at IS NOT NULL;
CREATE INDEX idx_tokens_issued_at ON tokens(issued_at DESC);

-- Token blacklist table: Revoked tokens (synced with Redis)
CREATE TABLE token_blacklist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_id VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    reason VARCHAR(255),
    revoked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_by VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    UNIQUE(token_id, tenant_id)
);

-- Indexes for token_blacklist
CREATE INDEX idx_blacklist_token_id ON token_blacklist(token_id);
CREATE INDEX idx_blacklist_tenant_id ON token_blacklist(tenant_id);
CREATE INDEX idx_blacklist_expires_at ON token_blacklist(expires_at);

-- ============================================================================
-- AUTHORIZATION ENGINE
-- ============================================================================

-- Policies table: RBAC/ABAC/PBAC policies
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    policy_name VARCHAR(255) NOT NULL,
    policy_type VARCHAR(50) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    -- Policy Definition
    description TEXT,
    rules JSONB NOT NULL,
    conditions JSONB,
    actions TEXT[],
    resources TEXT[],
    effect VARCHAR(20) NOT NULL,
    
    -- Lifecycle
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_until TIMESTAMP WITH TIME ZONE,
    
    -- Priority
    priority INTEGER DEFAULT 100,
    
    CONSTRAINT valid_policy_type CHECK (policy_type IN ('rbac', 'abac', 'pbac', 'custom')),
    CONSTRAINT valid_effect CHECK (effect IN ('allow', 'deny')),
    CONSTRAINT valid_status CHECK (status IN ('active', 'draft', 'archived')),
    UNIQUE(tenant_id, policy_name, version)
);

-- Indexes for policies
CREATE INDEX idx_policies_tenant_id ON policies(tenant_id);
CREATE INDEX idx_policies_status ON policies(status);
CREATE INDEX idx_policies_policy_type ON policies(policy_type);
CREATE INDEX idx_policies_priority ON policies(priority DESC);

-- Policy attributes table: Context attributes for ABAC
CREATE TABLE policy_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    attribute_name VARCHAR(255) NOT NULL,
    attribute_type VARCHAR(50) NOT NULL,
    source VARCHAR(100) NOT NULL,
    value_type VARCHAR(50) NOT NULL,
    description TEXT,
    sample_value TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_attribute_type CHECK (attribute_type IN ('user', 'resource', 'environment', 'action')),
    CONSTRAINT valid_value_type CHECK (value_type IN ('string', 'number', 'boolean', 'array', 'object')),
    UNIQUE(tenant_id, attribute_name)
);

-- Indexes for policy_attributes
CREATE INDEX idx_policy_attributes_tenant_id ON policy_attributes(tenant_id);
CREATE INDEX idx_policy_attributes_type ON policy_attributes(attribute_type);

-- Authorization logs table: Policy decision logs
CREATE TABLE authorization_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    decision VARCHAR(20) NOT NULL,
    policy_id UUID REFERENCES policies(id) ON DELETE SET NULL,
    policy_name VARCHAR(255),
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    context JSONB DEFAULT '{}'::jsonb,
    
    -- Performance
    evaluation_time_ms INTEGER,
    
    CONSTRAINT valid_decision CHECK (decision IN ('allow', 'deny', 'error'))
);

-- Indexes for authorization_logs
CREATE INDEX idx_authz_logs_tenant_id ON authorization_logs(tenant_id);
CREATE INDEX idx_authz_logs_timestamp ON authorization_logs(timestamp DESC);
CREATE INDEX idx_authz_logs_user_id ON authorization_logs(user_id);
CREATE INDEX idx_authz_logs_decision ON authorization_logs(decision);

-- ============================================================================
-- POWER OF ATTORNEY
-- ============================================================================

-- Power of Attorney records table
CREATE TABLE poa_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    poa_name VARCHAR(255) NOT NULL,
    grantor_id VARCHAR(255) NOT NULL,
    grantor_name VARCHAR(255) NOT NULL,
    representative_id VARCHAR(255) NOT NULL,
    representative_name VARCHAR(255) NOT NULL,
    representative_type VARCHAR(100),
    
    -- Scope
    scope_type VARCHAR(50) NOT NULL,
    actions TEXT[] NOT NULL,
    geographic_regions TEXT[],
    
    -- Lifecycle
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by VARCHAR(255),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Conditions
    conditions JSONB,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'pending', 'revoked', 'expired')),
    CONSTRAINT valid_scope_type CHECK (scope_type IN ('full', 'limited', 'financial', 'healthcare', 'legal', 'administrative'))
);

-- Indexes for poa_records
CREATE INDEX idx_poa_tenant_id ON poa_records(tenant_id);
CREATE INDEX idx_poa_grantor_id ON poa_records(grantor_id);
CREATE INDEX idx_poa_representative_id ON poa_records(representative_id);
CREATE INDEX idx_poa_status ON poa_records(status);
CREATE INDEX idx_poa_valid_from ON poa_records(valid_from);
CREATE INDEX idx_poa_valid_until ON poa_records(valid_until);

-- PoA templates table
CREATE TABLE poa_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    template_name VARCHAR(255) NOT NULL,
    description TEXT,
    scope_type VARCHAR(50) NOT NULL,
    default_actions TEXT[],
    default_duration_days INTEGER,
    conditions_schema JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    is_system_template BOOLEAN DEFAULT false
);

-- Indexes for poa_templates
CREATE INDEX idx_poa_templates_tenant_id ON poa_templates(tenant_id);

-- ============================================================================
-- EVENT SYSTEM
-- ============================================================================

-- Event types table
CREATE TABLE event_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    severity VARCHAR(50),
    schema JSONB,
    retention_days INTEGER DEFAULT 90,
    is_system_event BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_severity CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    UNIQUE(tenant_id, event_type)
);

-- Indexes for event_types
CREATE INDEX idx_event_types_tenant_id ON event_types(tenant_id);
CREATE INDEX idx_event_types_category ON event_types(category);

-- Events table: Event stream
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Event data
    source VARCHAR(255),
    user_id VARCHAR(255),
    resource VARCHAR(255),
    action VARCHAR(255),
    status VARCHAR(50),
    message TEXT,
    payload JSONB DEFAULT '{}'::jsonb,
    
    -- Metadata
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    correlation_id VARCHAR(255),
    
    CONSTRAINT valid_severity CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info'))
);

-- Partitioning strategy for events (by month)
-- CREATE TABLE events_2025_11 PARTITION OF events FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

-- Indexes for events
CREATE INDEX idx_events_tenant_id ON events(tenant_id);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_event_type ON events(event_type);
CREATE INDEX idx_events_severity ON events(severity);
CREATE INDEX idx_events_user_id ON events(user_id);
CREATE INDEX idx_events_correlation_id ON events(correlation_id) WHERE correlation_id IS NOT NULL;

-- Event handlers table
CREATE TABLE event_handlers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    handler_name VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    handler_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    -- Configuration
    endpoint_url TEXT,
    http_method VARCHAR(10),
    headers JSONB,
    retry_config JSONB,
    timeout_seconds INTEGER DEFAULT 30,
    
    -- Statistics
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_handler_type CHECK (handler_type IN ('webhook', 'email', 'sms', 'slack', 'custom')),
    CONSTRAINT valid_status CHECK (status IN ('active', 'paused', 'disabled')),
    UNIQUE(tenant_id, handler_name)
);

-- Indexes for event_handlers
CREATE INDEX idx_event_handlers_tenant_id ON event_handlers(tenant_id);
CREATE INDEX idx_event_handlers_event_type ON event_handlers(event_type);
CREATE INDEX idx_event_handlers_status ON event_handlers(status);

-- ============================================================================
-- RESILIENCE PATTERNS
-- ============================================================================

-- Circuit breakers table
CREATE TABLE circuit_breakers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    breaker_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL DEFAULT 'closed',
    
    -- Configuration
    failure_threshold INTEGER NOT NULL DEFAULT 5,
    success_threshold INTEGER NOT NULL DEFAULT 2,
    timeout_seconds INTEGER NOT NULL DEFAULT 60,
    half_open_max_requests INTEGER DEFAULT 3,
    
    -- Statistics
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    consecutive_successes INTEGER DEFAULT 0,
    last_failure_time TIMESTAMP WITH TIME ZONE,
    last_success_time TIMESTAMP WITH TIME ZONE,
    last_state_change TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_state CHECK (state IN ('closed', 'open', 'half-open')),
    UNIQUE(tenant_id, breaker_name)
);

-- Indexes for circuit_breakers
CREATE INDEX idx_circuit_breakers_tenant_id ON circuit_breakers(tenant_id);
CREATE INDEX idx_circuit_breakers_state ON circuit_breakers(state);
CREATE INDEX idx_circuit_breakers_service ON circuit_breakers(service_name);

-- Rate limiters table
CREATE TABLE rate_limiters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    limiter_name VARCHAR(255) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    
    -- Configuration
    max_requests INTEGER NOT NULL,
    window_seconds INTEGER NOT NULL,
    burst_size INTEGER,
    
    -- Statistics (aggregate)
    total_requests BIGINT DEFAULT 0,
    allowed_requests BIGINT DEFAULT 0,
    rejected_requests BIGINT DEFAULT 0,
    last_request_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_algorithm CHECK (algorithm IN ('token_bucket', 'sliding_window', 'fixed_window', 'leaky_bucket')),
    UNIQUE(tenant_id, limiter_name)
);

-- Indexes for rate_limiters
CREATE INDEX idx_rate_limiters_tenant_id ON rate_limiters(tenant_id);
CREATE INDEX idx_rate_limiters_endpoint ON rate_limiters(endpoint);

-- Retry policies table
CREATE TABLE retry_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    policy_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    
    -- Configuration
    max_attempts INTEGER NOT NULL DEFAULT 3,
    backoff_type VARCHAR(50) NOT NULL DEFAULT 'exponential',
    initial_delay_ms INTEGER NOT NULL DEFAULT 100,
    max_delay_ms INTEGER NOT NULL DEFAULT 10000,
    multiplier NUMERIC(5,2) DEFAULT 2.0,
    jitter_enabled BOOLEAN DEFAULT true,
    retryable_errors TEXT[],
    
    -- Statistics
    total_retries BIGINT DEFAULT 0,
    successful_retries BIGINT DEFAULT 0,
    failed_retries BIGINT DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_backoff_type CHECK (backoff_type IN ('exponential', 'linear', 'constant', 'fibonacci')),
    UNIQUE(tenant_id, policy_name)
);

-- Indexes for retry_policies
CREATE INDEX idx_retry_policies_tenant_id ON retry_policies(tenant_id);
CREATE INDEX idx_retry_policies_service ON retry_policies(service_name);

-- Bulkheads table
CREATE TABLE bulkheads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    bulkhead_name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    
    -- Configuration
    max_concurrent INTEGER NOT NULL DEFAULT 10,
    max_queue INTEGER NOT NULL DEFAULT 100,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    
    -- Statistics
    current_active INTEGER DEFAULT 0,
    current_queued INTEGER DEFAULT 0,
    total_executed BIGINT DEFAULT 0,
    total_rejected BIGINT DEFAULT 0,
    total_timeout BIGINT DEFAULT 0,
    peak_concurrent INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, bulkhead_name)
);

-- Indexes for bulkheads
CREATE INDEX idx_bulkheads_tenant_id ON bulkheads(tenant_id);
CREATE INDEX idx_bulkheads_service ON bulkheads(service_name);

-- ============================================================================
-- AUDIT TRAIL (7-year retention for compliance)
-- ============================================================================

-- Audit events table
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    
    -- Actor
    user_id VARCHAR(255) NOT NULL,
    user_name VARCHAR(255),
    user_role VARCHAR(100),
    
    -- Action
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),
    
    -- Outcome
    status VARCHAR(50) NOT NULL,
    status_code INTEGER,
    error_message TEXT,
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    correlation_id VARCHAR(255),
    
    -- Changes
    before_state JSONB,
    after_state JSONB,
    changes JSONB,
    
    -- Compliance
    compliance_framework VARCHAR(100),
    risk_level VARCHAR(50),
    requires_review BOOLEAN DEFAULT false,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by VARCHAR(255),
    
    -- Tamper protection
    hash VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64),
    
    CONSTRAINT valid_severity CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    CONSTRAINT valid_status CHECK (status IN ('success', 'failure', 'partial', 'error'))
);

-- Partitioning strategy for audit_events (by quarter for 7-year retention)
-- CREATE TABLE audit_events_2025_q4 PARTITION OF audit_events FOR VALUES FROM ('2025-10-01') TO ('2026-01-01');

-- Indexes for audit_events
CREATE INDEX idx_audit_events_tenant_id ON audit_events(tenant_id);
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_category ON audit_events(category);
CREATE INDEX idx_audit_events_severity ON audit_events(severity);
CREATE INDEX idx_audit_events_requires_review ON audit_events(requires_review) WHERE requires_review = true;
CREATE INDEX idx_audit_events_correlation_id ON audit_events(correlation_id) WHERE correlation_id IS NOT NULL;

-- Compliance reports table
CREATE TABLE compliance_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    report_name VARCHAR(255) NOT NULL,
    framework VARCHAR(100) NOT NULL,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Report data
    total_events INTEGER NOT NULL,
    compliant_events INTEGER NOT NULL,
    non_compliant_events INTEGER NOT NULL,
    critical_violations INTEGER NOT NULL,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'generated',
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    generated_by VARCHAR(255),
    
    -- Details
    summary TEXT,
    recommendations TEXT[],
    violations JSONB,
    report_data JSONB,
    
    CONSTRAINT valid_framework CHECK (framework IN ('SOC2', 'HIPAA', 'GDPR', 'PCI-DSS', 'ISO27001')),
    CONSTRAINT valid_status CHECK (status IN ('generated', 'reviewed', 'approved', 'archived'))
);

-- Indexes for compliance_reports
CREATE INDEX idx_compliance_reports_tenant_id ON compliance_reports(tenant_id);
CREATE INDEX idx_compliance_reports_framework ON compliance_reports(framework);
CREATE INDEX idx_compliance_reports_period ON compliance_reports(period_start, period_end);

-- Event correlation patterns table
CREATE TABLE event_correlation_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    pattern_name VARCHAR(255) NOT NULL,
    pattern_type VARCHAR(50) NOT NULL,
    description TEXT,
    
    -- Pattern definition
    event_sequence TEXT[],
    time_window_minutes INTEGER NOT NULL,
    min_occurrences INTEGER DEFAULT 1,
    conditions JSONB,
    
    -- Actions
    severity VARCHAR(50) NOT NULL,
    alert_enabled BOOLEAN DEFAULT true,
    alert_recipients TEXT[],
    
    -- Statistics
    matches_count INTEGER DEFAULT 0,
    last_match_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_pattern_type CHECK (pattern_type IN ('sequential', 'concurrent', 'frequency', 'anomaly')),
    CONSTRAINT valid_severity CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    UNIQUE(tenant_id, pattern_name)
);

-- Indexes for event_correlation_patterns
CREATE INDEX idx_correlation_patterns_tenant_id ON event_correlation_patterns(tenant_id);

-- SIEM integrations table
CREATE TABLE siem_integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    integration_name VARCHAR(255) NOT NULL,
    siem_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    -- Configuration
    endpoint_url TEXT NOT NULL,
    auth_type VARCHAR(50),
    api_key TEXT,
    format VARCHAR(50) NOT NULL,
    batch_size INTEGER DEFAULT 100,
    flush_interval_seconds INTEGER DEFAULT 60,
    
    -- Filtering
    event_types TEXT[],
    min_severity VARCHAR(50),
    
    -- Statistics
    events_sent BIGINT DEFAULT 0,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    last_error_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_siem_type CHECK (siem_type IN ('splunk', 'elastic', 'qradar', 'sentinel', 'sumo_logic', 'datadog')),
    CONSTRAINT valid_status CHECK (status IN ('active', 'paused', 'error', 'disabled')),
    CONSTRAINT valid_format CHECK (format IN ('json', 'cef', 'leef', 'syslog')),
    UNIQUE(tenant_id, integration_name)
);

-- Indexes for siem_integrations
CREATE INDEX idx_siem_integrations_tenant_id ON siem_integrations(tenant_id);
CREATE INDEX idx_siem_integrations_status ON siem_integrations(status);

-- ============================================================================
-- CONFIGURATION MANAGEMENT
-- ============================================================================

-- Configuration variables table
CREATE TABLE config_variables (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    variable_key VARCHAR(255) NOT NULL,
    variable_value TEXT NOT NULL,
    variable_type VARCHAR(50) NOT NULL,
    scope VARCHAR(50) NOT NULL DEFAULT 'global',
    
    -- Metadata
    description TEXT,
    is_sensitive BOOLEAN DEFAULT false,
    is_encrypted BOOLEAN DEFAULT false,
    category VARCHAR(100),
    
    -- Lifecycle
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    
    CONSTRAINT valid_variable_type CHECK (variable_type IN ('string', 'number', 'boolean', 'json')),
    CONSTRAINT valid_scope CHECK (scope IN ('global', 'tenant', 'environment')),
    UNIQUE(tenant_id, variable_key, scope)
);

-- Indexes for config_variables
CREATE INDEX idx_config_variables_tenant_id ON config_variables(tenant_id);
CREATE INDEX idx_config_variables_scope ON config_variables(scope);
CREATE INDEX idx_config_variables_category ON config_variables(category);

-- Configuration files table
CREATE TABLE config_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_format VARCHAR(50) NOT NULL,
    file_content TEXT NOT NULL,
    
    -- Metadata
    description TEXT,
    checksum VARCHAR(64),
    size_bytes INTEGER,
    
    -- Lifecycle
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    
    CONSTRAINT valid_format CHECK (file_format IN ('yaml', 'json', 'toml', 'properties')),
    UNIQUE(tenant_id, file_name, version)
);

-- Indexes for config_files
CREATE INDEX idx_config_files_tenant_id ON config_files(tenant_id);
CREATE INDEX idx_config_files_name ON config_files(file_name);

-- Service configurations table
CREATE TABLE service_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    config_version VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    
    -- Configuration
    config_data JSONB NOT NULL,
    environment VARCHAR(50),
    
    -- Lifecycle
    deployed_at TIMESTAMP WITH TIME ZONE,
    last_reload_at TIMESTAMP WITH TIME ZONE,
    requires_restart BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_status CHECK (status IN ('pending', 'deployed', 'active', 'failed')),
    UNIQUE(tenant_id, service_name, config_version)
);

-- Indexes for service_configs
CREATE INDEX idx_service_configs_tenant_id ON service_configs(tenant_id);
CREATE INDEX idx_service_configs_service ON service_configs(service_name);
CREATE INDEX idx_service_configs_status ON service_configs(status);

-- Tenant configuration overrides table
CREATE TABLE tenant_config_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    config_key VARCHAR(255) NOT NULL,
    override_value TEXT NOT NULL,
    override_type VARCHAR(50) NOT NULL,
    
    -- Lifecycle
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    
    CONSTRAINT valid_override_type CHECK (override_type IN ('string', 'number', 'boolean', 'json')),
    UNIQUE(tenant_id, config_key)
);

-- Indexes for tenant_config_overrides
CREATE INDEX idx_tenant_overrides_tenant_id ON tenant_config_overrides(tenant_id);
CREATE INDEX idx_tenant_overrides_enabled ON tenant_config_overrides(enabled);

-- Feature flags table
CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    flag_key VARCHAR(255) NOT NULL,
    flag_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- State
    enabled BOOLEAN DEFAULT false,
    rollout_percentage INTEGER DEFAULT 0,
    targeting_rules JSONB,
    
    -- Metadata
    category VARCHAR(100),
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    
    CONSTRAINT valid_rollout CHECK (rollout_percentage >= 0 AND rollout_percentage <= 100),
    UNIQUE(tenant_id, flag_key)
);

-- Indexes for feature_flags
CREATE INDEX idx_feature_flags_tenant_id ON feature_flags(tenant_id);
CREATE INDEX idx_feature_flags_enabled ON feature_flags(enabled);
CREATE INDEX idx_feature_flags_category ON feature_flags(category);

-- ============================================================================
-- REVOCATION TRANSPARENCY (Merkle Tree & Append-Only Log)
-- ============================================================================

-- Merkle tree nodes table
CREATE TABLE merkle_tree_nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    tree_version INTEGER NOT NULL DEFAULT 1,
    node_hash VARCHAR(64) NOT NULL,
    level INTEGER NOT NULL,
    position INTEGER NOT NULL,
    
    -- Tree structure
    is_leaf BOOLEAN NOT NULL DEFAULT false,
    left_child_hash VARCHAR(64),
    right_child_hash VARCHAR(64),
    parent_hash VARCHAR(64),
    
    -- Leaf data (for leaf nodes only)
    token_id VARCHAR(255),
    leaf_data JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, tree_version, level, position)
);

-- Indexes for merkle_tree_nodes
CREATE INDEX idx_merkle_nodes_tenant_id ON merkle_tree_nodes(tenant_id);
CREATE INDEX idx_merkle_nodes_tree_version ON merkle_tree_nodes(tree_version);
CREATE INDEX idx_merkle_nodes_hash ON merkle_tree_nodes(node_hash);
CREATE INDEX idx_merkle_nodes_level ON merkle_tree_nodes(level);
CREATE INDEX idx_merkle_nodes_token_id ON merkle_tree_nodes(token_id) WHERE token_id IS NOT NULL;

-- Merkle proofs table
CREATE TABLE merkle_proofs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    proof_id VARCHAR(255) UNIQUE NOT NULL,
    token_id VARCHAR(255) NOT NULL,
    tree_version INTEGER NOT NULL,
    
    -- Proof data
    leaf_hash VARCHAR(64) NOT NULL,
    root_hash VARCHAR(64) NOT NULL,
    proof_path JSONB NOT NULL,
    
    -- Verification
    verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, token_id, tree_version)
);

-- Indexes for merkle_proofs
CREATE INDEX idx_merkle_proofs_tenant_id ON merkle_proofs(tenant_id);
CREATE INDEX idx_merkle_proofs_token_id ON merkle_proofs(token_id);
CREATE INDEX idx_merkle_proofs_verified ON merkle_proofs(verified);

-- Revocations table
CREATE TABLE revocations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    token_id VARCHAR(255) NOT NULL,
    revocation_reason VARCHAR(255),
    
    -- Merkle tree integration
    leaf_hash VARCHAR(64) NOT NULL,
    merkle_root VARCHAR(64) NOT NULL,
    block_height INTEGER NOT NULL,
    tree_version INTEGER NOT NULL,
    
    -- Status
    verified BOOLEAN DEFAULT false,
    
    -- Lifecycle
    revoked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_by VARCHAR(255) NOT NULL,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    UNIQUE(tenant_id, token_id, tree_version)
);

-- Indexes for revocations
CREATE INDEX idx_revocations_tenant_id ON revocations(tenant_id);
CREATE INDEX idx_revocations_token_id ON revocations(token_id);
CREATE INDEX idx_revocations_verified ON revocations(verified);
CREATE INDEX idx_revocations_revoked_at ON revocations(revoked_at DESC);

-- Append-only log table
CREATE TABLE append_only_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    log_index BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    operation VARCHAR(50) NOT NULL,
    
    -- Entry data
    entry_data JSONB NOT NULL,
    
    -- Hash chain
    entry_hash VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,
    
    -- Verification
    verified BOOLEAN DEFAULT true,
    
    CONSTRAINT valid_operation CHECK (operation IN ('append', 'verify', 'audit')),
    UNIQUE(tenant_id, log_index)
);

-- Indexes for append_only_log
CREATE INDEX idx_append_log_tenant_id ON append_only_log(tenant_id);
CREATE INDEX idx_append_log_index ON append_only_log(log_index);
CREATE INDEX idx_append_log_timestamp ON append_only_log(timestamp DESC);

-- ============================================================================
-- ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

-- Enable RLS on all tenant-scoped tables
ALTER TABLE subscribers ENABLE ROW LEVEL SECURITY;
ALTER TABLE tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE token_blacklist ENABLE ROW LEVEL SECURITY;
ALTER TABLE policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_attributes ENABLE ROW LEVEL SECURITY;
ALTER TABLE authorization_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE poa_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE poa_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE events ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_handlers ENABLE ROW LEVEL SECURITY;
ALTER TABLE circuit_breakers ENABLE ROW LEVEL SECURITY;
ALTER TABLE rate_limiters ENABLE ROW LEVEL SECURITY;
ALTER TABLE retry_policies ENABLE ROW LEVEL SECURITY;
ALTER TABLE bulkheads ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_correlation_patterns ENABLE ROW LEVEL SECURITY;
ALTER TABLE siem_integrations ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_variables ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_config_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE merkle_tree_nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE merkle_proofs ENABLE ROW LEVEL SECURITY;
ALTER TABLE revocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE append_only_log ENABLE ROW LEVEL SECURITY;

-- Create RLS policy function
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS VARCHAR AS $$
BEGIN
    RETURN current_setting('app.current_tenant_id', true);
END;
$$ LANGUAGE plpgsql STABLE;

-- Example RLS policies (repeat for all tables)
-- These policies ensure users can only access data for their tenant

CREATE POLICY tenant_isolation_policy ON tokens
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY tenant_isolation_policy ON audit_events
    FOR ALL
    USING (tenant_id = current_tenant_id());

-- Add similar policies for all other tables...

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to relevant tables
CREATE TRIGGER update_subscribers_updated_at BEFORE UPDATE ON subscribers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_config_variables_updated_at BEFORE UPDATE ON config_variables
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feature_flags_updated_at BEFORE UPDATE ON feature_flags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_circuit_breakers_updated_at BEFORE UPDATE ON circuit_breakers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bulkheads_updated_at BEFORE UPDATE ON bulkheads
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS
-- ============================================================================

-- Active tokens view (non-revoked, non-expired)
CREATE VIEW active_tokens AS
SELECT *
FROM tokens
WHERE revoked_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP;

-- Recent audit events view (last 30 days)
CREATE VIEW recent_audit_events AS
SELECT *
FROM audit_events
WHERE timestamp > CURRENT_TIMESTAMP - INTERVAL '30 days'
ORDER BY timestamp DESC;

-- Active policies view
CREATE VIEW active_policies AS
SELECT *
FROM policies
WHERE status = 'active'
  AND (valid_from IS NULL OR valid_from <= CURRENT_TIMESTAMP)
  AND (valid_until IS NULL OR valid_until >= CURRENT_TIMESTAMP)
ORDER BY priority DESC;

-- Open circuit breakers view
CREATE VIEW open_circuit_breakers AS
SELECT *
FROM circuit_breakers
WHERE state = 'open'
ORDER BY last_state_change DESC;

-- ============================================================================
-- GRANTS
-- ============================================================================

-- Create application user role
CREATE ROLE gauth_app_user WITH LOGIN PASSWORD 'change_me_in_production';

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO gauth_app_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO gauth_app_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO gauth_app_user;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE subscribers IS 'Core tenant/subscriber information with OIDC and key configuration';
COMMENT ON TABLE tokens IS 'Long-term token metadata storage (active tokens cached in Redis)';
COMMENT ON TABLE token_blacklist IS 'Revoked tokens synchronized with Redis blacklist';
COMMENT ON TABLE policies IS 'Authorization policies (RBAC/ABAC/PBAC)';
COMMENT ON TABLE audit_events IS '7-year retention audit trail for compliance (SOC2, HIPAA, GDPR, PCI-DSS)';
COMMENT ON TABLE merkle_tree_nodes IS 'Merkle tree structure for revocation transparency';
COMMENT ON TABLE append_only_log IS 'Cryptographically verifiable append-only log for revocations';
COMMENT ON TABLE circuit_breakers IS 'Circuit breaker state for resilience patterns (state cached in Redis)';
COMMENT ON TABLE rate_limiters IS 'Rate limiter configuration (counters in Redis)';

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
