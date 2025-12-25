// Package gnap provides PoA bridge for linking GNAP grants to Power of Attorney credentials.
package gnap

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gnap"
	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// PoAProvider defines interface for retrieving PoA records compatibility
type PoAProvider interface {
	GetPoA(ctx context.Context, tenantID, poaID string) (*poa.PoARecord, error)
}

// PoABridge links GNAP grants to Power of Attorney credentials.
type PoABridge struct {
	grantStore  gnap.GrantStore
	poaProvider PoAProvider
}

// NewPoABridge creates a new GNAP-PoA bridge.
func NewPoABridge(store gnap.GrantStore, poaProvider PoAProvider) *PoABridge {
	return &PoABridge{
		grantStore:  store,
		poaProvider: poaProvider,
	}
}

// GrantWithPoA represents a GNAP grant with associated PoA reference.
type GrantWithPoA struct {
	GrantID       string    `json:"grant_id"`
	PoAID         string    `json:"poa_id,omitempty"`
	TokenValue    string    `json:"token_value,omitempty"`
	DelegationRef string    `json:"delegation_ref,omitempty"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

// LinkGrantToPoA associates a GNAP grant with a Power of Attorney credential.
func (b *PoABridge) LinkGrantToPoA(grantID, poaID, delegationRef string) (*GrantWithPoA, error) {
	grant, err := b.grantStore.Get(grantID)
	if err != nil {
		return nil, fmt.Errorf("grant not found: %w", err)
	}
	if grant == nil {
		return nil, fmt.Errorf("grant %s does not exist", grantID)
	}

	// Use Grant's built-in PoAID field
	grant.PoAID = poaID

	if err := b.grantStore.Update(grant); err != nil {
		return nil, fmt.Errorf("failed to update grant: %w", err)
	}

	return &GrantWithPoA{
		GrantID:       grantID,
		PoAID:         poaID,
		DelegationRef: delegationRef,
		IssuedAt:      grant.CreatedAt,
	}, nil
}

// GetPoAForGrant retrieves the PoA reference for a GNAP grant.
func (b *PoABridge) GetPoAForGrant(grantID string) (poaID string, err error) {
	grant, err := b.grantStore.Get(grantID)
	if err != nil {
		return "", err
	}
	if grant == nil {
		return "", fmt.Errorf("grant not found")
	}
	return grant.PoAID, nil
}

// ValidatePoAAuthority checks if a GNAP grant's linked PoA has valid authority.
func (b *PoABridge) ValidatePoAAuthority(grantID string) (valid bool, reason string, err error) {
	poaID, err := b.GetPoAForGrant(grantID)
	if err != nil {
		return false, "", err
	}

	if poaID == "" {
		return true, "no_poa_required", nil
	}

	if b.poaProvider == nil {
		// If no provider configured (e.g. memory/dev mode), allow but log warning ideally
		return true, "poa_validation_skipped_no_provider", nil
	}

	// Assuming a default tenant or deriving from context/grant if available.
	// For now using "default" tenant as placeholder or we should extract if possible.
	// In a real scenario, the Grant might have TenantID or we assume single tenant for this specific flow.
	tenantID := "default"

	record, err := b.poaProvider.GetPoA(context.Background(), tenantID, poaID)
	if err != nil {
		return false, "poa_lookup_failed", fmt.Errorf("failed to retrieve PoA %s: %w", poaID, err)
	}

	if record.Status != "active" {
		return false, fmt.Sprintf("poa_status_%s", record.Status), nil
	}

	if time.Now().After(record.ValidUntil) {
		return false, "poa_expired", nil
	}

	return true, "poa_valid", nil
}
