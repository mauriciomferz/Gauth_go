-- Create API Keys table (Phase 22)
-- Supports web/handlers/admin/apikey_handler.go

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    key_name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    description TEXT,
    scopes TEXT[] NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    
    -- Usage & Limits
    total_requests BIGINT DEFAULT 0,
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'revoked', 'expired')),
    UNIQUE(key_prefix) -- Ensure prefixes are unique for lookup if needed, though mostly cosmetic
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);

-- RLS
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

-- RLS Policy (Allow usage for now, refine for multi-tenant later)
-- In a real scenario, this should check the current tenant context
CREATE POLICY tenant_isolation_policy ON api_keys 
    FOR ALL 
    USING (tenant_id = current_setting('app.current_tenant_id', true));

-- For local dev/admin without setting context, we might want a permissive policy or rely on the app setting it.
-- Adding a permissive policy for testing/admin access if the setting is missing might be risky but helpful for dev restoration.
-- CREATE POLICY admin_access ON api_keys FOR ALL USING (current_setting('app.current_tenant_id', true) IS NULL); 
