-- Migration to update power_of_attorney table to match repository expectations
-- This adds the missing columns needed by the PoA repository

-- Drop and recreate the power_of_attorney table with correct schema
DROP TABLE IF EXISTS delegation_chains CASCADE;
DROP TABLE IF EXISTS power_of_attorney CASCADE;

CREATE TABLE IF NOT EXISTS power_of_attorney (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    poa_name VARCHAR(255),
    grantor_id VARCHAR(255) NOT NULL,
    grantor_name VARCHAR(255),
    representative_id VARCHAR(255) NOT NULL,
    representative_name VARCHAR(255),
    representative_type VARCHAR(100) NOT NULL,
    scope_type VARCHAR(100),
    actions TEXT[] DEFAULT '{}'::TEXT[],
    geographic_regions TEXT[] DEFAULT '{}'::TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by VARCHAR(255),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    conditions JSONB DEFAULT '{}'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'pending', 'revoked', 'expired'))
);

-- Indexes for power_of_attorney
CREATE INDEX IF NOT EXISTS idx_poa_tenant_id ON power_of_attorney(tenant_id);
CREATE INDEX IF NOT EXISTS idx_poa_grantor_id ON power_of_attorney(grantor_id);
CREATE INDEX IF NOT EXISTS idx_poa_representative_id ON power_of_attorney(representative_id);
CREATE INDEX IF NOT EXISTS idx_poa_status ON power_of_attorney(status);
CREATE INDEX IF NOT EXISTS idx_poa_valid_from ON power_of_attorney(valid_from);
CREATE INDEX IF NOT EXISTS idx_poa_valid_until ON power_of_attorney(valid_until);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_poa_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER poa_updated_at_trigger
    BEFORE UPDATE ON power_of_attorney
    FOR EACH ROW
    EXECUTE FUNCTION update_poa_updated_at();

-- Recreate delegation_chains table
CREATE TABLE IF NOT EXISTS delegation_chains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    delegator_id VARCHAR(255) NOT NULL,
    delegatee_id VARCHAR(255) NOT NULL,
    depth INTEGER NOT NULL DEFAULT 1,
    chain_path VARCHAR(1000),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_depth CHECK (depth > 0)
);

CREATE INDEX IF NOT EXISTS idx_delegation_chains_tenant_id ON delegation_chains(tenant_id);
CREATE INDEX IF NOT EXISTS idx_delegation_chains_poa_id ON delegation_chains(poa_id);
CREATE INDEX IF NOT EXISTS idx_delegation_chains_delegator_id ON delegation_chains(delegator_id);
CREATE INDEX IF NOT EXISTS idx_delegation_chains_delegatee_id ON delegation_chains(delegatee_id);
