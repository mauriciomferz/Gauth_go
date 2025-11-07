package ai

import (
	"testing"
	"time"
)

// TestAICapabilityMatrix tests core AI capability matrix functionality
func TestAICapabilityMatrix(t *testing.T) {
	matrix := NewAICapabilityMatrix()
	matrix.SetEnforcementActive(true)

	t.Run("Human Entity Access", func(t *testing.T) {
		profile := AISystemProfile{
			EntityType:   AIEntityHuman,
			SystemID:     "human-user-123",
			Jurisdiction: "US",
			RiskLevel:    "low",
		}

		claims := map[string]any{
			"user_id": "human-123",
		}

		decision := matrix.EnforceAICapabilities(profile, "transaction:execute", claims)

		if decision.Decision != "allow" {
			t.Errorf("Expected human to be allowed, got: %s", decision.Decision)
		}

		if decision.RequiredHumanAuth {
			t.Error("Human should not require human auth")
		}
	})

	t.Run("AI Assistant Restrictions", func(t *testing.T) {
		profile := AISystemProfile{
			EntityType:   AIEntityAssistant,
			SystemID:     "assistant-gpt4",
			Jurisdiction: "US",
			RiskLevel:    "medium",
		}

		claims := map[string]any{
			"ai_entity_verified":         true,
			"algorithmic_accountability": true, // Required by US policy
		}

		// Should allow read operations
		decision := matrix.EnforceAICapabilities(profile, "transaction:read", claims)
		if decision.Decision != "allow" {
			t.Errorf("Expected assistant to be allowed for read, got: %s - %s", decision.Decision, decision.Reason)
		}

		// Should deny execution operations
		decision = matrix.EnforceAICapabilities(profile, "transaction:execute", claims)
		if decision.Decision != "deny" {
			t.Errorf("Expected assistant to be denied for execute, got: %s", decision.Decision)
		}
	})

	t.Run("AI Agent with Missing Claims", func(t *testing.T) {
		profile := AISystemProfile{
			EntityType:   AIEntityAgent,
			SystemID:     "agent-autonomous-1",
			Jurisdiction: "US",
			RiskLevel:    "high",
		}

		claims := map[string]any{
			// Missing required claims
		}

		decision := matrix.EnforceAICapabilities(profile, "transaction:read", claims)
		if decision.Decision != "deny" {
			t.Errorf("Expected agent to be denied without required claims, got: %s", decision.Decision)
		}

		if len(decision.MissingCapabilities) == 0 {
			t.Error("Expected missing capabilities to be reported")
		}
	})

	t.Run("EU AI Act Compliance", func(t *testing.T) {
		profile := AISystemProfile{
			EntityType:      AIEntityAgent,
			SystemID:        "eu-agent-1",
			Jurisdiction:    "EU",
			RiskLevel:       "high",
			IndustryContext: "general",
		}

		claims := map[string]any{
			"ai_entity_verified":  true,
			"ai_agent_registered": true,
			// Missing EU-specific claims
		}

		decision := matrix.EnforceAICapabilities(profile, "transaction:execute", claims)
		if decision.Decision != "deny" {
			t.Errorf("Expected EU agent to be denied without EU compliance claims")
		}

		// Should require EU compliance
		foundEUPolicy := false
		for _, policyID := range decision.AppliedPolicies {
			if policyID == "eu_ai_act_2025" {
				foundEUPolicy = true
				break
			}
		}
		if !foundEUPolicy {
			t.Error("Expected EU AI Act policy to be applied")
		}
	})

	t.Run("Healthcare Industry Restrictions", func(t *testing.T) {
		profile := AISystemProfile{
			EntityType:      AIEntityAnalytics,
			SystemID:        "healthcare-ai-analytics",
			Jurisdiction:    "US",
			RiskLevel:       "critical",
			IndustryContext: "healthcare",
		}

		claims := map[string]any{
			"ai_analytics_approved": true,
			"ai_entity_verified":    true,
			// Missing HIPAA compliance
		}

		decision := matrix.EnforceAICapabilities(profile, "transaction:read", claims)
		if decision.Decision != "deny" {
			t.Errorf("Expected healthcare AI to be denied without HIPAA compliance")
		}

		// Add HIPAA compliance and other required claims
		claims["hipaa_compliance"] = true
		claims["phi_protection"] = true
		claims["healthcare_cert"] = true
		claims["de_identification"] = true
		claims["algorithmic_accountability"] = true // Required by US default policy

		decision = matrix.EnforceAICapabilities(profile, "transaction:read", claims)
		if decision.Decision != "allow" {
			t.Errorf("Expected healthcare AI to be allowed with HIPAA compliance, got: %s - %s", decision.Decision, decision.Reason)
		}

		if !decision.RequiredHumanAuth {
			t.Error("Healthcare AI should require human authorization")
		}
	})

	t.Run("Time Window Restrictions", func(t *testing.T) {
		// Create automation AI with time restrictions
		rule := AICapabilityRule{
			EntityType:     AIEntityAutomation,
			AllowedActions: []string{"transaction:execute"},
			RequiredClaims: []string{"ai_automation_certified", "ai_entity_verified"},
			TimeWindows:    []string{"09:00-17:00"}, // Business hours only
			AuditLevel:     "realtime",
		}

		matrix.UpdateEntityRule(AIEntityAutomation, rule)

		profile := AISystemProfile{
			EntityType:   AIEntityAutomation,
			SystemID:     "automation-system-1",
			Jurisdiction: "US",
			RiskLevel:    "medium",
		}

		claims := map[string]any{
			"ai_automation_certified": true,
			"ai_entity_verified":      true,
		}

		// Test enforcement with time restrictions (in real scenario we'd mock time)
		decision := matrix.EnforceAICapabilities(profile, "transaction:execute", claims)

		// For now, just verify the rule was updated and decision was made
		retrievedRule, exists := matrix.GetEntityRule(AIEntityAutomation)

		// Verify decision was processed (may be denied due to time window)
		if decision.DecisionID == "" {
			t.Error("Expected decision ID to be generated")
		}
		if !exists {
			t.Error("Expected automation rule to exist")
		}

		if len(retrievedRule.TimeWindows) != 1 || retrievedRule.TimeWindows[0] != "09:00-17:00" {
			t.Error("Expected time window restriction to be set")
		}
	})

	t.Run("Enforcement Disabled", func(t *testing.T) {
		matrix.SetEnforcementActive(false)

		profile := AISystemProfile{
			EntityType:   AIEntityModel,
			SystemID:     "dangerous-model",
			Jurisdiction: "US",
			RiskLevel:    "critical",
		}

		claims := map[string]any{
			// No claims at all
		}

		decision := matrix.EnforceAICapabilities(profile, "admin:delete", claims)
		if decision.Decision != "allow" {
			t.Errorf("Expected to allow when enforcement disabled, got: %s", decision.Decision)
		}

		if decision.Reason != "AI capability enforcement disabled" {
			t.Errorf("Expected disabled reason, got: %s", decision.Reason)
		}

		// Re-enable for other tests
		matrix.SetEnforcementActive(true)
	})

	t.Run("Policy Management", func(t *testing.T) {
		policies := matrix.GetGovernancePolicies()
		if len(policies) == 0 {
			t.Error("Expected default policies to be loaded")
		}

		// Test adding new policy
		customPolicy := AIGovernancePolicy{
			PolicyID:            "test_policy_123",
			Jurisdiction:        "CA",
			ComplianceFramework: "PIPEDA",
			ProhibitedActions:   []string{"transaction:issue"},
			MandatoryClaims:     []string{"canada_compliance"},
			EffectiveDate:       time.Now().Format(time.RFC3339),
			LastUpdated:         time.Now().Format(time.RFC3339),
		}

		matrix.AddGovernancePolicy(customPolicy)

		updatedPolicies := matrix.GetGovernancePolicies()
		if len(updatedPolicies) <= len(policies) {
			t.Error("Expected policy count to increase after adding custom policy")
		}

		// Verify Canadian jurisdiction gets the new policy
		profile := AISystemProfile{
			EntityType:   AIEntityAgent,
			SystemID:     "canada-agent",
			Jurisdiction: "CA",
			RiskLevel:    "medium",
		}

		claims := map[string]any{
			"ai_entity_verified":  true,
			"ai_agent_registered": true,
			// Missing canada_compliance
		}

		decision := matrix.EnforceAICapabilities(profile, "transaction:read", claims)
		foundCustomPolicy := false
		for _, policyID := range decision.AppliedPolicies {
			if policyID == "test_policy_123" {
				foundCustomPolicy = true
				break
			}
		}
		if !foundCustomPolicy {
			t.Error("Expected custom Canadian policy to be applied")
		}
	})
}

// TestServerIntegration tests the server integration layer
func TestServerIntegration(t *testing.T) {
	integration := NewServerIntegration()
	integration.EnableEnforcement(true)

	auditCalled := false
	metricsCalled := false

	integration.SetAuditCallback(func(action string, metadata map[string]any) {
		auditCalled = true
		if action != "ai_capability_enforcement" {
			t.Errorf("Expected audit action 'ai_capability_enforcement', got: %s", action)
		}
	})

	integration.SetMetricsCallback(func(metric string) {
		metricsCalled = true
	})

	t.Run("AI Profile Extraction", func(t *testing.T) {
		claims := map[string]any{
			"ai_entity_type":      "agent",
			"system_id":           "test-agent-1",
			"jurisdiction":        "EU",
			"industry_context":    "finance",
			"risk_level":          "high",
			"model_name":          "gpt-4",
			"compliance_flags":    []string{"GDPR", "MiFID"},
			"ai_agent_registered": true,
		}

		allowed, missing, metadata := integration.EnforceAICapabilities("transaction:read", claims)

		// Should be denied due to missing EU compliance claims
		if allowed {
			t.Error("Expected EU financial agent to be denied without proper compliance")
		}

		if len(missing) == 0 && !allowed {
			t.Error("Expected missing capabilities to be reported when denied")
		}

		if metadata == nil {
			t.Error("Expected metadata to be returned")
		}

		if entityType, ok := metadata["entity_type"].(string); !ok || entityType != "agent" {
			t.Error("Expected entity_type in metadata")
		}

		// Give the goroutine a moment to execute the audit callback
		time.Sleep(10 * time.Millisecond)

		if !auditCalled {
			t.Error("Expected audit callback to be called")
		}

		if !metricsCalled {
			t.Error("Expected metrics callback to be called")
		}
	})

	t.Run("Human User Bypass", func(t *testing.T) {
		claims := map[string]any{
			"user_id": "human-123",
			// No AI indicators
		}

		allowed, missing, metadata := integration.EnforceAICapabilities("admin:delete", claims)

		if !allowed {
			t.Error("Expected human user to be allowed")
		}

		if len(missing) > 0 {
			t.Error("Expected no missing capabilities for human")
		}

		if enforcement, ok := metadata["ai_enforcement"].(string); !ok || enforcement != "skipped" {
			t.Error("Expected AI enforcement to be skipped for human")
		}
	})

	t.Run("Profile Validation", func(t *testing.T) {
		validProfile := AISystemProfile{
			EntityType:   AIEntityAgent,
			SystemID:     "valid-agent",
			Jurisdiction: "US",
			RiskLevel:    "medium",
		}

		errors := integration.ValidateAIProfile(validProfile)
		if len(errors) > 0 {
			t.Errorf("Expected valid profile to have no errors, got: %v", errors)
		}

		invalidProfile := AISystemProfile{
			EntityType: AIEntityModel,
			// Missing required fields
		}

		errors = integration.ValidateAIProfile(invalidProfile)
		if len(errors) == 0 {
			t.Error("Expected invalid profile to have validation errors")
		}
	})

	t.Run("Status Information", func(t *testing.T) {
		status := integration.GetAICapabilityStatus()

		if active, ok := status["enforcement_active"].(bool); !ok || !active {
			t.Error("Expected enforcement to be active")
		}

		if entityTypes, ok := status["supported_entity_types"].([]string); !ok || len(entityTypes) == 0 {
			t.Error("Expected supported entity types to be listed")
		}

		if policyCount, ok := status["loaded_policies"].(int); !ok || policyCount == 0 {
			t.Error("Expected policies to be loaded")
		}
	})

	t.Run("Test Profile Creation", func(t *testing.T) {
		testProfile := integration.CreateTestProfile("assistant", "EU")

		if testProfile.EntityType != AIEntityAssistant {
			t.Error("Expected assistant entity type")
		}

		if testProfile.Jurisdiction != "EU" {
			t.Error("Expected EU jurisdiction")
		}

		if testProfile.SystemID == "" {
			t.Error("Expected system ID to be generated")
		}
	})
}

// TestWildcardMatching tests action pattern matching
func TestWildcardMatching(t *testing.T) {
	matrix := NewAICapabilityMatrix()

	testCases := []struct {
		action   string
		pattern  string
		expected bool
	}{
		{"transaction:execute", "*", true},
		{"transaction:execute", "transaction:execute", true},
		{"transaction:execute", "transaction:*", true},
		{"transaction:execute", "admin:*", false},
		{"admin:delete", "admin:*", true},
		{"info:read", "info:*", true},
		{"info:read", "transaction:*", false},
		{"delegation:create", "delegation:*", true},
		{"anything", "*", true},
	}

	for _, tc := range testCases {
		result := matrix.matchesActionPattern(tc.action, tc.pattern)
		if result != tc.expected {
			t.Errorf("Pattern matching %s against %s: expected %v, got %v",
				tc.action, tc.pattern, tc.expected, result)
		}
	}
}

// TestClaimValidation tests claim presence validation
func TestClaimValidation(t *testing.T) {
	matrix := NewAICapabilityMatrix()

	testCases := []struct {
		claims    map[string]any
		claimName string
		expected  bool
	}{
		{map[string]any{"test": true}, "test", true},
		{map[string]any{"test": false}, "test", false},
		{map[string]any{"test": "yes"}, "test", true},
		{map[string]any{"test": ""}, "test", false},
		{map[string]any{"test": "false"}, "test", false},
		{map[string]any{"test": []string{"value"}}, "test", true},
		{map[string]any{"test": []string{}}, "test", false},
		{map[string]any{}, "missing", false},
		{map[string]any{"test": nil}, "test", false},
		{map[string]any{"test": "0"}, "test", false},
	}

	for i, tc := range testCases {
		result := matrix.hasClaimValue(tc.claims, tc.claimName)
		if result != tc.expected {
			t.Errorf("Test case %d: claim validation for %s: expected %v, got %v",
				i, tc.claimName, tc.expected, result)
		}
	}
}

// TestAuditLevelPriority tests audit level priority calculation
func TestAuditLevelPriority(t *testing.T) {
	matrix := NewAICapabilityMatrix()

	testCases := []struct {
		level    string
		expected int
	}{
		{"none", 0},
		{"basic", 1},
		{"detailed", 2},
		{"realtime", 3},
		{"unknown", 1}, // default to basic
	}

	for _, tc := range testCases {
		result := matrix.getAuditLevelPriority(tc.level)
		if result != tc.expected {
			t.Errorf("Audit level priority for %s: expected %d, got %d",
				tc.level, tc.expected, result)
		}
	}
}

// TestEntityTypeSupport tests that all expected entity types are supported
func TestEntityTypeSupport(t *testing.T) {
	matrix := NewAICapabilityMatrix()
	matrix.SetEnforcementActive(true)

	expectedTypes := []AIEntityType{
		AIEntityHuman, AIEntityAssistant, AIEntityAgent, AIEntityModel,
		AIEntitySystem, AIEntityRobot, AIEntityAnalytics, AIEntityAutomation,
	}

	supportedTypes := matrix.GetEntityTypes()

	if len(supportedTypes) != len(expectedTypes) {
		t.Errorf("Expected %d entity types, got %d", len(expectedTypes), len(supportedTypes))
	}

	// Test that each expected type has a rule
	for _, expectedType := range expectedTypes {
		_, exists := matrix.GetEntityRule(expectedType)
		if !exists {
			t.Errorf("Expected rule to exist for entity type: %s", expectedType)
		}
	}
}

// BenchmarkAIEnforcement benchmarks the AI enforcement decision process
func BenchmarkAIEnforcement(b *testing.B) {
	matrix := NewAICapabilityMatrix()
	matrix.SetEnforcementActive(true)

	profile := AISystemProfile{
		EntityType:   AIEntityAgent,
		SystemID:     "benchmark-agent",
		Jurisdiction: "US",
		RiskLevel:    "medium",
	}

	claims := map[string]any{
		"ai_entity_verified":  true,
		"ai_agent_registered": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matrix.EnforceAICapabilities(profile, "transaction:read", claims)
	}
}

// BenchmarkServerIntegration benchmarks the server integration layer
func BenchmarkServerIntegration(b *testing.B) {
	integration := NewServerIntegration()
	integration.EnableEnforcement(true)

	claims := map[string]any{
		"ai_entity_type":      "agent",
		"system_id":           "benchmark-agent",
		"jurisdiction":        "US",
		"ai_entity_verified":  true,
		"ai_agent_registered": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		integration.EnforceAICapabilities("transaction:read", claims)
	}
}
