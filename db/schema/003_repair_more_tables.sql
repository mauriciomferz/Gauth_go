-- Repair missing tables Part 2

-- Tokens table
CREATE TABLE IF NOT EXISTS tokens (
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

CREATE INDEX IF NOT EXISTS idx_tokens_token_id ON tokens(token_id);
CREATE INDEX IF NOT EXISTS idx_tokens_tenant_id ON tokens(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tokens_subject ON tokens(subject);
CREATE INDEX IF NOT EXISTS idx_tokens_expires_at ON tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_tokens_revoked_at ON tokens(revoked_at) WHERE revoked_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tokens_issued_at ON tokens(issued_at DESC);

-- Token blacklist table
CREATE TABLE IF NOT EXISTS token_blacklist (
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

CREATE INDEX IF NOT EXISTS idx_blacklist_token_id ON token_blacklist(token_id);
CREATE INDEX IF NOT EXISTS idx_blacklist_tenant_id ON token_blacklist(tenant_id);
CREATE INDEX IF NOT EXISTS idx_blacklist_expires_at ON token_blacklist(expires_at);

-- PoA Records table (New Schema)
CREATE TABLE IF NOT EXISTS poa_records (
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

CREATE INDEX IF NOT EXISTS idx_poa_tenant_id ON poa_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_poa_grantor_id ON poa_records(grantor_id);
CREATE INDEX IF NOT EXISTS idx_poa_representative_id ON poa_records(representative_id);
CREATE INDEX IF NOT EXISTS idx_poa_status ON poa_records(status);
CREATE INDEX IF NOT EXISTS idx_poa_valid_from ON poa_records(valid_from);
CREATE INDEX IF NOT EXISTS idx_poa_valid_until ON poa_records(valid_until);

-- PoA Templates table
CREATE TABLE IF NOT EXISTS poa_templates (
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

CREATE INDEX IF NOT EXISTS idx_poa_templates_tenant_id ON poa_templates(tenant_id);

-- Fix Audit Events IP Address Type
ALTER TABLE audit_events ALTER COLUMN ip_address TYPE VARCHAR(45);

-- Enable RLS
ALTER TABLE tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE token_blacklist ENABLE ROW LEVEL SECURITY;
ALTER TABLE poa_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE poa_templates ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY tenant_isolation_policy ON tokens FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON token_blacklist FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON poa_records FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON poa_templates FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
