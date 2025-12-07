package modellimits

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestModelLimitsSnapshotHashChange ensures snapshot hash changes after limits file reload.
func TestModelLimitsSnapshotHashChange(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "model_limits_snapshot_*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	initial := `{"model_limits":{"snap-model":{"max_input_tokens":100}}}`
	_, _ = tmp.Write([]byte(initial))
	tmp.Close()

	h := NewHandler(tmp.Name(), "", "")
	h.ReloadInterval = 100 * time.Millisecond // fast poll

	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	h1, _, _, _ := h.ComputeSnapshot()

	// Modify limits (tighten) to force different canonical representation
	updated := `{"model_limits":{"snap-model":{"max_input_tokens":80}}}`
	// Ensure mtime
	time.Sleep(1 * time.Second)
	if err := os.WriteFile(tmp.Name(), []byte(updated), 0600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	var h2 string
	changed := false
	for time.Now().Before(deadline) {
		h2, _, _, _ = h.ComputeSnapshot()
		if h2 != h1 {
			changed = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !changed {
		t.Fatalf("expected hash change after reload h1=%s h2=%s", h1, h2)
	}
}
