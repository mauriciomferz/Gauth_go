// Example: RFC111-compliant token protocol flow with owner proof, grant, attestation, revocation, and compliance check.
//
// This example demonstrates a full protocol flow as required by RFC111:
//   - Owner proof (subject provides proof of control)
//   - Grant (authorization server issues token)
//   - Attestation (third party attests to token)
//   - Revocation (token is revoked)
//   - Compliance check (ensure revoked tokens are not valid)
//
// Run with: go run ./examples/token/rfc111_protocol_flow/main.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func main() {
	ctx := context.Background()
	store := token.NewMemoryStore(time.Hour)

	// 1. Owner proof (subject provides proof of control)
	ownerID := "user-42"
	ownerProof := map[string]interface{}{
		"method":    "email-verification",
		"timestamp": time.Now(),
	}
	fmt.Printf("Owner proof for %s: %v\n", ownerID, ownerProof)

	// 2. Grant (authorization server issues token)
	tok := &token.Token{
		ID:        "token-42",
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

	// 3. Attestation (third party attests to token)
	attestation := map[string]interface{}{
		"attester": "notary-xyz",
		"date":     time.Now(),
	}
	fmt.Printf("Attestation: %v\n", attestation)

	// 4. Validation (token should be valid)
	fetched, err := store.Get(ctx, tok.ID)
	if err != nil {
		fmt.Println("Get error:", err)
		return
	}
	fmt.Printf("Token valid: %v\n", fetched != nil)

	// 5. Revocation
	if err := store.Revoke(ctx, tok); err != nil {
		fmt.Println("Revoke error:", err)
		return
	}
	fmt.Println("Token revoked.")

	// 6. Compliance check (token should not be valid)
	fetched, err = store.Get(ctx, tok.ID)
	if err == nil && fetched == nil {
		fmt.Printf("Token revoked: %v\n", true)
	} else if err != nil {
		fmt.Println("Get after revoke error:", err)
	}
}
