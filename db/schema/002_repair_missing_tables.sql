-- Repair migration to create missing tables
-- Extracted from 001_initial_schema.sql

-- Audit events table
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    
    -- Actor
    user_id VARCHAR(255) NOT NULL,
    user_name VARCHAR(255),
    user_role VARCHAR(100),
    
    -- Action
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),
    
    -- Outcome
    status VARCHAR(50) NOT NULL,
    status_code INTEGER,
    error_message TEXT,
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    correlation_id VARCHAR(255),
    
    -- Changes
    before_state JSONB,
    after_state JSONB,
    changes JSONB,
    
    -- Compliance
    compliance_framework VARCHAR(100),
    risk_level VARCHAR(50),
    requires_review BOOLEAN DEFAULT false,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by VARCHAR(255),
    
    -- Tamper protection
    hash VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64),
    
    CONSTRAINT valid_severity CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    CONSTRAINT valid_status CHECK (status IN ('success', 'failure', 'partial', 'error'))
);

CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_id ON audit_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events(action);
CREATE INDEX IF NOT EXISTS idx_audit_events_category ON audit_events(category);
CREATE INDEX IF NOT EXISTS idx_audit_events_severity ON audit_events(severity);
CREATE INDEX IF NOT EXISTS idx_audit_events_requires_review ON audit_events(requires_review) WHERE requires_review = true;
CREATE INDEX IF NOT EXISTS idx_audit_events_correlation_id ON audit_events(correlation_id) WHERE correlation_id IS NOT NULL;

-- Compliance reports table
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    report_name VARCHAR(255) NOT NULL,
    framework VARCHAR(100) NOT NULL,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Report data
    total_events INTEGER NOT NULL,
    compliant_events INTEGER NOT NULL,
    non_compliant_events INTEGER NOT NULL,
    critical_violations INTEGER NOT NULL,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'generated',
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    generated_by VARCHAR(255),
    
    -- Details
    summary TEXT,
    recommendations TEXT[],
    violations JSONB,
    report_data JSONB,
    
    CONSTRAINT valid_framework CHECK (framework IN ('SOC2', 'HIPAA', 'GDPR', 'PCI-DSS', 'ISO27001')),
    CONSTRAINT valid_status CHECK (status IN ('generated', 'reviewed', 'approved', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_compliance_reports_tenant_id ON compliance_reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_framework ON compliance_reports(framework);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_period ON compliance_reports(period_start, period_end);

-- Event correlation patterns table
CREATE TABLE IF NOT EXISTS event_correlation_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    pattern_name VARCHAR(255) NOT NULL,
    pattern_type VARCHAR(50) NOT NULL,
    description TEXT,
    
    -- Pattern definition
    event_sequence TEXT[],
    time_window_minutes INTEGER NOT NULL,
    min_occurrences INTEGER DEFAULT 1,
    conditions JSONB,
    
    -- Actions
    severity VARCHAR(50) NOT NULL,
    alert_enabled BOOLEAN DEFAULT true,
    alert_recipients TEXT[],
    
    -- Statistics
    matches_count INTEGER DEFAULT 0,
    last_match_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_pattern_type CHECK (pattern_type IN ('sequential', 'concurrent', 'frequency', 'anomaly')),
    CONSTRAINT valid_severity CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    UNIQUE(tenant_id, pattern_name)
);

CREATE INDEX IF NOT EXISTS idx_correlation_patterns_tenant_id ON event_correlation_patterns(tenant_id);

-- SIEM integrations table
CREATE TABLE IF NOT EXISTS siem_integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    integration_name VARCHAR(255) NOT NULL,
    siem_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    -- Configuration
    endpoint_url TEXT NOT NULL,
    auth_type VARCHAR(50),
    api_key TEXT,
    format VARCHAR(50) NOT NULL,
    batch_size INTEGER DEFAULT 100,
    flush_interval_seconds INTEGER DEFAULT 60,
    
    -- Filtering
    event_types TEXT[],
    min_severity VARCHAR(50),
    
    -- Statistics
    events_sent BIGINT DEFAULT 0,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    last_error_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_siem_type CHECK (siem_type IN ('splunk', 'elastic', 'qradar', 'sentinel', 'sumo_logic', 'datadog')),
    CONSTRAINT valid_status CHECK (status IN ('active', 'paused', 'error', 'disabled')),
    CONSTRAINT valid_format CHECK (format IN ('json', 'cef', 'leef', 'syslog')),
    UNIQUE(tenant_id, integration_name)
);

CREATE INDEX IF NOT EXISTS idx_siem_integrations_tenant_id ON siem_integrations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_siem_integrations_status ON siem_integrations(status);

-- Merkle tree nodes table
CREATE TABLE IF NOT EXISTS merkle_tree_nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    tree_version INTEGER NOT NULL DEFAULT 1,
    node_hash VARCHAR(64) NOT NULL,
    level INTEGER NOT NULL,
    position INTEGER NOT NULL,
    
    -- Tree structure
    is_leaf BOOLEAN NOT NULL DEFAULT false,
    left_child_hash VARCHAR(64),
    right_child_hash VARCHAR(64),
    parent_hash VARCHAR(64),
    
    -- Leaf data (for leaf nodes only)
    token_id VARCHAR(255),
    leaf_data JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, tree_version, level, position)
);

CREATE INDEX IF NOT EXISTS idx_merkle_nodes_tenant_id ON merkle_tree_nodes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_merkle_nodes_tree_version ON merkle_tree_nodes(tree_version);
CREATE INDEX IF NOT EXISTS idx_merkle_nodes_hash ON merkle_tree_nodes(node_hash);
CREATE INDEX IF NOT EXISTS idx_merkle_nodes_level ON merkle_tree_nodes(level);
CREATE INDEX IF NOT EXISTS idx_merkle_nodes_token_id ON merkle_tree_nodes(token_id) WHERE token_id IS NOT NULL;

-- Merkle proofs table
CREATE TABLE IF NOT EXISTS merkle_proofs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    proof_id VARCHAR(255) UNIQUE NOT NULL,
    token_id VARCHAR(255) NOT NULL,
    tree_version INTEGER NOT NULL,
    
    -- Proof data
    leaf_hash VARCHAR(64) NOT NULL,
    root_hash VARCHAR(64) NOT NULL,
    proof_path JSONB NOT NULL,
    
    -- Verification
    verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(tenant_id, token_id, tree_version)
);

CREATE INDEX IF NOT EXISTS idx_merkle_proofs_tenant_id ON merkle_proofs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_merkle_proofs_token_id ON merkle_proofs(token_id);
CREATE INDEX IF NOT EXISTS idx_merkle_proofs_verified ON merkle_proofs(verified);

-- Revocations table
CREATE TABLE IF NOT EXISTS revocations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    token_id VARCHAR(255) NOT NULL,
    revocation_reason VARCHAR(255),
    
    -- Merkle tree integration
    leaf_hash VARCHAR(64) NOT NULL,
    merkle_root VARCHAR(64) NOT NULL,
    block_height INTEGER NOT NULL,
    tree_version INTEGER NOT NULL,
    
    -- Status
    verified BOOLEAN DEFAULT false,
    
    -- Lifecycle
    revoked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_by VARCHAR(255) NOT NULL,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    UNIQUE(tenant_id, token_id, tree_version)
);

CREATE INDEX IF NOT EXISTS idx_revocations_tenant_id ON revocations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_revocations_token_id ON revocations(token_id);
CREATE INDEX IF NOT EXISTS idx_revocations_verified ON revocations(verified);
CREATE INDEX IF NOT EXISTS idx_revocations_revoked_at ON revocations(revoked_at DESC);

-- Append-only log table
CREATE TABLE IF NOT EXISTS append_only_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL REFERENCES subscribers(tenant_id) ON DELETE CASCADE,
    log_index BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    operation VARCHAR(50) NOT NULL,
    
    -- Entry data
    entry_data JSONB NOT NULL,
    
    -- Hash chain
    entry_hash VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,
    
    -- Verification
    verified BOOLEAN DEFAULT true,
    
    CONSTRAINT valid_operation CHECK (operation IN ('append', 'verify', 'audit')),
    UNIQUE(tenant_id, log_index)
);

CREATE INDEX IF NOT EXISTS idx_append_log_tenant_id ON append_only_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_append_log_index ON append_only_log(log_index);
CREATE INDEX IF NOT EXISTS idx_append_log_timestamp ON append_only_log(timestamp DESC);

-- Enable RLS
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_correlation_patterns ENABLE ROW LEVEL SECURITY;
ALTER TABLE siem_integrations ENABLE ROW LEVEL SECURITY;
ALTER TABLE merkle_tree_nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE merkle_proofs ENABLE ROW LEVEL SECURITY;
ALTER TABLE revocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE append_only_log ENABLE ROW LEVEL SECURITY;

-- RLS Policies
CREATE POLICY tenant_isolation_policy ON audit_events FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON compliance_reports FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON event_correlation_patterns FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON siem_integrations FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON merkle_tree_nodes FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON merkle_proofs FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON revocations FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));
CREATE POLICY tenant_isolation_policy ON append_only_log FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', true));

-- Views
CREATE OR REPLACE VIEW recent_audit_events AS
SELECT *
FROM audit_events
WHERE timestamp > CURRENT_TIMESTAMP - INTERVAL '30 days'
ORDER BY timestamp DESC;
