package pdp

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ConflictType categorizes the nature of policy conflicts
type ConflictType string

const (
	// ConflictPermitDeny occurs when one policy permits and another denies the same request
	ConflictPermitDeny ConflictType = "permit_deny"

	// ConflictScopeOverlap occurs when policies have overlapping subjects/actions/resources
	ConflictScopeOverlap ConflictType = "scope_overlap"

	// ConflictPriorityAmbiguity occurs when policies with same priority conflict
	ConflictPriorityAmbiguity ConflictType = "priority_ambiguity"

	// ConflictRuleContradiction occurs within a single policy when rules contradict
	ConflictRuleContradiction ConflictType = "rule_contradiction"
)

// ConflictSeverity indicates the impact level of a conflict
type ConflictSeverity string

const (
	SeverityCritical ConflictSeverity = "critical" // Conflicts that may lead to security vulnerabilities
	SeverityHigh     ConflictSeverity = "high"     // Conflicts that may cause unexpected denials
	SeverityMedium   ConflictSeverity = "medium"   // Conflicts that reduce policy clarity
	SeverityLow      ConflictSeverity = "low"      // Conflicts that are informational
)

// PolicyConflict represents a detected conflict between policies
type PolicyConflict struct {
	ID              string           `json:"id"`
	Type            ConflictType     `json:"type"`
	Severity        ConflictSeverity `json:"severity"`
	PolicyIDs       []string         `json:"policy_ids"`
	RuleIDs         []string         `json:"rule_ids,omitempty"`
	Subject         string           `json:"subject,omitempty"`
	Action          string           `json:"action,omitempty"`
	Resource        string           `json:"resource,omitempty"`
	Description     string           `json:"description"`
	Recommendation  string           `json:"recommendation"`
	DetectedAt      time.Time        `json:"detected_at"`
	ResolutionHint  string           `json:"resolution_hint,omitempty"`
	AffectedRequest *Request         `json:"affected_request,omitempty"`
}

// ConflictDetector analyzes policies for conflicts
type ConflictDetector struct {
	policies []Policy
	strategy CombiningStrategy
}

// NewConflictDetector creates a new conflict detection engine
func NewConflictDetector(policies []Policy, strategy CombiningStrategy) *ConflictDetector {
	return &ConflictDetector{
		policies: policies,
		strategy: strategy,
	}
}

// DetectConflicts performs comprehensive conflict analysis
func (cd *ConflictDetector) DetectConflicts() []PolicyConflict {
	var conflicts []PolicyConflict

	// Detect permit-deny conflicts
	conflicts = append(conflicts, cd.detectPermitDenyConflicts()...)

	// Detect scope overlaps
	conflicts = append(conflicts, cd.detectScopeOverlaps()...)

	// Detect rule contradictions within policies
	conflicts = append(conflicts, cd.detectRuleContradictions()...)

	// Sort by severity
	sort.Slice(conflicts, func(i, j int) bool {
		severityOrder := map[ConflictSeverity]int{
			SeverityCritical: 0,
			SeverityHigh:     1,
			SeverityMedium:   2,
			SeverityLow:      3,
		}
		return severityOrder[conflicts[i].Severity] < severityOrder[conflicts[j].Severity]
	})

	return conflicts
}

// detectPermitDenyConflicts finds policies that conflict on same request
func (cd *ConflictDetector) detectPermitDenyConflicts() []PolicyConflict {
	var conflicts []PolicyConflict
	conflictID := 0

	// Build a map of subject/action/resource combinations
	type triple struct{ subj, act, res string }
	allowPolicies := make(map[triple][]string)
	denyPolicies := make(map[triple][]string)

	for _, policy := range cd.policies {
		for _, subject := range policy.Subjects {
			for _, rule := range policy.Rules {
				for _, action := range rule.Actions {
					for _, resource := range rule.Resources {
						t := triple{subject, action, resource}

						if rule.Effect == outcomeAllow {
							allowPolicies[t] = append(allowPolicies[t], policy.ID)
						} else if rule.Effect == outcomeDeny {
							denyPolicies[t] = append(denyPolicies[t], policy.ID)
						}
					}
				}
			}
		}
	}

	// Find overlapping triples
	for t, allowIDs := range allowPolicies {
		if denyIDs, exists := denyPolicies[t]; exists {
			conflictID++
			severity := SeverityHigh

			// Critical if subject is not wildcard
			if t.subj != "*" {
				severity = SeverityCritical
			}

			allowIDs = append(allowIDs, denyIDs...)
			sort.Strings(allowIDs)

			conflicts = append(conflicts, PolicyConflict{
				ID:        fmt.Sprintf("conflict-%d", conflictID),
				Type:      ConflictPermitDeny,
				Severity:  severity,
				PolicyIDs: allowIDs,
				Subject:   t.subj,
				Action:    t.act,
				Resource:  t.res,
				Description: fmt.Sprintf("Policies %s permit while policies %s deny for subject=%s, action=%s, resource=%s",
					strings.Join(allowIDs, ","), strings.Join(denyIDs, ","), t.subj, t.act, t.res),
				Recommendation: cd.generatePermitDenyRecommendation(allowIDs, denyIDs),
				DetectedAt:     time.Now(),
				ResolutionHint: fmt.Sprintf("Strategy '%s' will resolve as: %s",
					cd.strategy.Name(), cd.predictResolution(allowIDs, denyIDs)),
			})
		}
	}

	return conflicts
}

// detectScopeOverlaps finds policies with overlapping scopes
func (cd *ConflictDetector) detectScopeOverlaps() []PolicyConflict {
	var conflicts []PolicyConflict
	conflictID := 0

	// Check each pair of policies for overlaps
	for i := 0; i < len(cd.policies); i++ {
		for j := i + 1; j < len(cd.policies); j++ {
			p1, p2 := cd.policies[i], cd.policies[j]

			// Check if subjects overlap
			if !hasOverlap(p1.Subjects, p2.Subjects) {
				continue
			}

			// Check if any rules overlap
			for _, r1 := range p1.Rules {
				for _, r2 := range p2.Rules {
					if hasOverlap(r1.Actions, r2.Actions) && hasOverlap(r1.Resources, r2.Resources) {
						// Different effects = higher severity
						severity := SeverityMedium
						if r1.Effect != r2.Effect {
							severity = SeverityHigh
						}

						conflictID++
						conflicts = append(conflicts, PolicyConflict{
							ID:             fmt.Sprintf("overlap-%d", conflictID),
							Type:           ConflictScopeOverlap,
							Severity:       severity,
							PolicyIDs:      []string{p1.ID, p2.ID},
							RuleIDs:        []string{r1.ID, r2.ID},
							Description:    fmt.Sprintf("Policies %s and %s have overlapping scopes with different effects", p1.ID, p2.ID),
							Recommendation: "Consider consolidating overlapping policies or making scopes mutually exclusive",
							DetectedAt:     time.Now(),
							ResolutionHint: "Review policy scope definitions and rule ordering",
						})
					}
				}
			}
		}
	}

	return conflicts
}

// detectRuleContradictions finds conflicting rules within single policies
func (cd *ConflictDetector) detectRuleContradictions() []PolicyConflict {
	var conflicts []PolicyConflict
	conflictID := 0

	for _, policy := range cd.policies {
		// Check each pair of rules within the policy
		for i := 0; i < len(policy.Rules); i++ {
			for j := i + 1; j < len(policy.Rules); j++ {
				r1, r2 := policy.Rules[i], policy.Rules[j]

				// Check if rules have overlapping scope but different effects
				if r1.Effect != r2.Effect &&
					hasOverlap(r1.Actions, r2.Actions) &&
					hasOverlap(r1.Resources, r2.Resources) {
					conflictID++
					conflicts = append(conflicts, PolicyConflict{
						ID:        fmt.Sprintf("contradiction-%d", conflictID),
						Type:      ConflictRuleContradiction,
						Severity:  SeverityHigh,
						PolicyIDs: []string{policy.ID},
						RuleIDs:   []string{r1.ID, r2.ID},
						Description: fmt.Sprintf("Policy %s has contradicting rules %s (%s) and %s (%s)",
							policy.ID, r1.ID, r1.Effect, r2.ID, r2.Effect),
						Recommendation: "Refactor rules to be mutually exclusive or remove contradiction",
						DetectedAt:     time.Now(),
						ResolutionHint: "Consider using more specific action/resource patterns or expressions",
					})
				}
			}
		}
	}

	return conflicts
}

// generatePermitDenyRecommendation generates specific recommendations for permit-deny conflicts
func (cd *ConflictDetector) generatePermitDenyRecommendation(allowIDs, denyIDs []string) string {
	switch cd.strategy.Name() {
	case "deny_overrides":
		return fmt.Sprintf("Deny-overrides strategy will DENY (policies %s take precedence). Consider: 1) Making deny policies more specific, 2) Removing redundant allow policies, 3) Using expressions to disambiguate", strings.Join(denyIDs, ","))
	case "permit_overrides":
		return fmt.Sprintf("Permit-overrides strategy will ALLOW (policies %s take precedence). Consider: 1) Making allow policies more specific, 2) Removing redundant deny policies, 3) Adding mandatory obligations for logging", strings.Join(allowIDs, ","))
	case "first_applicable":
		return "First-applicable strategy will use the first matching policy. Consider: 1) Reordering policies by priority, 2) Making policies mutually exclusive, 3) Using explicit ordering metadata"
	default:
		return "Review combining strategy behavior and adjust policy definitions accordingly"
	}
}

// predictResolution predicts how the combining strategy will resolve a conflict
func (cd *ConflictDetector) predictResolution(allowIDs, denyIDs []string) string {
	switch cd.strategy.Name() {
	case "deny_overrides":
		return "DENY (deny policies override)"
	case "permit_overrides":
		return "ALLOW (permit policies override)"
	case "first_applicable":
		return "depends on policy evaluation order"
	default:
		return "unknown"
	}
}

// hasOverlap checks if two string slices have any common elements
func hasOverlap(a, b []string) bool {
	for _, aVal := range a {
		for _, bVal := range b {
			if aVal == bVal || aVal == "*" || bVal == "*" {
				return true
			}
		}
	}
	return false
}

// ConflictDiagnostics provides structured conflict analysis results
type ConflictDiagnostics struct {
	TotalConflicts     int              `json:"total_conflicts"`
	CriticalCount      int              `json:"critical_count"`
	HighCount          int              `json:"high_count"`
	MediumCount        int              `json:"medium_count"`
	LowCount           int              `json:"low_count"`
	Conflicts          []PolicyConflict `json:"conflicts"`
	Strategy           string           `json:"strategy"`
	PolicyCount        int              `json:"policy_count"`
	GeneratedAt        time.Time        `json:"generated_at"`
	RecommendedActions []string         `json:"recommended_actions"`
}

// AnalyzePolicies performs full policy conflict analysis and returns diagnostics
func AnalyzePolicies(policies []Policy, strategy CombiningStrategy) *ConflictDiagnostics {
	detector := NewConflictDetector(policies, strategy)
	conflicts := detector.DetectConflicts()

	diag := &ConflictDiagnostics{
		TotalConflicts: len(conflicts),
		Conflicts:      conflicts,
		Strategy:       strategy.Name(),
		PolicyCount:    len(policies),
		GeneratedAt:    time.Now(),
	}

	// Count by severity
	for _, c := range conflicts {
		switch c.Severity {
		case SeverityCritical:
			diag.CriticalCount++
		case SeverityHigh:
			diag.HighCount++
		case SeverityMedium:
			diag.MediumCount++
		case SeverityLow:
			diag.LowCount++
		}
	}

	// Generate recommended actions
	if diag.CriticalCount > 0 {
		diag.RecommendedActions = append(diag.RecommendedActions,
			"URGENT: Address critical conflicts immediately - potential security vulnerabilities")
	}
	if diag.HighCount > 0 {
		diag.RecommendedActions = append(diag.RecommendedActions,
			"Review high-severity conflicts - may cause unexpected access denials")
	}
	if diag.TotalConflicts > diag.PolicyCount/2 {
		diag.RecommendedActions = append(diag.RecommendedActions,
			"Consider policy consolidation - high conflict ratio indicates fragmentation")
	}

	return diag
}
