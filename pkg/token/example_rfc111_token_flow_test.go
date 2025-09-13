package token

import (
	"context"
	"fmt"
	"time"
)

// ExampleRFC111TokenFlow demonstrates a full RFC111-compliant token lifecycle:
// - Token issuance (with power-of-attorney metadata)
// - Attestation (simulated)
// - Validation
// - Revocation
// - Compliance check after revocation
// ExampleRFC111TokenFlow demonstrates a full RFC111-compliant token lifecycle:
// - Token issuance (with power-of-attorney metadata)
// - Attestation (simulated)
// - Validation
// - Revocation
// - Compliance check after revocation
//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
//
// See the LICENSE file in the project root for details.
func Example_rfc111TokenFlow() {
	ctx := context.Background()

	// 1. Token issuance (with power-of-attorney metadata)
	tok := &Token{
		ID:        "token-1",
		Value:     "secure-token-value",
		Type:      Access,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Issuer:    "authz-server",
		Subject:   "ai-agent-1",
		Scopes:    []string{"sign", "read"},
		Metadata: &Metadata{
			AppData: map[string]string{
				"power_of_attorney": "AI Power of Attorney for doc-42",
			},
		},
	}
	store := NewMemoryStore(time.Hour)
	if err := store.Save(ctx, tok.ID, tok); err != nil {
		fmt.Println("Save error:", err)
		return
	}

	// 2. Attestation (simulated)
	_ = map[string]interface{}{
		"witness": "notary-123",
		"date":    time.Now(),
	}
	// In a real system, attestation would be cryptographically bound to the token

	// 3. Validation
	fetched, err := store.Get(ctx, tok.ID)
	if err != nil {
		fmt.Println("Get error:", err)
		return
	}
	fmt.Printf("Token valid: %v\n", fetched != nil)

	// 4. Revocation
	if err := store.Revoke(ctx, tok); err != nil {
		fmt.Println("Revoke error:", err)
		return
	}

	// 5. Compliance check after revocation
	fetched, err = store.Get(ctx, tok.ID)
	if err == nil && fetched == nil {
		fmt.Printf("Token revoked: %v\n", true)
	} else if err != nil {
		fmt.Println("Get after revoke error:", err)
	}

	// Output:
	// Token valid: true
	// Get after revoke error: token not found
}
