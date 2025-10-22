package main

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
)

// TestRFC0111ConfigValidationErrors exercises all error branches of ValidateRFC0111Compliance.
func TestRFC0111ConfigValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		cfg  *rfc0111.RFC0111Config
	}{
		{"nil", nil},
		{"empty auth server", &rfc0111.RFC0111Config{TrustServiceProvider: "tsp", MaxDelegationDepth: 1, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"empty TSP", &rfc0111.RFC0111Config{AuthorizationServerURL: "https://auth", MaxDelegationDepth: 1, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"depth zero", &rfc0111.RFC0111Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 0, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"depth too large", &rfc0111.RFC0111Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 99, DefaultTokenValidity: time.Hour, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"token validity zero", &rfc0111.RFC0111Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 1, DefaultTokenValidity: 0, ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
		{"exclusions false", &rfc0111.RFC0111Config{AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp", MaxDelegationDepth: 1, DefaultTokenValidity: time.Hour, ExcludeWeb3: false, ExcludeAIOperators: true, ExcludeDNAIdentities: true}},
	}
	for _, tc := range cases {
		if err := rfc0111.ValidateRFC0111Compliance(tc.cfg); err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
	}
}

// TestRFC0111ConfigValidationSuccess ensures a fully valid config passes.
func TestRFC0111ConfigValidationSuccess(t *testing.T) {
	cfg := &rfc0111.RFC0111Config{
		AuthorizationServerURL: "https://auth", TrustServiceProvider: "tsp",
		MaxDelegationDepth: 3, DefaultTokenValidity: 2 * time.Hour,
		ExcludeWeb3: true, ExcludeAIOperators: true, ExcludeDNAIdentities: true,
	}
	if err := rfc0111.ValidateRFC0111Compliance(cfg); err != nil {
		t.Fatalf("unexpected validation failure: %v", err)
	}
}
