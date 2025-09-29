// Package authz provides authorization functionality for the GAuth framework.
//
// # Quick Start
//
//	import "github.com/Gimel-Foundation/gauth/pkg/authz"
//
//	az := authz.New(nil)
//	req := &authz.Request{
//	    Subject:  "user123",
//	    Resource: "users",
//	    Action:   "create",
//	}
//	err := az.Check(ctx, req)
//	// ...
//
// See runnable examples in examples/authz/basic/main.go and examples/authz/advanced_policy_flow/main.go.
//
// # See Also
//
//   - package token: for extended token management and validation
//   - package audit: for audit trail and event logging
//   - package events: for event types and event bus
//
// For advanced integration and protocol flows, see the examples/ directory and the project README.
//
// # RFC111 Mapping
//
// This package implements the GAuth authorization protocol as defined in GiFo-RfC 0111 (September 2025):
//
//   - Power-of-attorney modeling: Types Subject, Resource, Action, Policy, AccessRequest,
//     and AccessResponse represent the core entities and relationships for power delegation
//     and enforcement.
//   - P*P architecture: Interfaces and logic support Power Enforcement Point (PEP),
//     Power Decision Point (PDP), Power Information Point (PIP), Power Administration
//     Point (PAP), and Power Verification Point (PVP) roles.
//   - Centralized authorization: All authorization decisions are enforced centrally;
//     decentralized/team-based delegation is explicitly prevented (see enforcement
//     in pkg/auth/extended_controls.go).
//   - Exclusions: Web3/blockchain, DNA-based identity, and AI-controlled GAuth are strictly excluded, as required by RFC111.
//   - Extended tokens, attestation, and versioning: Supported via integration with pkg/token and metadata types.
//   - Audit, compliance, and transparency: All actions are auditable and traceable, supporting verifiability and compliance.
//
// For more details, see the RFC111 summary in the project root and the README.md for usage patterns.
//
// The authz package implements fine-grained authorization through multiple
// mechanisms including RBAC (Role-Based Access Control), ABAC (Attribute-Based
// Access Control), PBAC (Policy-Based Access Control), and custom policy engines.
//
// Authorization is managed separately from authentication to maintain separation
// of concerns and enable more flexible security models.
package authz

//
// # Core Concepts
//
// 1. Policies
//
// Policies define authorization rules:
//
//	policy := authz.NewPolicy().
//		WithRole("admin").
//		WithResource("users").
//		WithAction("create").
//		Allow()
//
// 2. Roles
//
// Roles group permissions:
//
//	admin := authz.NewRole("admin").
//		AddPermission("users:*").
//		AddPermission("systems:read")
//
// 3. Permissions
//
// Permissions define allowed operations:
//
//	perm := authz.Permission{
//		Resource: "users",
//		Action:   "create",
//		Effect:   authz.Allow,
//	}
//
// # Usage Examples
//
// Basic authorization check:
//
//	authz := authz.New(config)
//	err := authz.Check(ctx, &authz.Request{
//		Subject:  "user123",
//		Resource: "users",
//		Action:   "create",
//	})
//
// Role-based check:
//
//	err := authz.CheckRole(ctx, "user123", "admin")
//
// Policy evaluation:
//
//	allowed := authz.Evaluate(ctx, policy, request)
//
// # Thread Safety
//
// All public methods are thread-safe and can be called concurrently.
//
// # Error Handling
//
// The package uses strongly typed errors:
//
//	var (
//		ErrUnauthorized      = errors.New("unauthorized")
//		ErrInsufficientScope = errors.New("insufficient scope")
//		ErrPolicyNotFound    = errors.New("policy not found")
//	)
//
// Check errors using errors.Is():
//
//	if err := authz.Check(ctx, req); err != nil {
//		if errors.Is(err, authz.ErrUnauthorized) {
//			// Handle unauthorized access
//		}
//	}
//
// # Best Practices
//
//  1. Policy Design
//     - Keep policies simple and focused
//     - Use hierarchical resources
//     - Implement least privilege
//
//  2. Role Management
//     - Define clear role hierarchies
//     - Limit role proliferation
//     - Regular role audits
//
//  3. Performance
//     - Cache policy decisions
//     - Use bulk permission checks
//     - Monitor evaluation time
//
//  4. Security
//     - Validate all inputs
//     - Log authorization decisions
//     - Regular policy reviews
//
// # Configuration
//
//	type Config struct {
//		// DefaultEffect is the default policy effect (Allow/Deny)
//		DefaultEffect Effect
//
//		// PolicyStore defines where policies are stored
//		PolicyStore PolicyStore
//
//		// RoleManager handles role assignments
//		RoleManager RoleManager
//
//		// Cache configures decision caching
//		Cache *Cache
//	}
//
// # Extensions
//
// The package can be extended through interfaces:
//
//  1. PolicyStore - Custom policy storage
//  2. RoleManager - Custom role management
//  3. Evaluator - Custom policy evaluation
//  4. Auditor - Custom audit logging
//
// # Metrics
//
// The package exports Prometheus metrics:
//
//  - authz_checks_total: Total number of authorization checks
//  - authz_check_errors_total: Total number of failed checks
//  - authz_cache_hits_total: Total number of cache hits
//  - authz_evaluation_duration: Policy evaluation duration
//
// # Integration
//
// The package integrates with:
//
//  1. Standard middleware
//  2. OAuth2/OIDC providers
//  3. External policy engines
//  4. Audit systems
//
