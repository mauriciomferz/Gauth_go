# PDP Conflict Diagnostics (P0.2)

## Overview

The **Conflict Diagnostics** system provides comprehensive policy conflict detection and resolution reporting for the Policy Decision Point (PDP) engine. This capability addresses RFC-0111 § 3.8.1 requirements for transparent policy conflict identification and resolution.

## Features

### Conflict Detection Types

1. **Permit-Deny Conflicts** (`ConflictPermitDeny`)
   - Detects when one policy permits while another denies the same request
   - Severity: `Critical` for specific subjects, `High` for wildcard subjects
   - Example: Policy A allows `user:alice` to `read /documents/*`, Policy B denies the same

2. **Scope Overlap Conflicts** (`ConflictScopeOverlap`)
   - Identifies policies with overlapping subjects/actions/resources
   - Severity: `High` for different effects, `Medium` for same effects
   - Example: Two policies both apply to `read /files/*` but with different conditions

3. **Priority Ambiguity** (`ConflictPriorityAmbiguity`)
   - Detects when multiple policies match with unclear ordering (first-applicable strategy)
   - Severity: `Medium`
   - Example: Multiple policies match but resolution depends on evaluation order

4. **Rule Contradictions** (`ConflictRuleContradiction`)
   - Finds contradictory rules within a single policy
   - Severity: `High`
   - Example: Policy has Rule A allowing `delete /temp/*` and Rule B denying the same

## Architecture

### Core Components

```go
// PolicyConflict represents a detected conflict
type PolicyConflict struct {
    ID              string
    Type            ConflictType
    Severity        ConflictSeverity
    PolicyIDs       []string
    RuleIDs         []string
    Subject         string
    Action          string
    Resource        string
    Description     string
    Recommendation  string
    DetectedAt      time.Time
    ResolutionHint  string
    AffectedRequest *Request
}

// ConflictDiagnostics provides structured analysis
type ConflictDiagnostics struct {
    TotalConflicts      int
    CriticalCount       int
    HighCount           int
    MediumCount         int
    LowCount            int
    Conflicts           []PolicyConflict
    Strategy            string
    PolicyCount         int
    GeneratedAt         time.Time
    RecommendedActions  []string
}
```

### Enhanced CombiningStrategy Interface

All combining strategies now implement `CombineWithDiagnostics()`:

```go
type CombiningStrategy interface {
    Combine(steps []EvaluationStep) (Effect, []string, []string, string)
    CombineWithDiagnostics(steps []EvaluationStep, policies []Policy) (Effect, []string, []string, string, []PolicyConflict)
    Name() string
}
```

## Usage

### 1. Static Policy Analysis

Analyze policies for conflicts before deployment:

```go
policies := []Policy{
    {
        ID:       "policy-allow-read",
        Subjects: []string{"user:alice"},
        Rules: []Rule{
            {Actions: []string{"read"}, Resources: []string{"/docs/*"}, Effect: "allow"},
        },
    },
    {
        ID:       "policy-deny-read",
        Subjects: []string{"user:alice"},
        Rules: []Rule{
            {Actions: []string{"read"}, Resources: []string{"/docs/*"}, Effect: "deny"},
        },
    },
}

strategy := DenyOverridesStrategy{}
diagnostics := AnalyzePolicies(policies, strategy)

fmt.Printf("Total conflicts: %d\n", diagnostics.TotalConflicts)
fmt.Printf("Critical: %d, High: %d, Medium: %d, Low: %d\n",
    diagnostics.CriticalCount, diagnostics.HighCount,
    diagnostics.MediumCount, diagnostics.LowCount)

for _, conflict := range diagnostics.Conflicts {
    fmt.Printf("[%s] %s\n", conflict.Severity, conflict.Description)
    fmt.Printf("  Recommendation: %s\n", conflict.Recommendation)
}
```

### 2. Runtime Conflict Detection

Detect conflicts during policy evaluation:

```go
steps := []EvaluationStep{
    {PolicyID: "policy-allow", Effect: "allow", Matched: true},
    {PolicyID: "policy-deny", Effect: "deny", Matched: true},
}

strategy := DenyOverridesStrategy{}
effect, allowIDs, denyIDs, reason, conflicts := strategy.CombineWithDiagnostics(steps, policies)

if len(conflicts) > 0 {
    for _, c := range conflicts {
        log.Printf("Conflict detected: %s - %s", c.Type, c.Description)
        log.Printf("Resolution: %s", c.ResolutionHint)
    }
}
```

### 3. Integration with Decision Output

Conflicts are included in the Decision struct:

```go
type Decision struct {
    Allow        bool
    Reason       string
    Policies     []string
    DenyPolicies []string
    Obligations  []Obligation
    Trace        []EvaluationStep
    Metadata     map[string]string
    Conflicts    []PolicyConflict  // P0.2: Conflict diagnostics
    HasConflicts bool              // P0.2: Quick check flag
}
```

## Severity Levels

| Severity | Description | Action Required |
|----------|-------------|-----------------|
| `Critical` | Conflicts that may lead to security vulnerabilities (e.g., permit-override allows access despite deny policies) | **URGENT**: Review immediately |
| `High` | Conflicts causing unexpected denials or ambiguous resolutions | Review within 24 hours |
| `Medium` | Conflicts reducing policy clarity or causing redundancy | Review during next policy audit |
| `Low` | Informational conflicts that don't impact functionality | Optional review |

## Combining Strategy Behavior

### Deny-Overrides Strategy

- **Resolution**: Deny if any deny policy matches
- **Conflict Detection**: Reports permit-deny conflicts with recommendation to make deny policies more specific
- **Example Output**:
  ```
  [HIGH] Deny-overrides resolved conflict: 2 policies allowed, 1 policy denied
  Recommendation: Deny policies took precedence. Consider making deny policies 
  more specific or removing redundant allow policies.
  Resolution Hint: DENY effect applied (deny-overrides strategy)
  ```

### Permit-Overrides Strategy

- **Resolution**: Allow if any allow policy matches
- **Conflict Detection**: Reports permit-deny conflicts as `CRITICAL` (security concern)
- **Example Output**:
  ```
  [CRITICAL] Permit-overrides resolved conflict: 1 policy allowed, 2 policies denied
  Recommendation: Allow policies took precedence. Consider adding mandatory 
  obligations for audit logging or making allow policies more restrictive.
  Resolution Hint: ALLOW effect applied (permit-overrides strategy) - ensure this is intended
  ```

### First-Applicable Strategy

- **Resolution**: Use first matching policy
- **Conflict Detection**: Reports priority ambiguity when multiple policies match
- **Example Output**:
  ```
  [MEDIUM] First-applicable strategy: 3 policies matched, using first (policy-admin-deny)
  Recommendation: Consider making policies mutually exclusive or using explicit 
  priority ordering to avoid ambiguity.
  Resolution Hint: Policy policy-admin-deny applied; remaining 2 policies ignored
  ```

## Best Practices

### 1. Pre-Deployment Validation

Always run `AnalyzePolicies()` before deploying new policies:

```go
// In CI/CD pipeline
diagnostics := AnalyzePolicies(newPolicies, strategy)
if diagnostics.CriticalCount > 0 {
    return errors.New("critical policy conflicts detected - deployment blocked")
}
if diagnostics.HighCount > 5 {
    log.Warn("High number of high-severity conflicts - review recommended")
}
```

### 2. Monitoring Runtime Conflicts

Track conflict metrics in production:

```go
if decision.HasConflicts {
    metrics.ConflictDetected(decision.Conflicts)
    for _, c := range decision.Conflicts {
        auditLog.Record(c.ID, c.Type, c.Severity, c.PolicyIDs)
    }
}
```

### 3. Policy Consolidation

If conflicts exceed 50% of policy count:

```go
if diagnostics.TotalConflicts > diagnostics.PolicyCount/2 {
    // Trigger policy consolidation workflow
    suggestPolicyMerge(diagnostics.Conflicts)
}
```

### 4. Handling Recommendations

Process recommended actions programmatically:

```go
for _, action := range diagnostics.RecommendedActions {
    switch {
    case strings.Contains(action, "URGENT"):
        notifySecurityTeam(action)
    case strings.Contains(action, "consolidation"):
        schedulePolicyAudit()
    }
}
```

## Examples

### Example 1: Permit-Deny Conflict with Deny-Overrides

**Policies**:
```json
{
  "policies": [
    {
      "id": "allow-engineers-read",
      "subjects": ["group:engineers"],
      "rules": [{"actions": ["read"], "resources": ["/api/v1/*"], "effect": "allow"}]
    },
    {
      "id": "deny-api-v1-deprecated",
      "subjects": ["*"],
      "rules": [{"actions": ["read"], "resources": ["/api/v1/*"], "effect": "deny"}]
    }
  ]
}
```

**Conflict Detected**:
```
ID: conflict-1
Type: permit_deny
Severity: high
PolicyIDs: [allow-engineers-read, deny-api-v1-deprecated]
Description: Policies allow-engineers-read permit while policies 
  deny-api-v1-deprecated deny for subject=group:engineers, action=read, 
  resource=/api/v1/*
Recommendation: Deny-overrides strategy will DENY (policies 
  deny-api-v1-deprecated take precedence). Consider: 1) Making deny 
  policies more specific, 2) Removing redundant allow policies, 3) Using 
  expressions to disambiguate
ResolutionHint: DENY (deny policies override)
```

### Example 2: Rule Contradiction

**Policy**:
```json
{
  "id": "policy-temp-files",
  "subjects": ["user:admin"],
  "rules": [
    {"id": "allow-delete", "actions": ["delete"], "resources": ["/temp/*"], "effect": "allow"},
    {"id": "deny-delete", "actions": ["delete"], "resources": ["/temp/*"], "effect": "deny"}
  ]
}
```

**Conflict Detected**:
```
ID: contradiction-1
Type: rule_contradiction
Severity: high
PolicyIDs: [policy-temp-files]
RuleIDs: [allow-delete, deny-delete]
Description: Policy policy-temp-files has contradicting rules allow-delete 
  (allow) and deny-delete (deny)
Recommendation: Refactor rules to be mutually exclusive or remove contradiction
ResolutionHint: Consider using more specific action/resource patterns or expressions
```

### Example 3: Priority Ambiguity (First-Applicable)

**Policies**:
```json
{
  "policies": [
    {"id": "p1", "subjects": ["user:bob"], "rules": [{"actions": ["write"], "resources": ["/data"], "effect": "deny"}]},
    {"id": "p2", "subjects": ["user:bob"], "rules": [{"actions": ["write"], "resources": ["/data"], "effect": "allow"}]},
    {"id": "p3", "subjects": ["user:bob"], "rules": [{"actions": ["write"], "resources": ["/data"], "effect": "allow"}]}
  ]
}
```

**Conflict Detected**:
```
ID: runtime-conflict-1
Type: priority_ambiguity
Severity: medium
PolicyIDs: [p1, p2, p3]
Description: First-applicable strategy: 3 policies matched, using first (p1)
Recommendation: Consider making policies mutually exclusive or using explicit 
  priority ordering to avoid ambiguity.
ResolutionHint: Policy p1 applied; remaining 2 policies ignored
```

## Metrics

Track conflict diagnostics metrics:

```prometheus
# Total conflicts detected
gauth_pdp_conflicts_total{type="permit_deny"} 12
gauth_pdp_conflicts_total{type="scope_overlap"} 5
gauth_pdp_conflicts_total{type="rule_contradiction"} 2

# Conflicts by severity
gauth_pdp_conflicts_by_severity{severity="critical"} 3
gauth_pdp_conflicts_by_severity{severity="high"} 8
gauth_pdp_conflicts_by_severity{severity="medium"} 6
gauth_pdp_conflicts_by_severity{severity="low"} 2

# Resolution strategy usage
gauth_pdp_strategy_conflicts{strategy="deny_overrides"} 10
gauth_pdp_strategy_conflicts{strategy="permit_overrides"} 5
gauth_pdp_strategy_conflicts{strategy="first_applicable"} 4
```

## Environment Variables

None required - conflict diagnostics are always enabled and add minimal overhead (~2-5% evaluation time).

## Performance

- **Static Analysis**: O(n²) for n policies (acceptable for typical policy counts < 1000)
- **Runtime Detection**: O(m) for m evaluation steps (negligible overhead)
- **Memory**: ~200 bytes per conflict detected

## Migration Path

### Phase 1: Static Analysis (Week 1-2)
1. Run `AnalyzePolicies()` on existing policy sets
2. Review and categorize conflicts
3. Fix critical and high-severity conflicts

### Phase 2: Monitoring (Week 3-4)
1. Enable runtime conflict logging in production
2. Track conflict metrics
3. Identify frequently conflicting policies

### Phase 3: Automation (Week 5-6)
1. Integrate conflict checks into CI/CD
2. Block deployments with critical conflicts
3. Auto-generate policy consolidation recommendations

## Testing

All conflict detection features are covered by 12+ test scenarios:

```bash
# Run conflict diagnostics tests
go test ./pkg/pdp -run TestDetect -v
go test ./pkg/pdp -run TestCombineWithDiagnostics -v
go test ./pkg/pdp -run TestAnalyzePolicies -v

# All tests
go test ./pkg/pdp -v
```

## References

- RFC-0111 § 3.8.1: Conflict Resolution Requirements
- RFC-0111 § 3.5: Combining Algorithms
- `pkg/pdp/engine.go`: Core PDP implementation
- `pkg/pdp/conflict_diagnostics.go`: Conflict detection engine
- `pkg/pdp/conflict_diagnostics_test.go`: Test suite
- `docs/P0_IMPLEMENTATION_PLAN.md`: P0.2 implementation plan

## Changelog

**v0.2.0 (2025-01-19)** - P0.2 Implementation
- Added comprehensive conflict detection (permit-deny, scope overlap, rule contradictions, priority ambiguity)
- Enhanced `CombiningStrategy` interface with `CombineWithDiagnostics()`
- Implemented conflict detection for all three combining strategies
- Added severity levels (Critical, High, Medium, Low) with recommendations
- Created 12+ test scenarios with 100% pass rate
- Integrated diagnostics into `Decision` struct

## License

Copyright © 2024 Gimel Foundation. Licensed under Apache 2.0.
