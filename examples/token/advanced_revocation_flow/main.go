// Example: RFC111 advanced revocation scenario with multi-attestation and chained delegation.
//
// This example demonstrates:
//   - Token issuance with multi-attestation
//   - Delegation to multiple agents
//   - Selective revocation (revoke one delegate, others remain valid)
//   - Compliance check for all tokens
//
// Run with: go run ./examples/token/advanced_revocation_flow/main.go
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

	// 1. Token issuance with multi-attestation
	ownerID := "user-300"
	attestations := []map[string]interface{}{
		{"attester": "notary-1", "date": time.Now()},
		{"attester": "supervisor-2", "date": time.Now()},
	}
	tok := &token.Token{
		ID:        "token-300",
		Value:     "secure-token-value",
		Type:      token.Access,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Issuer:    "authz-server",
		Subject:   ownerID,
		Scopes:    []string{"read", "write"},
		Metadata: &token.Metadata{
			AppData: map[string]string{
				"attestation_count": fmt.Sprintf("%d", len(attestations)),
			},
		},
	}
	if err := store.Save(ctx, tok.ID, tok); err != nil {
		fmt.Println("Save error:", err)
		return
	}
	fmt.Println("Token granted with multi-attestation.")
	for _, att := range attestations {
		fmt.Printf("Attestation: %v\n", att)
	}

	// 2. Delegation to multiple agents
	delegates := []string{"agent-1", "agent-2", "agent-3"}
	delegateTokens := make([]*token.Token, len(delegates))
	for i, agent := range delegates {
		delegateTokens[i] = &token.Token{
			ID:        fmt.Sprintf("token-300-delegate-%d", i+1),
			Value:     fmt.Sprintf("delegate-token-value-%d", i+1),
			Type:      token.Access,
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Issuer:    tok.Issuer,
			Subject:   agent,
			Scopes:    tok.Scopes,
			Metadata: &token.Metadata{
				AppData: map[string]string{
					"delegated_by": ownerID,
				},
			},
		}
		if err := store.Save(ctx, delegateTokens[i].ID, delegateTokens[i]); err != nil {
			fmt.Printf("Delegate save error for %s: %v\n", agent, err)
			return
		}
		fmt.Printf("Token delegated to %s.\n", agent)
	}

	// 3. Selective revocation (revoke agent-2's token)
	revokeIdx := 1 // agent-2
	if err := store.Revoke(ctx, delegateTokens[revokeIdx]); err != nil {
		fmt.Printf("Revoke error for %s: %v\n", delegates[revokeIdx], err)
		return
	}
	fmt.Printf("Delegated token for %s revoked.\n", delegates[revokeIdx])

	// 4. Compliance check for all tokens
	for i, agent := range delegates {
		fetched, err := store.Get(ctx, delegateTokens[i].ID)
		if err == nil && fetched == nil {
			fmt.Printf("Token for %s revoked: %v\n", agent, true)
		} else if err != nil {
			fmt.Printf("Get after revoke error for %s: %v\n", agent, err)
		} else {
			fmt.Printf("Token for %s valid: %v\n", agent, fetched != nil)
		}
	}
}
