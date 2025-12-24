-- Migration: Create SAML Providers table
-- Version: 012
-- Created: 2025-12-21

CREATE TABLE IF NOT EXISTS saml_providers (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    provider_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    
    -- IdP Configuration
    entity_id VARCHAR(2048) NOT NULL,       -- IdP Issuer URI
    sso_url VARCHAR(2048) NOT NULL,         -- IdP Single Sign-On Service URL
    slo_url VARCHAR(2048),                  -- IdP Single Logout Service URL
    certificate TEXT NOT NULL,              -- IdP X.509 Certificate (PEM)
    
    -- SP Configuration (Validation & Mapping)
    request_signing_cert TEXT,              -- SP Certificate for signing requests (optional)
    request_signing_key TEXT,               -- SP Private Key for signing requests (optional, encrypted)
    sign_requests BOOLEAN DEFAULT FALSE,    -- Whether to sign AuthnRequests
    want_assertions_signed BOOLEAN DEFAULT TRUE,
    want_response_signed BOOLEAN DEFAULT FALSE,
    attribute_mapping JSONB DEFAULT '{}',   -- Map SAML attributes to User fields
    
    -- Metadata
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    
    UNIQUE(tenant_id, provider_name),
    UNIQUE(tenant_id, entity_id)
);

CREATE INDEX idx_saml_providers_tenant ON saml_providers(tenant_id);
