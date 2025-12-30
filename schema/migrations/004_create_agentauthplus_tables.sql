-- Migration to create AgentAuth+ (Advanced Features) tables

-- 1. Successor Activations
CREATE TABLE successor_activations (
    id TEXT PRIMARY KEY,
    poa_id TEXT NOT NULL,
    primary_agent_id TEXT NOT NULL,
    successor_agent_id TEXT NOT NULL,
    activation_reason TEXT,
    activated_at TIMESTAMPTZ NOT NULL,
    activated_by TEXT NOT NULL,
    status TEXT NOT NULL,
    deactivated_at TIMESTAMPTZ,
    deactivated_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_successor_activations_poa_id ON successor_activations(poa_id);
CREATE INDEX idx_successor_activations_status ON successor_activations(status);

-- 2. AI Delegations
CREATE TABLE ai_delegations (
    id TEXT PRIMARY KEY,
    source_poa_id TEXT NOT NULL,
    source_agent_id TEXT NOT NULL,
    target_agent_id TEXT NOT NULL,
    delegated_scope JSONB, -- Array of permissions/scopes
    delegation_depth INT NOT NULL,
    max_allowed_depth INT NOT NULL,
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    status TEXT NOT NULL, -- active, revoked, expired
    delegation_policy JSONB, -- Embedded policy snapshot
    revoked_at TIMESTAMPTZ,
    revoked_by TEXT,
    revocation_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_delegations_source_poa_id ON ai_delegations(source_poa_id);
CREATE INDEX idx_ai_delegations_target_agent_id ON ai_delegations(target_agent_id);
CREATE INDEX idx_ai_delegations_status ON ai_delegations(status);

-- 3. Dual Control Approvals
CREATE TABLE dual_control_approvals (
    id TEXT PRIMARY KEY,
    poa_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_description TEXT,
    requested_by TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL,
    required_approvers INT NOT NULL DEFAULT 1,
    approval_threshold TEXT NOT NULL DEFAULT 'all', -- all, majority, quorum, weighted
    status TEXT NOT NULL, -- pending, approved, rejected, expired
    approved_by JSONB, -- Array of approval records
    rejected_by JSONB, -- Array of rejection records
    expires_at TIMESTAMPTZ,
    metadata JSONB, -- Context data for decision
    decision_finalized_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dual_control_poa_id ON dual_control_approvals(poa_id);
CREATE INDEX idx_dual_control_status ON dual_control_approvals(status);

-- 4. Fiduciary Duty Violations
CREATE TABLE fiduciary_duty_violations (
    id TEXT PRIMARY KEY,
    poa_id TEXT,
    agent_id TEXT,
    duty_type TEXT NOT NULL, -- loyalty, care, obedience, disclosure
    violation_description TEXT,
    severity TEXT NOT NULL, -- minor, moderate, major, critical
    detected_at TIMESTAMPTZ NOT NULL,
    detected_by TEXT NOT NULL,
    resolution_status TEXT NOT NULL, -- open, investigating, resolved, dismissed
    consequences JSONB, -- Remedial actions taken
    evidence JSONB, -- Links to logs or proofs
    reviewed_by TEXT,
    reviewed_at TIMESTAMPTZ,
    resolution_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fiduciary_poa_id ON fiduciary_duty_violations(poa_id);
CREATE INDEX idx_fiduciary_agent_id ON fiduciary_duty_violations(agent_id);
CREATE INDEX idx_fiduciary_resolution_status ON fiduciary_duty_violations(resolution_status);

-- 5. AI Capability Assessments
CREATE TABLE ai_capability_assessments (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    assessed_by TEXT NOT NULL,
    assessment_date TIMESTAMPTZ NOT NULL,
    overall_level TEXT NOT NULL, -- L0 to L5
    domain_scores JSONB, -- Key-value map of domain->score
    risk_profile JSONB, -- Risk assessment data
    limitations JSONB, -- Known constraints
    certifications JSONB, -- Array of verified certs
    valid_until TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_capability_agent_id ON ai_capability_assessments(agent_id);
CREATE INDEX idx_capability_overall_level ON ai_capability_assessments(overall_level);
