-- Migration: Policy Templates System
-- Description: Complete policy template management with versioning, validation, analytics, and marketplace
-- Author: GitHub Copilot
-- Date: 2025-11-24

-- =============================================================================
-- 1. Policy Templates Core Table
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL, -- 'authorization', 'access_control', 'data_governance', etc.
    industry VARCHAR(100), -- 'healthcare', 'finance', 'government', 'retail', 'generic'
    
    -- Template content
    template_type VARCHAR(50) NOT NULL, -- 'abac', 'rbac', 'pbac', 'hybrid'
    policy_rules JSONB NOT NULL, -- The actual policy rules
    variables JSONB DEFAULT '[]'::jsonb, -- Template variables for customization
    
    -- Versioning
    version INTEGER NOT NULL DEFAULT 1,
    is_latest BOOLEAN NOT NULL DEFAULT true,
    parent_template_id UUID REFERENCES policy_templates(id) ON DELETE SET NULL,
    
    -- Status and visibility
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- 'draft', 'active', 'deprecated', 'archived'
    visibility VARCHAR(50) NOT NULL DEFAULT 'private', -- 'private', 'organization', 'public', 'marketplace'
    
    -- Marketplace fields
    is_marketplace_item BOOLEAN NOT NULL DEFAULT false,
    marketplace_rating DECIMAL(3,2) DEFAULT 0.00,
    marketplace_downloads INTEGER DEFAULT 0,
    marketplace_price DECIMAL(10,2) DEFAULT 0.00, -- 0 for free templates
    author_id VARCHAR(255),
    license VARCHAR(100) DEFAULT 'proprietary', -- 'MIT', 'Apache-2.0', 'proprietary', etc.
    
    -- Metadata
    tags TEXT[] DEFAULT '{}',
    compliance_frameworks TEXT[] DEFAULT '{}', -- 'GDPR', 'HIPAA', 'SOC2', etc.
    created_by VARCHAR(255) NOT NULL,
    updated_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ,
    deprecated_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT valid_template_status CHECK (status IN ('draft', 'active', 'deprecated', 'archived')),
    CONSTRAINT valid_template_visibility CHECK (visibility IN ('private', 'organization', 'public', 'marketplace')),
    CONSTRAINT valid_template_type CHECK (template_type IN ('abac', 'rbac', 'pbac', 'hybrid', 'custom')),
    CONSTRAINT valid_template_category CHECK (category IN ('authorization', 'access_control', 'data_governance', 'compliance', 'audit', 'custom')),
    CONSTRAINT valid_marketplace_rating CHECK (marketplace_rating >= 0.00 AND marketplace_rating <= 5.00),
    CONSTRAINT unique_template_version UNIQUE (tenant_id, name, version)
);

-- Indexes
CREATE INDEX idx_policy_templates_tenant_id ON policy_templates(tenant_id);
CREATE INDEX idx_policy_templates_category ON policy_templates(category);
CREATE INDEX idx_policy_templates_industry ON policy_templates(industry) WHERE industry IS NOT NULL;
CREATE INDEX idx_policy_templates_status ON policy_templates(status);
CREATE INDEX idx_policy_templates_visibility ON policy_templates(visibility);
CREATE INDEX idx_policy_templates_is_latest ON policy_templates(is_latest) WHERE is_latest = true;
CREATE INDEX idx_policy_templates_marketplace ON policy_templates(is_marketplace_item) WHERE is_marketplace_item = true;
CREATE INDEX idx_policy_templates_parent ON policy_templates(parent_template_id) WHERE parent_template_id IS NOT NULL;
CREATE INDEX idx_policy_templates_tags ON policy_templates USING gin(tags);
CREATE INDEX idx_policy_templates_compliance ON policy_templates USING gin(compliance_frameworks);
CREATE INDEX idx_policy_templates_created_at ON policy_templates(created_at DESC);

-- RLS Policy
ALTER TABLE policy_templates ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_templates
    USING (tenant_id::text = current_tenant_id()::text 
           OR visibility IN ('public', 'marketplace'));

-- =============================================================================
-- 2. Template Version History
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_template_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES policy_templates(id) ON DELETE CASCADE,
    tenant_id VARCHAR(100) NOT NULL,
    
    version_number INTEGER NOT NULL,
    changelog TEXT,
    
    -- Version snapshot
    template_snapshot JSONB NOT NULL, -- Full template data at this version
    policy_rules_diff JSONB, -- Diff from previous version
    
    -- Version metadata
    version_status VARCHAR(50) NOT NULL DEFAULT 'current', -- 'current', 'superseded', 'rolled_back'
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deployed_at TIMESTAMPTZ,
    superseded_at TIMESTAMPTZ,
    
    CONSTRAINT valid_version_status CHECK (version_status IN ('current', 'superseded', 'rolled_back')),
    CONSTRAINT unique_template_version_number UNIQUE (template_id, version_number)
);

CREATE INDEX idx_template_versions_template_id ON policy_template_versions(template_id);
CREATE INDEX idx_template_versions_tenant_id ON policy_template_versions(tenant_id);
CREATE INDEX idx_template_versions_created_at ON policy_template_versions(created_at DESC);
CREATE INDEX idx_template_versions_status ON policy_template_versions(version_status);

-- RLS Policy
ALTER TABLE policy_template_versions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_template_versions
    USING (tenant_id::text = current_tenant_id()::text);

-- =============================================================================
-- 3. Template Validation Rules
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_template_validation_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL,
    
    rule_name VARCHAR(255) NOT NULL,
    rule_description TEXT,
    rule_type VARCHAR(50) NOT NULL, -- 'syntax', 'semantic', 'security', 'compliance', 'performance'
    
    -- Rule definition
    validation_function TEXT NOT NULL, -- JavaScript/Lua/Go function for validation
    error_message_template TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'error', -- 'error', 'warning', 'info'
    
    -- Rule application
    applies_to_categories TEXT[] DEFAULT '{}', -- Which categories this rule applies to
    applies_to_types TEXT[] DEFAULT '{}', -- Which template types this applies to
    is_required BOOLEAN NOT NULL DEFAULT true,
    
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_rule_type CHECK (rule_type IN ('syntax', 'semantic', 'security', 'compliance', 'performance', 'custom')),
    CONSTRAINT valid_severity CHECK (severity IN ('error', 'warning', 'info'))
);

CREATE INDEX idx_validation_rules_tenant_id ON policy_template_validation_rules(tenant_id);
CREATE INDEX idx_validation_rules_type ON policy_template_validation_rules(rule_type);
CREATE INDEX idx_validation_rules_active ON policy_template_validation_rules(is_active) WHERE is_active = true;

-- RLS Policy
ALTER TABLE policy_template_validation_rules ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_template_validation_rules
    USING (tenant_id::text = current_tenant_id()::text);

-- =============================================================================
-- 4. Template Validation Results
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_template_validations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES policy_templates(id) ON DELETE CASCADE,
    tenant_id VARCHAR(100) NOT NULL,
    
    validation_status VARCHAR(50) NOT NULL, -- 'passed', 'failed', 'warning', 'skipped'
    
    -- Validation results
    total_rules_checked INTEGER NOT NULL DEFAULT 0,
    rules_passed INTEGER NOT NULL DEFAULT 0,
    rules_failed INTEGER NOT NULL DEFAULT 0,
    rules_warned INTEGER NOT NULL DEFAULT 0,
    
    validation_results JSONB NOT NULL DEFAULT '[]'::jsonb, -- Array of individual rule results
    
    -- Context
    validated_by VARCHAR(255) NOT NULL,
    validated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    validation_duration_ms INTEGER,
    
    CONSTRAINT valid_validation_status CHECK (validation_status IN ('passed', 'failed', 'warning', 'skipped'))
);

CREATE INDEX idx_template_validations_template_id ON policy_template_validations(template_id);
CREATE INDEX idx_template_validations_tenant_id ON policy_template_validations(tenant_id);
CREATE INDEX idx_template_validations_status ON policy_template_validations(validation_status);
CREATE INDEX idx_template_validations_validated_at ON policy_template_validations(validated_at DESC);

-- RLS Policy
ALTER TABLE policy_template_validations ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_template_validations
    USING (tenant_id::text = current_tenant_id()::text);

-- =============================================================================
-- 5. Template Analytics & Usage Tracking
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_template_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES policy_templates(id) ON DELETE CASCADE,
    tenant_id VARCHAR(100) NOT NULL,
    
    -- Usage metrics
    total_deployments INTEGER NOT NULL DEFAULT 0,
    active_deployments INTEGER NOT NULL DEFAULT 0,
    total_evaluations BIGINT NOT NULL DEFAULT 0,
    
    -- Performance metrics
    avg_evaluation_time_ms DECIMAL(10,2),
    p50_evaluation_time_ms DECIMAL(10,2),
    p95_evaluation_time_ms DECIMAL(10,2),
    p99_evaluation_time_ms DECIMAL(10,2),
    
    -- Effectiveness metrics
    total_denials BIGINT NOT NULL DEFAULT 0,
    total_approvals BIGINT NOT NULL DEFAULT 0,
    false_positive_rate DECIMAL(5,4),
    false_negative_rate DECIMAL(5,4),
    
    -- Time periods
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    
    -- Metadata
    last_updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_template_analytics_period UNIQUE (template_id, period_start)
);

CREATE INDEX idx_template_analytics_template_id ON policy_template_analytics(template_id);
CREATE INDEX idx_template_analytics_tenant_id ON policy_template_analytics(tenant_id);
CREATE INDEX idx_template_analytics_period ON policy_template_analytics(period_start DESC, period_end DESC);

-- RLS Policy
ALTER TABLE policy_template_analytics ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_template_analytics
    USING (tenant_id::text = current_tenant_id()::text);

-- =============================================================================
-- 6. Template Dynamic Switching Rules
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_template_switch_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id VARCHAR(100) NOT NULL,
    
    rule_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Source and target templates
    from_template_id UUID REFERENCES policy_templates(id) ON DELETE CASCADE,
    to_template_id UUID NOT NULL REFERENCES policy_templates(id) ON DELETE CASCADE,
    
    -- Switching conditions
    switch_conditions JSONB NOT NULL, -- Conditions that trigger the switch
    priority INTEGER NOT NULL DEFAULT 100, -- Higher priority rules evaluated first
    
    -- Context matching
    context_attributes TEXT[] DEFAULT '{}', -- Which attributes to evaluate
    time_based_rule JSONB, -- Schedule-based switching (e.g., business hours)
    
    -- Status and behavior
    is_active BOOLEAN NOT NULL DEFAULT true,
    switch_mode VARCHAR(50) NOT NULL DEFAULT 'replace', -- 'replace', 'augment', 'fallback'
    
    -- Metadata
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_triggered_at TIMESTAMPTZ,
    trigger_count INTEGER NOT NULL DEFAULT 0,
    
    CONSTRAINT valid_switch_mode CHECK (switch_mode IN ('replace', 'augment', 'fallback'))
);

CREATE INDEX idx_switch_rules_tenant_id ON policy_template_switch_rules(tenant_id);
CREATE INDEX idx_switch_rules_from_template ON policy_template_switch_rules(from_template_id) WHERE from_template_id IS NOT NULL;
CREATE INDEX idx_switch_rules_to_template ON policy_template_switch_rules(to_template_id);
CREATE INDEX idx_switch_rules_active ON policy_template_switch_rules(is_active) WHERE is_active = true;
CREATE INDEX idx_switch_rules_priority ON policy_template_switch_rules(priority DESC);

-- RLS Policy
ALTER TABLE policy_template_switch_rules ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_template_switch_rules
    USING (tenant_id::text = current_tenant_id()::text);

-- =============================================================================
-- 7. Template Marketplace Reviews & Ratings
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_template_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES policy_templates(id) ON DELETE CASCADE,
    tenant_id VARCHAR(100) NOT NULL,
    
    -- Review details
    reviewer_id VARCHAR(255) NOT NULL,
    reviewer_name VARCHAR(255),
    rating INTEGER NOT NULL, -- 1-5 stars
    title VARCHAR(255),
    review_text TEXT,
    
    -- Usefulness tracking
    helpful_count INTEGER NOT NULL DEFAULT 0,
    not_helpful_count INTEGER NOT NULL DEFAULT 0,
    
    -- Verification
    is_verified_purchase BOOLEAN NOT NULL DEFAULT false,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'published', -- 'pending', 'published', 'hidden', 'flagged'
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_rating CHECK (rating >= 1 AND rating <= 5),
    CONSTRAINT valid_review_status CHECK (status IN ('pending', 'published', 'hidden', 'flagged'))
);

CREATE INDEX idx_template_reviews_template_id ON policy_template_reviews(template_id);
CREATE INDEX idx_template_reviews_tenant_id ON policy_template_reviews(tenant_id);
CREATE INDEX idx_template_reviews_rating ON policy_template_reviews(rating);
CREATE INDEX idx_template_reviews_status ON policy_template_reviews(status);
CREATE INDEX idx_template_reviews_created_at ON policy_template_reviews(created_at DESC);

-- RLS Policy
ALTER TABLE policy_template_reviews ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_template_reviews
    USING (tenant_id::text = current_tenant_id()::text);

-- =============================================================================
-- 8. Template Cloning/Fork History
-- =============================================================================

CREATE TABLE IF NOT EXISTS policy_template_forks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_template_id UUID NOT NULL REFERENCES policy_templates(id) ON DELETE CASCADE,
    forked_template_id UUID NOT NULL REFERENCES policy_templates(id) ON DELETE CASCADE,
    tenant_id VARCHAR(100) NOT NULL,
    
    fork_type VARCHAR(50) NOT NULL, -- 'clone', 'inherit', 'customize'
    customizations JSONB, -- What was changed from original
    
    forked_by VARCHAR(255) NOT NULL,
    forked_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_fork_type CHECK (fork_type IN ('clone', 'inherit', 'customize', 'marketplace_import'))
);

CREATE INDEX idx_template_forks_original ON policy_template_forks(original_template_id);
CREATE INDEX idx_template_forks_forked ON policy_template_forks(forked_template_id);
CREATE INDEX idx_template_forks_tenant_id ON policy_template_forks(tenant_id);

-- RLS Policy
ALTER TABLE policy_template_forks ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON policy_template_forks
    USING (tenant_id::text = current_tenant_id()::text);

-- =============================================================================
-- 9. Pre-configured Industry Templates (Seed Data)
-- =============================================================================

-- Note: Actual seed data will be inserted via separate seeding script
-- This just ensures the structure exists

-- =============================================================================
-- 10. Triggers for Auto-updating Updated_at
-- =============================================================================

CREATE OR REPLACE FUNCTION update_policy_templates_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_policy_templates_updated_at
    BEFORE UPDATE ON policy_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_templates_updated_at();

CREATE TRIGGER update_validation_rules_updated_at
    BEFORE UPDATE ON policy_template_validation_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_templates_updated_at();

CREATE TRIGGER update_switch_rules_updated_at
    BEFORE UPDATE ON policy_template_switch_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_templates_updated_at();

CREATE TRIGGER update_template_reviews_updated_at
    BEFORE UPDATE ON policy_template_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_templates_updated_at();

-- =============================================================================
-- 11. Comments for Documentation
-- =============================================================================

COMMENT ON TABLE policy_templates IS 'Core table for policy template management with versioning and marketplace support';
COMMENT ON TABLE policy_template_versions IS 'Version history and changelog for policy templates';
COMMENT ON TABLE policy_template_validation_rules IS 'Configurable validation rules for template quality assurance';
COMMENT ON TABLE policy_template_validations IS 'Validation execution results and audit trail';
COMMENT ON TABLE policy_template_analytics IS 'Usage metrics and performance analytics for templates';
COMMENT ON TABLE policy_template_switch_rules IS 'Dynamic template switching rules based on context';
COMMENT ON TABLE policy_template_reviews IS 'Marketplace reviews and ratings for public templates';
COMMENT ON TABLE policy_template_forks IS 'Template cloning and inheritance relationship tracking';
