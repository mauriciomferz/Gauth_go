-- Migration: Add authorization_details to extended_tokens
-- RFC 9396 Rich Authorization Requests support
-- Version: 017
-- Created: 2025-12-10

ALTER TABLE extended_tokens
ADD COLUMN IF NOT EXISTS authorization_details JSONB;

COMMENT ON COLUMN extended_tokens.authorization_details IS 'RFC 9396 Authorization Details (Rich Authorization Requests)';

-- Index for searching by authorization type
CREATE INDEX IF NOT EXISTS idx_extended_tokens_auth_details_type ON extended_tokens USING gin ((authorization_details));
