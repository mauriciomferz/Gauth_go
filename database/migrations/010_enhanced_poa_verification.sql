-- Migration 010: Enhanced Power of Attorney Model with Verification Support
-- This migration adds comprehensive PoA fields for issuer/grantee, attestations,
-- version history, structured restrictions, and verification support.

-- Add new columns to power_of_attorney table
ALTER TABLE power_of_attorney
ADD COLUMN IF NOT EXISTS issuer_id TEXT,
ADD COLUMN IF NOT EXISTS grantee_id TEXT,
ADD COLUMN IF NOT EXISTS structured_scope JSONB,
ADD COLUMN IF NOT EXISTS restrictions JSONB,
ADD COLUMN IF NOT EXISTS attestations JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS version_number INTEGER DEFAULT 1,
ADD COLUMN IF NOT EXISTS version_history JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired', 'pending')),
ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS revoked_by TEXT,
ADD COLUMN IF NOT EXISTS revocation_reason TEXT;

-- Migrate existing data: copy grantor_id to issuer_id and representative_id to grantee_id
UPDATE power_of_attorney
SET issuer_id = grantor_id,
    grantee_id = representative_id
WHERE issuer_id IS NULL OR grantee_id IS NULL;

-- Create indexes for new fields
CREATE INDEX IF NOT EXISTS idx_poa_issuer_id ON power_of_attorney(issuer_id);
CREATE INDEX IF NOT EXISTS idx_poa_grantee_id ON power_of_attorney(grantee_id);
CREATE INDEX IF NOT EXISTS idx_poa_status ON power_of_attorney(status);
CREATE INDEX IF NOT EXISTS idx_poa_revoked_at ON power_of_attorney(revoked_at) WHERE revoked_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_poa_version_number ON power_of_attorney(version_number);

-- Create GIN index for JSONB fields to support queries
CREATE INDEX IF NOT EXISTS idx_poa_structured_scope_gin ON power_of_attorney USING gin(structured_scope);
CREATE INDEX IF NOT EXISTS idx_poa_restrictions_gin ON power_of_attorney USING gin(restrictions);
CREATE INDEX IF NOT EXISTS idx_poa_attestations_gin ON power_of_attorney USING gin(attestations);

-- Create verification_reports table for storing comprehensive verification reports
CREATE TABLE IF NOT EXISTS verification_reports (
    id TEXT PRIMARY KEY,
    poa_id TEXT NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    report_id TEXT NOT NULL UNIQUE,
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    requested_action JSONB NOT NULL,
    overall_valid BOOLEAN NOT NULL,
    poa_verification JSONB,
    scope_verification JSONB,
    principal_status JSONB,
    revocation_status JSONB,
    chain_of_authority JSONB,
    fiduciary_compliance JSONB,
    capability_check JSONB,
    warnings TEXT[],
    recommendations TEXT[],
    validity_period TEXT,
    next_review_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_verification_reports_poa_id ON verification_reports(poa_id);
CREATE INDEX IF NOT EXISTS idx_verification_reports_report_id ON verification_reports(report_id);
CREATE INDEX IF NOT EXISTS idx_verification_reports_generated_at ON verification_reports(generated_at);
CREATE INDEX IF NOT EXISTS idx_verification_reports_overall_valid ON verification_reports(overall_valid);

-- Create attestations table for tracking individual attestations
CREATE TABLE IF NOT EXISTS poa_attestations (
    id SERIAL PRIMARY KEY,
    poa_id TEXT NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    attestation_type TEXT NOT NULL CHECK (attestation_type IN ('notary', 'witness', 'electronic_signature', 'video_verification')),
    attestor_id TEXT NOT NULL,
    attestor_name TEXT NOT NULL,
    attestor_role TEXT,
    method TEXT NOT NULL CHECK (method IN ('in_person', 'video', 'electronic', 'document')),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    location TEXT,
    certificate_id TEXT,
    signature_hash TEXT,
    verified BOOLEAN DEFAULT FALSE,
    verification_date TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_attestations_poa_id ON poa_attestations(poa_id);
CREATE INDEX IF NOT EXISTS idx_attestations_type ON poa_attestations(attestation_type);
CREATE INDEX IF NOT EXISTS idx_attestations_attestor_id ON poa_attestations(attestor_id);
CREATE INDEX IF NOT EXISTS idx_attestations_verified ON poa_attestations(verified);

-- Create poa_versions table for detailed version history
CREATE TABLE IF NOT EXISTS poa_versions (
    id SERIAL PRIMARY KEY,
    poa_id TEXT NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by TEXT NOT NULL,
    change_type TEXT NOT NULL CHECK (change_type IN ('created', 'updated', 'revoked', 'renewed', 'amended')),
    changes TEXT[],
    previous_state JSONB,
    new_state JSONB,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(poa_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_poa_versions_poa_id ON poa_versions(poa_id);
CREATE INDEX IF NOT EXISTS idx_poa_versions_version_number ON poa_versions(poa_id, version_number);
CREATE INDEX IF NOT EXISTS idx_poa_versions_modified_at ON poa_versions(modified_at);

-- Create principal_status table for tracking principal legal capacity
CREATE TABLE IF NOT EXISTS principal_status (
    id SERIAL PRIMARY KEY,
    principal_id TEXT NOT NULL,
    legal_capacity TEXT NOT NULL CHECK (legal_capacity IN ('full', 'limited', 'incapacitated')),
    entity_type TEXT NOT NULL CHECK (entity_type IN ('individual', 'corporation', 'partnership', 'llc', 'trust', 'government', 'ai_agent')),
    status TEXT NOT NULL CHECK (status IN ('active', 'dissolved', 'suspended', 'inactive')),
    jurisdiction TEXT,
    jurisdiction_status TEXT,
    verified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    verification_method TEXT,
    verification_source TEXT,
    issues TEXT[],
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_principal_status_principal_id ON principal_status(principal_id);
CREATE INDEX IF NOT EXISTS idx_principal_status_entity_type ON principal_status(entity_type);
CREATE INDEX IF NOT EXISTS idx_principal_status_status ON principal_status(status);
CREATE INDEX IF NOT EXISTS idx_principal_status_expires_at ON principal_status(expires_at);

-- Create representative_positions table for tracking representative positions
CREATE TABLE IF NOT EXISTS representative_positions (
    id SERIAL PRIMARY KEY,
    representative_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    position TEXT NOT NULL,
    authorized_to_act BOOLEAN DEFAULT FALSE,
    signing_authority BOOLEAN DEFAULT FALSE,
    effective_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE,
    verified_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verification_source TEXT,
    issues TEXT[],
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_representative_positions_rep_id ON representative_positions(representative_id);
CREATE INDEX IF NOT EXISTS idx_representative_positions_org_id ON representative_positions(organization_id);
CREATE INDEX IF NOT EXISTS idx_representative_positions_rep_org ON representative_positions(representative_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_representative_positions_effective_date ON representative_positions(effective_date);

-- Create verification_cache table for caching verification results
CREATE TABLE IF NOT EXISTS verification_cache (
    id SERIAL PRIMARY KEY,
    cache_key TEXT NOT NULL UNIQUE,
    verification_type TEXT NOT NULL CHECK (verification_type IN ('poa', 'scope', 'principal', 'revocation', 'attestation', 'chain')),
    poa_id TEXT,
    result JSONB NOT NULL,
    cached_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    hit_count INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_verification_cache_key ON verification_cache(cache_key);
CREATE INDEX IF NOT EXISTS idx_verification_cache_type ON verification_cache(verification_type);
CREATE INDEX IF NOT EXISTS idx_verification_cache_poa_id ON verification_cache(poa_id);
CREATE INDEX IF NOT EXISTS idx_verification_cache_expires_at ON verification_cache(expires_at);

-- Create function to automatically update version history
CREATE OR REPLACE FUNCTION update_poa_version_history()
RETURNS TRIGGER AS $$
BEGIN
    -- Increment version number
    NEW.version_number := COALESCE(OLD.version_number, 0) + 1;
    NEW.updated_at := CURRENT_TIMESTAMP;
    
    -- Add to version history
    NEW.version_history := COALESCE(OLD.version_history, '[]'::jsonb) || 
        jsonb_build_object(
            'version_number', NEW.version_number,
            'modified_at', CURRENT_TIMESTAMP,
            'modified_by', COALESCE(current_setting('app.current_user', true), 'system'),
            'changes', ARRAY['Updated PoA fields']
        );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic version tracking
DROP TRIGGER IF EXISTS trigger_update_poa_version ON power_of_attorney;
CREATE TRIGGER trigger_update_poa_version
    BEFORE UPDATE ON power_of_attorney
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*)
    EXECUTE FUNCTION update_poa_version_history();

-- Create function to automatically update status to expired
CREATE OR REPLACE FUNCTION update_expired_poas()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE power_of_attorney
    SET status = 'expired'
    WHERE status = 'active'
    AND valid_until < CURRENT_TIMESTAMP
    AND status != 'revoked';
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to check geographic restrictions
CREATE OR REPLACE FUNCTION check_geographic_restriction(
    restrictions_json JSONB,
    country_code TEXT,
    region_code TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    geographic_restrictions JSONB;
    allowed_countries JSONB;
    excluded_regions JSONB;
BEGIN
    -- Extract geographic restrictions
    geographic_restrictions := restrictions_json -> 'geographic_restrictions';
    
    IF geographic_restrictions IS NULL THEN
        RETURN TRUE; -- No restrictions
    END IF;
    
    -- Check allowed countries
    allowed_countries := geographic_restrictions -> 'allowed_countries';
    IF allowed_countries IS NOT NULL AND jsonb_array_length(allowed_countries) > 0 THEN
        IF NOT (allowed_countries @> to_jsonb(country_code) THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    -- Check excluded regions
    excluded_regions := geographic_restrictions -> 'excluded_regions';
    IF excluded_regions IS NOT NULL THEN
        IF excluded_regions @> to_jsonb(country_code) THEN
            RETURN FALSE;
        END IF;
        IF region_code IS NOT NULL AND excluded_regions @> to_jsonb(region_code) THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Create function to check value limits
CREATE OR REPLACE FUNCTION check_value_limit(
    restrictions_json JSONB,
    transaction_value NUMERIC,
    currency_code TEXT DEFAULT 'USD'
)
RETURNS BOOLEAN AS $$
DECLARE
    value_limits JSONB;
    max_single_transaction NUMERIC;
BEGIN
    -- Extract value limits
    value_limits := restrictions_json -> 'value_limits';
    
    IF value_limits IS NULL THEN
        RETURN TRUE; -- No restrictions
    END IF;
    
    -- Check max single transaction
    max_single_transaction := (value_limits ->> 'max_single_transaction')::NUMERIC;
    
    IF max_single_transaction IS NOT NULL AND transaction_value > max_single_transaction THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Create view for active PoAs with full details
CREATE OR REPLACE VIEW active_poas_detailed AS
SELECT 
    p.*,
    CASE 
        WHEN p.status = 'revoked' THEN 'Revoked'
        WHEN p.valid_until < CURRENT_TIMESTAMP THEN 'Expired'
        WHEN p.valid_from > CURRENT_TIMESTAMP THEN 'Pending'
        ELSE 'Active'
    END AS computed_status,
    jsonb_array_length(COALESCE(p.attestations, '[]'::jsonb) AS attestation_count,
    jsonb_array_length(COALESCE(p.version_history, '[]'::jsonb) AS version_count
FROM power_of_attorney p;

-- Create view for PoAs requiring review
CREATE OR REPLACE VIEW poas_requiring_review AS
SELECT 
    p.*,
    CASE
        WHEN p.valid_until - CURRENT_TIMESTAMP < INTERVAL '30 days' THEN 'Expiring Soon'
        WHEN jsonb_array_length(COALESCE(p.attestations, '[]'::jsonb) = 0 THEN 'Missing Attestations'
        WHEN p.version_number > 5 THEN 'Multiple Amendments'
        ELSE 'Review Recommended'
    END AS review_reason
FROM power_of_attorney p
WHERE p.status = 'active'
AND (
    p.valid_until - CURRENT_TIMESTAMP < INTERVAL '30 days'
    OR jsonb_array_length(COALESCE(p.attestations, '[]'::jsonb) = 0
    OR p.version_number > 5
);

-- Add comments for documentation
COMMENT ON TABLE verification_reports IS 'Stores comprehensive verification reports for relying parties';
COMMENT ON TABLE poa_attestations IS 'Tracks legal attestations (notary, witnesses) for PoAs';
COMMENT ON TABLE poa_versions IS 'Maintains detailed version history for PoA changes';
COMMENT ON TABLE principal_status IS 'Tracks legal capacity and status of principals';
COMMENT ON TABLE representative_positions IS 'Tracks positions and authority of representatives';
COMMENT ON TABLE verification_cache IS 'Caches verification results for performance';

COMMENT ON COLUMN power_of_attorney.issuer_id IS 'Entity granting the power of attorney';
COMMENT ON COLUMN power_of_attorney.grantee_id IS 'Entity receiving the power of attorney (AI agent or representative)';
COMMENT ON COLUMN power_of_attorney.structured_scope IS 'Detailed scope with transactions, decisions, actions, and geographic constraints';
COMMENT ON COLUMN power_of_attorney.restrictions IS 'Value limits, geographic restrictions, temporal restrictions';
COMMENT ON COLUMN power_of_attorney.attestations IS 'Array of attestations (notary, witnesses, etc.)';
COMMENT ON COLUMN power_of_attorney.version_number IS 'Current version number of the PoA';
COMMENT ON COLUMN power_of_attorney.version_history IS 'Array of version history entries';
COMMENT ON COLUMN power_of_attorney.status IS 'Current status: active, revoked, expired, pending';

-- Grant permissions (adjust as needed for your security model)
GRANT SELECT ON active_poas_detailed TO PUBLIC;
GRANT SELECT ON poas_requiring_review TO PUBLIC;
