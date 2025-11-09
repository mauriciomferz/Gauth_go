// Package authz provides authorization enforcement and policy evaluation for GAuth.
//
// This package implements the authorization layer, including policy-based access
// control, decision caching, and obligation execution. It supports both
// role-based access control (RBAC) and attribute-based access control (ABAC)
// through a flexible policy evaluation engine.
//
// # Authorization Models
//
// The package supports multiple authorization models:
//   - RBAC (Role-Based Access Control) - Permission based on roles
//   - ABAC (Attribute-Based Access Control) - Permission based on attributes
//   - Hybrid models - Combination of RBAC and ABAC
//
// # Basic Authorization
//
// Simple authorization check:
//
//	authorizer := authz.NewMemoryAuthorizer()
//
//	// Add a policy
//	policy := &authz.Policy{
//	    ID:       "read-docs",
//	    Effect:   authz.EffectAllow,
//	    Subjects: []string{"user:alice"},
//	    Resources: []string{"document:*"},
//	    Actions:  []string{"read"},
//	}
//	authorizer.AddPolicy(policy)
//
//	// Check authorization
//	request := &authz.Request{
//	    Subject:  "user:alice",
//	    Resource: "document:123",
//	    Action:   "read",
//	}
//
//	decision, err := authorizer.Authorize(request)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if decision.Effect == authz.EffectAllow {
//	    // Authorization granted
//	}
//
// # Policy-Based Access Control
//
// Define complex policies with conditions:
//
//	policy := &authz.Policy{
//	    ID:       "time-based-access",
//	    Effect:   authz.EffectAllow,
//	    Subjects: []string{"role:employee"},
//	    Resources: []string{"system:*"},
//	    Actions:  []string{"access"},
//	    Conditions: []authz.Condition{
//	        {
//	            Type: "time-range",
//	            Parameters: map[string]interface{}{
//	                "start": "09:00",
//	                "end":   "17:00",
//	            },
//	        },
//	        {
//	            Type: "ip-whitelist",
//	            Parameters: map[string]interface{}{
//	                "ranges": []string{"192.168.1.0/24"},
//	            },
//	        },
//	    },
//	}
//
// # Pattern Matching
//
// Policies support wildcards and patterns:
//
//	// Match all documents
//	Resources: []string{"document:*"}
//
//	// Match specific prefix
//	Resources: []string{"document:confidential:*"}
//
//	// Match any resource type
//	Resources: []string{"*:123"}
//
// # Expression Evaluation
//
// Use expressions for complex authorization logic:
//
//	policy := &authz.Policy{
//	    ID:     "attr-based",
//	    Effect: authz.EffectAllow,
//	    Expression: `
//	        subject.role == "manager" &&
//	        resource.department == subject.department &&
//	        action == "approve"
//	    `,
//	}
//
// Supported operators:
//   - Comparison: ==, !=, <, >, <=, >=
//   - Logical: &&, ||, !
//   - Membership: in
//   - Parentheses for grouping
//
// # Decision Caching
//
// Enable caching for improved performance:
//
//	cache := authz.NewAuthorizationCache(1000, 5*time.Minute)
//	authorizer.SetDecisionCache(cache)
//
//	// Subsequent identical requests will use cached decisions
//	decision1, _ := authorizer.Authorize(request)
//	decision2, _ := authorizer.Authorize(request) // Cache hit
//
// Cache invalidation on policy changes:
//
//	// Add/remove policy automatically invalidates relevant cache entries
//	authorizer.AddPolicy(newPolicy)   // Cache invalidated
//	authorizer.RemovePolicy(policyID) // Cache invalidated
//
// # Obligation Execution
//
// Policies can include obligations that must be executed:
//
//	policy := &authz.Policy{
//	    ID:       "audit-required",
//	    Effect:   authz.EffectAllow,
//	    Subjects: []string{"user:*"},
//	    Resources: []string{"sensitive:*"},
//	    Actions:  []string{"read"},
//	    Obligations: []authz.Obligation{
//	        {
//	            Type: "log-access",
//	            Parameters: map[string]interface{}{
//	                "level": "audit",
//	            },
//	        },
//	        {
//	            Type: "notify-admin",
//	            Parameters: map[string]interface{}{
//	                "email": "security@example.com",
//	            },
//	        },
//	    },
//	}
//
//	// Set obligation executor
//	executor := authz.NewObligationExecutor()
//	authorizer.SetObligationExecutor(executor)
//
//	// Obligations are automatically executed after Allow decisions
//	decision, _ := authorizer.Authorize(request)
//	// Obligations executed if decision is Allow
//
// # Combining Policies
//
// Control how multiple policies are combined:
//
//	// Deny-overrides: Any deny trumps allows
//	authorizer.SetCombiningStrategy(authz.DenyOverrides)
//
//	// Allow-overrides: Any allow trumps denies
//	authorizer.SetCombiningStrategy(authz.AllowOverrides)
//
//	// Unanimous: All applicable policies must allow
//	authorizer.SetCombiningStrategy(authz.Unanimous)
//
// # Jurisdiction Support
//
// Different authorization rules for different jurisdictions:
//
//	authorizer.SetJurisdiction("US")
//
//	// Policies can be jurisdiction-specific
//	policy := &authz.Policy{
//	    ID:            "gdpr-access",
//	    Effect:        authz.EffectAllow,
//	    Jurisdictions: []string{"EU"},
//	    // ... other fields
//	}
//
// # Metrics and Monitoring
//
// Track authorization metrics:
//
//	metrics := authorizer.AuthorizationCacheMetrics()
//	fmt.Printf("Hit rate: %.2f%%\n", metrics.HitRate)
//	fmt.Printf("Total requests: %d\n", metrics.Requests)
//
// # Performance Optimization
//
// For high-throughput authorization:
//
//   - Enable decision caching (5-10 minute TTL recommended)
//   - Use memory-based policy stores for fast lookups
//   - Pre-compile policy expressions
//   - Batch authorization checks when possible
//   - Use simple patterns instead of complex expressions where possible
//
// # Security Considerations
//
//   - Always use explicit policy definitions (avoid overly broad wildcards)
//   - Implement policy version control and audit trails
//   - Regularly review and prune unused policies
//   - Use deny-overrides for security-critical resources
//   - Log all authorization decisions for security monitoring
//   - Implement rate limiting on authorization endpoints
//   - Validate policy expressions before deployment
//
// # Thread Safety
//
// All Authorizer methods are safe for concurrent use by multiple goroutines.
// The underlying policy store and decision cache must also be thread-safe.
//
// # Test Coverage
//
// This package has 84.3% test coverage, including:
//   - Policy evaluation with all combining strategies
//   - Pattern matching and wildcards
//   - Expression evaluation
//   - Decision caching and invalidation
//   - Obligation execution
//   - Error path coverage
//   - Concurrent access scenarios
package authz
