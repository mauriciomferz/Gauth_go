package poa

import (
	"context"
	"encoding/json"
	"fmt"
)

// MultiSigCoordinator handles the coordination of multi-signature collection.
// It relies on the repository implementation for concurrency safety (DB locking).
type MultiSigCoordinator struct {
	repo PoARepository
}

// NewMultiSigCoordinator creates a new coordinator.
func NewMultiSigCoordinator(repo PoARepository) *MultiSigCoordinator {
	return &MultiSigCoordinator{
		repo: repo,
	}
}

// CollectSignature adds a partial signature to a pending PoA and transitions it to active if threshold is met.
// The `signature` argument can be any serializable map/struct representing the signature (e.g. from AAP-001).
func (c *MultiSigCoordinator) CollectSignature(ctx context.Context, tenantID, poaID, signerID string, signature interface{}, threshold int) (*PoARecord, error) {
	if c.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	// Normalize signature to map[string]interface{} for storage
	var sigMap map[string]interface{}

	// If it's already a map, use it. Otherwise, marshal/unmarshal to normalize.
	if m, ok := signature.(map[string]interface{}); ok {
		sigMap = m
	} else {
		b, err := json.Marshal(signature)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal signature: %w", err)
		}
		if err := json.Unmarshal(b, &sigMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal signature to map: %w", err)
		}
	}

	return c.repo.AddMultiSignature(ctx, tenantID, poaID, signerID, sigMap, threshold)
}
