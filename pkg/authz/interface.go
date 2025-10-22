package authz

import "context"

// Authorizer defines the minimal interface required by components needing authorization decisions.
// It is intentionally small to allow alternate implementations (e.g., remote PDP, policy engine adapters)
// without pulling in the full MemoryAuthorizer surface.
type Authorizer interface {
	Authorize(ctx context.Context, request Request) (Decision, error)
}
