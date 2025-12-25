-- Add authorization_details column to extended_tokens if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='extended_tokens' AND column_name='authorization_details') THEN
        ALTER TABLE extended_tokens ADD COLUMN authorization_details JSONB;
    END IF;
END $$;

-- Create index for querying by authorization detail type
CREATE INDEX IF NOT EXISTS idx_extended_tokens_auth_details_type ON extended_tokens ((authorization_details->>'type'));
