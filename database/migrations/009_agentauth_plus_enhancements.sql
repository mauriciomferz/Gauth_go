-- Migration 009: AgentAuth+ Enhancements
-- Adds successor AI, delegation policy, dual control, and fiduciary duties support

-- Add AgentAuth+ fields to power_of_attorney table
ALTER TABLE power_of_attorney 
ADD COLUMN IF NOT EXISTS successor_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS delegation_policy JSONB,
ADD COLUMN IF NOT EXISTS fiduciary_duties JSONB,
ADD COLUMN IF NOT EXISTS obligation_type VARCHAR(50) DEFAULT 'permissive',
ADD COLUMN IF NOT EXISTS capability_requirements JSONB,
ADD COLUMN IF NOT EXISTS dual_control_required BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS approval_workflow_id VARCHAR(255);

-- Create successor activation tracking table
CREATE TABLE IF NOT EXISTS successor_activations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    primary_agent_id VARCHAR(255) NOT NULL,
    successor_agent_id VARCHAR(255) NOT NULL,
    activation_reason VARCHAR(100) NOT NULL, -- 'unavailable', 'failure', 'manual', 'timeout'
    activated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    activated_by VARCHAR(255) NOT NULL,
    deactivated_at TIMESTAMP,
    deactivated_by VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active', -- 'active', 'deactivated', 'superseded'
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create delegation tracking table for AI-to-AI delegations
CREATE TABLE IF NOT EXISTS ai_delegations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    source_agent_id VARCHAR(255) NOT NULL,
    target_agent_id VARCHAR(255) NOT NULL,
    delegated_scope JSONB NOT NULL,
    delegation_depth INT NOT NULL DEFAULT 1,
    max_allowed_depth INT NOT NULL,
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    status VARCHAR(50) DEFAULT 'active', -- 'active', 'revoked', 'expired'
    delegation_policy JSONB,
    revoked_at TIMESTAMP,
    revoked_by VARCHAR(255),
    revocation_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create dual control approvals table
CREATE TABLE IF NOT EXISTS dual_control_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    action_type VARCHAR(100) NOT NULL,
    action_description TEXT NOT NULL,
    requested_by VARCHAR(255) NOT NULL,
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
    required_approvers INT NOT NULL DEFAULT 2,
    approval_threshold VARCHAR(50) DEFAULT 'all', -- 'all', 'majority', 'quorum', 'weighted'
    status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'approved', 'rejected', 'expired'
    approved_by JSONB DEFAULT '[]'::JSONB,
    rejected_by JSONB DEFAULT '[]'::JSONB,
    decision_finalized_at TIMESTAMP,
    expires_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create fiduciary duty violations tracking table
CREATE TABLE IF NOT EXISTS fiduciary_duty_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poa_id UUID NOT NULL REFERENCES power_of_attorney(id) ON DELETE CASCADE,
    agent_id VARCHAR(255) NOT NULL,
    duty_type VARCHAR(100) NOT NULL, -- 'care', 'loyalty', 'good_faith', 'disclosure', 'confidentiality'
    violation_description TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL, -- 'minor', 'moderate', 'major', 'critical'
    detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    detected_by VARCHAR(255) NOT NULL,
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP,
    resolution_status VARCHAR(50) DEFAULT 'open', -- 'open', 'investigating', 'resolved', 'dismissed'
    resolution_notes TEXT,
    consequences JSONB,
    evidence JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create AI capability assessments table
CREATE TABLE IF NOT EXISTS ai_capability_assessments (
    id VARCHAR(255) PRIMARY KEY,
    agent_id VARCHAR(255) NOT NULL,
    assessment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    capability_level VARCHAR(10) NOT NULL, -- 'L0' through 'L5'
    capability_domains JSONB NOT NULL, -- {financial: 0.95, legal: 0.80, medical: 0.0}
    risk_score DECIMAL(3,2) NOT NULL, -- 0.00 to 1.00
    certification_status VARCHAR(50) DEFAULT 'uncertified',
    certifications JSONB DEFAULT '[]'::JSONB,
    limitations JSONB,
    recommended_restrictions JSONB,
    assessed_by VARCHAR(255) NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    superseded_by VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_successor_activations_poa_id ON successor_activations(poa_id);
CREATE INDEX IF NOT EXISTS idx_successor_activations_status ON successor_activations(status);
CREATE INDEX IF NOT EXISTS idx_successor_activations_activated_at ON successor_activations(activated_at);

CREATE INDEX IF NOT EXISTS idx_ai_delegations_source_poa ON ai_delegations(source_poa_id);
CREATE INDEX IF NOT EXISTS idx_ai_delegations_source_agent ON ai_delegations(source_agent_id);
CREATE INDEX IF NOT EXISTS idx_ai_delegations_target_agent ON ai_delegations(target_agent_id);
CREATE INDEX IF NOT EXISTS idx_ai_delegations_status ON ai_delegations(status);
CREATE INDEX IF NOT EXISTS idx_ai_delegations_depth ON ai_delegations(delegation_depth);

CREATE INDEX IF NOT EXISTS idx_dual_control_poa_id ON dual_control_approvals(poa_id);
CREATE INDEX IF NOT EXISTS idx_dual_control_status ON dual_control_approvals(status);
CREATE INDEX IF NOT EXISTS idx_dual_control_requested_at ON dual_control_approvals(requested_at);

CREATE INDEX IF NOT EXISTS idx_fiduciary_violations_poa_id ON fiduciary_duty_violations(poa_id);
CREATE INDEX IF NOT EXISTS idx_fiduciary_violations_agent_id ON fiduciary_duty_violations(agent_id);
CREATE INDEX IF NOT EXISTS idx_fiduciary_violations_severity ON fiduciary_duty_violations(severity);
CREATE INDEX IF NOT EXISTS idx_fiduciary_violations_status ON fiduciary_duty_violations(resolution_status);

CREATE INDEX IF NOT EXISTS idx_capability_assessments_agent_id ON ai_capability_assessments(agent_id);
CREATE INDEX IF NOT EXISTS idx_capability_assessments_valid_until ON ai_capability_assessments(valid_until);
CREATE INDEX IF NOT EXISTS idx_capability_assessments_risk_score ON ai_capability_assessments(risk_score);

-- Add comments for documentation
COMMENT ON COLUMN power_of_attorney.successor_id IS 'AgentAuth+: Backup AI agent to activate if primary agent fails or is unavailable';
COMMENT ON COLUMN power_of_attorney.delegation_policy IS 'AgentAuth+: JSON policy defining delegation rules {can_delegate:bool, max_depth:int, allowed_delegates:[]}';
COMMENT ON COLUMN power_of_attorney.fiduciary_duties IS 'AgentAuth+: JSON array of fiduciary duties this agent must uphold [care, loyalty, good_faith, etc.]';
COMMENT ON COLUMN power_of_attorney.obligation_type IS 'AgentAuth+: permissive (do-unless) or mandatory (need-to-do)';
COMMENT ON COLUMN power_of_attorney.capability_requirements IS 'AgentAuth+: Required AI capability levels for this PoA';
COMMENT ON COLUMN power_of_attorney.dual_control_required IS 'AgentAuth+: Whether high-risk actions require second-level approval';

COMMENT ON TABLE successor_activations IS 'AgentAuth+: Tracks when successor AIs are activated to replace primary agents';
COMMENT ON TABLE ai_delegations IS 'AgentAuth+: Tracks AI-to-AI delegations with depth limits and policy enforcement';
COMMENT ON TABLE dual_control_approvals IS 'AgentAuth+: Second-level approval workflow for high-risk AI actions';
COMMENT ON TABLE fiduciary_duty_violations IS 'AgentAuth+: Tracks and manages fiduciary duty breaches by AI agents';
COMMENT ON TABLE ai_capability_assessments IS 'AgentAuth+: Periodic capability assessments to match AI capabilities with authorization scope';
