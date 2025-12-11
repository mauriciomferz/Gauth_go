-- Create Config Subsystem Tables

-- 1. config_files
CREATE TABLE IF NOT EXISTS config_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_format VARCHAR(50) NOT NULL DEFAULT 'yaml',
    file_content TEXT NOT NULL,
    description TEXT,
    checksum VARCHAR(64),
    size_bytes INTEGER,
    version INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    UNIQUE(tenant_id, file_name, version)
);

CREATE INDEX IF NOT EXISTS idx_config_files_tenant ON config_files(tenant_id);

-- 2. service_configs
CREATE TABLE IF NOT EXISTS service_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    config_version VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    config_data JSONB NOT NULL,
    environment VARCHAR(50),
    deployed_at TIMESTAMP WITH TIME ZONE,
    last_reload_at TIMESTAMP WITH TIME ZONE,
    requires_restart BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, service_name, config_version)
);

CREATE INDEX IF NOT EXISTS idx_service_configs_tenant ON service_configs(tenant_id);

-- 3. tenant_config_overrides
CREATE TABLE IF NOT EXISTS tenant_config_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    config_key VARCHAR(255) NOT NULL,
    override_value TEXT NOT NULL,
    override_type VARCHAR(50) NOT NULL DEFAULT 'string',
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    UNIQUE(tenant_id, config_key)
);

CREATE INDEX IF NOT EXISTS idx_tenant_overrides_tenant ON tenant_config_overrides(tenant_id);

-- 4. feature_flags
CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    flag_key VARCHAR(255) NOT NULL,
    flag_name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT false,
    rollout_percentage INTEGER DEFAULT 0,
    targeting_rules JSONB,
    category VARCHAR(100),
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    UNIQUE(tenant_id, flag_key)
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_tenant ON feature_flags(tenant_id);
