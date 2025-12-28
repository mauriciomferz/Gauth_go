package sync

import (
	"context"

	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// PolicySource defines the interface for fetching policy updates.
type PolicySource interface {
	// Fetch returns the latest set of policies and a version string (e.g., hash or timestamp).
	// If the source hasn't changed, it may return the same version.
	// Errors during fetch should be returned.
	Fetch(ctx context.Context) ([]authz.Policy, string, error)
}

// PolicyAuthorizer defines the subset of authorizer interface required for syncing.
type PolicyAuthorizer interface {
	ReplacePolicies(ctx context.Context, policies []authz.Policy) error
}
