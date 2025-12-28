package pdp

import (
	"context"

	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// AuthzAdapter wraps an authz.Authorizer to satisfy the pdp.Engine interface
type AuthzAdapter struct {
	authorizer authz.Authorizer
}

// NewAuthzAdapter creates a new adapter for the given authorizer
func NewAuthzAdapter(a authz.Authorizer) *AuthzAdapter {
	return &AuthzAdapter{
		authorizer: a,
	}
}

// Evaluate implements the Engine interface
func (a *AuthzAdapter) Evaluate(ctx context.Context, req Request) (Decision, error) {
	// Map pdp.Request to authz.Request
	authzReq := authz.Request{
		Subject:  req.Subject,
		Action:   req.Action,
		Resource: req.Resource,
		Context:  req.Attributes,
	}

	// Call the authorizer
	authzDec, err := a.authorizer.Authorize(ctx, authzReq)
	if err != nil {
		return Decision{}, err
	}

	// Map authz.Decision back to pdp.Decision
	return Decision{
		Allow:    authzDec.Allow,
		Reason:   authzDec.Reason,
		Policies: authzDec.Policies,
		Metadata: authzDec.Metadata,
	}, nil
}

// Metrics implements the Engine interface
func (a *AuthzAdapter) Metrics() MetricsSnapshot {
	// Return empty metrics if the authorizer snapshot is not easily mappable
	// or implement mapping if needed.
	return MetricsSnapshot{
		PolicyMatches: make(map[string]uint64),
	}
}
