-- Migration: Create extended_tokens table
-- RFC-0111 Extended Token Storage
-- Version: 001
-- Created: 2025-11-11

CREATE TABLE IF NOT EXISTS extended_tokens (
    -- Primary key
    access_token VARCHAR(512) PRIMARY KEY,
    
    -- OAuth 2.0 fields
    token_type VARCHAR(50) NOT NULL DEFAULT 'GAuth-Extended-Token',
    expires_in BIGINT NOT NULL,
    refresh_token VARCHAR(512),
    scope TEXT[], -- Array of scope strings
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- RFC-0111 Extended Token Data (stored as JSONB for flexibility)
    power_of_attorney JSONB,
    authorization_chain JSONB NOT NULL,
    client_owner JSONB,
    owners_authorizer JSONB,
    resource_owner JSONB,
    legal_framework JSONB NOT NULL,
    restrictions JSONB,
    issued_by JSONB NOT NULL,
    verification_proof JSONB NOT NULL,
    
    -- Request Context
    request_id VARCHAR(255),
    grant_id VARCHAR(255) NOT NULL,
    transaction_context JSONB,
    
    -- Compliance & Audit
    compliance_level VARCHAR(100) NOT NULL,
    audit_trail JSONB,
    jurisdiction_context JSONB,
    
    -- Token Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    use_count INTEGER NOT NULL DEFAULT 0,
    
    -- Indexes for common queries
    CONSTRAINT refresh_token_unique UNIQUE (refresh_token)
);

-- Create indexes for performance
CREATE INDEX idx_extended_tokens_issued_at ON extended_tokens(issued_at, expires_in) WHERE revoked_at IS NULL;
CREATE INDEX idx_extended_tokens_revoked ON extended_tokens(revoked_at) WHERE revoked_at IS NOT NULL;
CREATE INDEX idx_extended_tokens_client_id ON extended_tokens(((authorization_chain->'client'->>'entity_id')));
CREATE INDEX idx_extended_tokens_grant_id ON extended_tokens(grant_id);
CREATE INDEX idx_extended_tokens_created_at ON extended_tokens(created_at DESC);

-- Add comments
COMMENT ON TABLE extended_tokens IS 'RFC-0111 Extended Authorization Tokens with comprehensive compliance metadata';
COMMENT ON COLUMN extended_tokens.access_token IS 'OAuth 2.0 access token (primary key)';
COMMENT ON COLUMN extended_tokens.authorization_chain IS 'RFC-0111 authorization chain (owners authorizer -> client owner -> client)';
COMMENT ON COLUMN extended_tokens.legal_framework IS 'Legal framework including applicable laws and jurisdiction';
COMMENT ON COLUMN extended_tokens.verification_proof IS 'Identity verification chain for all parties';
COMMENT ON COLUMN extended_tokens.compliance_level IS 'RFC-0111 compliance level (e.g., rfc-0111-compliant)';
