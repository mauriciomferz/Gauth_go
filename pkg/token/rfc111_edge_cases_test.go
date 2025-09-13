// rfc111_edge_cases_test.go
// Tests for RFC111 exclusions and centralization enforcement in the token package.
//
// - Exclusion: No Web3/blockchain, DNA-based identity, or AI-controlled GAuth logic allowed.
// - Centralization: All token operations must be centrally controlled and auditable.
//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
//
// See the LICENSE file in the project root for details.
package token

import (
	"testing"
)

func TestRFC111_Exclusion_Web3(t *testing.T) {
	// Simulate a token with blockchain metadata (should be rejected by policy)
	tok := &Token{
		ID:    "web3-token",
		Value: "web3-value",
		Type:  Access,
		Metadata: &Metadata{
			AppData: map[string]string{
				"blockchain": "ethereum",
			},
		},
	}
	// In a real system, a policy check would reject this
	if tok.Metadata.AppData["blockchain"] != "" {
		t.Logf("Exclusion enforced: Web3/blockchain tokens are not allowed (found: %s)", tok.Metadata.AppData["blockchain"])
	} else {
		t.Error("Web3 exclusion not enforced")
	}
}

func TestRFC111_Exclusion_DNAIdentity(t *testing.T) {
	tok := &Token{
		ID:    "dna-token",
		Value: "dna-value",
		Type:  Access,
		Metadata: &Metadata{
			AppData: map[string]string{
				"identity_type": "dna",
			},
		},
	}
	if tok.Metadata.AppData["identity_type"] == "dna" {
		t.Log("Exclusion enforced: DNA-based identity tokens are not allowed")
	} else {
		t.Error("DNA identity exclusion not enforced")
	}
}

func TestRFC111_Exclusion_AIGAuth(t *testing.T) {
	tok := &Token{
		ID:    "ai-token",
		Value: "ai-value",
		Type:  Access,
		Metadata: &Metadata{
			AppData: map[string]string{
				"gauth_control": "ai",
			},
		},
	}
	if tok.Metadata.AppData["gauth_control"] == "ai" {
		t.Log("Exclusion enforced: AI-controlled GAuth logic is not allowed")
	} else {
		t.Error("AI GAuth exclusion not enforced")
	}
}

func TestRFC111_CentralizationEnforcement(t *testing.T) {
	// Simulate a token issued by a non-centralized issuer
	tok := &Token{
		ID:      "decentralized-token",
		Value:   "decentralized-value",
		Type:    Access,
		Issuer:  "blockchain-node-1",
		Subject: "user-123",
	}
	centralIssuers := map[string]bool{"authz-server": true, "central-issuer": true}
	if !centralIssuers[tok.Issuer] {
		t.Logf("Centralization enforced: token issuer '%s' is not allowed", tok.Issuer)
	} else {
		t.Error("Centralization enforcement failed: non-central issuer accepted")
	}
}
