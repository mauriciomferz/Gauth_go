-- Create Events Subsystem Tables

-- 1. Create event_types table
CREATE TABLE IF NOT EXISTS event_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    severity VARCHAR(50) NOT NULL,
    schema JSONB,
    retention_days INTEGER DEFAULT 30,
    is_system_event BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, event_type)
);

CREATE INDEX IF NOT EXISTS idx_event_types_tenant ON event_types(tenant_id);

-- 2. Create event_handlers table
CREATE TABLE IF NOT EXISTS event_handlers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    handler_name VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL, -- Logical link to event_types via name, or FK? Code uses name.
    handler_type VARCHAR(50) NOT NULL DEFAULT 'webhook',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    endpoint VARCHAR(2048),
    method VARCHAR(10) DEFAULT 'POST',
    headers JSONB,
    timeout_seconds INTEGER DEFAULT 10,
    retry_count INTEGER DEFAULT 3,
    enabled BOOLEAN DEFAULT true,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_event_handlers_tenant ON event_handlers(tenant_id);

-- 3. Update events table with missing columns
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45),
    ADD COLUMN IF NOT EXISTS user_agent TEXT,
    ADD COLUMN IF NOT EXISTS request_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS session_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(255);

-- 4. Enable RLS
ALTER TABLE event_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_handlers ENABLE ROW LEVEL SECURITY;

-- Policy for event_types
DROP POLICY IF EXISTS event_types_isolation_policy ON event_types;
CREATE POLICY event_types_isolation_policy ON event_types
    USING (tenant_id = current_setting('app.current_tenant', true)::text);

-- Policy for event_handlers
DROP POLICY IF EXISTS event_handlers_isolation_policy ON event_handlers;
CREATE POLICY event_handlers_isolation_policy ON event_handlers
    USING (tenant_id = current_setting('app.current_tenant', true)::text);
