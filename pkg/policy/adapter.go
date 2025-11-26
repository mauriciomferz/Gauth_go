package policy

import (
	"context"
	"time"

	authz "github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// AuthorizerAdapter bridges the new chain-based policy engine to the legacy simple Authorizer interface.
// It evaluates using deny-overrides semantics and returns a Decision compatible with existing code.
type AuthorizerAdapter struct {
	engine *ChainEngine
	// attributeProviders allows injecting dynamic attributes keyed by request (optional future use)
}

func NewAuthorizerAdapter(e *ChainEngine) *AuthorizerAdapter { return &AuthorizerAdapter{engine: e} }

// Authorize implements authz.Authorizer by translating the request into an EvalRequest.
func (a *AuthorizerAdapter) Authorize(ctx context.Context, r authz.Request) (authz.Decision, error) {
	attrs := map[string]string{}
	for k, v := range r.Context {
		attrs[k] = v
	}
	evalReq := EvalRequest{
		Subject:  r.Subject,
		Action:   r.Action,
		Resource: r.Resource,
		Attrs:    attrs,
		Now:      time.Now().UTC(),
	}
	dec, err := a.engine.Evaluate(ctx, evalReq)
	if err != nil {
		return authz.Decision{}, err
	}
	// Map to legacy decision type (Allow field; Reason string into Reason; Matched policies into Policies slice)
	return authz.Decision{Allow: dec.Allow, Allowed: dec.Allow, Policies: dec.Matched, Reason: dec.Reason, Metadata: map[string]string{"bundle_hash": dec.BundleHash, "chain_head": dec.ChainHead}}, nil
}
