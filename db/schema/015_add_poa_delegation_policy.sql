-- Add delegation_policy to power_of_attorney table
-- Required by GAuth+ subsystem for AI-to-AI delegation validation

ALTER TABLE power_of_attorney
    ADD COLUMN IF NOT EXISTS delegation_policy JSONB;
