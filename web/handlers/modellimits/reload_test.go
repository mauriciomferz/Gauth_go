package modellimits

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestModelLimitsDynamicReload verifies that tightening a limit in the JSON file is applied after reload interval.
func TestModelLimitsDynamicReload(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "model_limits_reload_*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	initial := `{"model_limits":{"reload-model":{"max_input_tokens":500}}}`
	_, _ = tmp.Write([]byte(initial))
	tmp.Close()

	h := NewHandler(tmp.Name(), "", "")
	h.ReloadInterval = 100 * time.Millisecond // poll often

	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Initial request within 400 tokens allowed (limit 500)
	res := h.CheckLimit("reload-model", "", 400, 0)
	if !res.Allowed {
		t.Fatalf("expected initial allow, got denied: %v", res.Error)
	}

	// Overwrite file with tighter limit 300
	tightened := `{"model_limits":{"reload-model":{"max_input_tokens":300}}}`
	// Ensure mtime changes
	time.Sleep(1 * time.Second)
	if err := os.WriteFile(tmp.Name(), []byte(tightened), 0600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	// Wait up to 3 seconds for reload to apply
	deadline := time.Now().Add(3 * time.Second)
	reloaded := false
	for time.Now().Before(deadline) {
		res = h.CheckLimit("reload-model", "", 400, 0)
		if !res.Allowed && res.Error == "model_limit_exceeded" {
			reloaded = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !reloaded {
		t.Fatalf("limit did not tighten after reload window, allowed=true")
	}

	// Confirm lower allowed request passes
	res = h.CheckLimit("reload-model", "", 250, 0)
	if !res.Allowed {
		t.Fatalf("expected allow under tightened limit, got denied: %v", res.Error)
	}
}
