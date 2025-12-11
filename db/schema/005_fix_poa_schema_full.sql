-- Fix PoA Schema (Phase 21)
-- Recreate power_of_attorney table to match backend code expectations (replacing legacy schema)

DROP TABLE IF EXISTS power_of_attorney CASCADE;

-- Create Power of Attorney table with correct schema (from db/schema/003_repair_more_tables.sql poa_records definition)
CREATE TABLE power_of_attorney (
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
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'pending', 'revoked', 'expired')),
    CONSTRAINT valid_scope_type CHECK (scope_type IN ('full', 'limited', 'financial', 'healthcare', 'legal', 'administrative'))
);

-- Indexes
CREATE INDEX idx_poa_tenant_id ON power_of_attorney(tenant_id);
CREATE INDEX idx_poa_grantor_id ON power_of_attorney(grantor_id);
CREATE INDEX idx_poa_representative_id ON power_of_attorney(representative_id);
CREATE INDEX idx_poa_status ON power_of_attorney(status);
CREATE INDEX idx_poa_valid_from ON power_of_attorney(valid_from);
CREATE INDEX idx_poa_valid_until ON power_of_attorney(valid_until);

-- RLS
ALTER TABLE power_of_attorney ENABLE ROW LEVEL SECURITY;
CREATE POLICY allow_all_poa ON power_of_attorney FOR ALL USING (true);
