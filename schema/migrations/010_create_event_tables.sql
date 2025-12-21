-- Migration: Create Event System tables
-- Covers: Event Types, Events, and Event Handlers
-- Version: 010
-- Created: 2025-12-21

-- ============================================================================
-- EVENT TYPES
-- ============================================================================

CREATE TABLE IF NOT EXISTS event_types (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(50) NOT NULL,
    schema JSONB,
    retention_days INTEGER DEFAULT 30,
    is_system_event BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, event_type)
);

CREATE INDEX idx_event_types_category ON event_types(tenant_id, category);

-- ============================================================================
-- EVENTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS events (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source VARCHAR(255),
    user_id VARCHAR(255),
    resource VARCHAR(255),
    action VARCHAR(255),
    status VARCHAR(50),
    message TEXT,
    payload JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    correlation_id VARCHAR(255)
);

CREATE INDEX idx_events_tenant_time ON events(tenant_id, timestamp DESC);
CREATE INDEX idx_events_type ON events(tenant_id, event_type);
CREATE INDEX idx_events_category ON events(tenant_id, category);
CREATE INDEX idx_events_user ON events(tenant_id, user_id);

-- ============================================================================
-- EVENT HANDLERS
-- ============================================================================

CREATE TABLE IF NOT EXISTS event_handlers (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    handler_name VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    endpoint VARCHAR(2048),
    method VARCHAR(10) DEFAULT 'POST',
    headers JSONB,
    timeout_seconds INTEGER DEFAULT 10,
    retry_count INTEGER DEFAULT 3,
    enabled BOOLEAN DEFAULT TRUE,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_handlers_tenant ON event_handlers(tenant_id);

-- ============================================================================
-- SEED DATA
-- ============================================================================

INSERT INTO event_types (tenant_id, event_type, category, description, severity, is_system_event)
VALUES 
('test-tenant-1', 'auth.login.success', 'authentication', 'User logged in successfully', 'info', true),
('test-tenant-1', 'auth.login.failed', 'authentication', 'User login failed', 'warning', true),
('test-tenant-1', 'auth.logout', 'authentication', 'User logged out', 'info', true),
('test-tenant-1', 'user.created', 'identity', 'New user account created', 'info', true),
('test-tenant-1', 'user.updated', 'identity', 'User account details updated', 'info', true),
('test-tenant-1', 'system.config.change', 'system', 'System configuration changed', 'warning', true)
ON CONFLICT (tenant_id, event_type) DO NOTHING;
