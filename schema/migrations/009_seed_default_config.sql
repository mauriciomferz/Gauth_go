-- Migration: Seed default configuration
-- Covers: Inserting default gauth-config for test-tenant-1
-- Version: 009
-- Created: 2025-12-21

INSERT INTO config_files (
    tenant_id, file_name, file_format, file_content, description, 
    checksum, size_bytes, version, updated_by
) VALUES (
    'test-tenant-1', 
    'gauth-config', 
    'yaml', 
    'system:
  log_level: info
  environment: production
features:
  audit_logging: true
  mfa_enabled: true
  revocation_check: true
', 
    'Default configuration seed', 
    NULL, -- checksum
    120, -- approximate size
    1, 
    'system'
);
