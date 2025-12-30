package modellimits

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestAttestationStreamSurgeTrigger forces a surge condition and expects a surge_trigger attestation.
func TestAttestationStreamSurgeTrigger(t *testing.T) {
	t.Setenv("AGENTAUTH_ATTEST_STREAM_ENABLE", "1")
	t.Setenv("AGENTAUTH_MODEL_LIMIT_ATTEST_SIGN", "0")
	t.Setenv("AGENTAUTH_MODEL_LIMIT_ATTEST_NOTARIZE", "0")
	// Lower surge thresholds for deterministic trigger
	t.Setenv("AGENTAUTH_MODEL_LIMIT_SURGE_FACTOR", "0.5")
	t.Setenv("AGENTAUTH_MODEL_LIMIT_SURGE_MIN_EVENTS", "2")

	// Prepare limits file (required to trigger exceed)
	limitsFile, err := os.CreateTemp(t.TempDir(), "limits-*.json")
	if err != nil {
		t.Fatalf("limits temp: %v", err)
	}
	// Set a small limit for "surge-model"
	limitsJSON := `{"model_limits":{"surge-model":{"max_input_tokens":10}}}`
	_, _ = limitsFile.WriteString(limitsJSON)
	_ = limitsFile.Close()

	// Prepare audit file (required for configured=true)
	auditFile, err := os.CreateTemp(t.TempDir(), "audit-*.jsonl")
	if err != nil {
		t.Fatalf("audit temp: %v", err)
	}
	_, _ = auditFile.WriteString("{\"hash\":\"h0\"}\n")
	_ = auditFile.Close()

	// Create Handler
	h := NewHandler(limitsFile.Name(), auditFile.Name(), "")
	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("handler init: %v", err)
	}

	// Subscribe
	ch := h.SubscribeAttestation()
	defer h.UnsubscribeAttestation(ch)

	// Wait for any initial events or ensure stream is ready?
	// The original test read "open" event. Handler emits "open" logic?
	// Handler does not automatically emit "open" on subscription start in SubscribeAttestation?
	// BetaServer's `apiModelLimitsAttestationStream` emitted "open".
	// Handler.SubscribeAttestation just returns channel.
	// But `h.EmitAttestation` is public (or effectively used).
	// WE NEED to simulate the surge.

	// Fire enough exceed events to meet surge (simulate input exceed repeatedly)
	// We call CheckLimit with input > limit (limit=10).
	for i := 0; i < 3; i++ { // 3 events exceeds min_events=2 and factor baseline
		// Trigger "audit_append" by causing an exceed event
		// Input 200 > 10
		res := h.CheckLimit("surge-model", "", 200, 0)
		if res.Error != "" && res.Error != "model_limit_exceeded" { // We expect failure.
			// Actually we expect it to FAIL limit check.
		}

		if res.Allowed {
			t.Fatalf("expected allowed=false")
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	foundSurge := false
	for time.Now().Before(deadline) {
		select {
		case att := <-ch:
			// Marshal to check JSON content if needed, or inspect struct
			// att.Reason might be "surge_trigger"
			// Wait, att struct has Reason field?
			// Let's check struct definition.
			// Ideally we just check the struct fields.
			// But attestation struct might not have Reason field populated directly if it's separate?
			// BetaServer `emitAttestation` set `att.Reason = reason`.
			// `Handler` should do the same.
			if att.Reason == "surge_trigger" {
				foundSurge = true
				goto Verified
			}
			// Fallback check: check if Surge struct is populated
			if att.Surge != nil && att.Surge.Triggered {
				// Reason might optionally be set.
				// Depending on implementation.
				foundSurge = true
				goto Verified
			}
		case <-time.After(100 * time.Millisecond):
			// continue loop
		}
	}
Verified:
	if !foundSurge {
		t.Fatalf("did not observe surge_trigger attestation event")
	}
}
