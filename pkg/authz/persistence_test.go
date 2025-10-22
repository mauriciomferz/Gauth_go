package authz

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Reusable test JSON policy fragments to reduce duplication.
const (
	policyAllowAlice = `[{"id":"p1","subject":"alice","resource":"vault","actions":["read"],"effect":"allow"}]`
	policyDenyAlice  = `[{"id":"p2","subject":"alice","resource":"vault","actions":["read"],"effect":"deny"}]`
)

func writeTempPolicies(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "policies.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
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
	if policies[0].ID != "p1" {
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
	if err := os.WriteFile(path, []byte(policyDenyAlice), 0o644); err != nil {
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
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
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
	if err := os.WriteFile(tmpFile, initial, 0o644); err != nil {
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
	if err := os.WriteFile(tmpFile, updated, 0o644); err != nil {
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
	if err := os.WriteFile(tmpFile, third, 0o644); err != nil {
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
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
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
