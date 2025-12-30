package authz

import (
	"context"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// AuthorizerAdapter bridges the new chain-based policy engine to the legacy simple Authorizer interface.
// It evaluates using deny-overrides semantics and returns a Decision compatible with existing code.
type AuthorizerAdapter struct {
	engine *policy.ChainEngine
	// attributeProviders allows injecting dynamic attributes keyed by request (optional future use)
}

func NewAuthorizerAdapter(e *policy.ChainEngine) *AuthorizerAdapter {
	return &AuthorizerAdapter{engine: e}
}

// Authorize implements Authorizer by translating the request into an EvalRequest.
func (a *AuthorizerAdapter) Authorize(ctx context.Context, r Request) (Decision, error) {
	attrs := map[string]string{}
	for k, v := range r.Context {
		attrs[k] = v
	}
	evalReq := policy.EvalRequest{
		Subject:  r.Subject,
		Action:   r.Action,
		Resource: r.Resource,
		Attrs:    attrs,
		Now:      time.Now().UTC(),
	}
	dec, err := a.engine.Evaluate(ctx, evalReq)
	if err != nil {
		return Decision{}, err
	}
	// Map to legacy decision type (Allow field; Reason string into Reason; Matched policies into Policies slice)
	return Decision{Allow: dec.Allow, Allowed: dec.Allow, Policies: dec.Matched, Reason: dec.Reason, Metadata: map[string]string{"bundle_hash": dec.BundleHash, "chain_head": dec.ChainHead}}, nil
}
