// Package delegation lifecycle tests.
package delegation

import (
	"testing"
	"time"
)

func TestDelegation_Suspend(t *testing.T) {
	d := &Delegation{
		ID:        "test-1",
		Subject:   "user:alice",
		Delegate:  "user:bob",
		Scope:     map[string]string{"resource": "document", "action": "read"},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusActive,
	}

	err := d.Suspend("admin", "security_review", 2*time.Hour)
	if err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}

	if d.Status != StatusSuspended {
		t.Errorf("Expected status %s, got %s", StatusSuspended, d.Status)
	}

	if d.IsUsable() {
		t.Error("Suspended delegation should not be usable")
	}
}

func TestDelegation_Resume(t *testing.T) {
	d := &Delegation{
		ID:        "test-2",
		Subject:   "user:alice",
		Delegate:  "user:bob",
		Scope:     map[string]string{"resource": "document"},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusSuspended,
	}

	err := d.Resume("admin")
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if d.Status != StatusActive {
		t.Errorf("Expected status %s, got %s", StatusActive, d.Status)
	}

	if !d.IsUsable() {
		t.Error("Resumed delegation should be usable")
	}
}

func TestDelegation_PartiallyRevoke(t *testing.T) {
	d := &Delegation{
		ID:       "test-3",
		Subject:  "user:alice",
		Delegate: "user:bob",
		Scope: map[string]string{
			"resource": "document",
			"action":   "read",
			"action2":  "write",
		},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusActive,
	}

	// Revoke write action
	removedScope := map[string]string{"action2": "write"}
	err := d.PartiallyRevoke(removedScope, "admin", "policy_change")
	if err != nil {
		t.Fatalf("PartiallyRevoke failed: %v", err)
	}

	if d.Status != StatusPartiallyRevoked {
		t.Errorf("Expected status %s, got %s", StatusPartiallyRevoked, d.Status)
	}

	if _, exists := d.Scope["action2"]; exists {
		t.Error("Revoked scope key should be removed")
	}

	if _, exists := d.Scope["resource"]; !exists {
		t.Error("Non-revoked scope key should remain")
	}

	if !d.IsUsable() {
		t.Error("Partially revoked delegation with remaining scope should be usable")
	}
}

func TestDelegation_PartiallyRevoke_AllScope(t *testing.T) {
	d := &Delegation{
		ID:        "test-4",
		Subject:   "user:alice",
		Delegate:  "user:bob",
		Scope:     map[string]string{"resource": "document"},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusActive,
	}

	// Revoke all scope
	removedScope := map[string]string{"resource": "document"}
	err := d.PartiallyRevoke(removedScope, "admin", "full_revocation")
	if err != nil {
		t.Fatalf("PartiallyRevoke failed: %v", err)
	}

	// Should become terminated when all scope removed
	if d.Status != StatusTerminated {
		t.Errorf("Expected status %s when all scope removed, got %s", StatusTerminated, d.Status)
	}

	if d.IsUsable() {
		t.Error("Delegation with no remaining scope should not be usable")
	}
}

func TestDelegation_Terminate(t *testing.T) {
	d := &Delegation{
		ID:        "test-5",
		Subject:   "user:alice",
		Delegate:  "user:bob",
		Scope:     map[string]string{"resource": "document"},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusActive,
	}

	err := d.Terminate("admin", "policy_violation")
	if err != nil {
		t.Fatalf("Terminate failed: %v", err)
	}

	if d.Status != StatusTerminated {
		t.Errorf("Expected status %s, got %s", StatusTerminated, d.Status)
	}

	if d.IsUsable() {
		t.Error("Terminated delegation should not be usable")
	}
}

func TestDelegation_IsUsable_Expired(t *testing.T) {
	d := &Delegation{
		ID:        "test-6",
		Subject:   "user:alice",
		Delegate:  "user:bob",
		Scope:     map[string]string{"resource": "document"},
		IssuedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired 1 hour ago
		Status:    StatusActive,
	}

	if d.IsUsable() {
		t.Error("Expired delegation should not be usable")
	}
}

func TestDelegation_CanAccessResource(t *testing.T) {
	d := &Delegation{
		ID:       "test-7",
		Subject:  "user:alice",
		Delegate: "user:bob",
		Scope: map[string]string{
			"resource": "document",
			"action":   "read",
		},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusActive,
	}

	// Should allow exact match
	if !d.CanAccessResource("resource", "document") {
		t.Error("Should allow exact match")
	}

	if !d.CanAccessResource("action", "read") {
		t.Error("Should allow exact match for action")
	}

	// Should deny mismatch
	if d.CanAccessResource("resource", "other") {
		t.Error("Should deny mismatched value")
	}

	// Should deny missing key
	if d.CanAccessResource("other_key", "value") {
		t.Error("Should deny missing scope key")
	}
}

func TestDelegation_CanAccessResource_Wildcard(t *testing.T) {
	d := &Delegation{
		ID:       "test-8",
		Subject:  "user:alice",
		Delegate: "user:bob",
		Scope: map[string]string{
			"resource": "*",
			"action":   "read",
		},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusActive,
	}

	// Wildcard should match any value
	if !d.CanAccessResource("resource", "document") {
		t.Error("Wildcard should match document")
	}

	if !d.CanAccessResource("resource", "file") {
		t.Error("Wildcard should match file")
	}

	if !d.CanAccessResource("resource", "anything") {
		t.Error("Wildcard should match anything")
	}
}

func TestDelegation_StatusTransitions(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus string
		newStatus string
		wantErr   bool
	}{
		{"active to suspended", StatusActive, StatusSuspended, false},
		{"suspended to active", StatusSuspended, StatusActive, false},
		{"active to terminated", StatusActive, StatusTerminated, false},
		{"terminated to active", StatusTerminated, StatusActive, true},
		{"terminated to suspended", StatusTerminated, StatusSuspended, true},
		{"active to partially_revoked", StatusActive, StatusPartiallyRevoked, false},
		{"partially_revoked to terminated", StatusPartiallyRevoked, StatusTerminated, false},
		{"partially_revoked to active", StatusPartiallyRevoked, StatusActive, true},
		{"same status", StatusActive, StatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDelegationStatusTransition(tt.oldStatus, tt.newStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDelegationStatusTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDelegation_LifecycleWorkflow(t *testing.T) {
	// Test a complete lifecycle workflow
	d := &Delegation{
		ID:       "workflow-1",
		Subject:  "user:alice",
		Delegate: "user:bob",
		Scope: map[string]string{
			"resource": "document",
			"action1":  "read",
			"action2":  "write",
			"action3":  "delete",
		},
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Status:    StatusActive,
	}

	// 1. Delegation starts active
	if !d.IsUsable() {
		t.Fatal("New delegation should be usable")
	}

	// 2. Suspend for review
	if err := d.Suspend("admin", "security_review", 1*time.Hour); err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}
	if d.IsUsable() {
		t.Error("Suspended delegation should not be usable")
	}

	// 3. Resume after review
	if err := d.Resume("admin"); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	if !d.IsUsable() {
		t.Error("Resumed delegation should be usable")
	}

	// 4. Partially revoke write permission
	removedScope := map[string]string{"action2": "write"}
	if err := d.PartiallyRevoke(removedScope, "admin", "policy_update"); err != nil {
		t.Fatalf("PartiallyRevoke failed: %v", err)
	}
	if d.Status != StatusPartiallyRevoked {
		t.Error("Status should be partially_revoked")
	}
	if !d.IsUsable() {
		t.Error("Partially revoked delegation with remaining scope should be usable")
	}

	// 5. Can still read
	if !d.CanAccessResource("action1", "read") {
		t.Error("Should still allow read after partial revocation")
	}

	// 6. Cannot write (revoked)
	if d.CanAccessResource("action2", "write") {
		t.Error("Should not allow write after revocation")
	}

	// 7. Finally terminate
	if err := d.Terminate("admin", "end_of_project"); err != nil {
		t.Fatalf("Terminate failed: %v", err)
	}
	if d.IsUsable() {
		t.Error("Terminated delegation should not be usable")
	}

	t.Logf("Lifecycle workflow completed successfully for delegation %s", d.ID)
}

func TestChain_DepthEnforcement(t *testing.T) {
	// Test that depth limiting works as expected
	t.Setenv("AGENTAUTH_MAX_DELEGATION_DEPTH", "3")

	chain := NewChain()

	// Add 3 delegations (should succeed)
	for i := 1; i <= 3; i++ {
		d := Delegation{
			ID:        string(rune('A' + i - 1)),
			Subject:   "user:alice",
			Delegate:  "user:bob",
			Scope:     map[string]string{"resource": "doc"},
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		_, err := chain.Append(d)
		if err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	// 4th should fail (exceeds depth=3)
	d4 := Delegation{
		ID:        "D",
		Subject:   "user:alice",
		Delegate:  "user:bob",
		Scope:     map[string]string{"resource": "doc"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_, err := chain.Append(d4)
	if err == nil {
		t.Error("Expected depth exceeded error for 4th delegation")
	}

	t.Logf("Depth enforcement working: %v", err)
}
