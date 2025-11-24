-- Add missing columns to authorization_policies to match repository expectations

ALTER TABLE authorization_policies 
    ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1,
    ADD COLUMN IF NOT EXISTS conditions JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(255),
    ADD COLUMN IF NOT EXISTS valid_from TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS valid_until TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_authorization_policies_version ON authorization_policies(version);
CREATE INDEX IF NOT EXISTS idx_authorization_policies_valid_from ON authorization_policies(valid_from);
CREATE INDEX IF NOT EXISTS idx_authorization_policies_valid_until ON authorization_policies(valid_until);
