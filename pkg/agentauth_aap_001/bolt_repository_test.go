package agentauth_aap_001

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempBoltPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "poa.db")
}

// fabricatePOA creates a minimal PowerOfAttorney for testing.
func fabricatePOA(grantor, grantee string, dur time.Duration) *PowerOfAttorney {
	now := time.Now().UTC()
	return &PowerOfAttorney{
		ID:           generatePOAID(),
		Grantor:      grantor,
		Grantee:      grantee,
		Scope:        []string{"test:action", "*"},
		Restrictions: map[string]string{"max_amount": "100.00"},
		ValidFrom:    now,
		ValidUntil:   now.Add(dur),
		Status:       POAStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestBoltRepositoryCRUD(t *testing.T) {
	path := tempBoltPath(t)
	repo, err := NewBoltRepository(path)
	if err != nil {
		t.Fatalf("open bolt repo: %v", err)
	}
	defer func() { _ = repo.Close() }()
	p := fabricatePOA("alice", "bob", time.Hour)
	if err := repo.Create(p); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Get
	got, ok := repo.Get(p.ID)
	if !ok || got == nil {
		t.Fatalf("get failed")
	}
	if got.Grantor != p.Grantor {
		t.Fatalf("grantor mismatch")
	}
	// ListByPrincipal
	list := repo.ListByPrincipal("alice")
	if len(list) == 0 {
		t.Fatalf("expected listing for grantor")
	}
	list2 := repo.ListByPrincipal("bob")
	if len(list2) == 0 {
		t.Fatalf("expected listing for grantee")
	}
	// Update
	got.Status = POAStatusRevoked
	got.UpdatedAt = time.Now().UTC()
	if err := repo.Update(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, ok2 := repo.Get(p.ID)
	if !ok2 || updated.Status != POAStatusRevoked {
		t.Fatalf("update not reflected")
	}
}

func TestBoltRepositoryPersistenceAcrossReopen(t *testing.T) {
	path := tempBoltPath(t)
	repo, err := NewBoltRepository(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	p := fabricatePOA("carol", "dave", time.Hour)
	if err2 := repo.Create(p); err2 != nil {
		t.Fatalf("create: %v", err2)
	}
	_ = repo.Close()
	// Reopen
	repo2, err := NewBoltRepository(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = repo2.Close() }()
	got, ok := repo2.Get(p.ID)
	if !ok || got == nil {
		t.Fatalf("reloaded get failed")
	}
	if got.Grantor != p.Grantor {
		t.Fatalf("grantor mismatch after reopen")
	}
}

func TestBoltRepositoryMissingRecord(t *testing.T) {
	path := tempBoltPath(t)
	repo, err := NewBoltRepository(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = repo.Close() }()
	if _, ok := repo.Get("nonexistent"); ok {
		t.Fatalf("expected not found")
	}
	list := repo.ListByPrincipal("nobody")
	if len(list) != 0 {
		t.Fatalf("expected empty list")
	}
}

func TestBoltRepositoryConcurrentReads(t *testing.T) {
	path := tempBoltPath(t)
	repo, err := NewBoltRepository(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = repo.Close() }()
	// Seed several POAs
	for i := 0; i < 10; i++ {
		p := fabricatePOA("eve", "user%d"+string(rune('A'+i)), time.Hour)
		if err := repo.Create(p); err != nil {
			t.Fatalf("seed create: %v", err)
		}
	}
	done := make(chan struct{})
	for j := 0; j < 5; j++ {
		go func() {
			for k := 0; k < 50; k++ {
				_ = repo.ListByPrincipal("eve")
			}
			done <- struct{}{}
		}()
	}
	for j := 0; j < 5; j++ {
		<-done
	}
	// ensure at least one listing returned results
	if len(repo.ListByPrincipal("eve")) == 0 {
		t.Fatalf("expected listings for eve")
	}
}

func TestBoltRepositoryClose(t *testing.T) {
	path := tempBoltPath(t)
	repo, err := NewBoltRepository(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// operations after close should fail benignly
	if _, ok := repo.Get("anything"); ok {
		t.Fatalf("expected closed get false")
	}
	// ensure file still exists
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("db file missing after close: %v", statErr)
	}
}
