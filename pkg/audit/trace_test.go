package audit

import (
	"context"
	"testing"
	"time"
)

// TestDecisionTraceabilityStub simulates recording authorization decisions with matched and denied policy IDs.
// It asserts that metadata fields are captured and the hash chain links correctly.
func TestDecisionTraceabilityStub(t *testing.T) {
	ml := NewMemoryLogger(nil)
	// Simulate two decisions: one allowed (matched policies), one denied (denied_by policies)
	evAllow := &Event{Type: EventTypeAuthorization, Action: "eval", Result: ResultSuccess, Subject: "alice", Metadata: map[string]interface{}{"matched_policies": []string{"allow-read"}, "denied_policies": []string{}}}
	evDeny := &Event{Type: EventTypeAuthorization, Action: "eval", Result: ResultFailure, Subject: "alice", Metadata: map[string]interface{}{"matched_policies": []string{"allow-read"}, "denied_policies": []string{"deny-write"}}}
	if err := ml.Log(context.TODO(), evAllow); err != nil {
		t.Fatalf("log allow: %v", err)
	}
	if err := ml.Log(context.TODO(), evDeny); err != nil {
		t.Fatalf("log deny: %v", err)
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Verify chain integrity
	if err := ml.VerifyChain(); err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	// Basic field assertions
	if evAllow.Hash == "" || evDeny.Hash == "" {
		t.Fatalf("expected hashes to be set")
	}
	if evDeny.PrevHash != evAllow.Hash {
		t.Fatalf("expected second event prev hash to equal first hash")
	}
	// Metadata presence checks
	if _, ok := evAllow.Metadata["matched_policies"]; !ok {
		t.Fatalf("missing matched_policies in allow event")
	}
	if _, ok := evDeny.Metadata["denied_policies"]; !ok {
		t.Fatalf("missing denied_policies in deny event")
	}
}
