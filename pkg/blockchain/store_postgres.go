package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mauriciomferz/Gauth_go/pkg/database"
)

// PostgresPoAStore implements PoAStore using PostgreSQL
type PostgresPoAStore struct {
	db *database.DB
}

// NewPostgresPoAStore creates a new PostgresPoAStore
func NewPostgresPoAStore(db *database.DB) *PostgresPoAStore {
	return &PostgresPoAStore{
		db: db,
	}
}

// GetPoA retrieves a PoA by ID
func (s *PostgresPoAStore) GetPoA(ctx context.Context, poaID string) (*EnhancedPoA, error) {
	query := `
		SELECT 
			id, grantor_id, representative_id, 
			scope_type, actions, geographic_regions, 
			status, valid_from, valid_until, 
			revoked_at, revoked_by, revocation_reason,
			created_at, metadata
		FROM poa_records
		WHERE id = $1
	`

	var poa EnhancedPoA
	var id, grantorID, representativeID, scopeType, status, metadataJSON string
	var actions, regions []string
	var validFrom, validUntil, createdAt time.Time
	var revokedAt *time.Time
	var revokedBy, revocationReason *string
	// Note: using generic map for structured_scope reconstruction
	// In a real impl, we'd map scope_type/actions to StructuredScope

	err := s.db.Pool.QueryRow(ctx, query, poaID).Scan(
		&id, &grantorID, &representativeID,
		&scopeType, &actions, &regions,
		&status, &validFrom, &validUntil,
		&revokedAt, &revokedBy, &revocationReason,
		&createdAt, &metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("PoA not found: %s", poaID)
		}
		return nil, fmt.Errorf("failed to get PoA: %w", err)
	}

	// Unmarshal metadata to check for blockchain info
	var metadata map[string]interface{}
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// reconstruct EnhancedPoA
	poa = EnhancedPoA{
		ID:               id,
		IssuerID:         grantorID,
		GranteeID:        representativeID,
		Status:           status,
		ValidFrom:        validFrom,
		ValidUntil:       validUntil,
		CreatedAt:        createdAt,
		RevokedAt:        revokedAt,
		RevokedBy:        revokedBy,
		RevocationReason: revocationReason,
		StructuredScope: map[string]interface{}{
			"type":    scopeType,
			"actions": actions,
			"regions": regions,
		},
		// For now we mock these as they aren't strictly strictly in the table schema as distinct columns
		// or are inside metadata/JSONB
		Restrictions:  map[string]interface{}{},
		Attestations:  map[string]interface{}{},
		VersionNumber: 1,
	}

	if txHash, ok := metadata["blockchain_tx_hash"].(string); ok {
		poa.BlockchainTxHash = &txHash
	}
	if blockNum, ok := metadata["blockchain_block"].(float64); ok { // JSON numbers are floats
		block := int64(blockNum)
		poa.BlockchainBlock = &block
	}

	return &poa, nil
}

// GetPoAsByStatus retrieves all PoAs with a specific status
func (s *PostgresPoAStore) GetPoAsByStatus(ctx context.Context, status string) ([]*EnhancedPoA, error) {
	// Simplified implementation for example purposes
	return nil, fmt.Errorf("not implemented")
}

// UpdatePoABlockchainInfo updates the blockchain transaction info for a PoA
func (s *PostgresPoAStore) UpdatePoABlockchainInfo(ctx context.Context, poaID string, txHash string, blockNumber int64) error {
	// We store blockchain info in the metadata JSONB column to avoid schema changes
	query := `
		UPDATE poa_records
		SET metadata = jsonb_set(
			jsonb_set(metadata, '{blockchain_tx_hash}', to_jsonb($2::text)),
			'{blockchain_block}', to_jsonb($3::bigint)
		),
		updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.Pool.Exec(ctx, query, poaID, txHash, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to update blockchain info: %w", err)
	}

	return nil
}

// GetUnsyncedPoAs retrieves PoAs that haven't been synced to blockchain yet
func (s *PostgresPoAStore) GetUnsyncedPoAs(ctx context.Context, limit int) ([]*EnhancedPoA, error) {
	// Find active/revoked PoAs that don't have a blockchain_tx_hash in metadata
	query := `
		SELECT 
			id, grantor_id, representative_id, 
			scope_type, actions, geographic_regions, 
			status, valid_from, valid_until, 
			revoked_at, revoked_by, revocation_reason,
			created_at, metadata
		FROM poa_records
		WHERE (status = 'active' OR status = 'revoked')
		AND (metadata->>'blockchain_tx_hash') IS NULL
		LIMIT $1
	`

	rows, err := s.db.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unsynced PoAs: %w", err)
	}
	defer rows.Close()

	var results []*EnhancedPoA

	for rows.Next() {
		var id, grantorID, representativeID, scopeType, status, metadataJSON string
		var actions, regions []string
		var validFrom, validUntil, createdAt time.Time
		var revokedAt *time.Time
		var revokedBy, revocationReason *string

		if err := rows.Scan(
			&id, &grantorID, &representativeID,
			&scopeType, &actions, &regions,
			&status, &validFrom, &validUntil,
			&revokedAt, &revokedBy, &revocationReason,
			&createdAt, &metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		poa := &EnhancedPoA{
			ID:               id,
			IssuerID:         grantorID,
			GranteeID:        representativeID,
			Status:           status,
			ValidFrom:        validFrom,
			ValidUntil:       validUntil,
			CreatedAt:        createdAt,
			RevokedAt:        revokedAt,
			RevokedBy:        revokedBy,
			RevocationReason: revocationReason,
			StructuredScope: map[string]interface{}{
				"type":    scopeType,
				"actions": actions,
				"regions": regions,
			},
			VersionNumber: 1,
		}
		results = append(results, poa)
	}

	return results, nil
}
