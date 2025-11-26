-- Migration 010: Fix AI Capability Assessments Schema
-- Aligns table structure with service implementation

-- Rename columns to match service code expectations
ALTER TABLE ai_capability_assessments 
  RENAME COLUMN capability_level TO overall_level;

ALTER TABLE ai_capability_assessments 
  RENAME COLUMN capability_domains TO domain_scores;

-- Drop risk_score column and add risk_profile jsonb
ALTER TABLE ai_capability_assessments 
  DROP COLUMN risk_score;

ALTER TABLE ai_capability_assessments 
  ADD COLUMN risk_profile jsonb DEFAULT '{}'::jsonb;

-- Add notes column
ALTER TABLE ai_capability_assessments 
  ADD COLUMN notes text;

-- Update indexes
DROP INDEX IF EXISTS idx_capability_assessments_risk_score;
CREATE INDEX idx_capability_assessments_agent_valid ON ai_capability_assessments(agent_id, valid_until);

COMMENT ON COLUMN ai_capability_assessments.overall_level IS 'AI capability level: L0 through L5';
COMMENT ON COLUMN ai_capability_assessments.domain_scores IS 'JSON object mapping domain names to scores (0.0-1.0)';
COMMENT ON COLUMN ai_capability_assessments.risk_profile IS 'JSON object with risk assessment details';
COMMENT ON COLUMN ai_capability_assessments.notes IS 'Additional assessment notes and observations';
