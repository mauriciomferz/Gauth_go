package modellimits

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestAttestationStreamAuditTrigger ensures audit append causes an attestation emission with reason=audit_append.
func TestAttestationStreamAuditTrigger(t *testing.T) {
	t.Setenv("AGENTAUTH_ATTEST_STREAM_ENABLE", "1")
	t.Setenv("AGENTAUTH_MODEL_LIMIT_ATTEST_SIGN", "0")
	t.Setenv("AGENTAUTH_MODEL_LIMIT_ATTEST_NOTARIZE", "0")

	// Limits file with small limit to force exceed
	limitsFile, err := os.CreateTemp(t.TempDir(), "limits-*.json")
	if err != nil {
		t.Fatalf("limits temp: %v", err)
	}
	_, _ = limitsFile.WriteString(`{"model_limits":{"m1":{"max_input_tokens":100}}}`)
	_ = limitsFile.Close()

	// Audit file
	auditFile, err := os.CreateTemp(t.TempDir(), "audit-*.jsonl")
	if err != nil {
		t.Fatalf("audit temp: %v", err)
	}
	_, _ = auditFile.WriteString("{\"hash\":\"h0\"}\n")
	_ = auditFile.Close()

	h := NewHandler(limitsFile.Name(), auditFile.Name(), "")
	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("handler init: %v", err)
	}

	ch := h.SubscribeAttestation()
	defer h.UnsubscribeAttestation(ch)

	// Trigger "audit_append" by causing an exceed event
	// Input 200 > 100
	// CheckLimit(modelID, userID, input, output)
	_ = h.CheckLimit("m1", "", 200, 0)

	deadline := time.Now().Add(3 * time.Second)
	foundAudit := false
	for time.Now().Before(deadline) {
		select {
		case att := <-ch:
			if att.Reason == "audit_append" {
				foundAudit = true
				goto Verified
			}
		case <-time.After(100 * time.Millisecond):
		}
	}
Verified:
	if !foundAudit {
		t.Fatal("did not observe audit_append attestation event")
	}
}
