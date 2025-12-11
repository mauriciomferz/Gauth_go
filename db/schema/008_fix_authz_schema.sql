-- Fix Authorization Policies Schema
-- Adds missing columns required by pkg/authz/repository.go

ALTER TABLE authorization_policies
    ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1,
    ADD COLUMN IF NOT EXISTS conditions JSONB,
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(255),
    ADD COLUMN IF NOT EXISTS valid_from TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS valid_until TIMESTAMP WITH TIME ZONE;

-- Add indexes for valid_from/until if useful for querying active policies
CREATE INDEX IF NOT EXISTS idx_authz_policies_validity ON authorization_policies(valid_from, valid_until);
