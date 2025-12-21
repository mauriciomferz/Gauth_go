-- Migration: Create OIDC Providers table
-- Version: 011
-- Created: 2025-12-21

CREATE TABLE IF NOT EXISTS oidc_providers (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    provider_name VARCHAR(255) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    
    -- OIDC Configuration
    issuer_url VARCHAR(2048) NOT NULL,
    authorization_endpoint VARCHAR(2048),
    token_endpoint VARCHAR(2048),
    userinfo_endpoint VARCHAR(2048),
    jwks_uri VARCHAR(2048),
    end_session_endpoint VARCHAR(2048),
    
    -- Client Configuration
    client_id VARCHAR(255) NOT NULL,
    client_secret VARCHAR(2048),
    scopes TEXT[] DEFAULT '{}',
    redirect_uris TEXT[] DEFAULT '{}',
    post_logout_redirect_uris TEXT[] DEFAULT '{}',
    
    -- Mappings
    claims_mapping JSONB DEFAULT '{}',
    user_attribute_mapping JSONB DEFAULT '{}',
    additional_params JSONB DEFAULT '{}',
    
    -- Validation & Security
    validate_issuer BOOLEAN DEFAULT TRUE,
    validate_audience BOOLEAN DEFAULT TRUE,
    validate_signature BOOLEAN DEFAULT TRUE,
    clock_skew_seconds INTEGER DEFAULT 300,
    pkce_enabled BOOLEAN DEFAULT TRUE,
    
    -- Flow Configuration
    auto_provision_users BOOLEAN DEFAULT TRUE,
    default_role VARCHAR(255),
    response_type VARCHAR(50) DEFAULT 'code',
    response_mode VARCHAR(50) DEFAULT 'query',
    prompt VARCHAR(50),
    max_age INTEGER,
    
    -- Azure Specific
    azure_tenant_id VARCHAR(255),
    azure_resource VARCHAR(255),

    -- Metadata
    status VARCHAR(50) DEFAULT 'active',
    is_default BOOLEAN DEFAULT FALSE,
    priority INTEGER DEFAULT 0,
    
    -- Audit
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_by VARCHAR(255),
    
    -- Status & Validation
    last_sync_at TIMESTAMP WITH TIME ZONE,
    config_valid BOOLEAN DEFAULT FALSE,
    validation_errors TEXT[] DEFAULT '{}',
    last_validated_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    UNIQUE(tenant_id, provider_name)
);

CREATE INDEX idx_oidc_providers_tenant ON oidc_providers(tenant_id);
CREATE INDEX idx_oidc_providers_status ON oidc_providers(tenant_id, status);
