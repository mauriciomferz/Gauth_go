package limits

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestIncDefaultDelta verifies that delta=0 applies default increment of 1.
func TestIncDefaultDelta(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	if v := st.Inc("alpha", 0); v != 1 {
		t.Fatalf("expected alpha=1 got %d", v)
	}
	if v := st.Inc("alpha", 0); v != 2 {
		t.Fatalf("expected alpha=2 got %d", v)
	}
	if v := st.Get("alpha"); v != 2 {
		t.Fatalf("Get mismatch %d", v)
	}
}

// TestPersistenceRoundTrip ensures counters persist and reload correctly.
func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "limits.json")
	st, err := New(path)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	st.Inc("issued", 5)
	st.Inc("revoked", 2)
	if err := st.Persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	// Create new store loading same path
	st2, err := New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := st2.Get("issued"); got != 5 {
		t.Fatalf("issued mismatch %d", got)
	}
	if got := st2.Get("revoked"); got != 2 {
		t.Fatalf("revoked mismatch %d", got)
	}
}

// TestConcurrentIncrements validates race safety under goroutines.
func TestConcurrentIncrements(t *testing.T) {
	st, err := New("")
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	const goroutines = 50
	const iters = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				st.Inc("load", 0)
			}
		}()
	}
	wg.Wait()
	expected := uint64(goroutines * iters)
	if got := st.Get("load"); got != expected {
		t.Fatalf("expected %d got %d", expected, got)
	}
}

// TestLedgerEntryFormat ensures _type present and counters copied.
func TestLedgerEntryFormat(t *testing.T) {
	st, _ := New("")
	st.Inc("tokens", 3)
	st.Inc("delegations", 1)
	le := st.LedgerEntry()
	if le["_type"] != "limits_snapshot" {
		t.Fatalf("missing _type limits_snapshot: %#v", le)
	}
	if le["tokens"] != uint64(3) || le["delegations"] != uint64(1) {
		t.Fatalf("counter mismatch: %#v", le)
	}
}

// TestManagerSnapshotCallback ensures callback invoked after persistence tick.
func TestManagerSnapshotCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	// Override env interval small for test
	os.Setenv("GAUTH_LIMITS_PERSIST_INTERVAL_SEC", "1")
	defer os.Unsetenv("GAUTH_LIMITS_PERSIST_INTERVAL_SEC")
	dir := t.TempDir()
	path := filepath.Join(dir, "m.json")
	os.Setenv("GAUTH_LIMITS_PERSIST_PATH", path)
	defer os.Unsetenv("GAUTH_LIMITS_PERSIST_PATH")
	mgr, err := InitFromEnv()
	if err != nil {
		t.Fatalf("init mgr: %v", err)
	}
	called := make(chan struct{}, 2)
	mgr.SetSnapshotCallback(func(m map[string]any) {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	// increment some counters
	Inc("x", 0)
	Inc("y", 5)
	// wait for callback (timeout ~3s)
	select {
	case <-called:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("callback not invoked within timeout")
	}
	if err := mgr.Close(); err != nil {
		t.Fatalf("close mgr: %v", err)
	}
	// Confirm file created
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("persist file missing: %v", err)
	}
	// Guard race with GOOS not requiring specific check
	_ = runtime.GOOS
}
