// Example: Advanced RFC111 protocol flow with multi-attestation and chained delegation.
//
// This example demonstrates:
//   - Owner proof
//   - Grant (token issuance)
//   - Multiple attestations (e.g., notary, supervisor)
//   - Chained delegation (token delegated to another subject)
//   - Revocation and compliance check
//
// Run with: go run ./examples/token/advanced_rfc111_flow/main.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

func main() {
	ctx := context.Background()
	store := token.NewMemoryStore(time.Hour)

	// 1. Owner proof
	ownerID := "user-100"
	fmt.Printf("Owner proof for %s: email-verification\n", ownerID)

	// 2. Grant (token issuance)
	tok := &token.Token{
		ID:        "token-100",
		Value:     "secure-token-value",
		Type:      token.Access,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Issuer:    "authz-server",
		Subject:   ownerID,
		Scopes:    []string{"read", "write"},
		Metadata: &token.Metadata{
			AppData: map[string]string{
				"grant_type": "owner_proof",
			},
		},
	}
	if err := store.Save(ctx, tok.ID, tok); err != nil {
		fmt.Println("Save error:", err)
		return
	}
	fmt.Println("Token granted.")

	// 3. Multi-attestation (notary, supervisor)
	attestations := []map[string]interface{}{
		{"attester": "notary-abc", "date": time.Now()},
		{"attester": "supervisor-xyz", "date": time.Now()},
	}
	for _, att := range attestations {
		fmt.Printf("Attestation: %v\n", att)
	}

	// 4. Chained delegation (token delegated to another subject)
	delegatedToken := *tok // shallow copy
	delegatedToken.ID = "token-100-delegated"
	delegatedToken.Subject = "user-200"
	delegatedToken.Metadata.AppData["delegated_by"] = ownerID
	if err := store.Save(ctx, delegatedToken.ID, &delegatedToken); err != nil {
		fmt.Println("Delegation save error:", err)
		return
	}
	fmt.Printf("Token delegated to %s.\n", delegatedToken.Subject)

	// 5. Validation (delegated token should be valid)
	fetched, err := store.Get(ctx, delegatedToken.ID)
	if err != nil {
		fmt.Println("Get error:", err)
		return
	}
	fmt.Printf("Delegated token valid: %v\n", fetched != nil)

	// 6. Revocation (revoke delegated token)
	if err := store.Revoke(ctx, &delegatedToken); err != nil {
		fmt.Println("Revoke error:", err)
		return
	}
	fmt.Println("Delegated token revoked.")

	// 7. Compliance check (delegated token should not be valid)
	fetched, err = store.Get(ctx, delegatedToken.ID)
	if err == nil && fetched == nil {
		fmt.Printf("Delegated token revoked: %v\n", true)
	} else if err != nil {
		fmt.Println("Get after revoke error:", err)
	}
}
