-- Migration: Create Tokens and Blacklist Tables
-- Covers: Standard OAuth/Session Tokens and Token Blacklist
-- Version: 006
-- Created: 2025-12-21

-- ============================================================================
-- TOKENS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS tokens (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    token_id VARCHAR(512) NOT NULL UNIQUE, -- The actual token string or hash
    tenant_id VARCHAR(255) NOT NULL,
    token_type VARCHAR(50) NOT NULL, -- access, refresh, api_key
    subject VARCHAR(255) NOT NULL,
    audience TEXT[],
    issuer VARCHAR(255),
    scope TEXT[],
    
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    
    ip_address VARCHAR(255),
    user_agent TEXT,
    device_id VARCHAR(255),
    usage_count INTEGER DEFAULT 0
);

-- Indexes for Tokens
CREATE INDEX idx_tokens_tenant_id ON tokens(tenant_id);
CREATE INDEX idx_tokens_token_id ON tokens(token_id);
CREATE INDEX idx_tokens_subject ON tokens(subject);
CREATE INDEX idx_tokens_expires_at ON tokens(expires_at);
CREATE INDEX idx_tokens_revoked_at ON tokens(revoked_at);


-- ============================================================================
-- TOKEN BLACKLIST TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS token_blacklist (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    token_id VARCHAR(512) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    reason TEXT,
    revoked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_by VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    CONSTRAINT token_blacklist_unique UNIQUE (token_id, tenant_id)
);

-- Indexes for Blacklist
CREATE INDEX idx_blacklist_token_tenant ON token_blacklist(token_id, tenant_id);
CREATE INDEX idx_blacklist_expires_at ON token_blacklist(expires_at);
