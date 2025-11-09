package authz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Reusable test JSON policy fragments to reduce duplication.
const (
	testPolicyID     = "p1"
	policyAllowAlice = `[{"id":"` + testPolicyID + `","subject":"alice","resource":"vault","actions":["read"],"effect":"allow"}]`
	policyDenyAlice  = `[{"id":"p2","subject":"alice","resource":"vault","actions":["read"],"effect":"deny"}]`
)

func writeTempPolicies(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "policies.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to write test policy: %v", err)
	}
	return path
}

func TestFilePolicyStoreLoadEmptyMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")
	store, err := NewFilePolicyStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	policies, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(policies) != 0 {
		t.Fatalf("expected 0 policies got %d", len(policies))
	}
}

func TestFilePolicyStoreLoadValid(t *testing.T) {
	dir := t.TempDir()
	json := `[{"id":"p1","subject":"alice","resource":"vault","actions":["read"],"effect":"allow"}]`
	path := writeTempPolicies(t, dir, json)
	store, err := NewFilePolicyStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	policies, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy got %d", len(policies))
	}
	if policies[0].ID != testPolicyID {
		t.Fatalf("unexpected policy id %s", policies[0].ID)
	}
}

func TestFilePolicyStoreMalformed(t *testing.T) {
	dir := t.TempDir()
	path := writeTempPolicies(t, dir, `{"bad":true}`) // not an array
	store, err := NewFilePolicyStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	_, err = store.Load()
	if err == nil {
		t.Fatalf("expected unmarshal error for malformed JSON")
	}
}

func TestPersistentAuthorizerReload(t *testing.T) {
	dir := t.TempDir()
	path := writeTempPolicies(t, dir, policyAllowAlice)
	store, err := NewFilePolicyStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	pa, err := NewPersistentAuthorizer(store, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("persistent authorizer: %v", err)
	}
	// initial decision should allow
	dec, _ := pa.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if !dec.Allow {
		t.Fatalf("expected allow by p1")
	}

	// modify file to new policy set denying access
	if err := os.WriteFile(path, []byte(policyDenyAlice), 0o600); err != nil {
		t.Fatalf("write updated: %v", err)
	}
	// bump mod time explicitly
	if err := os.Chtimes(path, time.Now().Add(2*time.Second), time.Now().Add(2*time.Second)); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// emulate polling without starting goroutine (direct reload call)
	if err := pa.reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	dec2, _ := pa.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if dec2.Allow {
		t.Fatalf("expected deny after reload, got allow")
	}
}

func TestMemoryAuthorizerCacheBasic(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{ID: "p1", Subject: "alice", Resource: "vault", Actions: []string{"read"}, Effect: Allow})
	ma.EnableCaching(200 * time.Millisecond)
	// first call - miss
	dec1, _ := ma.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if dec1.Metadata["cache_hit"] != metadataCacheHitFalse {
		t.Fatalf("expected cache miss false, got %v", dec1.Metadata["cache_hit"])
	}
	// second call - hit
	dec2, _ := ma.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if dec2.Metadata["cache_hit"] != metadataCacheHitTrue {
		t.Fatalf("expected cache hit true, got %v", dec2.Metadata["cache_hit"])
	}
	// wait for expiry
	time.Sleep(250 * time.Millisecond)
	dec3, _ := ma.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if dec3.Metadata["cache_hit"] != metadataCacheHitFalse {
		t.Fatalf("expected cache miss after expiry, got %v", dec3.Metadata["cache_hit"])
	}
}

func TestMemoryAuthorizerCacheInvalidation(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{ID: "p1", Subject: "alice", Resource: "vault", Actions: []string{"read"}, Effect: Allow})
	ma.EnableCaching(1 * time.Second)
	dec1, _ := ma.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if dec1.Metadata["cache_hit"] != metadataCacheHitFalse {
		t.Fatalf("expected initial miss")
	}
	dec2, _ := ma.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if dec2.Metadata["cache_hit"] != metadataCacheHitTrue {
		t.Fatalf("expected second hit")
	}
	// invalidate and re-authorize
	ma.InvalidateAll()
	dec3, _ := ma.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if dec3.Metadata["cache_hit"] != metadataCacheHitFalse {
		t.Fatalf("expected miss after invalidation")
	}
}

func TestCombiningStrategies(t *testing.T) {
	// conflicting policies: one allow, one deny
	allow := Policy{ID: "allow", Subject: "alice", Resource: "obj", Actions: []string{"read"}, Effect: Allow}
	deny := Policy{ID: "deny", Subject: "alice", Resource: "obj", Actions: []string{"read"}, Effect: Deny}

	// DenyOverrides should deny
	ma1 := NewMemoryAuthorizer()
	ma1.AddPolicy(allow)
	ma1.AddPolicy(deny)
	ma1.SetCombiningStrategy(DenyOverrides)
	d1, _ := ma1.Authorize(context.Background(), Request{Subject: "alice", Resource: "obj", Action: "read"})
	if d1.Allow {
		t.Fatalf("deny_overrides expected deny, got allow")
	}
	if d1.Reason == "" {
		t.Fatalf("expected reason populated")
	}

	// PermitOverrides should allow
	ma2 := NewMemoryAuthorizer()
	ma2.AddPolicy(allow)
	ma2.AddPolicy(deny)
	ma2.SetCombiningStrategy(PermitOverrides)
	d2, _ := ma2.Authorize(context.Background(), Request{Subject: "alice", Resource: "obj", Action: "read"})
	if !d2.Allow {
		t.Fatalf("permit_overrides expected allow, got deny")
	}

	// FirstApplicable (order dependent) - add deny first then allow should deny
	ma3 := NewMemoryAuthorizer()
	ma3.AddPolicy(deny)
	ma3.AddPolicy(allow)
	ma3.SetCombiningStrategy(FirstApplicable)
	d3, _ := ma3.Authorize(context.Background(), Request{Subject: "alice", Resource: "obj", Action: "read"})
	if d3.Allow {
		t.Fatalf("first_applicable expected deny due to first matching deny policy")
	}
}

func TestFsnotifyWatchReload(t *testing.T) {
	dir := t.TempDir()
	path := writeTempPolicies(t, dir, `[{"id":"p1","subject":"alice","resource":"vault","actions":["read"],"effect":"allow"}]`)
	store, err := NewFilePolicyStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	pa, err := NewPersistentAuthorizer(store, 5*time.Second) // long poll to ensure watcher triggers first
	if err != nil {
		t.Fatalf("pa: %v", err)
	}
	if err := pa.StartWatch(); err != nil {
		t.Skipf("fsnotify not available: %v", err)
	}

	// initial allow
	d1, _ := pa.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if !d1.Allow {
		t.Fatalf("expected initial allow")
	}

	// change policy to deny
	newContent := `[{"id":"p2","subject":"alice","resource":"vault","actions":["read"],"effect":"deny"}]`
	if err := os.WriteFile(path, []byte(newContent), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// wait for watcher to process (small sleep)
	time.Sleep(300 * time.Millisecond)
	d2, _ := pa.Authorize(context.Background(), Request{Subject: "alice", Resource: "vault", Action: "read"})
	if d2.Allow {
		t.Fatalf("expected deny after fsnotify reload")
	}
	pa.Stop()
}

// TestReloadMetric verifies that reload counter increments on polling and fsnotify paths.
func TestReloadMetric(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "policies.json")
	initial := []byte(`[{"id":"x1","subject":"alice","resource":"r1","actions":["read"],"effect":"allow"}]`)
	if err := os.WriteFile(tmpFile, initial, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	store, err := NewFilePolicyStore(tmpFile)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	pa, err := NewPersistentAuthorizer(store, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("new persistent: %v", err)
	}
	// Start polling
	pa.Start()
	// Modify file after brief delay
	time.Sleep(150 * time.Millisecond)
	updated := []byte(`[{"id":"x2","subject":"alice","resource":"r2","actions":["read"],"effect":"deny"}]`)
	if err := os.WriteFile(tmpFile, updated, 0o600); err != nil {
		t.Fatalf("write update: %v", err)
	}
	// Wait for poll to detect
	time.Sleep(250 * time.Millisecond)
	snap1 := pa.GetMetricsSnapshot()
	if snap1.Reloads == 0 {
		t.Fatalf("expected reload increment from polling, got %d", snap1.Reloads)
	}
	// Start watch and trigger another change
	if err := pa.StartWatch(); err != nil {
		t.Fatalf("start watch: %v", err)
	}
	third := []byte(`[{"id":"x3","subject":"alice","resource":"r3","actions":["read"],"effect":"allow"}]`)
	if err := os.WriteFile(tmpFile, third, 0o600); err != nil {
		t.Fatalf("write third: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	snap2 := pa.GetMetricsSnapshot()
	if snap2.Reloads <= snap1.Reloads {
		t.Fatalf("expected reloads to increase; before=%d after=%d", snap1.Reloads, snap2.Reloads)
	}
}

func TestPolicyDiffOnReload(t *testing.T) {
	dir := t.TempDir()
	path := writeTempPolicies(t, dir, `[{"id":"a","subject":"alice","resource":"r1","actions":["read"],"effect":"allow"},{"id":"b","subject":"alice","resource":"r2","actions":["read"],"effect":"allow"}]`)
	store, err := NewFilePolicyStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	pa, err := NewPersistentAuthorizer(store, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("new persistent: %v", err)
	}
	// initial diff should show added a,b (previous empty set)
	added, removed := pa.LastDiff()
	if len(added) != 2 || len(removed) != 0 {
		t.Fatalf("expected initial added 2 removed 0, got %v %v", added, removed)
	}
	// modify file: remove b, add c
	newContent := `[{"id":"a","subject":"alice","resource":"r1","actions":["read"],"effect":"allow"},{"id":"c","subject":"alice","resource":"r3","actions":["read"],"effect":"deny"}]`
	if err := os.WriteFile(path, []byte(newContent), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// trigger reload directly
	if err := pa.reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	added, removed = pa.LastDiff()
	// Expect added c, removed b
	if !(len(added) == 1 && added[0] == "c") {
		t.Fatalf("expected added [c], got %v", added)
	}
	if !(len(removed) == 1 && removed[0] == "b") {
		t.Fatalf("expected removed [b], got %v", removed)
	}
}

// TestStartWatchErrorCases tests error handling in StartWatch
func TestStartWatchErrorCases(t *testing.T) {
	t.Run("non_file_store", func(t *testing.T) {
		// Use a mock store that isn't FilePolicyStore
		mockStore := &mockPolicyStore{}
		pa := &PersistentAuthorizer{
			MemoryAuthorizer: NewMemoryAuthorizer(),
			store:            mockStore,
			stopCh:           make(chan struct{}),
		}
		err := pa.StartWatch()
		if err == nil {
			t.Fatal("expected error for non-FilePolicyStore")
		}
		if err.Error() != "watch only supported for FilePolicyStore" {
			t.Errorf("unexpected error: %v", err)
		}
		if pa.WatchErr() != nil {
			t.Errorf("watchErr should remain nil, got: %v", pa.WatchErr())
		}
	})

	t.Run("invalid_file_path", func(t *testing.T) {
		// Create a FilePolicyStore with an invalid path
		invalidPath := "/nonexistent/directory/that/does/not/exist/policies.json"
		store := &FilePolicyStore{path: invalidPath}
		pa := &PersistentAuthorizer{
			MemoryAuthorizer: NewMemoryAuthorizer(),
			store:            store,
			stopCh:           make(chan struct{}),
		}

		err := pa.StartWatch()
		if err == nil {
			t.Fatal("expected error for invalid path")
		}
		if pa.WatchErr() == nil {
			t.Error("watchErr should be set on failure")
		}
	})

	t.Run("watcher_close_on_error", func(t *testing.T) {
		// Test that watcher closes properly on Add error
		dir := t.TempDir()
		// Create valid file first
		path := writeTempPolicies(t, dir, `[]`)
		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		pa, err := NewPersistentAuthorizer(store, 1*time.Second)
		if err != nil {
			t.Fatalf("new persistent: %v", err)
		}

		// Remove file to cause watcher.Add to potentially fail
		os.Remove(path)

		// StartWatch may or may not fail depending on OS behavior
		// but it should handle the error gracefully
		err = pa.StartWatch()
		// Error is acceptable here
		if err != nil && pa.WatchErr() == nil {
			t.Error("if StartWatch returns error, watchErr should be set")
		}
	})
}

// TestWatchLoopEdgeCases tests watchLoop behavior
func TestWatchLoopEdgeCases(t *testing.T) {
	t.Run("watch_loop_stops_on_stopCh", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempPolicies(t, dir, `[]`)
		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		pa, err := NewPersistentAuthorizer(store, 10*time.Second)
		if err != nil {
			t.Fatalf("new persistent: %v", err)
		}

		if err := pa.StartWatch(); err != nil {
			t.Skipf("fsnotify not available: %v", err)
		}

		// Immediately stop
		pa.Stop()
		time.Sleep(100 * time.Millisecond)

		// Watcher should be closed
		// No way to directly verify goroutine exit, but Stop() should not hang
	})

	t.Run("watch_loop_handles_rename_events", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempPolicies(t, dir, `[{"id":"p1","subject":"alice","resource":"r1","actions":["read"],"effect":"allow"}]`)
		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		pa, err := NewPersistentAuthorizer(store, 10*time.Second)
		if err != nil {
			t.Fatalf("new persistent: %v", err)
		}
		defer pa.Stop()

		if err := pa.StartWatch(); err != nil {
			t.Skipf("fsnotify not available: %v", err)
		}

		// Rename file (some editors do this on save)
		tmpPath := path + ".tmp"
		newContent := `[{"id":"p2","subject":"bob","resource":"r2","actions":["write"],"effect":"deny"}]`
		if err := os.WriteFile(tmpPath, []byte(newContent), 0o600); err != nil {
			t.Fatalf("write temp: %v", err)
		}
		if err := os.Rename(tmpPath, path); err != nil {
			t.Fatalf("rename: %v", err)
		}

		time.Sleep(300 * time.Millisecond)

		// Policy should have reloaded
		if pa.PolicyCount() != 1 {
			t.Errorf("expected 1 policy after rename, got %d", pa.PolicyCount())
		}
		// Note: accessing specific policy requires holding lock, so just verify count
	})

	t.Run("watch_loop_handles_remove_events", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempPolicies(t, dir, `[{"id":"p1","subject":"alice","resource":"r1","actions":["read"],"effect":"allow"}]`)
		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		pa, err := NewPersistentAuthorizer(store, 10*time.Second)
		if err != nil {
			t.Fatalf("new persistent: %v", err)
		}
		defer pa.Stop()

		if err := pa.StartWatch(); err != nil {
			t.Skipf("fsnotify not available: %v", err)
		}

		// Remove file
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove: %v", err)
		}

		time.Sleep(300 * time.Millisecond)

		// Should reload with empty policy set
		if pa.PolicyCount() != 0 {
			t.Errorf("expected 0 policies after removal, got %d", pa.PolicyCount())
		}
	})

	t.Run("watch_loop_handles_multiple_rapid_changes", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "policies.json")
		// Start with file
		if err := os.WriteFile(path, []byte(`[{"id":"p1","subject":"alice","resource":"r1","actions":["read"],"effect":"allow"}]`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}
		pa, err := NewPersistentAuthorizer(store, 10*time.Second)
		if err != nil {
			t.Fatalf("new persistent: %v", err)
		}
		defer pa.Stop()

		if err := pa.StartWatch(); err != nil {
			t.Skipf("fsnotify not available: %v", err)
		}

		// Make multiple rapid changes
		for i := 2; i <= 4; i++ {
			time.Sleep(50 * time.Millisecond)
			content := fmt.Sprintf(`[{"id":"p%d","subject":"alice","resource":"r%d","actions":["read"],"effect":"allow"}]`, i, i)
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("write %d: %v", i, err)
			}
		}

		// Wait for last change to be processed
		time.Sleep(300 * time.Millisecond)

		// Should have reloaded to latest policy (exact ID depends on timing)
		if pa.PolicyCount() != 1 {
			t.Errorf("expected 1 policy after changes, got %d", pa.PolicyCount())
		}
		// Just verify we have a policy, don't check exact ID due to timing
	})
}

// TestLoadEdgeCases tests additional Load edge cases
func TestLoadEdgeCases(t *testing.T) {
	t.Run("read_permission_error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("running as root, cannot test permission errors")
		}

		dir := t.TempDir()
		path := writeTempPolicies(t, dir, `[]`)

		// Make file unreadable
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		defer os.Chmod(path, 0o600) // cleanup

		store, err := NewFilePolicyStore(path)
		if err != nil {
			// refreshModTime might fail with permission error
			t.Skipf("NewFilePolicyStore failed (expected on permission error): %v", err)
		}

		_, err = store.Load()
		if err == nil {
			t.Fatal("expected error loading unreadable file")
		}
	})

	t.Run("whitespace_only_file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "whitespace.json")
		if err := os.WriteFile(path, []byte("   \n\t  \n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}

		_, err = store.Load()
		if err == nil {
			t.Fatal("expected error for whitespace-only file")
		}
	})

	t.Run("empty_object_instead_of_array", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempPolicies(t, dir, `{}`)
		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}

		_, err = store.Load()
		if err == nil {
			t.Fatal("expected error for object instead of array")
		}
	})
}

// TestNewFilePolicyStoreEdgeCases tests constructor edge cases
func TestNewFilePolicyStoreEdgeCases(t *testing.T) {
	t.Run("directory_instead_of_file", func(t *testing.T) {
		dir := t.TempDir()
		store, err := NewFilePolicyStore(dir)
		if err != nil {
			// Some systems may error on Stat(directory)
			return
		}

		// lastModified should be set to directory mod time
		if store.lastModified.IsZero() {
			t.Error("lastModified should not be zero for existing directory")
		}
	})

	t.Run("symlink_to_file", func(t *testing.T) {
		dir := t.TempDir()
		target := writeTempPolicies(t, dir, `[]`)
		symlink := filepath.Join(dir, "symlink.json")

		if err := os.Symlink(target, symlink); err != nil {
			t.Skipf("symlink not supported: %v", err)
		}

		store, err := NewFilePolicyStore(symlink)
		if err != nil {
			t.Fatalf("new store with symlink: %v", err)
		}

		policies, err := store.Load()
		if err != nil {
			t.Fatalf("load through symlink: %v", err)
		}
		if len(policies) != 0 {
			t.Errorf("expected 0 policies, got %d", len(policies))
		}
	})
}

// TestStartEdgeCases tests Start polling edge cases
func TestStartEdgeCases(t *testing.T) {
	t.Run("refresh_modtime_error_continues", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempPolicies(t, dir, `[]`)
		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}

		pa, err := NewPersistentAuthorizer(store, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("new persistent: %v", err)
		}
		defer pa.Stop()

		pa.Start()
		time.Sleep(30 * time.Millisecond)

		// Delete file to cause refreshModTime error
		os.Remove(path)

		// Wait for polling to attempt refresh
		time.Sleep(100 * time.Millisecond)

		// Polling should continue despite error (logged to stderr)
		// Just verify no panic occurred
	})

	t.Run("concurrent_starts", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempPolicies(t, dir, `[]`)
		store, err := NewFilePolicyStore(path)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}

		pa, err := NewPersistentAuthorizer(store, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("new persistent: %v", err)
		}
		defer pa.Stop()

		// Start multiple times (should spawn multiple goroutines)
		pa.Start()
		pa.Start()
		pa.Start()

		time.Sleep(50 * time.Millisecond)

		// No panic is the success criteria
	})
}

// mockPolicyStore for testing non-FilePolicyStore scenarios
type mockPolicyStore struct {
	policies     []Policy
	lastModified time.Time
	loadErr      error
}

func (m *mockPolicyStore) Load() ([]Policy, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return append([]Policy{}, m.policies...), nil
}

func (m *mockPolicyStore) LastModified() time.Time {
	return m.lastModified
}
