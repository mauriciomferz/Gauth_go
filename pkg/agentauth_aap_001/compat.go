package agentauth_aap_001

// Compatibility layer for legacy AAP-001 example code.
// The official_aap001_implementation example expects a configuration
// struct and a validation helper that previously lived in an earlier
// iteration of the project. Instead of rewriting the example to use the
// newer consolidated aap package config we provide a thin shim here.

import (
	"fmt"
	"time"
)

// AAP001Config represents high-level configuration required by the
// official AAP-001 implementation demo. All fields are intentionally
// kept exactly as referenced in the example for backwards compatibility.
type AAP001Config struct {
	AuthorizationServerURL    string        `json:"authorization_server_url"`
	TrustServiceProvider      string        `json:"trust_service_provider"`
	RequireNotarization       bool          `json:"require_notarization"`
	MaxDelegationDepth        int           `json:"max_delegation_depth"`
	DefaultTokenValidity      time.Duration `json:"default_token_validity"`
	AuditingEnabled           bool          `json:"auditing_enabled"`
	ComplianceTrackingEnabled bool          `json:"compliance_tracking_enabled"`

	// Mandatory open‑source exclusions as per the narrative in the example.
	ExcludeWeb3          bool `json:"exclude_web3"`
	ExcludeAIOperators   bool `json:"exclude_ai_operators"`
	ExcludeDNAIdentities bool `json:"exclude_dna_identities"`
}

// ValidateAAP001Compliance performs minimal semantic checks required by the
// example. It intentionally does NOT try to replicate deeper domain logic –
// the goal is to confirm that mandatory exclusions are enforced and that
// key numeric / duration parameters are sensible.
func ValidateAAP001Compliance(cfg *AAP001Config) error {
	if cfg == nil {
		return fmt.Errorf("aap001 config cannot be nil")
	}
	if cfg.AuthorizationServerURL == "" {
		return fmt.Errorf("authorization server URL required")
	}
	if cfg.TrustServiceProvider == "" {
		return fmt.Errorf("trust service provider required")
	}
	if cfg.MaxDelegationDepth <= 0 {
		return fmt.Errorf("max delegation depth must be > 0")
	}
	if cfg.MaxDelegationDepth > 8 { // arbitrary safeguard
		return fmt.Errorf("max delegation depth too large: %d", cfg.MaxDelegationDepth)
	}
	if cfg.DefaultTokenValidity <= 0 {
		return fmt.Errorf("default token validity must be > 0")
	}
	// Mandatory exclusions (per example commentary) must be TRUE in open source build.
	if !cfg.ExcludeWeb3 || !cfg.ExcludeAIOperators || !cfg.ExcludeDNAIdentities {
		return fmt.Errorf("all mandatory exclusions (web3, ai operators, dna identities) must be enabled")
	}
	return nil
}
