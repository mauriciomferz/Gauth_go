-- Seed Data: Pre-configured Industry-Specific Policy Templates
-- Description: Ready-to-use policy templates for various industries
-- Author: GitHub Copilot
-- Date: 2025-11-24

-- =============================================================================
-- Healthcare Industry Templates (HIPAA, PHI Protection)
-- =============================================================================

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system', -- System templates available to all tenants
    'HIPAA PHI Access Control',
    'Attribute-based access control template enforcing HIPAA compliance for Protected Health Information (PHI) access',
    'access_control',
    'healthcare',
    'abac',
    '{
        "rules": [
            {
                "name": "PHI Access Requires Authorization",
                "condition": "resource.type == \"phi\" && subject.role in [\"doctor\", \"nurse\", \"authorized_personnel\"]",
                "effect": "allow",
                "priority": 100
            },
            {
                "name": "Patient Can Access Own Records",
                "condition": "resource.type == \"phi\" && resource.patient_id == subject.patient_id",
                "effect": "allow",
                "priority": 90
            },
            {
                "name": "Emergency Override",
                "condition": "context.emergency_mode == true && subject.role in [\"doctor\", \"nurse\"]",
                "effect": "allow",
                "obligations": ["log_emergency_access", "notify_compliance"],
                "priority": 110
            },
            {
                "name": "Audit All PHI Access",
                "condition": "resource.type == \"phi\"",
                "effect": "allow",
                "obligations": ["audit_log", "encrypt_audit"],
                "priority": 1
            }
        ],
        "default_effect": "deny"
    }'::jsonb,
    '[
        {"name": "authorized_roles", "type": "array", "default": ["doctor", "nurse", "authorized_personnel"]},
        {"name": "emergency_access_enabled", "type": "boolean", "default": true},
        {"name": "audit_retention_days", "type": "integer", "default": 2555}
    ]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.8, 1247,
    'system', 'Apache-2.0',
    ARRAY['hipaa', 'healthcare', 'phi', 'access-control', 'compliance'],
    ARRAY['HIPAA', 'HITECH'],
    'system'
);

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system',
    'Healthcare Data Governance',
    'Comprehensive data governance template for healthcare organizations managing patient data lifecycle',
    'data_governance',
    'healthcare',
    'hybrid',
    '{
        "rules": [
            {
                "name": "Data Minimization",
                "condition": "action == \"collect_data\"",
                "effect": "allow",
                "obligations": ["validate_necessity", "document_purpose", "set_retention"],
                "priority": 100
            },
            {
                "name": "Patient Consent Required",
                "condition": "action in [\"share_data\", \"export_data\"] && !resource.consent_obtained",
                "effect": "deny",
                "priority": 110
            },
            {
                "name": "Automatic De-identification",
                "condition": "action == \"export_for_research\" && context.purpose == \"research\"",
                "effect": "allow",
                "obligations": ["deidentify_phi", "remove_direct_identifiers"],
                "priority": 90
            }
        ],
        "default_effect": "deny"
    }'::jsonb,
    '[]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.6, 892,
    'system', 'Apache-2.0',
    ARRAY['healthcare', 'data-governance', 'patient-data', 'consent'],
    ARRAY['HIPAA', 'GDPR'],
    'system'
);

-- =============================================================================
-- Financial Services Templates (PCI-DSS, SOX)
-- =============================================================================

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system',
    'PCI-DSS Payment Data Protection',
    'PCI-DSS compliant access control for cardholder data environment (CDE)',
    'access_control',
    'finance',
    'abac',
    '{
        "rules": [
            {
                "name": "CDE Access Restricted",
                "condition": "resource.environment == \"cde\" && subject.clearance_level >= 3",
                "effect": "allow",
                "obligations": ["mfa_required", "log_access"],
                "priority": 100
            },
            {
                "name": "Cardholder Data Encryption",
                "condition": "resource.contains_pan == true",
                "effect": "allow",
                "obligations": ["encrypt_at_rest", "encrypt_in_transit", "mask_display"],
                "priority": 110
            },
            {
                "name": "Segregation of Duties",
                "condition": "action in [\"approve_transaction\", \"execute_transaction\"] && subject.user_id == resource.initiator_id",
                "effect": "deny",
                "priority": 120
            },
            {
                "name": "Time-based Access Control",
                "condition": "resource.environment == \"cde\" && !context.within_business_hours",
                "effect": "deny",
                "priority": 90
            }
        ],
        "default_effect": "deny"
    }'::jsonb,
    '[
        {"name": "business_hours_start", "type": "time", "default": "08:00"},
        {"name": "business_hours_end", "type": "time", "default": "18:00"},
        {"name": "min_clearance_level", "type": "integer", "default": 3}
    ]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.9, 2134,
    'system', 'Apache-2.0',
    ARRAY['pci-dss', 'finance', 'payment', 'cardholder-data', 'encryption'],
    ARRAY['PCI-DSS', 'SOX'],
    'system'
);

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system',
    'Financial Transaction Authorization',
    'Multi-level authorization template for financial transactions with fraud detection',
    'authorization',
    'finance',
    'hybrid',
    '{
        "rules": [
            {
                "name": "Transaction Amount Thresholds",
                "condition": "action == \"approve_transaction\" && resource.amount > 10000 && subject.approval_limit < resource.amount",
                "effect": "deny",
                "priority": 100
            },
            {
                "name": "Dual Authorization for High Value",
                "condition": "resource.amount > 100000 && count(resource.approvals) < 2",
                "effect": "deny",
                "priority": 110
            },
            {
                "name": "Velocity Check",
                "condition": "context.transactions_last_hour > 50 || context.total_amount_last_hour > 500000",
                "effect": "deny",
                "obligations": ["trigger_fraud_alert", "notify_security"],
                "priority": 120
            },
            {
                "name": "Geographic Restriction",
                "condition": "resource.destination_country in context.high_risk_countries",
                "effect": "deny",
                "obligations": ["manual_review_required"],
                "priority": 105
            }
        ],
        "default_effect": "allow"
    }'::jsonb,
    '[
        {"name": "threshold_low", "type": "number", "default": 10000},
        {"name": "threshold_high", "type": "number", "default": 100000},
        {"name": "velocity_max_transactions", "type": "integer", "default": 50},
        {"name": "high_risk_countries", "type": "array", "default": []}
    ]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.7, 1678,
    'system', 'Apache-2.0',
    ARRAY['finance', 'transaction', 'authorization', 'fraud-detection'],
    ARRAY['SOX', 'ISO27001'],
    'system'
);

-- =============================================================================
-- Government/Public Sector Templates
-- =============================================================================

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system',
    'Government Classified Data Access',
    'Multi-level security (MLS) access control for classified government information',
    'access_control',
    'government',
    'abac',
    '{
        "rules": [
            {
                "name": "Security Clearance Level",
                "condition": "subject.clearance_level >= resource.classification_level",
                "effect": "allow",
                "priority": 100
            },
            {
                "name": "Need to Know",
                "condition": "resource.compartment in subject.authorized_compartments",
                "effect": "allow",
                "priority": 110
            },
            {
                "name": "No Write Down",
                "condition": "action == \"write\" && subject.clearance_level < resource.classification_level",
                "effect": "deny",
                "priority": 120
            },
            {
                "name": "No Read Up",
                "condition": "action == \"read\" && subject.clearance_level < resource.classification_level",
                "effect": "deny",
                "priority": 120
            }
        ],
        "default_effect": "deny"
    }'::jsonb,
    '[
        {"name": "clearance_levels", "type": "object", "default": {"unclassified": 0, "confidential": 1, "secret": 2, "top_secret": 3}}
    ]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.5, 564,
    'system', 'Apache-2.0',
    ARRAY['government', 'classified', 'security-clearance', 'mls', 'bell-lapadula'],
    ARRAY['FedRAMP', 'NIST'],
    'system'
);

-- =============================================================================
-- Retail/E-commerce Templates
-- =============================================================================

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system',
    'E-commerce Customer Data Protection',
    'Privacy-focused access control for customer personal data and purchase history',
    'access_control',
    'retail',
    'abac',
    '{
        "rules": [
            {
                "name": "Customer Service Access",
                "condition": "subject.role == \"customer_service\" && action in [\"read\", \"update\"] && resource.type == \"customer_profile\"",
                "effect": "allow",
                "obligations": ["log_access", "mask_pii"],
                "priority": 100
            },
            {
                "name": "Marketing Department Restrictions",
                "condition": "subject.department == \"marketing\" && resource.pii_sensitivity == \"high\"",
                "effect": "deny",
                "priority": 110
            },
            {
                "name": "Customer Self-Service",
                "condition": "subject.customer_id == resource.customer_id && action != \"delete\"",
                "effect": "allow",
                "priority": 90
            },
            {
                "name": "GDPR Right to Erasure",
                "condition": "action == \"delete\" && subject.customer_id == resource.customer_id && context.gdpr_request == true",
                "effect": "allow",
                "obligations": ["verify_identity", "cascade_delete", "notify_processors"],
                "priority": 120
            }
        ],
        "default_effect": "deny"
    }'::jsonb,
    '[
        {"name": "pii_masking_enabled", "type": "boolean", "default": true},
        {"name": "audit_retention_days", "type": "integer", "default": 365}
    ]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.4, 1893,
    'system', 'Apache-2.0',
    ARRAY['retail', 'ecommerce', 'customer-data', 'privacy', 'gdpr'],
    ARRAY['GDPR', 'CCPA'],
    'system'
);

-- =============================================================================
-- Generic/Cross-Industry Templates
-- =============================================================================

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system',
    'Role-Based Access Control (RBAC) Basic',
    'Simple role-based access control template suitable for any organization',
    'access_control',
    'generic',
    'rbac',
    '{
        "rules": [
            {
                "name": "Admin Full Access",
                "condition": "subject.role == \"admin\"",
                "effect": "allow",
                "priority": 100
            },
            {
                "name": "Manager Department Access",
                "condition": "subject.role == \"manager\" && subject.department == resource.department",
                "effect": "allow",
                "priority": 90
            },
            {
                "name": "User Read Only",
                "condition": "subject.role == \"user\" && action == \"read\"",
                "effect": "allow",
                "priority": 80
            }
        ],
        "default_effect": "deny"
    }'::jsonb,
    '[
        {"name": "roles", "type": "array", "default": ["admin", "manager", "user", "guest"]},
        {"name": "enable_audit", "type": "boolean", "default": true}
    ]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.6, 5472,
    'system', 'MIT',
    ARRAY['rbac', 'basic', 'generic', 'access-control'],
    ARRAY['ISO27001'],
    'system'
);

INSERT INTO policy_templates (
    id, tenant_id, name, description, category, industry, template_type,
    policy_rules, variables, version, is_latest, status, visibility,
    is_marketplace_item, marketplace_rating, marketplace_downloads,
    author_id, license, tags, compliance_frameworks, created_by
) VALUES (
    uuid_generate_v4(),
    'system',
    'Time-Based Access Control',
    'Temporal access control template restricting access based on time windows and schedules',
    'access_control',
    'generic',
    'abac',
    '{
        "rules": [
            {
                "name": "Business Hours Access",
                "condition": "context.current_time >= context.business_hours_start && context.current_time <= context.business_hours_end",
                "effect": "allow",
                "priority": 100
            },
            {
                "name": "Weekend Restriction",
                "condition": "context.day_of_week in [\"saturday\", \"sunday\"] && subject.role != \"admin\"",
                "effect": "deny",
                "priority": 110
            },
            {
                "name": "Maintenance Window",
                "condition": "context.maintenance_mode == true && subject.role != \"sysadmin\"",
                "effect": "deny",
                "priority": 120
            },
            {
                "name": "Emergency Override",
                "condition": "context.emergency_access == true && subject.role in [\"admin\", \"security_officer\"]",
                "effect": "allow",
                "obligations": ["log_emergency_access", "require_justification"],
                "priority": 130
            }
        ],
        "default_effect": "allow"
    }'::jsonb,
    '[
        {"name": "business_hours_start", "type": "time", "default": "08:00"},
        {"name": "business_hours_end", "type": "time", "default": "18:00"},
        {"name": "timezone", "type": "string", "default": "UTC"}
    ]'::jsonb,
    1, true, 'active', 'marketplace',
    true, 4.3, 2341,
    'system', 'MIT',
    ARRAY['temporal', 'time-based', 'schedule', 'access-control'],
    ARRAY['ISO27001'],
    'system'
);

-- =============================================================================
-- Comments
-- =============================================================================

COMMENT ON TABLE policy_templates IS 'Policy template seed data includes 8 industry-specific templates ready for production use';
