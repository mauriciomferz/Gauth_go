-- Add missing columns to events table to match repository expectations

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45),
    ADD COLUMN IF NOT EXISTS user_agent TEXT,
    ADD COLUMN IF NOT EXISTS request_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS session_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_events_request_id ON events(request_id);
CREATE INDEX IF NOT EXISTS idx_events_session_id ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_events_correlation_id ON events(correlation_id);

-- Update event_types table to match repository expectations
ALTER TABLE event_types
    DROP COLUMN IF EXISTS type_name,
    DROP COLUMN IF EXISTS enabled,
    ADD COLUMN IF NOT EXISTS event_type VARCHAR(255),
    ADD COLUMN IF NOT EXISTS category VARCHAR(100),
    ADD COLUMN IF NOT EXISTS severity VARCHAR(50) DEFAULT 'info',
    ADD COLUMN IF NOT EXISTS retention_days INTEGER DEFAULT 90,
    ADD COLUMN IF NOT EXISTS is_system_event BOOLEAN DEFAULT false;

-- Add unique constraint
ALTER TABLE event_types 
    DROP CONSTRAINT IF EXISTS event_types_tenant_id_type_name_key,
    ADD CONSTRAINT event_types_tenant_id_event_type_key UNIQUE(tenant_id, event_type);

-- Update indexes
DROP INDEX IF EXISTS idx_event_types_type_name;
CREATE INDEX IF NOT EXISTS idx_event_types_event_type ON event_types(event_type);
CREATE INDEX IF NOT EXISTS idx_event_types_category ON event_types(category);
CREATE INDEX IF NOT EXISTS idx_event_types_severity ON event_types(severity);
