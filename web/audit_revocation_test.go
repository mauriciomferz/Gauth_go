package web

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
)

// TestRevocationAuditLogging verifies RFC111-R7 compliance.
// It ensures that appending to the revocation chain triggers a centralized audit entry with correct metadata.
func TestRevocationAuditLogging(t *testing.T) {
	// Save/Restore global hook to avoid polluting other tests
	originalHook := delegation.OnRevocationAppended
	t.Cleanup(func() {
		delegation.OnRevocationAppended = originalHook
	})

	gin.SetMode(gin.TestMode)
	km, _ := crypto.NewManager(1 * time.Hour)

	// Initialize server which wires the hook
	s := NewBetaServer("", WithKeyProvider(km))
	t.Cleanup(func() { s.Shutdown() })

	// Ensure chain is ready
	if s.revocationChain == nil {
		s.revocationChain = delegation.NewRevocationChain()
	}

	// 1. Trigger revocation append
	ev := delegation.RevocationEvent{
		ID:           "audit-rev-1",
		DelegationID: "audit-del-1",
		Reason:       "compromise",
	}

	// Append directly (simulating handler action or background process)
	// The hook wired in NewBetaServer should fire.
	if _, err := s.revocationChain.Append(ev); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	// 2. Verify Audit Log
	// Wait a tiny bit if hook was async? No, hook call in Append is synchronous.
	// Audit Append is thread-safe.

	// Use List(0) to get all entries
	entries := s.audit.List(0)
	var found *AuditEntry
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Action == "revocation_appended" && entries[i].Resource == "revocation_chain" {
			found = entries[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected audit entry 'revocation_appended' not found. Total entries: %d", len(entries))
	}

	// 3. Validate Metadata (RFC111-R7)
	// Meta is 'any', so we must assert it to map[string]any
	meta, ok := found.Meta.(map[string]any)
	if !ok {
		t.Fatalf("meta is not map[string]any, got %T", found.Meta)
	}

	if meta["revocation_id"] != ev.ID {
		t.Errorf("revocation_id mismatch: got %v want %v", meta["revocation_id"], ev.ID)
	}
	if meta["delegation_id"] != ev.DelegationID {
		t.Errorf("delegation_id mismatch: got %v want %v", meta["delegation_id"], ev.DelegationID)
	}
	if meta["reason"] != "compromise" {
		t.Errorf("reason mismatch: got %v", meta["reason"])
	}

	// Chain length should be at least 1 (since we just appended)
	if l, ok := meta["chain_length"].(int); !ok || l < 1 {
		t.Errorf("invalid chain_length: %v", meta["chain_length"])
	}

	// Aggregate hash should be present
	if agg, ok := meta["aggregate_hash"].(string); !ok || agg == "" {
		t.Error("aggregate_hash missing or empty")
	}

	// Event hash should match
	if h, ok := meta["event_hash"].(string); !ok || h == "" {
		t.Error("event_hash missing")
	}
}
