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
	if err := tmp.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	h := NewHandler(tmp.Name(), "", "")
	// Don't rely on background polling (it can be disabled in CI); explicitly reload.
	h.ReloadInterval = 0

	if err := h.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	h1, at1, models1, users1 := h.ComputeSnapshot()
	_ = at1
	_ = models1
	_ = users1

	// Modify limits (tighten) to force different canonical representation
	updated := `{"model_limits":{"snap-model":{"max_input_tokens":80}}}`
	if err := os.WriteFile(tmp.Name(), []byte(updated), 0600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	// Best-effort mtime bump (avoids flakes on coarse timestamp filesystems).
	if err := os.Chtimes(tmp.Name(), time.Now().Add(2*time.Second), time.Now().Add(2*time.Second)); err != nil {
		// If chtimes isn't supported, rewrite after a delay to force mtime to advance.
		time.Sleep(1100 * time.Millisecond)
		if err := os.WriteFile(tmp.Name(), []byte(updated), 0600); err != nil {
			t.Fatalf("rewrite2: %v", err)
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	loaded := false
	for time.Now().Before(deadline) {
		if h.LoadFromDisk() {
			loaded = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !loaded {
		t.Fatalf("expected reload to detect updated limits")
	}

	h2, at2, models2, users2 := h.ComputeSnapshot()
	_ = at2
	_ = models2
	_ = users2
	if h1 == h2 {
		t.Fatalf("expected hash change after reload h1=%s h2=%s", h1, h2)
	}
}
