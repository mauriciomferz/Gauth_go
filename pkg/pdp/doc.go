// Package pdp provides the Policy Decision Point for AgentAuth authorization.
//
// The PDP is the central authorization decision engine that evaluates policies
// against authorization requests. It supports ABAC (Attribute-Based Access Control),
// RBAC (Role-Based Access Control), and hybrid models with advanced policy
// evaluation including obligations and advice.
//
// # PDP Architecture
//
// The PDP follows the XACML architecture:
//   - PEP (Policy Enforcement Point) - Intercepts requests
//   - PDP (Policy Decision Point) - Makes authorization decisions
//   - PAP (Policy Administration Point) - Manages policies
//   - PIP (Policy Information Point) - Provides attributes
//
// # Basic Usage
//
// Create and use a PDP engine:
//
//	engine := pdp.NewEngine(policyStore, keyProvider)
//
//	// Create authorization request
//	req := &pdp.Request{
//	    Subject:  "user:alice",
//	    Resource: "document:123",
//	    Action:   "read",
//	    Context: map[string]interface{}{
//	        "time":       time.Now(),
//	        "ip_address": "192.168.1.100",
//	        "department": "engineering",
//	    },
//	}
//
//	// Evaluate
//	decision := engine.Evaluate(req)
//
//	switch decision.Effect {
//	case pdp.Allow:
//	    // Authorization granted
//	    for _, obligation := range decision.Obligations {
//	        // Execute obligations (e.g., audit logging)
//	    }
//	case pdp.Deny:
//	    // Authorization denied
//	    log.Printf("Denied: %s", decision.Reason)
//	case pdp.NotApplicable:
//	    // No policies matched
//	    log.Printf("No applicable policies")
//	}
//
// # Policy Evaluation
//
// The PDP evaluates policies in multiple phases:
//
//  1. Policy retrieval - Find applicable policies
//  2. Target matching - Check if policy applies to request
//  3. Condition evaluation - Evaluate policy conditions
//  4. Effect determination - Combine policy effects
//  5. Obligation collection - Gather obligations
//
// Example with detailed evaluation:
//
//	req := &pdp.Request{
//	    Subject:  "user:alice",
//	    Resource: "document:confidential-123",
//	    Action:   "read",
//	    Context: map[string]interface{}{
//	        "clearance_level": "SECRET",
//	        "department":      "engineering",
//	        "time_of_day":     time.Now().Hour(),
//	    },
//	}
//
//	decision := engine.EvaluateWithTrace(req)
//
//	// Examine evaluation trace
//	for _, step := range decision.Trace {
//	    log.Printf("Policy %s: %s (reason: %s)",
//	        step.PolicyID, step.Effect, step.Reason)
//	}
//
// # ABAC Policies
//
// Attribute-based policies using request context:
//
//	policy := &pdp.Policy{
//	    ID:     "abac-clearance",
//	    Effect: pdp.Allow,
//	    Target: &pdp.Target{
//	        Resources: []string{"document:confidential:*"},
//	        Actions:   []string{"read"},
//	    },
//	    Condition: &pdp.Condition{
//	        Expression: `
//	            subject.clearance_level == "SECRET" &&
//	            resource.classification <= subject.clearance_level &&
//	            context.time_of_day >= 9 &&
//	            context.time_of_day <= 17
//	        `,
//	    },
//	}
//
// # RBAC Policies
//
// Role-based policies:
//
//	policy := &pdp.Policy{
//	    ID:     "rbac-admin",
//	    Effect: pdp.Allow,
//	    Target: &pdp.Target{
//	        Subjects:  []string{"role:admin"},
//	        Resources: []string{"system:*"},
//	        Actions:   []string{"*"},
//	    },
//	}
//
// # Policy Combining Algorithms
//
// Control how multiple policies are combined:
//
//	// Deny-overrides: Any deny trumps all allows
//	engine.SetCombiningAlgorithm(pdp.DenyOverrides)
//
//	// Permit-overrides: Any allow trumps all denies
//	engine.SetCombiningAlgorithm(pdp.PermitOverrides)
//
//	// First-applicable: Use first matching policy
//	engine.SetCombiningAlgorithm(pdp.FirstApplicable)
//
//	// Only-one-applicable: Error if multiple match
//	engine.SetCombiningAlgorithm(pdp.OnlyOneApplicable)
//
// # Obligations and Advice
//
// Policies can include obligations (must be fulfilled) and advice (optional):
//
//	policy := &pdp.Policy{
//	    ID:     "audit-required",
//	    Effect: pdp.Allow,
//	    Target: &pdp.Target{
//	        Resources: []string{"sensitive:*"},
//	    },
//	    Obligations: []pdp.Obligation{
//	        {
//	            ID:   "log-access",
//	            Type: "audit",
//	            Attributes: map[string]interface{}{
//	                "severity": "HIGH",
//	                "retention": "7y",
//	            },
//	        },
//	    },
//	    Advice: []pdp.Advice{
//	        {
//	            ID:   "notify-owner",
//	            Type: "notification",
//	        },
//	    },
//	}
//
//	decision := engine.Evaluate(req)
//	if decision.Effect == pdp.Allow {
//	    // Execute obligations (required)
//	    for _, obl := range decision.Obligations {
//	        obligationExecutor.Execute(obl)
//	    }
//	    // Execute advice (optional)
//	    for _, adv := range decision.Advice {
//	        adviceExecutor.Execute(adv)
//	    }
//	}
//
// # Attribute Providers (PIP)
//
// Extend context with additional attributes:
//
//	// Register attribute provider
//	engine.RegisterAttributeProvider("user", func(id string) map[string]interface{} {
//	    user := userService.Get(id)
//	    return map[string]interface{}{
//	        "department":      user.Department,
//	        "clearance_level": user.ClearanceLevel,
//	        "manager":         user.ManagerID,
//	    }
//	})
//
//	// Attributes automatically enriched during evaluation
//	req := &pdp.Request{
//	    Subject: "user:alice",
//	    // ... other fields
//	}
//	// PDP automatically fetches user attributes
//
// # Caching
//
// Enable decision caching for performance:
//
//	cache := pdp.NewDecisionCache(1000, 5*time.Minute)
//	engine.SetCache(cache)
//
//	// Subsequent identical requests use cache
//	decision1 := engine.Evaluate(req) // Cache miss, full evaluation
//	decision2 := engine.Evaluate(req) // Cache hit, instant return
//
// # Performance Optimization
//
//   - Enable decision caching (5-10 min TTL)
//   - Use efficient policy storage (indexed by target)
//   - Pre-compile policy expressions
//   - Batch attribute retrievals from PIPs
//   - Use short-circuit evaluation in conditions
//   - Monitor PDP metrics for slow policies
//
// # Security Considerations
//
//   - Validate all input attributes
//   - Sanitize attribute values to prevent injection
//   - Implement timeouts for PIP queries
//   - Log all authorization decisions
//   - Monitor for unusual access patterns
//   - Regularly audit policies
//   - Test policies before deployment
//
// # Thread Safety
//
// All Engine methods are safe for concurrent use.
package pdp
