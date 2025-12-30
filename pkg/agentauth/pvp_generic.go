package agentauth

import (
	"context"
	"fmt"
	"time"
)

// GenericPVPStub is a placeholder for PVP providers that are not yet fully implemented.
// It returns a standard error explaining that the integration is pending.
type GenericPVPStub struct {
	providerName string
	docsURL      string
}

// NewGenericPVPStub creates a new generic stub
func NewGenericPVPStub(providerName, docsURL string) *GenericPVPStub {
	return &GenericPVPStub{
		providerName: providerName,
		docsURL:      docsURL,
	}
}

// VerifyIdentityProof returns a not implemented error
func (c *GenericPVPStub) VerifyIdentityProof(ctx context.Context, request *IdentityProofRequest) (*IdentityProofResult, error) {
	// Simulate a small network delay to be realistic even in failure
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}

	return nil, fmt.Errorf("%s PVP integration not yet implemented. Please refer to %s for API details", c.providerName, c.docsURL)
}
