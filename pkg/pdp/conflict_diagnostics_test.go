package pdp

import (
	"testing"
	"time"
)

// TestDetectPermitDenyConflicts verifies permit-deny conflict detection
func TestDetectPermitDenyConflicts(t *testing.T) {
	policies := []Policy{
		{
			ID:       "policy-allow-read",
			Subjects: []string{"user:alice"},
			Rules: []Rule{
				{
					ID:        "rule-1",
					Actions:   []string{"read"},
					Resources: []string{"/documents/*"},
					Effect:    "allow",
				},
			},
		},
		{
			ID:       "policy-deny-read",
			Subjects: []string{"user:alice"},
			Rules: []Rule{
				{
					ID:        "rule-2",
					Actions:   []string{"read"},
					Resources: []string{"/documents/*"},
					Effect:    "deny",
				},
			},
		},
	}

	detector := NewConflictDetector(policies, DenyOverridesStrategy{})
	conflicts := detector.detectPermitDenyConflicts()

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 permit-deny conflict, got %d", len(conflicts))
	}

	c := conflicts[0]
	if c.Type != ConflictPermitDeny {
		t.Errorf("Expected conflict type %s, got %s", ConflictPermitDeny, c.Type)
	}

	if c.Severity != SeverityCritical {
		t.Errorf("Expected severity %s for non-wildcard subject, got %s", SeverityCritical, c.Severity)
	}

	if len(c.PolicyIDs) != 2 {
		t.Errorf("Expected 2 policies in conflict, got %d", len(c.PolicyIDs))
	}

	if c.Subject != "user:alice" {
		t.Errorf("Expected subject 'user:alice', got %s", c.Subject)
	}

	if c.Action != "read" {
		t.Errorf("Expected action 'read', got %s", c.Action)
	}

	if c.Resource != "/documents/*" {
		t.Errorf("Expected resource '/documents/*', got %s", c.Resource)
	}
}

// TestDetectPermitDenyConflicts_WildcardSubject tests severity adjustment for wildcard subjects
func TestDetectPermitDenyConflicts_WildcardSubject(t *testing.T) {
	policies := []Policy{
		{
			ID:       "policy-allow-all",
			Subjects: []string{"*"},
			Rules: []Rule{
				{
					ID:        "rule-1",
					Actions:   []string{"write"},
					Resources: []string{"/data"},
					Effect:    "allow",
				},
			},
		},
		{
			ID:       "policy-deny-all",
			Subjects: []string{"*"},
			Rules: []Rule{
				{
					ID:        "rule-2",
					Actions:   []string{"write"},
					Resources: []string{"/data"},
					Effect:    "deny",
				},
			},
		},
	}

	detector := NewConflictDetector(policies, DenyOverridesStrategy{})
	conflicts := detector.detectPermitDenyConflicts()

	if len(conflicts) == 0 {
		t.Fatal("Expected at least 1 conflict")
	}

	// Wildcard subjects should have lower severity
	if conflicts[0].Severity != SeverityHigh {
		t.Errorf("Expected severity %s for wildcard subject, got %s", SeverityHigh, conflicts[0].Severity)
	}
}

// TestDetectScopeOverlaps verifies scope overlap detection
func TestDetectScopeOverlaps(t *testing.T) {
	policies := []Policy{
		{
			ID:       "policy-1",
			Subjects: []string{"user:bob"},
			Rules: []Rule{
				{
					ID:        "rule-1",
					Actions:   []string{"read", "write"},
					Resources: []string{"/files/*"},
					Effect:    "allow",
				},
			},
		},
		{
			ID:       "policy-2",
			Subjects: []string{"user:bob"},
			Rules: []Rule{
				{
					ID:        "rule-2",
					Actions:   []string{"read"},
					Resources: []string{"/files/*"},
					Effect:    "deny",
				},
			},
		},
	}

	detector := NewConflictDetector(policies, DenyOverridesStrategy{})
	conflicts := detector.detectScopeOverlaps()

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 scope overlap, got %d", len(conflicts))
	}

	c := conflicts[0]
	if c.Type != ConflictScopeOverlap {
		t.Errorf("Expected conflict type %s, got %s", ConflictScopeOverlap, c.Type)
	}

	// Different effects = higher severity
	if c.Severity != SeverityHigh {
		t.Errorf("Expected severity %s for different effects, got %s", SeverityHigh, c.Severity)
	}

	if len(c.PolicyIDs) != 2 {
		t.Errorf("Expected 2 policies in overlap, got %d", len(c.PolicyIDs))
	}

	if len(c.RuleIDs) != 2 {
		t.Errorf("Expected 2 rules in overlap, got %d", len(c.RuleIDs))
	}
}

// TestDetectScopeOverlaps_SameEffect tests lower severity for same-effect overlaps
func TestDetectScopeOverlaps_SameEffect(t *testing.T) {
	policies := []Policy{
		{
			ID:       "policy-1",
			Subjects: []string{"user:charlie"},
			Rules: []Rule{
				{
					ID:        "rule-1",
					Actions:   []string{"execute"},
					Resources: []string{"/scripts/*"},
					Effect:    "allow",
				},
			},
		},
		{
			ID:       "policy-2",
			Subjects: []string{"user:charlie"},
			Rules: []Rule{
				{
					ID:        "rule-2",
					Actions:   []string{"execute"},
					Resources: []string{"/scripts/*"},
					Effect:    "allow",
				},
			},
		},
	}

	detector := NewConflictDetector(policies, DenyOverridesStrategy{})
	conflicts := detector.detectScopeOverlaps()

	if len(conflicts) == 0 {
		t.Fatal("Expected at least 1 overlap")
	}

	// Same effect = medium severity
	if conflicts[0].Severity != SeverityMedium {
		t.Errorf("Expected severity %s for same effect, got %s", SeverityMedium, conflicts[0].Severity)
	}
}

// TestDetectRuleContradictions verifies intra-policy rule conflicts
func TestDetectRuleContradictions(t *testing.T) {
	policies := []Policy{
		{
			ID:       "policy-contradictory",
			Subjects: []string{"user:dave"},
			Rules: []Rule{
				{
					ID:        "rule-allow",
					Actions:   []string{"delete"},
					Resources: []string{"/temp/*"},
					Effect:    "allow",
				},
				{
					ID:        "rule-deny",
					Actions:   []string{"delete"},
					Resources: []string{"/temp/*"},
					Effect:    "deny",
				},
			},
		},
	}

	detector := NewConflictDetector(policies, DenyOverridesStrategy{})
	conflicts := detector.detectRuleContradictions()

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 rule contradiction, got %d", len(conflicts))
	}

	c := conflicts[0]
	if c.Type != ConflictRuleContradiction {
		t.Errorf("Expected conflict type %s, got %s", ConflictRuleContradiction, c.Type)
	}

	if c.Severity != SeverityHigh {
		t.Errorf("Expected severity %s, got %s", SeverityHigh, c.Severity)
	}

	if len(c.PolicyIDs) != 1 {
		t.Errorf("Expected 1 policy ID, got %d", len(c.PolicyIDs))
	}

	if len(c.RuleIDs) != 2 {
		t.Errorf("Expected 2 rule IDs, got %d", len(c.RuleIDs))
	}
}

// TestDetectConflicts verifies comprehensive conflict detection
func TestDetectConflicts(t *testing.T) {
	policies := []Policy{
		{
			ID:       "policy-1",
			Subjects: []string{"user:eve"},
			Rules: []Rule{
				{
					ID:        "rule-1",
					Actions:   []string{"admin"},
					Resources: []string{"/system"},
					Effect:    "allow",
				},
			},
		},
		{
			ID:       "policy-2",
			Subjects: []string{"user:eve"},
			Rules: []Rule{
				{
					ID:        "rule-2",
					Actions:   []string{"admin"},
					Resources: []string{"/system"},
					Effect:    "deny",
				},
			},
		},
		{
			ID:       "policy-3",
			Subjects: []string{"user:frank"},
			Rules: []Rule{
				{
					ID:        "rule-3a",
					Actions:   []string{"read"},
					Resources: []string{"/logs"},
					Effect:    "allow",
				},
				{
					ID:        "rule-3b",
					Actions:   []string{"read"},
					Resources: []string{"/logs"},
					Effect:    "deny",
				},
			},
		},
	}

	detector := NewConflictDetector(policies, DenyOverridesStrategy{})
	conflicts := detector.DetectConflicts()

	// Should detect both permit-deny and rule contradiction
	if len(conflicts) < 2 {
		t.Fatalf("Expected at least 2 conflicts, got %d", len(conflicts))
	}

	// Verify conflicts are sorted by severity (critical first)
	for i := 1; i < len(conflicts); i++ {
		prev := getSeverityOrder(conflicts[i-1].Severity)
		curr := getSeverityOrder(conflicts[i].Severity)
		if prev > curr {
			t.Errorf("Conflicts not sorted by severity: %s before %s", conflicts[i-1].Severity, conflicts[i].Severity)
		}
	}
}

// TestAnalyzePolicies verifies full policy analysis with diagnostics
func TestAnalyzePolicies(t *testing.T) {
	policies := []Policy{
		{
			ID:       "p1",
			Subjects: []string{"user:test"},
			Rules: []Rule{
				{ID: "r1", Actions: []string{"view"}, Resources: []string{"/api"}, Effect: "allow"},
			},
		},
		{
			ID:       "p2",
			Subjects: []string{"user:test"},
			Rules: []Rule{
				{ID: "r2", Actions: []string{"view"}, Resources: []string{"/api"}, Effect: "deny"},
			},
		},
	}

	diag := AnalyzePolicies(policies, DenyOverridesStrategy{})

	if diag.TotalConflicts == 0 {
		t.Error("Expected conflicts to be detected")
	}

	if diag.PolicyCount != 2 {
		t.Errorf("Expected 2 policies, got %d", diag.PolicyCount)
	}

	if diag.Strategy != "deny_overrides" {
		t.Errorf("Expected strategy 'deny_overrides', got %s", diag.Strategy)
	}

	if diag.CriticalCount+diag.HighCount+diag.MediumCount+diag.LowCount != diag.TotalConflicts {
		t.Error("Severity counts don't match total conflicts")
	}

	if diag.CriticalCount > 0 && len(diag.RecommendedActions) == 0 {
		t.Error("Expected recommended actions for critical conflicts")
	}

	if time.Since(diag.GeneratedAt) > time.Second {
		t.Error("GeneratedAt timestamp seems incorrect")
	}
}

// TestCombineWithDiagnostics_DenyOverrides verifies deny-overrides diagnostics
func TestCombineWithDiagnostics_DenyOverrides(t *testing.T) {
	steps := []EvaluationStep{
		{PolicyID: "policy-allow", Effect: "allow", Matched: true},
		{PolicyID: "policy-deny", Effect: "deny", Matched: true},
	}

	strategy := DenyOverridesStrategy{}
	effect, allowIDs, denyIDs, reason, conflicts := strategy.CombineWithDiagnostics(steps, nil)

	if effect != EffectDeny {
		t.Errorf("Expected EffectDeny, got %v", effect)
	}

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	c := conflicts[0]
	if c.Type != ConflictPermitDeny {
		t.Errorf("Expected ConflictPermitDeny, got %s", c.Type)
	}

	if c.Severity != SeverityHigh {
		t.Errorf("Expected SeverityHigh, got %s", c.Severity)
	}

	if len(c.PolicyIDs) != 2 {
		t.Errorf("Expected 2 policy IDs in conflict, got %d", len(c.PolicyIDs))
	}

	if len(allowIDs) != 1 || len(denyIDs) != 1 {
		t.Errorf("Expected 1 allow and 1 deny policy, got %d allow, %d deny", len(allowIDs), len(denyIDs))
	}

	if reason != denyPolicyReason {
		t.Errorf("Expected reason '%s', got '%s'", denyPolicyReason, reason)
	}
}

// TestCombineWithDiagnostics_PermitOverrides verifies permit-overrides diagnostics
func TestCombineWithDiagnostics_PermitOverrides(t *testing.T) {
	steps := []EvaluationStep{
		{PolicyID: "policy-allow", Effect: "allow", Matched: true},
		{PolicyID: "policy-deny", Effect: "deny", Matched: true},
	}

	strategy := PermitOverridesStrategy{}
	effect, allowIDs, denyIDs, reason, conflicts := strategy.CombineWithDiagnostics(steps, nil)

	if effect != EffectAllow {
		t.Errorf("Expected EffectAllow, got %v", effect)
	}

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	c := conflicts[0]
	if c.Severity != SeverityCritical {
		t.Errorf("Expected SeverityCritical for permit-overrides, got %s", c.Severity)
	}

	if len(allowIDs) != 1 || len(denyIDs) != 1 {
		t.Errorf("Expected 1 allow and 1 deny policy, got %d allow, %d deny", len(allowIDs), len(denyIDs))
	}

	if reason != allowPolicyReason {
		t.Errorf("Expected reason '%s', got '%s'", allowPolicyReason, reason)
	}
}

// TestCombineWithDiagnostics_FirstApplicable verifies first-applicable diagnostics
func TestCombineWithDiagnostics_FirstApplicable(t *testing.T) {
	steps := []EvaluationStep{
		{PolicyID: "policy-1", Effect: "deny", Matched: true},
		{PolicyID: "policy-2", Effect: "allow", Matched: true},
		{PolicyID: "policy-3", Effect: "allow", Matched: true},
	}

	strategy := FirstApplicableStrategy{}
	effect, allowIDs, denyIDs, _, conflicts := strategy.CombineWithDiagnostics(steps, nil)

	if effect != EffectDeny {
		t.Errorf("Expected EffectDeny (first policy), got %v", effect)
	}

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict (priority ambiguity), got %d", len(conflicts))
	}

	c := conflicts[0]
	if c.Type != ConflictPriorityAmbiguity {
		t.Errorf("Expected ConflictPriorityAmbiguity, got %s", c.Type)
	}

	if c.Severity != SeverityMedium {
		t.Errorf("Expected SeverityMedium, got %s", c.Severity)
	}

	if len(c.PolicyIDs) != 3 {
		t.Errorf("Expected 3 policy IDs in conflict, got %d", len(c.PolicyIDs))
	}

	if len(denyIDs) != 1 || denyIDs[0] != "policy-1" {
		t.Errorf("Expected deny policy 'policy-1', got %v", denyIDs)
	}

	if len(allowIDs) != 0 {
		t.Errorf("Expected no allow policies for first-deny, got %v", allowIDs)
	}
}

// TestCombineWithDiagnostics_NoConflict verifies no conflicts when only one effect
func TestCombineWithDiagnostics_NoConflict(t *testing.T) {
	steps := []EvaluationStep{
		{PolicyID: "policy-1", Effect: "allow", Matched: true},
		{PolicyID: "policy-2", Effect: "allow", Matched: true},
	}

	strategy := DenyOverridesStrategy{}
	effect, _, _, _, conflicts := strategy.CombineWithDiagnostics(steps, nil)

	if effect != EffectAllow {
		t.Errorf("Expected EffectAllow, got %v", effect)
	}

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts for same-effect policies, got %d", len(conflicts))
	}
}

// TestHasOverlap verifies overlap detection utility
func TestHasOverlap(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{"exact match", []string{"read"}, []string{"read"}, true},
		{"wildcard in a", []string{"*"}, []string{"write"}, true},
		{"wildcard in b", []string{"delete"}, []string{"*"}, true},
		{"no overlap", []string{"read"}, []string{"write"}, false},
		{"partial overlap", []string{"read", "write"}, []string{"write", "delete"}, true},
		{"empty a", []string{}, []string{"read"}, false},
		{"empty b", []string{"read"}, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOverlap(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("hasOverlap(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Helper function for severity ordering
func getSeverityOrder(s ConflictSeverity) int {
	order := map[ConflictSeverity]int{
		SeverityCritical: 0,
		SeverityHigh:     1,
		SeverityMedium:   2,
		SeverityLow:      3,
	}
	return order[s]
}
