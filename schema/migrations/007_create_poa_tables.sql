-- Migration: Create Power of Attorney Tables
-- Covers: Power of Attorney Records and Templates
-- Version: 007
-- Created: 2025-12-21

-- ============================================================================
-- POWER OF ATTORNEY TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS power_of_attorney (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    poa_name VARCHAR(255) NOT NULL,
    
    -- Parties
    grantor_id VARCHAR(255) NOT NULL,
    grantor_name VARCHAR(255) NOT NULL,
    representative_id VARCHAR(255) NOT NULL,
    representative_name VARCHAR(255) NOT NULL,
    representative_type VARCHAR(100) NOT NULL,
    
    -- Scope & Authority
    scope_type VARCHAR(100) NOT NULL,
    actions TEXT[],
    geographic_regions TEXT[],
    
    -- Status & Lifecycle
    status VARCHAR(50) NOT NULL, -- active, pending, revoked, expired
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Metadata & Extensibility
    conditions JSONB,
    metadata JSONB,
    
    -- Audit Fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by VARCHAR(255),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT
);

-- Indexes for PoA
CREATE INDEX idx_poa_tenant_id ON power_of_attorney(tenant_id);
CREATE INDEX idx_poa_grantor_id ON power_of_attorney(grantor_id);
CREATE INDEX idx_poa_representative_id ON power_of_attorney(representative_id);
CREATE INDEX idx_poa_status ON power_of_attorney(status);
CREATE INDEX idx_poa_validity ON power_of_attorney(valid_from, valid_until);
CREATE INDEX idx_poa_created_at ON power_of_attorney(created_at DESC);


-- ============================================================================
-- POA TEMPLATES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS poa_templates (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255), -- Nullable for system templates
    template_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    scope_type VARCHAR(100) NOT NULL,
    default_actions TEXT[],
    default_duration_days INTEGER,
    conditions_schema JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    is_system_template BOOLEAN DEFAULT FALSE
);

-- Indexes for PoA Templates
CREATE INDEX idx_poa_templates_tenant_id ON poa_templates(tenant_id);
CREATE INDEX idx_poa_templates_system ON poa_templates(is_system_template);
