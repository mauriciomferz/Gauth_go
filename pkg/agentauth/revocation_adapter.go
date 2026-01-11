package agentauth

import (
	"context"
	"fmt"

	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// DelegationRevocationAdapter adapts a delegation.RevocationChain to the RevocationChecker interface.
// It bridges the gap between the low-level blockchain-style revocation log and the high-level validator.
type DelegationRevocationAdapter struct {
	chain *delegation.RevocationChain
}

// NewDelegationRevocationAdapter creates a new adapter instance.
func NewDelegationRevocationAdapter(chain *delegation.RevocationChain) *DelegationRevocationAdapter {
	return &DelegationRevocationAdapter{
		chain: chain,
	}
}

// IsRevoked checks if an entity's authorization has been revoked.
// Currently returns false as RevocationChain tracks delegations, not entity identities directly.
// Entity revocation would be handled by a different mechanism (e.g. CRL/OCSP adapter).
func (a *DelegationRevocationAdapter) IsRevoked(ctx context.Context, entityID string) (bool, error) {
	// TODO: Integate with entity revocation list if needed.
	// For now, we assume entity identity validity is checked via TrustServiceProvider.
	return false, nil
}

// IsDelegationRevoked checks if a specific delegation ID has been revoked.
func (a *DelegationRevocationAdapter) IsDelegationRevoked(ctx context.Context, delegationID string) (bool, error) {
	if a.chain == nil {
		return false, fmt.Errorf("revocation chain not configured")
	}
	// The chain's IsDelegationRevoked is synchronous and in-memory (fast).
	// In a distributed setup, this would query a localized read-replica or cache.
	return a.chain.IsDelegationRevoked(delegationID, ""), nil
}

// GetRevocationInfo retrieves detailed revocation information.
// Not fully implemented for prototype.
func (a *DelegationRevocationAdapter) GetRevocationInfo(ctx context.Context, entityID string) (*RevocationInfo, error) {
	return &RevocationInfo{
		EntityID: entityID,
		Revoked:  false,
	}, nil
}

// CheckCertificateRevocation checks certificate revocation status.
// Not implemented for this adapter.
func (a *DelegationRevocationAdapter) CheckCertificateRevocation(
	ctx context.Context, certID string,
) (*CertificateRevocationStatus, error) {
	return &CertificateRevocationStatus{
		CertificateID: certID,
		Revoked:       false,
		CheckMethod:   "none",
	}, nil
}
