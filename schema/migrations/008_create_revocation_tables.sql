-- Migration: Create Revocation tables
-- Covers: Merkle Tree Nodes, Proofs, and Revocations
-- Version: 008
-- Created: 2025-12-21

-- ============================================================================
-- MERKLE TREE NODES
-- ============================================================================

CREATE TABLE IF NOT EXISTS merkle_tree_nodes (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    tree_version INTEGER NOT NULL,
    node_hash VARCHAR(64) NOT NULL,
    level INTEGER NOT NULL,
    position INTEGER NOT NULL,
    is_leaf BOOLEAN NOT NULL DEFAULT FALSE,
    left_child_hash VARCHAR(64),
    right_child_hash VARCHAR(64),
    parent_hash VARCHAR(64),
    token_id VARCHAR(255),
    leaf_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_merkle_tree ON merkle_tree_nodes(tenant_id, tree_version);
CREATE INDEX idx_merkle_node_hash ON merkle_tree_nodes(node_hash);

-- ============================================================================
-- MERKLE PROOFS
-- ============================================================================

CREATE TABLE IF NOT EXISTS merkle_proofs (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    proof_id VARCHAR(255) NOT NULL UNIQUE,
    token_id VARCHAR(255) NOT NULL,
    tree_version INTEGER NOT NULL,
    leaf_hash VARCHAR(64) NOT NULL,
    root_hash VARCHAR(64) NOT NULL,
    proof_path JSONB NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_merkle_proofs_token ON merkle_proofs(tenant_id, token_id);

-- ============================================================================
-- REVOCATIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS revocations (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::varchar,
    tenant_id VARCHAR(255) NOT NULL,
    token_id VARCHAR(255) NOT NULL,
    revocation_reason VARCHAR(255),
    leaf_hash VARCHAR(64) NOT NULL,
    merkle_root VARCHAR(64) NOT NULL,
    block_height INTEGER,
    tree_version INTEGER NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    revoked_by VARCHAR(255),
    metadata JSONB,
    revoked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_revocations_token ON revocations(tenant_id, token_id);
CREATE INDEX idx_revocations_date ON revocations(revoked_at DESC);
