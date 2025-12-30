-- Create AgentAuth+ Subsystem Tables

-- 1. successor_activations
CREATE TABLE IF NOT EXISTS successor_activations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    primary_agent_id VARCHAR(255) NOT NULL,
    successor_agent_id VARCHAR(255) NOT NULL,
    activation_reason TEXT NOT NULL,
    activated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    activated_by VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'deactivated')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    deactivated_by VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_successor_poa ON successor_activations(poa_id);
CREATE INDEX IF NOT EXISTS idx_successor_active ON successor_activations(status) WHERE status = 'active';

-- 2. ai_delegations
CREATE TABLE IF NOT EXISTS ai_delegations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    source_agent_id VARCHAR(255) NOT NULL,
    target_agent_id VARCHAR(255) NOT NULL,
    delegated_scope JSONB NOT NULL,
    delegation_depth INTEGER NOT NULL,
    max_allowed_depth INTEGER NOT NULL,
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_until TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    delegation_policy JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_delegation_poa ON ai_delegations(source_poa_id);
CREATE INDEX IF NOT EXISTS idx_delegation_target ON ai_delegations(target_agent_id);

-- 3. ai_capability_assessments
CREATE TABLE IF NOT EXISTS ai_capability_assessments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id VARCHAR(255) NOT NULL,
    assessed_by VARCHAR(255) NOT NULL,
    assessment_date TIMESTAMP WITH TIME ZONE NOT NULL,
    overall_level VARCHAR(10) NOT NULL,
    domain_scores JSONB NOT NULL,
    risk_profile JSONB NOT NULL,
    limitations JSONB,
    certifications JSONB,
    valid_until TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_assessment_agent ON ai_capability_assessments(agent_id);

-- 4. dual_control_approvals
CREATE TABLE IF NOT EXISTS dual_control_approvals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    action_type VARCHAR(100) NOT NULL,
    action_description TEXT,
    requested_by VARCHAR(255) NOT NULL,
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL,
    required_approvers INTEGER NOT NULL DEFAULT 2,
    approval_threshold VARCHAR(50) NOT NULL DEFAULT 'majority',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    approved_by JSONB DEFAULT '[]',
    rejected_by JSONB DEFAULT '[]',
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    decision_finalized_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_dual_control_poa ON dual_control_approvals(poa_id);
CREATE INDEX IF NOT EXISTS idx_dual_control_status ON dual_control_approvals(status);

-- 5. fiduciary_duty_violations
CREATE TABLE IF NOT EXISTS fiduciary_duty_violations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    agent_id VARCHAR(255) NOT NULL,
    duty_type VARCHAR(100) NOT NULL,
    violation_description TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    detected_by VARCHAR(255) NOT NULL,
    resolution_status VARCHAR(50) DEFAULT 'open',
    consequences JSONB,
    evidence JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_fiduciary_poa ON fiduciary_duty_violations(poa_id);
CREATE INDEX IF NOT EXISTS idx_fiduciary_agent ON fiduciary_duty_violations(agent_id);
