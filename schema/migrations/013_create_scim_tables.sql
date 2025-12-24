-- Migration: Create SCIM Clients table
-- Version: 013
-- Created: 2025-12-21

CREATE TABLE IF NOT EXISTS scim_clients (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    client_name VARCHAR(255) NOT NULL,
    
    -- Authentication
    token_id VARCHAR(255) NOT NULL,         -- Reference to an access token or specialized SCIM token
    scim_base_url VARCHAR(2048),            -- Optional: configured base URL for this client
    
    -- Permissions & Scope
    is_active BOOLEAN DEFAULT TRUE,
    permissions JSONB DEFAULT '["read", "write"]', -- Granular SCIM permissions
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(tenant_id, client_name)
);

CREATE INDEX idx_scim_clients_tenant ON scim_clients(tenant_id);
CREATE INDEX idx_scim_clients_token ON scim_clients(token_id);
