-- Add missing tenant_config_overrides table
-- Version: 008

-- This table was missing from the original admin handlers migration
-- but is required by the config handler for tenant override functionality

CREATE TABLE IF NOT EXISTS tenant_config_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    config_key VARCHAR(255) NOT NULL,
    override_value TEXT NOT NULL,
    override_type VARCHAR(50) NOT NULL,
    
    -- Lifecycle
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    
    CONSTRAINT valid_override_type CHECK (override_type IN ('string', 'number', 'boolean', 'json')),
    UNIQUE(tenant_id, config_key)
);

-- Indexes for tenant_config_overrides
CREATE INDEX IF NOT EXISTS idx_tenant_overrides_tenant_id ON tenant_config_overrides(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_overrides_enabled ON tenant_config_overrides(enabled);

-- Enable RLS for tenant isolation
ALTER TABLE tenant_config_overrides ENABLE ROW LEVEL SECURITY;

-- Create RLS policy
CREATE POLICY IF NOT EXISTS tenant_isolation_policy ON tenant_config_overrides
    FOR ALL
    USING (tenant_id IS NULL OR tenant_id = current_tenant_id());

-- Create updated_at trigger
CREATE TRIGGER IF NOT EXISTS update_tenant_overrides_updated_at 
    BEFORE UPDATE ON tenant_config_overrides
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();