package main

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/AGENTAUTH_aap_001"
)

// TestAAP001ConfigValidationErrors exercises all error branches of ValidateAAP001Compliance.
func TestAAP001ConfigValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		cfg  *AGENTAUTH_aap_001.AAP001Config
	}{
		{"nil", nil},
		{"empty auth server", &AGENTAUTH_aap_001.AAP001Config{TrustServiceProvider: "tsp", MaxDelegationDepth: 1, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"empty TSP", &AGENTAUTH_aap_001.AAP001Config{AuthorizationServerURL: "https://auth", MaxDelegationDepth: 1, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"depth zero", &AGENTAUTH_aap_001.AAP001Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 0, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"depth too large", &AGENTAUTH_aap_001.AAP001Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 99, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"token validity zero", &AGENTAUTH_aap_001.AAP001Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 1, DefaultTokenValidity: 0, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"exclusions false", &AGENTAUTH_aap_001.AAP001Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 1, DefaultTokenValidity: time.Hour, ExcludeWeb3: false, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
	}
	for _, tc := range cases {
		if err := AGENTAUTH_aap_001.ValidateAAP001Compliance(tc.cfg); err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
	}
}

// TestAAP001ConfigValidationSuccess ensures a fully valid config passes.
func TestAAP001ConfigValidationSuccess(t *testing.T) {
	cfg := &AGENTAUTH_aap_001.AAP001Config{
		AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp",
		MaxDelegationDepth: 3, DefaultTokenValidity: 2 * time.Hour,
		ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true,
	}
	if err := AGENTAUTH_aap_001.ValidateAAP001Compliance(cfg); err != nil {
		t.Fatalf("unexpected validation failure: %v", err)
	}
}
