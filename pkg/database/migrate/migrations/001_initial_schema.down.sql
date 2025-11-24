-- Rollback: Initial schema migration
-- Version: 001

-- Drop all tables in reverse dependency order
DROP TABLE IF EXISTS append_only_log CASCADE;
DROP TABLE IF EXISTS revocations CASCADE;
DROP TABLE IF EXISTS merkle_proofs CASCADE;
DROP TABLE IF EXISTS merkle_tree_nodes CASCADE;

DROP TABLE IF EXISTS feature_flags CASCADE;
DROP TABLE IF EXISTS tenant_config_overrides CASCADE;
DROP TABLE IF EXISTS service_configs CASCADE;
DROP TABLE IF EXISTS config_files CASCADE;
DROP TABLE IF EXISTS config_variables CASCADE;

DROP TABLE IF EXISTS siem_integrations CASCADE;
DROP TABLE IF EXISTS event_correlation_patterns CASCADE;
DROP TABLE IF EXISTS compliance_reports CASCADE;
DROP TABLE IF EXISTS audit_events CASCADE;

DROP TABLE IF EXISTS bulkheads CASCADE;
DROP TABLE IF EXISTS retry_policies CASCADE;
DROP TABLE IF EXISTS rate_limiters CASCADE;
DROP TABLE IF EXISTS circuit_breakers CASCADE;

DROP TABLE IF EXISTS event_handlers CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS event_types CASCADE;

DROP TABLE IF EXISTS poa_templates CASCADE;
DROP TABLE IF EXISTS poa_records CASCADE;

DROP TABLE IF EXISTS authorization_logs CASCADE;
DROP TABLE IF EXISTS policy_attributes CASCADE;
DROP TABLE IF EXISTS policies CASCADE;

DROP TABLE IF EXISTS token_blacklist CASCADE;
DROP TABLE IF EXISTS tokens CASCADE;
DROP TABLE IF EXISTS subscribers CASCADE;

-- Drop views
DROP VIEW IF EXISTS active_tokens;
DROP VIEW IF EXISTS recent_audit_events;
DROP VIEW IF EXISTS active_policies;
DROP VIEW IF EXISTS open_circuit_breakers;

-- Drop functions
DROP FUNCTION IF EXISTS current_tenant_id();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop extensions (optional - may be used by other schemas)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
-- DROP EXTENSION IF EXISTS "pgcrypto";
