-- API Keys and Security Configuration Schema
-- PostgreSQL 14+ with Row-Level Security (RLS)
-- Created: 2025-11-24
-- Tables: 2 (API Keys, Security Settings)

-- Enable required extensions (if not already enabled)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- API KEYS MANAGEMENT
-- ============================================================================

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    
    -- Key identification
    key_name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL, -- First 8 chars for display (e.g., "agentauth_pk_...")
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- SHA-256 hash of the full key
    
    -- Key metadata
    description TEXT,
    scopes TEXT[] NOT NULL DEFAULT '{}', -- Permissions: read, write, admin, etc.
    permissions JSONB DEFAULT '{}'::jsonb, -- Detailed permissions configuration
    
    -- Status and lifecycle
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    
    -- Rate limiting
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    
    -- Usage tracking
    total_requests BIGINT DEFAULT 0,
    last_request_ip VARCHAR(45),
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    CONSTRAINT valid_api_key_status CHECK (status IN ('active', 'revoked', 'expired', 'suspended')),
    CONSTRAINT valid_expiration CHECK (expires_at IS NULL OR expires_at > created_at),
    UNIQUE(tenant_id, key_name)
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_api_keys_created_at ON api_keys(created_at DESC);

COMMENT ON TABLE api_keys IS 'API key management for programmatic access';
COMMENT ON COLUMN api_keys.key_prefix IS 'Display-safe prefix (first 8 chars)';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of full key for validation';
COMMENT ON COLUMN api_keys.scopes IS 'High-level permission scopes';
COMMENT ON COLUMN api_keys.permissions IS 'Detailed permission configuration';

-- ============================================================================
-- SECURITY SETTINGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS security_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    
    -- Multi-Factor Authentication (MFA)
    mfa_enabled BOOLEAN DEFAULT false,
    mfa_methods TEXT[] DEFAULT '{}', -- totp, sms, email, webauthn
    mfa_required_for_admin BOOLEAN DEFAULT false,
    mfa_grace_period_hours INTEGER DEFAULT 24,
    
    -- IP Whitelisting
    ip_whitelist_enabled BOOLEAN DEFAULT false,
    ip_whitelist TEXT[] DEFAULT '{}', -- Array of CIDR ranges or IPs
    ip_whitelist_mode VARCHAR(20) DEFAULT 'allow', -- allow, deny
    
    -- Token Expiration Policies
    access_token_ttl_minutes INTEGER DEFAULT 60,
    refresh_token_ttl_days INTEGER DEFAULT 30,
    id_token_ttl_minutes INTEGER DEFAULT 60,
    api_key_default_ttl_days INTEGER DEFAULT 365,
    allow_token_refresh BOOLEAN DEFAULT true,
    max_token_lifetime_days INTEGER DEFAULT 90,
    
    -- Session Management
    session_timeout_minutes INTEGER DEFAULT 120,
    session_idle_timeout_minutes INTEGER DEFAULT 30,
    max_concurrent_sessions INTEGER DEFAULT 5,
    session_pinning_enabled BOOLEAN DEFAULT false,
    force_logout_on_password_change BOOLEAN DEFAULT true,
    
    -- Password Policy
    password_min_length INTEGER DEFAULT 12,
    password_require_uppercase BOOLEAN DEFAULT true,
    password_require_lowercase BOOLEAN DEFAULT true,
    password_require_numbers BOOLEAN DEFAULT true,
    password_require_special_chars BOOLEAN DEFAULT true,
    password_expiry_days INTEGER DEFAULT 90,
    password_history_count INTEGER DEFAULT 5,
    
    -- Login Security
    max_login_attempts INTEGER DEFAULT 5,
    lockout_duration_minutes INTEGER DEFAULT 30,
    suspicious_activity_detection BOOLEAN DEFAULT true,
    
    -- Advanced Security
    require_https BOOLEAN DEFAULT true,
    cors_allowed_origins TEXT[] DEFAULT '{}',
    csrf_protection_enabled BOOLEAN DEFAULT true,
    rate_limiting_enabled BOOLEAN DEFAULT true,
    
    -- Audit and Compliance
    audit_all_requests BOOLEAN DEFAULT false,
    log_pii_access BOOLEAN DEFAULT true,
    data_retention_days INTEGER DEFAULT 90,
    
    -- Notifications
    notify_on_new_device BOOLEAN DEFAULT true,
    notify_on_suspicious_login BOOLEAN DEFAULT true,
    notify_on_api_key_usage BOOLEAN DEFAULT false,
    security_contact_email VARCHAR(255),
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    last_reviewed_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_mfa_methods CHECK (
        mfa_methods <@ ARRAY['totp', 'sms', 'email', 'webauthn', 'backup_codes']
    ),
    CONSTRAINT valid_ip_whitelist_mode CHECK (ip_whitelist_mode IN ('allow', 'deny')),
    CONSTRAINT valid_token_ttls CHECK (
        access_token_ttl_minutes > 0 AND
        refresh_token_ttl_days > 0 AND
        id_token_ttl_minutes > 0
    ),
    CONSTRAINT valid_session_timeout CHECK (
        session_timeout_minutes > 0 AND
        session_idle_timeout_minutes > 0 AND
        session_idle_timeout_minutes <= session_timeout_minutes
    ),
    UNIQUE(tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_security_settings_tenant_id ON security_settings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_security_settings_mfa_enabled ON security_settings(mfa_enabled);
CREATE INDEX IF NOT EXISTS idx_security_settings_updated_at ON security_settings(updated_at DESC);

COMMENT ON TABLE security_settings IS 'Comprehensive security configuration per tenant';
COMMENT ON COLUMN security_settings.mfa_methods IS 'Enabled MFA methods: totp, sms, email, webauthn';
COMMENT ON COLUMN security_settings.ip_whitelist IS 'Array of allowed/blocked IP addresses or CIDR ranges';
COMMENT ON COLUMN security_settings.session_timeout_minutes IS 'Absolute session timeout';
COMMENT ON COLUMN security_settings.session_idle_timeout_minutes IS 'Idle timeout (must be <= session_timeout)';

-- ============================================================================
-- API KEY USAGE LOGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS api_key_usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    
    -- Request details
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    endpoint VARCHAR(500) NOT NULL,
    http_method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER,
    
    -- Client information
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    
    -- Usage metadata
    request_id VARCHAR(100),
    error_message TEXT,
    
    CONSTRAINT valid_http_method CHECK (http_method IN ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD'))
);

CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_api_key_id ON api_key_usage_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_tenant_id ON api_key_usage_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_timestamp ON api_key_usage_logs(timestamp DESC);

COMMENT ON TABLE api_key_usage_logs IS 'API key usage tracking and audit trail';

-- ============================================================================
-- SECURITY AUDIT LOGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS security_audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    
    -- Event details
    event_type VARCHAR(100) NOT NULL, -- mfa_enabled, ip_whitelisted, token_expired, etc.
    event_category VARCHAR(50) NOT NULL, -- authentication, authorization, configuration
    severity VARCHAR(20) NOT NULL, -- info, warning, critical
    
    -- Actor information
    actor_id VARCHAR(255),
    actor_type VARCHAR(50), -- user, admin, system, api_key
    actor_ip VARCHAR(45),
    
    -- Target information
    target_resource VARCHAR(255),
    target_id VARCHAR(255),
    
    -- Event data
    description TEXT NOT NULL,
    details JSONB DEFAULT '{}'::jsonb,
    
    -- Metadata
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    session_id VARCHAR(255),
    request_id VARCHAR(255),
    
    CONSTRAINT valid_security_event_category CHECK (
        event_category IN ('authentication', 'authorization', 'configuration', 'data_access', 'system')
    ),
    CONSTRAINT valid_security_severity CHECK (severity IN ('info', 'warning', 'critical'))
);

CREATE INDEX IF NOT EXISTS idx_security_audit_logs_tenant_id ON security_audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_security_audit_logs_timestamp ON security_audit_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_security_audit_logs_event_type ON security_audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_security_audit_logs_severity ON security_audit_logs(severity);

COMMENT ON TABLE security_audit_logs IS 'Security-specific audit trail for compliance';

-- ============================================================================
-- ROW-LEVEL SECURITY (RLS)
-- ============================================================================

ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE security_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_key_usage_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE security_audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON api_keys
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON security_settings
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON api_key_usage_logs
    FOR ALL
    USING (tenant_id = current_tenant_id());

CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON security_audit_logs
    FOR ALL
    USING (tenant_id = current_tenant_id());

-- ============================================================================
-- TRIGGERS
-- ============================================================================

CREATE TRIGGER IF NOT EXISTS update_security_settings_updated_at BEFORE UPDATE ON security_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update last_used_at on API key usage
CREATE OR REPLACE FUNCTION update_api_key_last_used()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE api_keys 
    SET 
        last_used_at = NEW.timestamp,
        total_requests = total_requests + 1,
        last_request_ip = NEW.ip_address
    WHERE id = NEW.api_key_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER IF NOT EXISTS update_api_key_usage AFTER INSERT ON api_key_usage_logs
    FOR EACH ROW EXECUTE FUNCTION update_api_key_last_used();

-- Trigger to log security configuration changes
CREATE OR REPLACE FUNCTION log_security_settings_change()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO security_audit_logs (
        tenant_id,
        event_type,
        event_category,
        severity,
        actor_id,
        actor_type,
        description,
        details
    ) VALUES (
        NEW.tenant_id,
        'security_settings_updated',
        'configuration',
        'warning',
        NEW.updated_by,
        'admin',
        'Security settings modified',
        jsonb_build_object(
            'changes', jsonb_build_object(
                'mfa_enabled', jsonb_build_object('old', OLD.mfa_enabled, 'new', NEW.mfa_enabled),
                'ip_whitelist_enabled', jsonb_build_object('old', OLD.ip_whitelist_enabled, 'new', NEW.ip_whitelist_enabled)
            )
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER IF NOT EXISTS log_security_changes AFTER UPDATE ON security_settings
    FOR EACH ROW EXECUTE FUNCTION log_security_settings_change();

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Create default security settings for test tenant
INSERT INTO security_settings (tenant_id, security_contact_email, updated_by)
VALUES ('test-tenant-1', 'security@test-tenant.com', 'system')
ON CONFLICT (tenant_id) DO NOTHING;

-- ============================================================================
-- GRANTS
-- ============================================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON api_keys TO agentauth_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON security_settings TO agentauth_app;
GRANT SELECT, INSERT ON api_key_usage_logs TO agentauth_app;
GRANT SELECT, INSERT ON security_audit_logs TO agentauth_app;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO agentauth_app;

-- ============================================================================
-- SCHEMA COMPLETE
-- ============================================================================

-- Summary:
-- ✅ 4 tables created (api_keys, security_settings, api_key_usage_logs, security_audit_logs)
-- ✅ Row-Level Security (RLS) enabled for multi-tenant isolation
-- ✅ Indexes created for performance
-- ✅ Triggers for automatic updates and audit logging
-- ✅ Default security settings for test tenant
-- ✅ Application role permissions granted

