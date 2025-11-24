-- Migration: Initial schema for GAuth Admin Portal
-- Version: 001
-- Description: Creates all tables for multi-tenant admin portal with RLS

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Import full schema from db/schema/001_initial_schema.sql
-- This migration creates all core tables for the admin portal

-- Note: In production, copy the content from db/schema/001_initial_schema.sql
-- For now, this is a placeholder that references the main schema file

-- Tables created:
-- - subscribers (tenant management)
-- - tokens & token_blacklist
-- - policies, policy_attributes, authorization_logs
-- - poa_records, poa_templates
-- - event_types, events, event_handlers
-- - circuit_breakers, rate_limiters, retry_policies, bulkheads
-- - audit_events, compliance_reports, event_correlation_patterns, siem_integrations
-- - config_variables, config_files, service_configs, tenant_config_overrides, feature_flags
-- - merkle_tree_nodes, merkle_proofs, revocations, append_only_log

-- All tables have Row-Level Security enabled for multi-tenant isolation
