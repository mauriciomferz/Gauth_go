package revocation

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

// setupTestRedis creates an in-memory Redis server for testing
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.ClusterClient) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// Create a single-node "cluster" client
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{mr.Addr()},
	})

	return mr, client
}

func TestTwoPhaseRevocation_DisablePoA(t *testing.T) {
	mr, _ := setupTestRedis(t)
	defer mr.Close()

	logger := NewSimpleLogger("TEST")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	ctx := context.Background()
	poaID := "poa_test_123"
	principal := "0xPrincipal"
	reason := "Suspicious activity detected"

	// Test: Disable PoA
	err = tpr.DisablePoA(ctx, poaID, principal, reason)
	if err != nil {
		t.Errorf("DisablePoA failed: %v", err)
	}

	// Verify state
	state, err := tpr.GetPoAState(ctx, poaID)
	if err != nil {
		t.Errorf("GetPoAState failed: %v", err)
	}

	if state == nil {
		t.Fatal("State is nil")
	}

	if state.Status != PoAStatusDisabled {
		t.Errorf("Expected status DISABLED, got %s", state.Status)
	}

	if state.DisableReason != reason {
		t.Errorf("Expected reason %s, got %s", reason, state.DisableReason)
	}

	if state.Principal != principal {
		t.Errorf("Expected principal %s, got %s", principal, state.Principal)
	}

	// Verify PoA is not usable
	usable, msg, err := tpr.IsPoAUsable(ctx, poaID)
	if err != nil {
		t.Errorf("IsPoAUsable failed: %v", err)
	}

	if usable {
		t.Error("PoA should not be usable after disable")
	}

	if msg == "" {
		t.Error("Expected non-empty message")
	}

	t.Logf("✅ PoA disabled successfully: %s", msg)
}

func TestTwoPhaseRevocation_RevokePoA(t *testing.T) {
	mr, _ := setupTestRedis(t)
	defer mr.Close()

	logger := NewSimpleLogger("TEST")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	ctx := context.Background()
	poaID := "poa_test_456"
	principal := "0xPrincipal"

	// First disable
	err = tpr.DisablePoA(ctx, poaID, principal, "Test disable")
	if err != nil {
		t.Fatalf("DisablePoA failed: %v", err)
	}

	// Then revoke
	err = tpr.RevokePoA(ctx, poaID, "Confirmed malicious activity")
	if err != nil {
		t.Errorf("RevokePoA failed: %v", err)
	}

	// Verify state
	state, err := tpr.GetPoAState(ctx, poaID)
	if err != nil {
		t.Errorf("GetPoAState failed: %v", err)
	}

	if state == nil {
		t.Fatal("State is nil")
	}

	if state.Status != PoAStatusRevoked {
		t.Errorf("Expected status REVOKED, got %s", state.Status)
	}

	if state.RevokeReason == "" {
		t.Error("Expected non-empty revoke reason")
	}

	if state.RevokedAt.IsZero() {
		t.Error("Expected non-zero revoked timestamp")
	}

	// Verify PoA is not usable
	usable, msg, err := tpr.IsPoAUsable(ctx, poaID)
	if err != nil {
		t.Errorf("IsPoAUsable failed: %v", err)
	}

	if usable {
		t.Error("PoA should not be usable after revoke")
	}

	t.Logf("✅ PoA revoked successfully: %s", msg)
}

func TestTwoPhaseRevocation_CancelDisable(t *testing.T) {
	mr, _ := setupTestRedis(t)
	defer mr.Close()

	logger := NewSimpleLogger("TEST")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	// Set longer timeout for testing
	tpr.SetDisableTimeout(5 * time.Second)

	ctx := context.Background()
	poaID := "poa_test_789"
	principal := "0xPrincipal"

	// Disable PoA
	err = tpr.DisablePoA(ctx, poaID, principal, "Accidental disable")
	if err != nil {
		t.Fatalf("DisablePoA failed: %v", err)
	}

	// Cancel disable (within window)
	err = tpr.CancelDisable(ctx, poaID)
	if err != nil {
		t.Errorf("CancelDisable failed: %v", err)
	}

	// Verify PoA is usable again
	usable, msg, err := tpr.IsPoAUsable(ctx, poaID)
	if err != nil {
		t.Errorf("IsPoAUsable failed: %v", err)
	}

	if !usable {
		t.Errorf("PoA should be usable after cancel: %s", msg)
	}

	t.Logf("✅ PoA re-enabled successfully: %s", msg)
}

func TestTwoPhaseRevocation_AutoRevoke(t *testing.T) {
	mr, _ := setupTestRedis(t)
	defer mr.Close()

	logger := NewSimpleLogger("TEST")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		t.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	// Set short timeout for testing
	tpr.SetDisableTimeout(200 * time.Millisecond)

	ctx := context.Background()
	poaID := "poa_test_auto_revoke"
	principal := "0xPrincipal"

	// Disable PoA
	err = tpr.DisablePoA(ctx, poaID, principal, "Test auto-revoke")
	if err != nil {
		t.Fatalf("DisablePoA failed: %v", err)
	}

	// Verify initially disabled
	state, _ := tpr.GetPoAState(ctx, poaID)
	if state.Status != PoAStatusDisabled {
		t.Errorf("Expected status DISABLED, got %s", state.Status)
	}

	// Wait for auto-revoke
	time.Sleep(300 * time.Millisecond)

	// Verify auto-revoked
	state, err = tpr.GetPoAState(ctx, poaID)
	if err != nil {
		t.Errorf("GetPoAState failed: %v", err)
	}

	if state.Status != PoAStatusRevoked {
		t.Errorf("Expected status REVOKED after timeout, got %s", state.Status)
	}

	t.Logf("✅ Auto-revoke triggered successfully after %v", tpr.GetDisableTimeout())
}
