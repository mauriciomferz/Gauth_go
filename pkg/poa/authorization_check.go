// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: MIT

package poa

import (
	"context"
	"fmt"
)

// AuthorizationChecker defines the interface for verifying a principal's actual permissions.
// This prevents privilege escalation attacks where a PoA requests scopes beyond what
// the grantor (principal) actually possesses.
//
// Security Context: Addresses Critical Vulnerability - Weak Delegation Constraints (CVE-2025-AGENTAUTH-005)
//
// Attack Scenario Prevented:
//  1. User "bob" has role="Editor" with permissions=["read", "write", "update"]
//  2. AI Agent "agent-malicious" requests PoA from Bob with scope=["read", "write", "admin", "delete"]
//  3. WITHOUT THIS CHECK: Signature validates (Bob signed it) → PoA granted → Agent has admin rights
//  4. WITH THIS CHECK: System queries Bob's actual permissions → ["read", "write", "update"]
//     → Requested scope ["admin", "delete"] NOT SUBSET → PoA rejected
//
// Implementation Requirements:
//   - MUST query authoritative source (RBAC system, database, IAM)
//   - MUST perform INTERSECTION check: requested_scope ⊆ principal_actual_permissions
//   - SHOULD cache results with short TTL (30-60s) for performance
//   - MUST fail-closed: If permission lookup fails, reject the PoA
type AuthorizationChecker interface {
	// GetPrincipalPermissions retrieves the complete set of permissions/scopes
	// that the principal (grantor) actually holds in the system.
	//
	// Parameters:
	//   - ctx: context for cancellation/timeout
	//   - tenantID: multi-tenant isolation
	//   - principalID: the grantor/user whose permissions are being checked
	//
	// Returns:
	//   - []string: List of permission scopes (e.g., ["read", "write", "update"])
	//   - error: Non-nil if permission lookup fails (system error, user not found, etc.)
	//
	// Security Note: This function MUST query real-time or cached (< 60s TTL) data,
	// not stale snapshots, to prevent TOCTOU vulnerabilities.
	GetPrincipalPermissions(ctx context.Context, tenantID, principalID string) ([]string, error)
}

// ValidateScopeAuthorization verifies that the requested PoA scope is a strict subset
// of the principal's actual permissions. This is the core authorization check that
// prevents privilege escalation attacks.
//
// Algorithm:
//  1. Query principal's actual permissions from authoritative source
//  2. For each requested action in PoA:
//     a. Check if action exists in principal's permission set
//     b. If ANY action is missing → REJECT (fail-closed)
//  3. If all actions are authorized → APPROVE
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - checker: AuthorizationChecker implementation (RBAC system, database, etc.)
//   - tenantID: multi-tenant isolation
//   - principalID: the grantor whose permissions are being validated
//   - requestedActions: the actions/scopes requested in the PoA
//
// Returns:
//   - bool: true if all requested actions are authorized, false otherwise
//   - []string: list of unauthorized actions (empty if valid=true)
//   - error: non-nil if permission lookup fails (treat as authorization failure)
//
// Example Usage:
//
//	valid, unauthorized, err := ValidateScopeAuthorization(
//	    ctx,
//	    authChecker,
//	    "tenant-123",
//	    "user-bob",
//	    []string{"read", "write", "admin"}, // PoA requests these
//	)
//	if err != nil {
//	    return fmt.Errorf("authorization check failed: %w", err)
//	}
//	if !valid {
//	    return fmt.Errorf("privilege escalation detected: unauthorized actions %v", unauthorized)
//	}
func ValidateScopeAuthorization(
	ctx context.Context,
	checker AuthorizationChecker,
	tenantID, principalID string,
	requestedActions []string,
) (bool, []string, error) {
	if checker == nil {
		return false, nil, fmt.Errorf("authorization checker not configured")
	}

	// Fetch principal's actual permissions from authoritative source
	principalPermissions, err := checker.GetPrincipalPermissions(ctx, tenantID, principalID)
	if err != nil {
		// Fail-closed: If we can't determine permissions, reject the request
		return false, nil, fmt.Errorf("failed to fetch principal permissions: %w", err)
	}

	// Build permission lookup map for O(1) checks
	permissionSet := make(map[string]bool, len(principalPermissions))
	for _, perm := range principalPermissions {
		permissionSet[perm] = true
	}

	// Check if all requested actions are subset of principal's permissions
	var unauthorized []string
	for _, action := range requestedActions {
		if !permissionSet[action] {
			unauthorized = append(unauthorized, action)
		}
	}

	if len(unauthorized) > 0 {
		return false, unauthorized, nil
	}

	return true, nil, nil
}

// DefaultAuthorizationChecker provides a simple database-backed implementation.
// For production use, integrate with your existing RBAC/IAM system.
type DefaultAuthorizationChecker struct {
	repo *Repository // Assumes Repository has method to query user permissions
}

// NewDefaultAuthorizationChecker creates a checker using the PoA repository.
func NewDefaultAuthorizationChecker(repo *Repository) *DefaultAuthorizationChecker {
	return &DefaultAuthorizationChecker{repo: repo}
}

// GetPrincipalPermissions queries the database for the user's assigned permissions.
// This is a placeholder implementation - integrate with your actual RBAC system.
func (d *DefaultAuthorizationChecker) GetPrincipalPermissions(ctx context.Context, tenantID, principalID string) ([]string, error) {
	// TODO: Replace with actual RBAC query
	// Example SQL:
	//   SELECT DISTINCT p.action
	//   FROM user_roles ur
	//   JOIN role_permissions rp ON ur.role_id = rp.role_id
	//   JOIN permissions p ON rp.permission_id = p.id
	//   WHERE ur.tenant_id = $1 AND ur.user_id = $2 AND ur.status = 'active'

	// Placeholder: Return empty set (fail-closed)
	return []string{}, fmt.Errorf("permission lookup not implemented - integrate with RBAC system")
}
