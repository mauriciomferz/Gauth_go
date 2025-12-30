package jurisdiction

import (
	"context"
	"os"
	"testing"
)

// TestDetachedSignatureEnforcement ensures requests without detached signature are denied when flag enabled.
func TestDetachedSignatureEnforcement(t *testing.T) {
	os.Setenv("AGENTAUTH_REQUIRE_DETACHED_SIGNATURE", "true")
	eng := NewEnforcementEngine()
	// Build context without detached_signature claim.
	ctx := &EnforcementContext{
		RequestID:    "req-1",
		Subject:      "alice",
		Resource:     "acct:123",
		Action:       "transfer",
		Jurisdiction: "US",
		Claims:       map[string]interface{}{"jurisdiction": "US"},
	}
	dec, err := eng.Enforce(context.Background(), ctx)
	if err != nil {
		t.Fatalf("enforce error: %v", err)
	}
	if dec.Allowed {
		t.Fatalf("expected denial due to missing detached signature")
	}
	found := false
	for _, v := range dec.Violations {
		if v == "missing_detached_signature" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("violation missing_detached_signature not recorded")
	}
	// Now supply signature claim
	ctx.Claims["detached_signature"] = "dummy.sig" // placeholder
	dec2, err := eng.Enforce(context.Background(), ctx)
	if err != nil {
		t.Fatalf("enforce error: %v", err)
	}
	if !dec2.Allowed {
		t.Fatalf("expected allow when signature present; violations=%v", dec2.Violations)
	}
	os.Unsetenv("AGENTAUTH_REQUIRE_DETACHED_SIGNATURE")
}
