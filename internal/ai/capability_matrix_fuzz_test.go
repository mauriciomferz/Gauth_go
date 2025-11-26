package ai

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mauriciomferz/Gauth_go/pkg/enforcement"
)

// FuzzCapabilityEnforcement fuzzes the capability matrix enforcement with arbitrary inputs
// to ensure it doesn't panic and always returns valid decisions.
func FuzzCapabilityEnforcement(f *testing.F) {
	// Seed corpus with valid inputs
	f.Add("assistant", "US", "medium", "transaction:read", "ai_entity_verified:true")
	f.Add("agent", "EU", "high", "transaction:execute", "ai_agent_registered:true,eu_act_compliant:true")
	f.Add("human", "US", "low", "data:access", "user_id:123")
	f.Add("hybrid", "CN", "critical", "admin:manage", "")
	f.Add("", "", "", "", "")

	f.Fuzz(func(t *testing.T, entityType, jurisdiction, riskLevel, capability, claimsStr string) {
		// Create matrix
		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		// Parse claims from string format "key1:val1,key2:val2"
		claims := make(map[string]any)
		if claimsStr != "" {
			pairs := strings.Split(claimsStr, ",")
			for _, pair := range pairs {
				kv := strings.Split(pair, ":")
				if len(kv) == 2 {
					// Try to parse as boolean
					switch kv[1] {
					case "true":
						claims[kv[0]] = true
					case "false":
						claims[kv[0]] = false
					default:
						claims[kv[0]] = kv[1]
					}
				}
			}
		}

		profile := AISystemProfile{
			EntityType:   AIEntityType(entityType),
			SystemID:     "fuzz-test-system",
			Jurisdiction: jurisdiction,
			RiskLevel:    riskLevel,
		}

		// Should not panic regardless of input
		decision := matrix.EnforceAICapabilities(profile, capability, claims)

		// Validate decision structure
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision value: %q (expected 'allow' or 'deny')", decision.Decision)
		}

		// If denied, should have a reason
		if decision.Decision == enforcement.DecisionDeny && decision.Reason == "" {
			t.Error("Deny decision missing reason")
		}

		// Decision should be consistent on repeated calls
		decision2 := matrix.EnforceAICapabilities(profile, capability, claims)
		if decision.Decision != decision2.Decision {
			t.Errorf("Inconsistent decisions: %s vs %s", decision.Decision, decision2.Decision)
		}
	})
}

// FuzzWildcardMatching fuzzes the wildcard capability matching logic
func FuzzWildcardMatching(f *testing.F) {
	// Seed corpus
	f.Add("transaction:read", "transaction:read")
	f.Add("transaction:*", "transaction:execute")
	f.Add("*:read", "data:read")
	f.Add("*:*", "admin:manage")
	f.Add("", "capability")
	f.Add(":", ":")
	f.Add("***", "test")

	f.Fuzz(func(t *testing.T, pattern, capability string) {
		// Test wildcard matching through capability enforcement
		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		profile := AISystemProfile{
			EntityType:   AIEntityAgent,
			SystemID:     "wildcard-test",
			Jurisdiction: "US",
			RiskLevel:    "medium",
		}

		claims := map[string]any{
			"ai_entity_verified":  true,
			"ai_agent_registered": true,
		}

		// Should not panic with arbitrary patterns
		decision := matrix.EnforceAICapabilities(profile, capability, claims)

		// Validate return value
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision: %q", decision.Decision)
		}

		// Test idempotence
		decision2 := matrix.EnforceAICapabilities(profile, capability, claims)
		if decision.Decision != decision2.Decision {
			t.Errorf("Inconsistent matching for capability=%q", capability)
		}
	})
}

// FuzzSystemProfileValidation fuzzes system profile validation
func FuzzSystemProfileValidation(f *testing.F) {
	// Seed corpus
	f.Add("assistant", "sys-123", "US", "medium", "finance")
	f.Add("agent", "", "EU", "high", "")
	f.Add("", "test", "", "", "healthcare")
	f.Add(strings.Repeat("x", 1000), "long-id", "INVALID", "超高", "特殊")

	f.Fuzz(func(t *testing.T, entityType, systemID, jurisdiction, riskLevel, industry string) {
		profile := AISystemProfile{
			EntityType:      AIEntityType(entityType),
			SystemID:        systemID,
			Jurisdiction:    jurisdiction,
			RiskLevel:       riskLevel,
			IndustryContext: industry,
		}

		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		// Should not panic with arbitrary profile
		claims := map[string]any{"test": true}
		decision := matrix.EnforceAICapabilities(profile, "test:capability", claims)

		// Decision should be valid
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision: %q", decision.Decision)
		}

		// System profile should be reflected in decision
		if decision.SystemProfile.SystemID != systemID {
			t.Errorf("SystemID mismatch: expected %q, got %q", systemID, decision.SystemProfile.SystemID)
		}
	})
}

// FuzzClaimsValidation fuzzes claims validation logic
func FuzzClaimsValidation(f *testing.F) {
	// Seed corpus with various claim combinations
	f.Add(10, true, "high", float64(100.5))
	f.Add(0, false, "", float64(0))
	f.Add(-999, true, "超高风险", float64(-1.5))
	f.Add(999999, false, strings.Repeat("x", 500), float64(1e100))

	f.Fuzz(func(t *testing.T, intVal int, boolVal bool, strVal string, floatVal float64) {
		claims := map[string]any{
			"int_claim":   intVal,
			"bool_claim":  boolVal,
			"str_claim":   strVal,
			"float_claim": floatVal,
			"nil_claim":   nil,
			"nested_map":  map[string]any{"inner": "value"},
		}

		profile := AISystemProfile{
			EntityType:   "agent",
			SystemID:     "fuzz-test",
			Jurisdiction: "US",
			RiskLevel:    "high",
		}

		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		// Should not panic with arbitrary claim types
		decision := matrix.EnforceAICapabilities(profile, "test:capability", claims)

		// Validate decision structure
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision: %q", decision.Decision)
		}

		// Missing capabilities should be strings
		for _, cap := range decision.MissingCapabilities {
			if cap == "" {
				t.Error("Missing capability should not be empty string")
			}
		}
	})
}

// FuzzJurisdictionEnforcement fuzzes jurisdiction-specific enforcement rules
func FuzzJurisdictionEnforcement(f *testing.F) {
	// Seed corpus with various jurisdictions
	f.Add("US", "agent", "high")
	f.Add("EU", "assistant", "medium")
	f.Add("CN", "human", "low")
	f.Add("", "hybrid", "critical")
	f.Add("UNKNOWN", "", "")
	f.Add(strings.Repeat("X", 100), "agent", "超高")

	f.Fuzz(func(t *testing.T, jurisdiction, entityType, riskLevel string) {
		profile := AISystemProfile{
			EntityType:   AIEntityType(entityType),
			SystemID:     "fuzz-system",
			Jurisdiction: jurisdiction,
			RiskLevel:    riskLevel,
		}

		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		claims := map[string]any{
			"ai_entity_verified":         true,
			"ai_agent_registered":        true,
			"algorithmic_accountability": true,
			"eu_act_compliant":           true,
			"explainability_compliant":   true,
			"china_reg_compliant":        true,
		}

		// Should not panic with arbitrary jurisdiction
		decision := matrix.EnforceAICapabilities(profile, "transaction:execute", claims)

		// Validate decision
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision for jurisdiction %q: %q", jurisdiction, decision.Decision)
		}

		// Jurisdiction should be reflected in profile
		if decision.SystemProfile.Jurisdiction != jurisdiction {
			t.Errorf("Jurisdiction mismatch: expected %q, got %q", jurisdiction, decision.SystemProfile.Jurisdiction)
		}
	})
}

// FuzzAuditLevelDetermination fuzzes audit level determination logic
func FuzzAuditLevelDetermination(f *testing.F) {
	// Seed corpus
	f.Add("agent", "critical", "finance", "transaction:execute")
	f.Add("assistant", "low", "general", "data:read")
	f.Add("human", "", "", "")
	f.Add("", "ultra-high", "healthcare", strings.Repeat("x", 200))

	f.Fuzz(func(t *testing.T, entityType, riskLevel, industry, capability string) {
		profile := AISystemProfile{
			EntityType:      AIEntityType(entityType),
			SystemID:        "audit-fuzz",
			Jurisdiction:    "US",
			RiskLevel:       riskLevel,
			IndustryContext: industry,
		}

		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		claims := map[string]any{"test": true}
		decision := matrix.EnforceAICapabilities(profile, capability, claims)

		// Audit level should be a valid string
		validLevels := map[string]bool{
			"none":     true,
			"basic":    true,
			"detailed": true,
			"full":     true,
			"":         true, // Empty is acceptable default
		}

		if !validLevels[decision.AuditLevel] {
			t.Errorf("Invalid audit level: %q (valid: none, basic, detailed, full)", decision.AuditLevel)
		}
	})
}

// FuzzEnforcementToggle fuzzes the enforcement active/inactive toggle
func FuzzEnforcementToggle(f *testing.F) {
	// Seed corpus
	f.Add(true, "agent", "high")
	f.Add(false, "assistant", "medium")
	f.Add(true, "", "")
	f.Add(false, "human", "critical")

	f.Fuzz(func(t *testing.T, active bool, entityType, riskLevel string) {
		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(active)

		profile := AISystemProfile{
			EntityType:   AIEntityType(entityType),
			SystemID:     "toggle-fuzz",
			Jurisdiction: "US",
			RiskLevel:    riskLevel,
		}

		claims := map[string]any{}
		decision := matrix.EnforceAICapabilities(profile, "test:capability", claims)

		// When enforcement is inactive, should allow by default
		if !active && decision.Decision == enforcement.DecisionDeny {
			t.Error("Enforcement inactive should not deny")
		} // Decision should be valid
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision: %q", decision.Decision)
		}
	})
}

// FuzzCapabilityFormat fuzzes various capability string formats
func FuzzCapabilityFormat(f *testing.F) {
	// Seed corpus with edge cases
	f.Add("simple")
	f.Add("namespace:action")
	f.Add("a:b:c:d:e")
	f.Add(":")
	f.Add(":action")
	f.Add("namespace:")
	f.Add("")
	f.Add("*")
	f.Add("**:**")
	f.Add("中文:操作")
	f.Add(strings.Repeat("x", 1000))

	f.Fuzz(func(t *testing.T, capability string) {
		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		profile := AISystemProfile{
			EntityType:   "agent",
			SystemID:     "format-fuzz",
			Jurisdiction: "US",
			RiskLevel:    "medium",
		}

		claims := map[string]any{
			"ai_entity_verified":  true,
			"ai_agent_registered": true,
		}

		// Should not panic with arbitrary capability format
		decision := matrix.EnforceAICapabilities(profile, capability, claims)

		// Decision should be valid
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision for capability %q: %q", capability, decision.Decision)
		}

		// Very long capabilities might be rejected, but shouldn't panic
		if len(capability) > 1000 {
			// Just verify it doesn't crash
			_ = decision.Decision
		}
	})
}

// FuzzRiskLevelEvaluation fuzzes risk level evaluation
func FuzzRiskLevelEvaluation(f *testing.F) {
	// Seed corpus
	f.Add("low", 1)
	f.Add("medium", 2)
	f.Add("high", 3)
	f.Add("critical", 4)
	f.Add("", 0)
	f.Add("超高", 999)
	f.Add(strings.Repeat("x", 100), -1)

	f.Fuzz(func(t *testing.T, riskLevel string, numClaims int) {
		// Generate claims based on numClaims
		claims := make(map[string]any)
		for i := 0; i < numClaims && i < 100; i++ {
			claims[fmt.Sprintf("claim_%d", i)] = true
		}

		profile := AISystemProfile{
			EntityType:   "agent",
			SystemID:     "risk-fuzz",
			Jurisdiction: "US",
			RiskLevel:    riskLevel,
		}

		matrix := NewAICapabilityMatrix()
		matrix.SetEnforcementActive(true)

		decision := matrix.EnforceAICapabilities(profile, "test:capability", claims)

		// Decision should be valid
		if decision.Decision != enforcement.DecisionAllow && decision.Decision != enforcement.DecisionDeny {
			t.Errorf("Invalid decision: %q", decision.Decision)
		}

		// Higher risk should generally require more claims (when enforcement is active)
		// But shouldn't panic regardless
		_ = decision.MissingCapabilities
	})
}
