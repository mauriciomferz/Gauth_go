// Copyright 2025 Gimel Foundation
// SPDX-License-Identifier: Apache-2.0

package poa

import (
	"context"
	"testing"
)

// mockAuthorizationChecker for testing
type mockAuthorizationChecker struct {
	permissions map[string][]string // principalID -> permissions
	shouldError bool
}

func (m *mockAuthorizationChecker) GetPrincipalPermissions(ctx context.Context, tenantID, principalID string) ([]string, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}
	if perms, ok := m.permissions[principalID]; ok {
		return perms, nil
	}
	return []string{}, nil
}

// TestValidateScopeAuthorization_PrivilegeEscalation tests the core security fix
// for CVE-2025-GAUTH-005 (Weak Delegation Constraints).
func TestValidateScopeAuthorization_PrivilegeEscalation(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid_Subset_Authorization", func(t *testing.T) {
		// User "bob" has Editor permissions: ["read", "write", "update"]
		checker := &mockAuthorizationChecker{
			permissions: map[string][]string{
				"user-bob": {"read", "write", "update"},
			},
		}

		// PoA requests subset: ["read", "write"]
		valid, unauthorized, err := ValidateScopeAuthorization(
			ctx,
			checker,
			"tenant-123",
			"user-bob",
			[]string{"read", "write"},
		)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !valid {
			t.Fatalf("Expected valid=true for subset permissions, got unauthorized=%v", unauthorized)
		}
		if len(unauthorized) != 0 {
			t.Fatalf("Expected no unauthorized actions, got %v", unauthorized)
		}
	})

	t.Run("PrivilegeEscalation_Blocked", func(t *testing.T) {
		// User "bob" has Editor permissions: ["read", "write", "update"]
		checker := &mockAuthorizationChecker{
			permissions: map[string][]string{
				"user-bob": {"read", "write", "update"},
			},
		}

		// PoA attempts to request ADMIN rights: ["read", "write", "admin", "delete"]
		valid, unauthorized, err := ValidateScopeAuthorization(
			ctx,
			checker,
			"tenant-123",
			"user-bob",
			[]string{"read", "write", "admin", "delete"},
		)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if valid {
			t.Fatal("Expected privilege escalation to be BLOCKED (valid=false)")
		}
		if len(unauthorized) != 2 {
			t.Fatalf("Expected 2 unauthorized actions [admin, delete], got %v", unauthorized)
		}

		// Verify correct unauthorized actions identified
		foundAdmin, foundDelete := false, false
		for _, action := range unauthorized {
			if action == "admin" {
				foundAdmin = true
			}
			if action == "delete" {
				foundDelete = true
			}
		}
		if !foundAdmin || !foundDelete {
			t.Fatalf("Expected unauthorized=[admin, delete], got %v", unauthorized)
		}
	})

	t.Run("EmptyPermissions_BlocksAll", func(t *testing.T) {
		// User "alice" has NO permissions
		checker := &mockAuthorizationChecker{
			permissions: map[string][]string{
				"user-alice": {},
			},
		}

		// PoA requests basic actions
		valid, unauthorized, err := ValidateScopeAuthorization(
			ctx,
			checker,
			"tenant-123",
			"user-alice",
			[]string{"read"},
		)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if valid {
			t.Fatal("Expected rejection when principal has no permissions")
		}
		if len(unauthorized) != 1 || unauthorized[0] != "read" {
			t.Fatalf("Expected unauthorized=[read], got %v", unauthorized)
		}
	})

	t.Run("PermissionLookupFailure_FailsClosed", func(t *testing.T) {
		// Simulate permission service unavailable
		checker := &mockAuthorizationChecker{
			shouldError: true,
		}

		valid, _, err := ValidateScopeAuthorization(
			ctx,
			checker,
			"tenant-123",
			"user-bob",
			[]string{"read"},
		)

		if err == nil {
			t.Fatal("Expected error when permission lookup fails")
		}
		if valid {
			t.Fatal("Expected fail-closed behavior: valid=false when lookup fails")
		}
	})

	t.Run("NilChecker_ReturnsError", func(t *testing.T) {
		valid, _, err := ValidateScopeAuthorization(
			ctx,
			nil, // No authorization checker configured
			"tenant-123",
			"user-bob",
			[]string{"read"},
		)

		if err == nil {
			t.Fatal("Expected error when checker is nil")
		}
		if valid {
			t.Fatal("Expected valid=false when checker is nil")
		}
	})
}

// TestAttackScenario_RealisticPrivilegeEscalation demonstrates real-world attack
func TestAttackScenario_RealisticPrivilegeEscalation(t *testing.T) {
	ctx := context.Background()

	// Scenario: Company has 3 roles
	// - Viewer: ["read"]
	// - Editor: ["read", "write", "update"]
	// - Admin: ["read", "write", "update", "delete", "admin", "manage_users"]

	checker := &mockAuthorizationChecker{
		permissions: map[string][]string{
			"user-alice": {"read"},                                    // Viewer
			"user-bob":   {"read", "write", "update"},                 // Editor
			"user-carol": {"read", "write", "update", "delete", "admin", "manage_users"}, // Admin
		},
	}

	t.Run("Attack1_ViewerRequestsAdmin", func(t *testing.T) {
		// Alice (Viewer) tries to delegate Admin permissions to an AI agent
		valid, _, _ := ValidateScopeAuthorization(
			ctx,
			checker,
			"tenant-123",
			"user-alice",
			[]string{"admin", "delete", "manage_users"},
		)

		if valid {
			t.Fatal("❌ SECURITY BREACH: Viewer successfully escalated to Admin")
		}
		t.Log("✅ Attack blocked: Viewer cannot delegate Admin permissions")
	})

	t.Run("Attack2_EditorRequestsDelete", func(t *testing.T) {
		// Bob (Editor) tries to include "delete" permission (Admin-only)
		valid, unauthorized, _ := ValidateScopeAuthorization(
			ctx,
			checker,
			"tenant-123",
			"user-bob",
			[]string{"read", "write", "delete"}, // "delete" requires Admin
		)

		if valid {
			t.Fatal("❌ SECURITY BREACH: Editor successfully escalated to Admin privileges")
		}
		if len(unauthorized) != 1 || unauthorized[0] != "delete" {
			t.Fatalf("Expected unauthorized=[delete], got %v", unauthorized)
		}
		t.Log("✅ Attack blocked: Editor cannot delegate delete permission")
	})

	t.Run("Legitimate_AdminDelegatesSubset", func(t *testing.T) {
		// Carol (Admin) delegates read/write to AI agent (legitimate use)
		valid, _, _ := ValidateScopeAuthorization(
			ctx,
			checker,
			"tenant-123",
			"user-carol",
			[]string{"read", "write"},
		)

		if !valid {
			t.Fatal("❌ FALSE POSITIVE: Admin should be able to delegate subset of permissions")
		}
		t.Log("✅ Legitimate delegation: Admin → read/write subset")
	})
}
