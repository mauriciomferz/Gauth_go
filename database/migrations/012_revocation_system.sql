-- Migration: 012_revocation_system.sql
-- Purpose: Create tables for revocation transparency system with Merkle tree support
-- Date: 2025-11-28

-- Create enum for revocation reason
CREATE TYPE revocation_reason_enum AS ENUM (
    'user_request',
    'security_breach',
    'compliance_violation',
    'admin_revocation',
    'token_expiration',
    'other'
);

-- Merkle Tree Nodes Table
CREATE TABLE IF NOT EXISTS merkle_tree_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    tree_version INTEGER NOT NULL,
    node_hash VARCHAR(64) NOT NULL,
    level INTEGER NOT NULL,
    position INTEGER NOT NULL,
    is_leaf BOOLEAN NOT NULL DEFAULT FALSE,
    left_child_hash VARCHAR(64),
    right_child_hash VARCHAR(64),
    parent_hash VARCHAR(64),
    token_id TEXT,
    leaf_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(tenant_id, tree_version, level, position)
);

CREATE INDEX IF NOT EXISTS idx_merkle_tree_nodes_tenant_version 
    ON merkle_tree_nodes(tenant_id, tree_version);

CREATE INDEX IF NOT EXISTS idx_merkle_tree_nodes_token_id 
    ON merkle_tree_nodes(token_id);

-- Merkle Proofs Table
CREATE TABLE IF NOT EXISTS merkle_proofs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    proof_id TEXT NOT NULL,
    token_id TEXT NOT NULL,
    tree_version INTEGER NOT NULL,
    leaf_hash VARCHAR(64) NOT NULL,
    root_hash VARCHAR(64) NOT NULL,
    proof_path JSONB NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(tenant_id, proof_id),
    UNIQUE(tenant_id, token_id, tree_version)
);

CREATE INDEX IF NOT EXISTS idx_merkle_proofs_tenant_token 
    ON merkle_proofs(tenant_id, token_id);

CREATE INDEX IF NOT EXISTS idx_merkle_proofs_created_at 
    ON merkle_proofs(created_at);

-- Revocations Table
CREATE TABLE IF NOT EXISTS revocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    token_id TEXT NOT NULL,
    revocation_reason VARCHAR(255),
    leaf_hash VARCHAR(64),
    merkle_root VARCHAR(64),
    block_height INTEGER DEFAULT 0,
    tree_version INTEGER DEFAULT 0,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_by TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(tenant_id, token_id)
);

CREATE INDEX IF NOT EXISTS idx_revocations_tenant 
    ON revocations(tenant_id);

CREATE INDEX IF NOT EXISTS idx_revocations_token 
    ON revocations(token_id);

CREATE INDEX IF NOT EXISTS idx_revocations_revoked_at 
    ON revocations(revoked_at DESC);

CREATE INDEX IF NOT EXISTS idx_revocations_verified 
    ON revocations(verified);

-- Revocation Audit Log Table
CREATE TABLE IF NOT EXISTS revocation_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    operation TEXT NOT NULL, -- 'create', 'verify', 'delete', etc.
    entity_type TEXT NOT NULL, -- 'revocation', 'merkle_proof', 'merkle_tree'
    entity_id TEXT,
    changed_by TEXT,
    change_details JSONB,
    log_entry_hash VARCHAR(64),
    previous_hash VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_revocation_audit_log_tenant 
    ON revocation_audit_log(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_revocation_audit_log_entity 
    ON revocation_audit_log(entity_type);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_merkle_tree_nodes_leaf 
    ON merkle_tree_nodes(tenant_id, tree_version) 
    WHERE is_leaf = TRUE;

-- Trigger to automatically update updated_at on merkle_tree_nodes
CREATE OR REPLACE FUNCTION update_merkle_tree_nodes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_merkle_tree_nodes_updated_at ON merkle_tree_nodes;
CREATE TRIGGER trigger_merkle_tree_nodes_updated_at
    BEFORE UPDATE ON merkle_tree_nodes
    FOR EACH ROW
    EXECUTE FUNCTION update_merkle_tree_nodes_updated_at();

-- Trigger to automatically update updated_at on merkle_proofs
CREATE OR REPLACE FUNCTION update_merkle_proofs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_merkle_proofs_updated_at ON merkle_proofs;
CREATE TRIGGER trigger_merkle_proofs_updated_at
    BEFORE UPDATE ON merkle_proofs
    FOR EACH ROW
    EXECUTE FUNCTION update_merkle_proofs_updated_at();

-- Trigger to automatically update updated_at on revocations
CREATE OR REPLACE FUNCTION update_revocations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_revocations_updated_at ON revocations;
CREATE TRIGGER trigger_revocations_updated_at
    BEFORE UPDATE ON revocations
    FOR EACH ROW
    EXECUTE FUNCTION update_revocations_updated_at();
