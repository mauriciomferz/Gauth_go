-- Migration: Add webhook support for event notifications
-- Description: Creates tables for webhook registration, delivery tracking, and event management

-- Create webhooks table for webhook registration
CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    secret VARCHAR(255) NOT NULL, -- HMAC secret for signature verification
    user_id VARCHAR(255) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    
    -- Event filtering
    events TEXT[] NOT NULL, -- Array of event types to subscribe to
    
    -- Delivery settings
    retry_count INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 30,
    
    -- Metadata
    description TEXT,
    headers JSONB DEFAULT '{}', -- Custom headers to include in requests
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_triggered_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_url CHECK (url ~* '^https?://'),
    CONSTRAINT valid_retry_count CHECK (retry_count >= 0 AND retry_count <= 10),
    CONSTRAINT valid_timeout CHECK (timeout_seconds >= 1 AND timeout_seconds <= 300)
);

-- Create index on user_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_enabled ON webhooks(enabled) WHERE enabled = true;

-- Create webhook_deliveries table for tracking delivery attempts
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    
    -- Request details
    url TEXT NOT NULL,
    http_method VARCHAR(10) DEFAULT 'POST',
    headers JSONB,
    payload JSONB NOT NULL,
    
    -- Response details
    status_code INTEGER,
    response_body TEXT,
    response_headers JSONB,
    
    -- Delivery status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, success, failed, retrying
    attempt_number INTEGER DEFAULT 1,
    max_attempts INTEGER DEFAULT 3,
    
    -- Error tracking
    error_message TEXT,
    
    -- Timing
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP,
    completed_at TIMESTAMP,
    next_retry_at TIMESTAMP,
    duration_ms INTEGER,
    
    -- Constraints
    CONSTRAINT valid_status CHECK (status IN ('pending', 'success', 'failed', 'retrying')),
    CONSTRAINT valid_attempt CHECK (attempt_number >= 1 AND attempt_number <= max_attempts),
    CONSTRAINT valid_http_method CHECK (http_method IN ('POST', 'PUT', 'PATCH'))
);

-- Create indexes for webhook_deliveries
CREATE INDEX IF NOT EXISTS idx_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_event_id ON webhook_deliveries(event_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_status ON webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_deliveries_next_retry ON webhook_deliveries(next_retry_at) WHERE status = 'retrying';
CREATE INDEX IF NOT EXISTS idx_deliveries_created_at ON webhook_deliveries(created_at DESC);

-- Create webhook_events table for event definitions
CREATE TABLE IF NOT EXISTS webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    payload_schema JSONB, -- JSON Schema for payload validation
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default event types
INSERT INTO webhook_events (event_type, description) VALUES
    ('poa.created', 'Power of Attorney document created'),
    ('poa.updated', 'Power of Attorney document updated'),
    ('poa.revoked', 'Power of Attorney document revoked'),
    ('poa.verified', 'Power of Attorney verification performed'),
    ('poa.expired', 'Power of Attorney document expired'),
    ('successor.activated', 'Successor designation activated'),
    ('delegation.created', 'Delegation created'),
    ('delegation.revoked', 'Delegation revoked'),
    ('dual_control.approval_required', 'Dual control approval required'),
    ('dual_control.approved', 'Dual control request approved'),
    ('dual_control.rejected', 'Dual control request rejected'),
    ('blockchain.sync_completed', 'Blockchain synchronization completed'),
    ('blockchain.sync_failed', 'Blockchain synchronization failed')
ON CONFLICT (event_type) DO NOTHING;

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_webhook_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for webhooks table
CREATE TRIGGER update_webhooks_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW
    EXECUTE FUNCTION update_webhook_timestamp();

-- Create view for webhook statistics
CREATE OR REPLACE VIEW webhook_stats AS
SELECT 
    w.id as webhook_id,
    w.name,
    w.url,
    w.enabled,
    COUNT(wd.id) as total_deliveries,
    SUM(CASE WHEN wd.status = 'success' THEN 1 ELSE 0 END) as successful_deliveries,
    SUM(CASE WHEN wd.status = 'failed' THEN 1 ELSE 0 END) as failed_deliveries,
    SUM(CASE WHEN wd.status = 'retrying' THEN 1 ELSE 0 END) as retrying_deliveries,
    AVG(CASE WHEN wd.duration_ms IS NOT NULL THEN wd.duration_ms ELSE NULL END) as avg_duration_ms,
    MAX(wd.created_at) as last_delivery_at,
    ROUND(
        100.0 * SUM(CASE WHEN wd.status = 'success' THEN 1 ELSE 0 END) / 
        NULLIF(COUNT(wd.id), 0), 2
    ) as success_rate_percent
FROM webhooks w
LEFT JOIN webhook_deliveries wd ON w.id = wd.webhook_id
GROUP BY w.id, w.name, w.url, w.enabled;

-- Add comments for documentation
COMMENT ON TABLE webhooks IS 'Webhook registration and configuration';
COMMENT ON TABLE webhook_deliveries IS 'Webhook delivery attempts and results';
COMMENT ON TABLE webhook_events IS 'Available webhook event types';
COMMENT ON VIEW webhook_stats IS 'Webhook delivery statistics and success rates';

-- Grant permissions (adjust based on your user setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON webhooks TO gauth_app;
-- GRANT SELECT, INSERT, UPDATE ON webhook_deliveries TO gauth_app;
-- GRANT SELECT ON webhook_events TO gauth_app;
-- GRANT SELECT ON webhook_stats TO gauth_app;
