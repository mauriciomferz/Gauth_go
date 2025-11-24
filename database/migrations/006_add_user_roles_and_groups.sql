-- Migration: Add user roles and groups tables for OIDC provisioning
-- Created: 2025-11-24

-- User roles table for RBAC
CREATE TABLE IF NOT EXISTS user_roles (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    role VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_role UNIQUE (user_id, tenant_id, role)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_id ON user_roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role);

-- OIDC user groups table for group synchronization
CREATE TABLE IF NOT EXISTS oidc_user_groups (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL,
    group_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_provider_group UNIQUE (user_id, provider_id, group_name),
    CONSTRAINT fk_oidc_user_groups_provider 
        FOREIGN KEY (provider_id) 
        REFERENCES oidc_providers(id) 
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_oidc_user_groups_user_id ON oidc_user_groups(user_id);
CREATE INDEX IF NOT EXISTS idx_oidc_user_groups_provider_id ON oidc_user_groups(provider_id);
CREATE INDEX IF NOT EXISTS idx_oidc_user_groups_group_name ON oidc_user_groups(group_name);

-- Add comments
COMMENT ON TABLE user_roles IS 'Stores user role assignments for RBAC';
COMMENT ON TABLE oidc_user_groups IS 'Stores OIDC group memberships synchronized from identity providers';

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON user_roles TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON oidc_user_groups TO postgres;
