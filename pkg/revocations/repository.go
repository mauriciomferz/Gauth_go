package revocations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles revocation database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new revocation repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	ID             string
	TenantID       string
	TreeVersion    int
	NodeHash       string
	Level          int
	Position       int
	IsLeaf         bool
	LeftChildHash  *string
	RightChildHash *string
	ParentHash     *string
	TokenID        *string
	LeafData       map[string]interface{}
	CreatedAt      time.Time
}

// MerkleProof represents a Merkle proof
type MerkleProof struct {
	ID          string
	TenantID    string
	ProofID     string
	TokenID     string
	TreeVersion int
	LeafHash    string
	RootHash    string
	ProofPath   []map[string]interface{}
	Verified    bool
	VerifiedAt  *time.Time
	CreatedAt   time.Time
}

// Revocation represents a token revocation entry
type Revocation struct {
	ID               string
	TenantID         string
	TokenID          string
	RevocationReason string
	LeafHash         string
	MerkleRoot       string
	BlockHeight      int
	TreeVersion      int
	Verified         bool
	RevokedAt        time.Time
	RevokedBy        string
	Metadata         map[string]interface{}
}

// RevocationStats holds aggregated statistics
type RevocationStats struct {
	TotalRevocations    int
	VerifiedRevocations int
	RevocationsLast24h  int
	RevocationsLast7d   int
	LatestBlockHeight   int
	LatestTreeVersion   int
	VerificationRate    float64
}

// CreateMerkleNode inserts a new Merkle tree node
func (r *Repository) CreateMerkleNode(ctx context.Context, node *MerkleNode) error {
	query := `
		INSERT INTO merkle_tree_nodes (
			tenant_id, tree_version, node_hash, level, position,
			is_leaf, left_child_hash, right_child_hash, parent_hash,
			token_id, leaf_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		node.TenantID, node.TreeVersion, node.NodeHash, node.Level, node.Position,
		node.IsLeaf, node.LeftChildHash, node.RightChildHash, node.ParentHash,
		node.TokenID, node.LeafData,
	).Scan(&node.ID, &node.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create merkle node: %w", err)
	}

	return nil
}

// GetMerkleTree retrieves all nodes for a specific tree version
func (r *Repository) GetMerkleTree(ctx context.Context, tenantID string, treeVersion int) ([]MerkleNode, error) {
	if r.db == nil {
		return []MerkleNode{}, nil
	}
	query := `
		SELECT 
			id, tenant_id, tree_version, node_hash, level, position,
			is_leaf, left_child_hash, right_child_hash, parent_hash,
			token_id, leaf_data, created_at
		FROM merkle_tree_nodes
		WHERE tenant_id = $1 AND tree_version = $2
		ORDER BY level ASC, position ASC`

	rows, err := r.db.Query(ctx, query, tenantID, treeVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get merkle tree: %w", err)
	}
	defer rows.Close()

	nodes := make([]MerkleNode, 0)
	for rows.Next() {
		var node MerkleNode
		err := rows.Scan(
			&node.ID, &node.TenantID, &node.TreeVersion, &node.NodeHash, &node.Level, &node.Position,
			&node.IsLeaf, &node.LeftChildHash, &node.RightChildHash, &node.ParentHash,
			&node.TokenID, &node.LeafData, &node.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan merkle node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetLatestTreeVersion retrieves the latest tree version for a tenant
func (r *Repository) GetLatestTreeVersion(ctx context.Context, tenantID string) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	query := `
		SELECT COALESCE(MAX(tree_version), 0)
		FROM merkle_tree_nodes
		WHERE tenant_id = $1`

	var version int
	err := r.db.QueryRow(ctx, query, tenantID).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest tree version: %w", err)
	}

	return version, nil
}

// CreateMerkleProof inserts a new Merkle proof
func (r *Repository) CreateMerkleProof(ctx context.Context, proof *MerkleProof) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		INSERT INTO merkle_proofs (
			tenant_id, proof_id, token_id, tree_version,
			leaf_hash, root_hash, proof_path, verified, verified_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		proof.TenantID, proof.ProofID, proof.TokenID, proof.TreeVersion,
		proof.LeafHash, proof.RootHash, proof.ProofPath, proof.Verified, proof.VerifiedAt,
	).Scan(&proof.ID, &proof.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create merkle proof: %w", err)
	}

	return nil
}

// GetMerkleProof retrieves a proof by token ID
func (r *Repository) GetMerkleProof(ctx context.Context, tenantID, tokenID string, treeVersion int) (*MerkleProof, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	query := `
		SELECT 
			id, tenant_id, proof_id, token_id, tree_version,
			leaf_hash, root_hash, proof_path, verified, verified_at, created_at
		FROM merkle_proofs
		WHERE tenant_id = $1 AND token_id = $2 AND tree_version = $3`

	var proof MerkleProof
	err := r.db.QueryRow(ctx, query, tenantID, tokenID, treeVersion).Scan(
		&proof.ID, &proof.TenantID, &proof.ProofID, &proof.TokenID, &proof.TreeVersion,
		&proof.LeafHash, &proof.RootHash, &proof.ProofPath, &proof.Verified, &proof.VerifiedAt, &proof.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get merkle proof: %w", err)
	}

	return &proof, nil
}

// ListMerkleProofs retrieves proofs with pagination
func (r *Repository) ListMerkleProofs(ctx context.Context, tenantID string, limit, offset int) ([]MerkleProof, int, error) {
	if r.db == nil {
		return []MerkleProof{}, 0, nil
	}
	// Count total
	countQuery := `SELECT COUNT(*) FROM merkle_proofs WHERE tenant_id = $1`
	var total int
	err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count proofs: %w", err)
	}

	// Get paginated results
	query := `
		SELECT 
			id, tenant_id, proof_id, token_id, tree_version,
			leaf_hash, root_hash, proof_path, verified, verified_at, created_at
		FROM merkle_proofs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list proofs: %w", err)
	}
	defer rows.Close()

	proofs := make([]MerkleProof, 0)
	for rows.Next() {
		var proof MerkleProof
		err := rows.Scan(
			&proof.ID, &proof.TenantID, &proof.ProofID, &proof.TokenID, &proof.TreeVersion,
			&proof.LeafHash, &proof.RootHash, &proof.ProofPath, &proof.Verified, &proof.VerifiedAt, &proof.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan proof: %w", err)
		}
		proofs = append(proofs, proof)
	}

	return proofs, total, nil
}

// UpdateProofVerification updates the verification status of a proof
func (r *Repository) UpdateProofVerification(ctx context.Context, proofID string, verified bool) error {
	if r.db == nil {
		return nil // No-op, or return an error if preferred: fmt.Errorf("database not available")
	}
	query := `
		UPDATE merkle_proofs
		SET verified = $1, verified_at = NOW()
		WHERE proof_id = $2`

	_, err := r.db.Exec(ctx, query, verified, proofID)
	if err != nil {
		return fmt.Errorf("failed to update proof verification: %w", err)
	}

	return nil
}

// CreateRevocation inserts a new revocation entry
func (r *Repository) CreateRevocation(ctx context.Context, revocation *Revocation) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}
	query := `
		INSERT INTO revocations (
			tenant_id, token_id, revocation_reason,
			leaf_hash, merkle_root, block_height, tree_version,
			verified, revoked_by, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, revoked_at`

	err := r.db.QueryRow(ctx, query,
		revocation.TenantID, revocation.TokenID, revocation.RevocationReason,
		revocation.LeafHash, revocation.MerkleRoot, revocation.BlockHeight, revocation.TreeVersion,
		revocation.Verified, revocation.RevokedBy, revocation.Metadata,
	).Scan(&revocation.ID, &revocation.RevokedAt)

	if err != nil {
		return fmt.Errorf("failed to create revocation: %w", err)
	}

	return nil
}

// ListRevocations retrieves revocations with pagination
func (r *Repository) ListRevocations(ctx context.Context, tenantID string, limit, offset int) ([]Revocation, int, error) {
	if r.db == nil {
		return []Revocation{}, 0, nil
	}
	// Count total
	countQuery := `SELECT COUNT(*) FROM revocations WHERE tenant_id = $1`
	var total int
	err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count revocations: %w", err)
	}

	// Get paginated results
	query := `
		SELECT 
			id, tenant_id, token_id, revocation_reason,
			leaf_hash, merkle_root, block_height, tree_version,
			verified, revoked_at, revoked_by, metadata
		FROM revocations
		WHERE tenant_id = $1
		ORDER BY revoked_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list revocations: %w", err)
	}
	defer rows.Close()

	revocations := make([]Revocation, 0)
	for rows.Next() {
		var rev Revocation
		err := rows.Scan(
			&rev.ID, &rev.TenantID, &rev.TokenID, &rev.RevocationReason,
			&rev.LeafHash, &rev.MerkleRoot, &rev.BlockHeight, &rev.TreeVersion,
			&rev.Verified, &rev.RevokedAt, &rev.RevokedBy, &rev.Metadata,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan revocation: %w", err)
		}
		revocations = append(revocations, rev)
	}

	return revocations, total, nil
}

// GetRevocation retrieves a single revocation by token ID
func (r *Repository) GetRevocation(ctx context.Context, tenantID, tokenID string) (*Revocation, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	query := `
		SELECT 
			id, tenant_id, token_id, revocation_reason,
			leaf_hash, merkle_root, block_height, tree_version,
			verified, revoked_at, revoked_by, metadata
		FROM revocations
		WHERE tenant_id = $1 AND token_id = $2
		ORDER BY revoked_at DESC
		LIMIT 1`

	var rev Revocation
	err := r.db.QueryRow(ctx, query, tenantID, tokenID).Scan(
		&rev.ID, &rev.TenantID, &rev.TokenID, &rev.RevocationReason,
		&rev.LeafHash, &rev.MerkleRoot, &rev.BlockHeight, &rev.TreeVersion,
		&rev.Verified, &rev.RevokedAt, &rev.RevokedBy, &rev.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get revocation: %w", err)
	}

	return &rev, nil
}

// GetRevocationStats retrieves aggregated statistics
func (r *Repository) GetRevocationStats(ctx context.Context, tenantID string) (*RevocationStats, error) {
	if r.db == nil {
		return &RevocationStats{
			TotalRevocations:    0,
			VerifiedRevocations: 0,
			RevocationsLast24h:  0,
			RevocationsLast7d:   0,
			LatestBlockHeight:   0,
			LatestTreeVersion:   0,
			VerificationRate:    0.0,
		}, nil
	}
	query := `
		SELECT 
			COUNT(*) AS total_revocations,
			COUNT(*) FILTER (WHERE verified = true) AS verified_revocations,
			COUNT(*) FILTER (WHERE revoked_at >= NOW() - INTERVAL '24 hours') AS revocations_last_24h,
			COUNT(*) FILTER (WHERE revoked_at >= NOW() - INTERVAL '7 days') AS revocations_last_7d,
			MAX(block_height) AS latest_block_height,
			MAX(tree_version) AS latest_tree_version
		FROM revocations
		WHERE tenant_id = $1`

	var stats RevocationStats
	var latestBlockHeight, latestTreeVersion sql.NullInt64

	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&stats.TotalRevocations,
		&stats.VerifiedRevocations,
		&stats.RevocationsLast24h,
		&stats.RevocationsLast7d,
		&latestBlockHeight,
		&latestTreeVersion,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get revocation stats: %w", err)
	}

	if latestBlockHeight.Valid {
		stats.LatestBlockHeight = int(latestBlockHeight.Int64)
	}
	if latestTreeVersion.Valid {
		stats.LatestTreeVersion = int(latestTreeVersion.Int64)
	}

	// Calculate verification rate
	if stats.TotalRevocations > 0 {
		stats.VerificationRate = float64(stats.VerifiedRevocations) / float64(stats.TotalRevocations)
	}

	return &stats, nil
}
