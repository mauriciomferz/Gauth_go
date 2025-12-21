-- Migration: Create Audit and Subscriber Tables
-- Covers: Audit Trail, Compliance Reports, Event Correlation, SIEM Integrations, and Subscribers
-- Version: 005
-- Created: 2025-12-21

-- ============================================================================
-- SUBSCRIBER TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscribers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_name VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL, -- active, suspended, pending, disabled
    tier VARCHAR(50) NOT NULL, -- free, standard, premium, enterprise
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    
    -- Legacy/Alias Columns (Required by Repository)
    subscriber_id VARCHAR(255),
    subscriber_name VARCHAR(255),
    
    -- OIDC Configuration
    oidc_provider VARCHAR(50),
    oidc_issuer VARCHAR(255),
    oidc_client_id VARCHAR(255),
    oidc_client_secret VARCHAR(255),
    oidc_scopes TEXT[],
    oidc_discovery_url VARCHAR(255),
    
    -- Key Management
    key_type VARCHAR(50),
    public_key TEXT,
    private_key_id VARCHAR(255),
    key_generated_at TIMESTAMP WITH TIME ZONE,
    key_expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Policy & Legal
    policy_template VARCHAR(255),
    legal_framework VARCHAR(255),
    
    -- Notification
    notification_channels TEXT[],
    notification_email VARCHAR(255),
    notification_webhook_url VARCHAR(255),
    
    -- Contact & Limits
    contact_email VARCHAR(255),
    contact_name VARCHAR(255),
    domain VARCHAR(255),
    max_users INTEGER DEFAULT 0,
    max_tokens INTEGER DEFAULT 0,
    metadata JSONB
);

-- Indexes for Subscribers
CREATE INDEX idx_subscribers_tenant_id ON subscribers(tenant_id);
CREATE INDEX idx_subscribers_status ON subscribers(status);
CREATE INDEX idx_subscribers_tier ON subscribers(tier);
CREATE INDEX idx_subscribers_contact_email ON subscribers(contact_email);


-- ============================================================================
-- AUDIT EVENTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS audit_events (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    
    -- Actor
    user_id VARCHAR(255),
    user_name VARCHAR(255),
    user_role VARCHAR(255),
    
    -- Action & Resource
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(255),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),
    
    -- Status
    status VARCHAR(50),
    status_code INTEGER,
    error_message TEXT,
    
    -- Context
    ip_address VARCHAR(255),
    user_agent TEXT,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    correlation_id VARCHAR(255),
    
    -- State Changes
    before_state JSONB,
    after_state JSONB,
    changes JSONB,
    
    -- Compliance & Review
    compliance_framework VARCHAR(255),
    risk_level VARCHAR(50),
    requires_review BOOLEAN DEFAULT FALSE,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by VARCHAR(255),
    
    -- Chain Integrity
    hash VARCHAR(255),
    previous_hash VARCHAR(255)
);

-- Indexes for Audit Events
CREATE INDEX idx_audit_events_tenant_id ON audit_events(tenant_id);
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX idx_audit_events_resource_id ON audit_events(resource_id);
CREATE INDEX idx_audit_events_category ON audit_events(category);
CREATE INDEX idx_audit_events_severity ON audit_events(severity);


-- ============================================================================
-- COMPLIANCE REPORTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS compliance_reports (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    report_name VARCHAR(255) NOT NULL,
    framework VARCHAR(100) NOT NULL,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Metrics
    total_events INTEGER DEFAULT 0,
    compliant_events INTEGER DEFAULT 0,
    non_compliant_events INTEGER DEFAULT 0,
    critical_violations INTEGER DEFAULT 0,
    
    status VARCHAR(50) NOT NULL,
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    generated_by VARCHAR(255),
    
    summary TEXT,
    recommendations JSONB,
    violations JSONB,
    report_data JSONB
);

-- Indexes for Compliance Reports
CREATE INDEX idx_compliance_reports_tenant_id ON compliance_reports(tenant_id);
CREATE INDEX idx_compliance_reports_framework ON compliance_reports(framework);
CREATE INDEX idx_compliance_reports_generated_at ON compliance_reports(generated_at DESC);


-- ============================================================================
-- EVENT CORRELATION PATTERNS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS event_correlation_patterns (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    pattern_name VARCHAR(255) NOT NULL,
    pattern_type VARCHAR(100) NOT NULL,
    description TEXT,
    
    event_sequence TEXT[],
    time_window_minutes INTEGER NOT NULL,
    min_occurrences INTEGER NOT NULL DEFAULT 1,
    conditions JSONB,
    
    severity VARCHAR(50) NOT NULL,
    alert_enabled BOOLEAN DEFAULT FALSE,
    alert_recipients TEXT[],
    
    matches_count INTEGER DEFAULT 0,
    last_match_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for Correlation Patterns
CREATE INDEX idx_correlation_patterns_tenant_id ON event_correlation_patterns(tenant_id);


-- ============================================================================
-- SIEM INTEGRATIONS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS siem_integrations (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    integration_name VARCHAR(255) NOT NULL,
    siem_type VARCHAR(100) NOT NULL, -- splunk, datadog, elastic, sentinel, etc.
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    endpoint_url VARCHAR(2048) NOT NULL,
    auth_type VARCHAR(50),
    api_key VARCHAR(255),
    format VARCHAR(50) DEFAULT 'json',
    
    batch_size INTEGER DEFAULT 100,
    flush_interval_seconds INTEGER DEFAULT 60,
    event_types TEXT[],
    min_severity VARCHAR(50),
    
    events_sent BIGINT DEFAULT 0,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    last_error_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for SIEM Integrations
CREATE INDEX idx_siem_integrations_tenant_id ON siem_integrations(tenant_id);
