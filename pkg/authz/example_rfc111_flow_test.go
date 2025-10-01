//go:build example
// +build example

// ExampleRFC111Flow demonstrates a full GAuth protocol flow as described in RFC111:
// - Owner/authorizer proof
// - Grant issuance
// - Attestation
// - Token issuance
// - Revocation
// - Compliance/audit
//
// This is a simplified, illustrative example for contributors and reviewers.
package authz_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/authz"
)

// ExampleRFC111Flow demonstrates a full GAuth protocol flow as described in RFC111:
// - Owner/authorizer proof
// - Grant issuance
// - Attestation
// - Token issuance
// - Revocation
// - Compliance/audit
//
// This is a simplified, illustrative example for contributors and reviewers.
func ExampleRFC111Flow() {
	ctx := context.Background()

	// 1. Owner/authorizer proof (simulated)
	owner := authz.Subject{ID: "owner1", Type: "human"}
	ai := authz.Subject{ID: "ai-agent-1", Type: "ai"}
	resource := authz.Resource{ID: "doc-42", Type: "document", Owner: owner.ID}
	action := authz.Action{Name: "sign"}

	// 2. Grant issuance (policy creation)
	policy := &authz.Policy{
		ID:        "grant-1",
		Version:   "v1",
		Name:      "AI Power of Attorney",
		Effect:    authz.Allow,
		Subjects:  []authz.Subject{ai},
		Resources: []authz.Resource{resource},
		Actions:   []authz.Action{action},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	authorizer := authz.NewMemoryAuthorizer()
	err := authorizer.AddPolicy(ctx, policy)
	if err != nil {
		fmt.Println("AddPolicy error:", err)
		return
	}

	// 3. Attestation (simulated, e.g. notary/witness)
	// In a real system, this would be a cryptographic or external attestation
	attestation := map[string]interface{}{
		"witness": "notary-123",
		"date":    time.Now(),
	}

	// 4. Token issuance (access request)
	request := &authz.AccessRequest{
		Subject:  ai,
		Resource: resource,
		Action:   action,
		Context:  map[string]interface{}{"attestation": attestation},
	}
	decision, err := authorizer.Authorize(ctx, ai, action, resource)
	if err != nil {
		fmt.Println("Authorize error:", err)
		return
	}
	fmt.Printf("Allowed: %v, Reason: %s\n", decision.Allowed, decision.Reason)

	// 5. Revocation (policy removal)
	err = authorizer.RemovePolicy(ctx, policy.ID)
	if err != nil {
		fmt.Println("RemovePolicy error:", err)
		return
	}

	// 6. Compliance/audit (decision after revocation)
	decision, err = authorizer.Authorize(ctx, ai, action, resource)
	if err != nil {
		fmt.Println("Authorize after revocation error:", err)
		return
	}
	fmt.Printf("Allowed after revocation: %v\n", decision.Allowed)

	// Output:
	// Allowed: true, Reason:
	// Allowed after revocation: false
}
