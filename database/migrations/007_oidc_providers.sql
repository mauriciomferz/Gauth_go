-- OIDC Providers Configuration Schema
-- PostgreSQL 14+ with Row-Level Security (RLS)
-- Created: 2025-11-24
-- Purpose: Store OpenID Connect provider configurations for authentication

-- Enable required extensions (if not already enabled)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- OIDC PROVIDERS
-- ============================================================================

CREATE TABLE IF NOT EXISTS oidc_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    
    -- Provider identification
    provider_name VARCHAR(255) NOT NULL, -- e.g., "Azure AD", "Google Workspace", "Okta"
    provider_type VARCHAR(50) NOT NULL, -- azure_ad, google, okta, auth0, custom
    display_name VARCHAR(255) NOT NULL, -- User-friendly name for UI
    
    -- OIDC Configuration (Discovery)
    issuer_url VARCHAR(500) NOT NULL, -- e.g., https://login.microsoftonline.com/{tenant}/v2.0
    authorization_endpoint VARCHAR(500),
    token_endpoint VARCHAR(500),
    userinfo_endpoint VARCHAR(500),
    jwks_uri VARCHAR(500),
    end_session_endpoint VARCHAR(500),
    
    -- Client credentials
    client_id VARCHAR(500) NOT NULL,
    client_secret TEXT, -- Encrypted client secret
    
    -- Scopes and claims
    scopes TEXT[] DEFAULT ARRAY['openid', 'profile', 'email'], -- OIDC scopes to request
    claims_mapping JSONB DEFAULT '{}'::jsonb, -- Map provider claims to internal user attributes
    
    -- Redirect URIs
    redirect_uris TEXT[] NOT NULL, -- Allowed redirect URIs after authentication
    post_logout_redirect_uris TEXT[], -- URIs to redirect after logout
    
    -- Token validation
    validate_issuer BOOLEAN DEFAULT true,
    validate_audience BOOLEAN DEFAULT true,
    validate_signature BOOLEAN DEFAULT true,
    clock_skew_seconds INTEGER DEFAULT 300, -- Allow 5 minutes clock skew
    
    -- User provisioning
    auto_provision_users BOOLEAN DEFAULT true, -- Automatically create users on first login
    user_attribute_mapping JSONB DEFAULT '{}'::jsonb, -- Map OIDC claims to user attributes
    default_role VARCHAR(100), -- Default role for auto-provisioned users
    
    -- Advanced settings
    pkce_enabled BOOLEAN DEFAULT true, -- Proof Key for Code Exchange
    response_type VARCHAR(50) DEFAULT 'code', -- code, id_token, token
    response_mode VARCHAR(50) DEFAULT 'query', -- query, fragment, form_post
    prompt VARCHAR(50), -- none, login, consent, select_account
    max_age INTEGER, -- Maximum authentication age in seconds
    
    -- Azure AD specific
    azure_tenant_id VARCHAR(100), -- For Azure AD multi-tenant apps
    azure_resource VARCHAR(500), -- Azure AD resource/audience
    
    -- Custom OIDC parameters
    additional_params JSONB DEFAULT '{}'::jsonb, -- Provider-specific parameters
    
    -- Status and lifecycle
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    is_default BOOLEAN DEFAULT false, -- Default provider for tenant
    priority INTEGER DEFAULT 0, -- Display order in UI
    
    -- Metadata
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    last_sync_at TIMESTAMP WITH TIME ZONE, -- Last metadata sync from discovery endpoint
    
    -- Configuration validation
    config_valid BOOLEAN DEFAULT true,
    validation_errors TEXT[],
    last_validated_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_oidc_provider_type CHECK (
        provider_type IN ('azure_ad', 'google', 'okta', 'auth0', 'custom')
    ),
    CONSTRAINT valid_oidc_status CHECK (
        status IN ('active', 'inactive', 'testing', 'error')
    ),
    CONSTRAINT valid_oidc_response_type CHECK (
        response_type IN ('code', 'id_token', 'token', 'code id_token', 'code token', 'id_token token', 'code id_token token')
    ),
    CONSTRAINT valid_oidc_response_mode CHECK (
        response_mode IN ('query', 'fragment', 'form_post')
    ),
    CONSTRAINT valid_oidc_prompt CHECK (
        prompt IS NULL OR prompt IN ('none', 'login', 'consent', 'select_account')
    ),
    UNIQUE(tenant_id, provider_name)
);

CREATE INDEX IF NOT EXISTS idx_oidc_providers_tenant_id ON oidc_providers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_oidc_providers_status ON oidc_providers(status);
CREATE INDEX IF NOT EXISTS idx_oidc_providers_provider_type ON oidc_providers(provider_type);
CREATE INDEX IF NOT EXISTS idx_oidc_providers_is_default ON oidc_providers(is_default) WHERE is_default = true;
CREATE INDEX IF NOT EXISTS idx_oidc_providers_priority ON oidc_providers(tenant_id, priority);

COMMENT ON TABLE oidc_providers IS 'OpenID Connect provider configurations for authentication';
COMMENT ON COLUMN oidc_providers.issuer_url IS 'OIDC issuer identifier (must match token iss claim)';
COMMENT ON COLUMN oidc_providers.claims_mapping IS 'Maps provider claims to internal user attributes';
COMMENT ON COLUMN oidc_providers.user_attribute_mapping IS 'Maps OIDC user info to internal user model';
COMMENT ON COLUMN oidc_providers.pkce_enabled IS 'Enable Proof Key for Code Exchange (recommended for security)';

-- ============================================================================
-- OIDC AUTHENTICATION SESSIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS oidc_auth_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES oidc_providers(id) ON DELETE CASCADE,
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    
    -- Session identification
    state VARCHAR(500) NOT NULL UNIQUE, -- OIDC state parameter
    nonce VARCHAR(500) NOT NULL, -- OIDC nonce for replay protection
    code_verifier VARCHAR(128), -- PKCE code verifier
    code_challenge VARCHAR(128), -- PKCE code challenge
    
    -- Request details
    redirect_uri VARCHAR(500) NOT NULL,
    scope TEXT[] NOT NULL,
    response_type VARCHAR(50) NOT NULL,
    prompt VARCHAR(50),
    
    -- Session state
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    user_id VARCHAR(255), -- Set after successful authentication
    
    -- Token data (after exchange)
    access_token TEXT,
    id_token TEXT,
    refresh_token TEXT,
    token_type VARCHAR(50),
    expires_in INTEGER,
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- User info (from userinfo endpoint)
    user_info JSONB DEFAULT '{}'::jsonb,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    
    CONSTRAINT valid_oidc_session_status CHECK (
        status IN ('pending', 'completed', 'failed', 'expired')
    ),
    CONSTRAINT valid_oidc_token_type CHECK (
        token_type IS NULL OR token_type IN ('Bearer', 'bearer')
    )
);

CREATE INDEX IF NOT EXISTS idx_oidc_auth_sessions_provider_id ON oidc_auth_sessions(provider_id);
CREATE INDEX IF NOT EXISTS idx_oidc_auth_sessions_tenant_id ON oidc_auth_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_oidc_auth_sessions_state ON oidc_auth_sessions(state);
CREATE INDEX IF NOT EXISTS idx_oidc_auth_sessions_status ON oidc_auth_sessions(status);
CREATE INDEX IF NOT EXISTS idx_oidc_auth_sessions_user_id ON oidc_auth_sessions(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_oidc_auth_sessions_created_at ON oidc_auth_sessions(created_at DESC);

COMMENT ON TABLE oidc_auth_sessions IS 'Tracks OIDC authentication flows and sessions';
COMMENT ON COLUMN oidc_auth_sessions.state IS 'Random state parameter for CSRF protection';
COMMENT ON COLUMN oidc_auth_sessions.nonce IS 'Random nonce for replay attack prevention';
COMMENT ON COLUMN oidc_auth_sessions.code_verifier IS 'PKCE code verifier (S256 hashed to create challenge)';

-- ============================================================================
-- OIDC USER MAPPINGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS oidc_user_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES oidc_providers(id) ON DELETE CASCADE,
    
    -- User identification
    user_id VARCHAR(255) NOT NULL, -- Internal user ID
    provider_user_id VARCHAR(500) NOT NULL, -- Provider's user ID (sub claim)
    provider_email VARCHAR(255), -- User's email from provider
    provider_username VARCHAR(255), -- Username from provider
    
    -- User attributes from provider
    given_name VARCHAR(255),
    family_name VARCHAR(255),
    name VARCHAR(500),
    email VARCHAR(255),
    email_verified BOOLEAN DEFAULT false,
    picture_url VARCHAR(500),
    locale VARCHAR(10),
    
    -- Provider-specific data
    provider_data JSONB DEFAULT '{}'::jsonb, -- Raw claims from provider
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMP WITH TIME ZONE,
    login_count INTEGER DEFAULT 0,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_oidc_mapping_status CHECK (
        status IN ('active', 'disabled', 'suspended')
    ),
    UNIQUE(provider_id, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_oidc_user_mappings_tenant_id ON oidc_user_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_oidc_user_mappings_provider_id ON oidc_user_mappings(provider_id);
CREATE INDEX IF NOT EXISTS idx_oidc_user_mappings_user_id ON oidc_user_mappings(user_id);
CREATE INDEX IF NOT EXISTS idx_oidc_user_mappings_provider_user_id ON oidc_user_mappings(provider_user_id);
CREATE INDEX IF NOT EXISTS idx_oidc_user_mappings_email ON oidc_user_mappings(email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_oidc_user_mappings_status ON oidc_user_mappings(status);

COMMENT ON TABLE oidc_user_mappings IS 'Maps external OIDC users to internal user accounts';
COMMENT ON COLUMN oidc_user_mappings.provider_user_id IS 'Unique identifier from provider (sub claim)';

-- ============================================================================
-- ROW-LEVEL SECURITY (RLS)
-- ============================================================================

ALTER TABLE oidc_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE oidc_auth_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE oidc_user_mappings ENABLE ROW LEVEL SECURITY;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE policyname = 'tenant_isolation_policy' AND tablename = 'oidc_providers') THEN
        CREATE POLICY tenant_isolation_policy ON oidc_providers FOR ALL USING (tenant_id = current_tenant_id());
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE policyname = 'tenant_isolation_policy' AND tablename = 'oidc_auth_sessions') THEN
        CREATE POLICY tenant_isolation_policy ON oidc_auth_sessions FOR ALL USING (tenant_id = current_tenant_id());
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE policyname = 'tenant_isolation_policy' AND tablename = 'oidc_user_mappings') THEN
        CREATE POLICY tenant_isolation_policy ON oidc_user_mappings FOR ALL USING (tenant_id = current_tenant_id());
    END IF;
END $$;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_oidc_providers_updated_at') THEN
        CREATE TRIGGER update_oidc_providers_updated_at BEFORE UPDATE ON oidc_providers
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_oidc_auth_sessions_updated_at') THEN
        CREATE TRIGGER update_oidc_auth_sessions_updated_at BEFORE UPDATE ON oidc_auth_sessions
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_oidc_user_mappings_updated_at') THEN
        CREATE TRIGGER update_oidc_user_mappings_updated_at BEFORE UPDATE ON oidc_user_mappings
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- Trigger to ensure only one default provider per tenant
CREATE OR REPLACE FUNCTION enforce_single_default_oidc_provider()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_default = true THEN
        UPDATE oidc_providers 
        SET is_default = false 
        WHERE tenant_id = NEW.tenant_id 
        AND id != NEW.id 
        AND is_default = true;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'ensure_single_default_provider') THEN
        CREATE TRIGGER ensure_single_default_provider BEFORE INSERT OR UPDATE ON oidc_providers
            FOR EACH ROW 
            WHEN (NEW.is_default = true)
            EXECUTE FUNCTION enforce_single_default_oidc_provider();
    END IF;
END $$;

-- Trigger to update login statistics on user mapping
CREATE OR REPLACE FUNCTION update_oidc_user_login_stats()
RETURNS TRIGGER AS $$
BEGIN
    NEW.login_count := NEW.login_count + 1;
    NEW.last_login_at := CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- GRANTS
-- ============================================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON oidc_providers TO agentauth_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON oidc_auth_sessions TO agentauth_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON oidc_user_mappings TO agentauth_app;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO agentauth_app;

-- ============================================================================
-- SCHEMA COMPLETE
-- ============================================================================

-- Summary:
-- ✅ 3 tables created (oidc_providers, oidc_auth_sessions, oidc_user_mappings)
-- ✅ Support for Azure AD, Google Workspace, Okta, Auth0, and custom OIDC providers
-- ✅ PKCE (Proof Key for Code Exchange) support for enhanced security
-- ✅ User provisioning and attribute mapping
-- ✅ Row-Level Security (RLS) for multi-tenant isolation
-- ✅ Indexes for performance
-- ✅ Triggers for data integrity
-- ✅ Application role permissions granted
