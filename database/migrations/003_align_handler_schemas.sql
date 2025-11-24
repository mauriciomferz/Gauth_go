-- Migration to align all handler table schemas with repository code expectations
-- This updates circuit_breakers, rate_limiters, retry_policies, bulkheads, and authorization_policies

-- Fix circuit_breakers table
ALTER TABLE circuit_breakers 
    RENAME COLUMN name TO breaker_name;

ALTER TABLE circuit_breakers 
    ADD COLUMN IF NOT EXISTS half_open_max_requests INTEGER DEFAULT 5,
    ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS consecutive_successes INTEGER DEFAULT 0;

-- Fix rate_limiters table (check if columns match)
-- Schema has: name, endpoint, algorithm - check what repository expects
-- (Will verify and add if needed in next step)

-- Fix retry_policies table (check if columns match)
-- (Will verify and add if needed in next step)

-- Fix bulkheads table (check if columns match)  
-- (Will verify and add if needed in next step)

-- Add event_types table (needed by events repository)
CREATE TABLE IF NOT EXISTS event_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    type_name VARCHAR(255) NOT NULL,
    description TEXT,
    schema JSONB,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, type_name)
);

CREATE INDEX IF NOT EXISTS idx_event_types_tenant_id ON event_types(tenant_id);
CREATE INDEX IF NOT EXISTS idx_event_types_type_name ON event_types(type_name);

-- Add event_handlers table (needed by events repository)
CREATE TABLE IF NOT EXISTS event_handlers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    handler_name VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) DEFAULT 'POST',
    headers JSONB DEFAULT '{}'::jsonb,
    timeout_seconds INTEGER DEFAULT 30,
    retry_count INTEGER DEFAULT 3,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, handler_name)
);

CREATE INDEX IF NOT EXISTS idx_event_handlers_tenant_id ON event_handlers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_event_handlers_event_type ON event_handlers(event_type);
CREATE INDEX IF NOT EXISTS idx_event_handlers_enabled ON event_handlers(enabled);

-- Trigger for event_types updated_at
CREATE OR REPLACE FUNCTION update_event_types_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER event_types_updated_at_trigger
    BEFORE UPDATE ON event_types
    FOR EACH ROW
    EXECUTE FUNCTION update_event_types_updated_at();

-- Trigger for event_handlers updated_at
CREATE OR REPLACE FUNCTION update_event_handlers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER event_handlers_updated_at_trigger
    BEFORE UPDATE ON event_handlers
    FOR EACH ROW
    EXECUTE FUNCTION update_event_handlers_updated_at();
