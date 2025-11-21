package authz

import "context"

// Authorizer defines the minimal interface required by components needing authorization decisions.
// It is intentionally small to allow alternate implementations (e.g., remote PDP, policy engine adapters)
// without pulling in the full MemoryAuthorizer surface.
//
// GetPermissions is included for security-critical operations like delegation scope validation (VULN-01 mitigation)
// where the system needs to verify that a subject actually possesses the permissions they're trying to delegate.
type Authorizer interface {
	Authorize(ctx context.Context, request Request) (Decision, error)
	GetPermissions(ctx context.Context, subject string) ([]Permission, error)
}
