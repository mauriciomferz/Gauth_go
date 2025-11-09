// Package delegation provides delegation chain management for GAuth authorization.
//
// This package implements authorization delegation, allowing principals to
// delegate their authority to other principals. It supports delegation chains,
// constraint propagation, and revocation of delegation rights.
//
// # Delegation Concepts
//
// Delegation allows a principal (delegator) to grant authorization to another
// principal (delegate) to act on their behalf. Key concepts:
//   - Delegator: Original authority holder
//   - Delegate: Principal receiving delegated authority
//   - Delegation chain: Series of delegations (A -> B -> C)
//   - Constraints: Restrictions on delegated authority
//
// # Creating Delegations
//
// Simple delegation:
//
//	mgr := delegation.NewManager(keyProvider, replayStore)
//
//	// Alice delegates to Bob
//	token, err := mgr.CreateDelegation(&delegation.Context{
//	    Delegator: "user:alice",
//	    Delegate:  "user:bob",
//	    Resource:  "document:123",
//	    Actions:   []string{"read"},
//	    NotBefore: time.Now(),
//	    NotAfter:  time.Now().Add(24 * time.Hour),
//	})
//
// # Delegation Chains
//
// Create multi-hop delegations:
//
//	// Alice -> Bob
//	token1, _ := mgr.CreateDelegation(&delegation.Context{
//	    Delegator: "user:alice",
//	    Delegate:  "user:bob",
//	    Resource:  "document:*",
//	    Actions:   []string{"read", "write"},
//	})
//
//	// Bob -> Charlie (using Alice's delegation)
//	token2, _ := mgr.CreateDelegationChain(token1, &delegation.Context{
//	    Delegator: "user:bob",
//	    Delegate:  "user:charlie",
//	    Resource:  "document:123",  // Must be subset of Bob's authority
//	    Actions:   []string{"read"}, // Must be subset of Bob's actions
//	})
//
// Chain validation ensures:
//   - Each link is properly signed
//   - Authority narrows at each hop
//   - No loops in the chain
//   - All links are unexpired
//
// # Constraints
//
// Add constraints to delegations:
//
//	token, _ := mgr.CreateDelegation(&delegation.Context{
//	    Delegator: "user:alice",
//	    Delegate:  "user:bob",
//	    Resource:  "document:*",
//	    Actions:   []string{"read"},
//	    Constraints: map[string]interface{}{
//	        "max_delegation_depth": 2,      // Max chain length
//	        "ip_whitelist":        []string{"192.168.1.0/24"},
//	        "time_window":         "09:00-17:00",
//	        "geo_restriction":     "US",
//	        "mfa_required":        true,
//	    },
//	})
//
// Constraints are inherited and enforced throughout the chain.
//
// # Validating Delegations
//
// Validate delegation tokens:
//
//	validator := delegation.NewValidator(keyProvider, revocationMgr)
//
//	// Validate delegation
//	chain, err := validator.Validate(token)
//	if err != nil {
//	    switch {
//	    case errors.Is(err, delegation.ErrInvalidChain):
//	        // Chain validation failed
//	    case errors.Is(err, delegation.ErrAuthorityExceeded):
//	        // Delegate exceeded delegator's authority
//	    case errors.Is(err, delegation.ErrConstraintViolation):
//	        // Constraint not satisfied
//	    }
//	}
//
//	// Access chain information
//	fmt.Printf("Chain length: %d\n", len(chain.Links))
//	fmt.Printf("Original delegator: %s\n", chain.Root)
//	fmt.Printf("Final delegate: %s\n", chain.Leaf)
//
// # Revocation
//
// Revoke delegations:
//
//	// Revoke specific delegation
//	err := mgr.Revoke(tokenID, "Security incident")
//
//	// Revoke all delegations by a principal
//	err := mgr.RevokeByPrincipal("user:bob")
//
//	// Revoke delegation chain from a point
//	err := mgr.RevokeChain(tokenID) // Revokes this and all downstream
//
// # Authority Narrowing
//
// Ensure delegates cannot exceed delegator's authority:
//
//	// Alice has: document:*, actions=[read,write]
//	aliceToken, _ := mgr.CreateDelegation(...)
//
//	// Bob can only delegate subset
//	bobToken, _ := mgr.CreateDelegationChain(aliceToken, &delegation.Context{
//	    Resource: "document:123",  // ✓ Subset of document:*
//	    Actions:  []string{"read"}, // ✓ Subset of [read,write]
//	})
//
//	// This would fail:
//	invalidToken, err := mgr.CreateDelegationChain(aliceToken, &delegation.Context{
//	    Resource: "system:*",        // ✗ Outside document:*
//	    Actions:  []string{"delete"}, // ✗ Not in [read,write]
//	})
//	// err == delegation.ErrAuthorityExceeded
//
// # Time-Bounded Delegations
//
// Create temporary delegations:
//
//	token, _ := mgr.CreateDelegation(&delegation.Context{
//	    Delegator: "user:alice",
//	    Delegate:  "user:bob",
//	    Resource:  "document:123",
//	    Actions:   []string{"read"},
//	    NotBefore: time.Now(),
//	    NotAfter:  time.Now().Add(1 * time.Hour), // Expires in 1 hour
//	})
//
// Child delegations cannot exceed parent's time bounds.
//
// # Performance Optimization
//
//   - Cache validated chains
//   - Use memory replay stores for short-lived delegations
//   - Batch revocation checks
//   - Pre-validate constraints before signing
//
// # Security Considerations
//
//   - Limit delegation chain depth (recommended: 3-5 hops max)
//   - Validate entire chain on every use
//   - Implement time bounds on all delegations
//   - Log all delegation creations and uses
//   - Monitor for suspicious delegation patterns
//   - Revoke immediately on security incidents
//   - Audit delegation chains regularly
//
// # Thread Safety
//
// All Manager and Validator methods are safe for concurrent use.
package delegation
