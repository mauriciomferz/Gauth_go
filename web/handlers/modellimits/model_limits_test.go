package modellimits

import (
	"context"
	"os"
	"testing"
)

// TestModelValidateLimits exercises unknown model (no limit), allowed within limit, and over-limit denial.
func TestModelValidateLimits(t *testing.T) {
	// Create temp limits file
	f, err := os.CreateTemp(t.TempDir(), "model_limits_*.json")
	if err != nil {
		t.Fatalf("temp file err=%v", err)
	}
	jsonData := []byte(`{"model_limits":{"demo-model":{"max_input_tokens":1024}}}`)
	_, _ = f.Write(jsonData)
	f.Close()

	h := NewHandler(f.Name(), "", "")
	h.StrictUnknown = false // Ensure default behavior explicitly or based on test requirement

	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Unknown model (should allow if not strict)
	res := h.CheckLimit("other-model", "", 500, 0)
	if !res.Allowed {
		t.Fatalf("expected allowed unknown model, got denied: %v", res.Error)
	}

	// Allowed within limit
	res = h.CheckLimit("demo-model", "", 100, 0)
	if !res.Allowed {
		t.Fatalf("expected allowed within limit, got denied: %v", res.Error)
	}

	// Over limit
	res = h.CheckLimit("demo-model", "", 2000, 0)
	if res.Allowed {
		t.Fatalf("expected denied over limit")
	}
	if res.Error != "model_limit_exceeded" {
		t.Fatalf("expected model_limit_exceeded got %s", res.Error)
	}
}
