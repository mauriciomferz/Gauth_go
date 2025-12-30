-- Fix AgentAuth+ Capability Assessment Schema

ALTER TABLE ai_capability_assessments
    ADD COLUMN IF NOT EXISTS certification_status VARCHAR(50),
    ADD COLUMN IF NOT EXISTS recommended_restrictions JSONB;
