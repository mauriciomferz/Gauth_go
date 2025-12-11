-- Create Authorization Framework Tables (Phase 23)
-- Supports pkg/authz/repository.go

-- 1. Policy Attributes (PIP - Policy Information Point)
CREATE TABLE IF NOT EXISTS policy_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    attribute_name VARCHAR(255) NOT NULL,
    attribute_type VARCHAR(50) NOT NULL,
    source VARCHAR(100) NOT NULL,
    value_type VARCHAR(50) NOT NULL,
    description TEXT,
    sample_value TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, attribute_name)
);

CREATE INDEX IF NOT EXISTS idx_policy_attributes_tenant_id ON policy_attributes(tenant_id);

-- 2. Authorization Logs (PEP - Policy Enforcement Point Decision Logs)
CREATE TABLE IF NOT EXISTS authorization_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    decision VARCHAR(50) NOT NULL,
    
    -- Policy details
    policy_id VARCHAR(255),
    policy_name VARCHAR(255),
    
    -- Context
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    context JSONB,
    evaluation_time_ms INTEGER,
    
    CONSTRAINT valid_decision CHECK (decision IN ('allow', 'deny', 'indeterminate'))
);

CREATE INDEX IF NOT EXISTS idx_authz_logs_tenant_id ON authorization_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_authz_logs_timestamp ON authorization_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_authz_logs_user_id ON authorization_logs(user_id);

-- RLS
ALTER TABLE policy_attributes ENABLE ROW LEVEL SECURITY;
ALTER TABLE authorization_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY tenant_isolation_policy ON policy_attributes 
    FOR ALL 
    USING (tenant_id = current_setting('app.current_tenant_id', true));

CREATE POLICY tenant_isolation_policy ON authorization_logs 
    FOR ALL 
    USING (tenant_id = current_setting('app.current_tenant_id', true));
