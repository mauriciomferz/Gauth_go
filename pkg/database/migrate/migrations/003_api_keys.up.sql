-- API Key Management Tables
-- Part of Phase 20: Operational Enhancements

-- Main API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id VARCHAR(64) UNIQUE NOT NULL,           -- Public key identifier (pk_...)
    key_hash VARCHAR(128) NOT NULL,                -- bcrypt hashed secret (sk_...)
    name VARCHAR(255) NOT NULL,
    description TEXT,
    user_id VARCHAR(255) NOT NULL,
    
    -- Rate limiting
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    rate_limit_per_day INTEGER DEFAULT 10000,
    
    -- Quotas
    quota_requests_total INTEGER,                  -- NULL = unlimited
    quota_requests_used INTEGER DEFAULT 0,
    
    -- Status
    enabled BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP,
    
    -- Security
    ip_whitelist TEXT[],                           -- Optional IP restrictions
    allowed_endpoints TEXT[],                      -- Optional endpoint restrictions
    
    -- Metadata
    metadata JSONB,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_id ON api_keys(key_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_enabled ON api_keys(enabled);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- API Key Usage Tracking
CREATE TABLE IF NOT EXISTS api_key_usage (
    id BIGSERIAL PRIMARY KEY,
    key_id VARCHAR(64) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER,
    request_ip VARCHAR(45),
    user_agent TEXT,
    error_message TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (key_id) REFERENCES api_keys(key_id) ON DELETE CASCADE
);

-- Indexes for usage queries
CREATE INDEX IF NOT EXISTS idx_api_key_usage_key_id ON api_key_usage(key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_timestamp ON api_key_usage(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_endpoint ON api_key_usage(endpoint);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_api_key_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER trigger_api_key_updated_at
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_api_key_updated_at();

-- View for API key statistics
CREATE OR REPLACE VIEW api_key_stats AS
SELECT 
    k.key_id,
    k.name,
    k.enabled,
    k.quota_requests_total,
    k.quota_requests_used,
    COUNT(u.id) FILTER (WHERE u.timestamp > NOW() - INTERVAL '24 hours') as requests_today,
    COUNT(u.id) FILTER (WHERE u.timestamp > NOW() - INTERVAL '7 days') as requests_this_week,
    COUNT(u.id) FILTER (WHERE u.timestamp > NOW() - INTERVAL '30 days') as requests_this_month,
    AVG(u.response_time_ms) FILTER (WHERE u.timestamp > NOW() - INTERVAL '24 hours') as avg_response_time_24h,
    MAX(u.timestamp) as last_used_at
FROM api_keys k
LEFT JOIN api_key_usage u ON k.key_id = u.key_id
GROUP BY k.key_id, k.name, k.enabled, k.quota_requests_total, k.quota_requests_used;

COMMENT ON TABLE api_keys IS 'API key management for programmatic access with rate limiting and quotas';
COMMENT ON TABLE api_key_usage IS 'Audit trail and analytics for API key usage';
COMMENT ON VIEW api_key_stats IS 'Aggregated statistics for API key usage analytics';
