package notary

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/notary"
	bolt "go.etcd.io/bbolt"
)

// mockNotarizer for testing
type mockNotarizer struct {
	mu       sync.Mutex
	receipts []notary.Receipt
	failNext bool
}

func (m *mockNotarizer) Notarize(hash string) (notary.Receipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return notary.Receipt{}, fmt.Errorf("mock notarization failed")
	}

	receipt := notary.Receipt{
		Hash:           hash,
		Timestamp:      time.Now().Format(time.RFC3339Nano),
		Provider:       "MockNotarizer",
		Version:        1,
		Success:        true,
		LatencySeconds: 0.001,
	}
	m.receipts = append(m.receipts, receipt)
	return receipt, nil
}

// createTestAdapter creates a test adapter with temporary BoltDB
func createTestAdapter(t *testing.T) (*RevocationAnchoringAdapter, *bolt.DB) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	store, err := NewReceiptStore(db)
	if err != nil {
		_ = db.Close()
		t.Fatalf("Failed to create store: %v", err)
	}

	mock := &mockNotarizer{}
	adapter := NewRevocationAnchoringAdapter(mock, store)
	return adapter, db
}

func TestRevocationAnchoringAdapter_Anchor(t *testing.T) {
	adapter, db := createTestAdapter(t)
	t.Cleanup(func() {
		closeErr := db.Close()
		if closeErr != nil {
			t.Errorf("db close: %v", closeErr)
		}
	})

	hash := "sha256:abc123"
	if err := adapter.Anchor(hash); err != nil {
		t.Fatalf("Anchor failed: %v", err)
	}

	receipt, found, err := adapter.GetReceipt(hash)
	if err != nil {
		t.Fatalf("GetReceipt failed: %v", err)
	}
	if !found {
		t.Error("Receipt not found")
	}
	if receipt.Hash != hash {
		t.Errorf("Hash mismatch: %s != %s", receipt.Hash, hash)
	}
}

func TestRevocationAnchoringAdapter_GetReceipt(t *testing.T) {
	adapter, db := createTestAdapter(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db close: %v", err)
		}
	})

	hash := "sha256:test123"
	if err := adapter.Anchor(hash); err != nil {
		t.Fatalf("Anchor failed: %v", err)
	}

	receipt, found, err := adapter.GetReceipt(hash)
	if err != nil {
		t.Errorf("GetReceipt failed: %v", err)
	}
	if !found {
		t.Error("Receipt not found")
	}
	if receipt.Provider != "MockNotarizer" {
		t.Errorf("Wrong provider: %s", receipt.Provider)
	}

	// Non-existent hash
	_, found, _ = adapter.GetReceipt("nonexistent")
	if found {
		t.Error("Should not find nonexistent hash")
	}
}

func TestRevocationAnchoringAdapter_GetStats(t *testing.T) {
	adapter, db := createTestAdapter(t)
	defer func() { _ = db.Close() }()

	// Anchor multiple hashes
	for i := 0; i < 3; i++ {
		hash := fmt.Sprintf("sha256:hash%d", i)
		if err := adapter.Anchor(hash); err != nil {
			t.Fatalf("Anchor failed: %v", err)
		}
	}

	stats, err := adapter.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalReceipts != 3 {
		t.Errorf("Expected 3 receipts, got %d", stats.TotalReceipts)
	}
}

func TestComputeRevocationHash(t *testing.T) {
	poaID := "poa-123"
	revoker := "user@example.com"
	ts := time.Unix(1234567890, 0)
	reason := "test-reason"

	hash1 := ComputeRevocationHash(poaID, revoker, ts, reason)
	hash2 := ComputeRevocationHash(poaID, revoker, ts, reason)

	// Should be deterministic
	if hash1 != hash2 {
		t.Error("Hash not deterministic")
	}

	// Should start with sha256:
	if len(hash1) < 7 || hash1[:7] != "sha256:" {
		t.Errorf("Hash should start with sha256:, got %s", hash1)
	}

	// Different inputs should produce different hashes
	hash3 := ComputeRevocationHash(poaID, revoker, ts, "different-reason")
	if hash1 == hash3 {
		t.Error("Different inputs should produce different hashes")
	}
}

func TestRevocationAnchoringAdapter_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persist.db")

	// Create and anchor
	db1, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}

	store1, err := NewReceiptStore(db1)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	adapter1 := NewRevocationAnchoringAdapter(&mockNotarizer{}, store1)
	hash := "sha256:persistent"
	if err2 := adapter1.Anchor(hash); err2 != nil {
		t.Fatalf("Anchor failed: %v", err2)
	}
	if closeErr := db1.Close(); closeErr != nil {
		t.Fatalf("db close: %v", closeErr)
	}

	// Reopen and verify
	db2, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("Failed to reopen DB: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db2.Close(); closeErr != nil {
			t.Errorf("db close: %v", err)
		}
	})

	store2, err := NewReceiptStore(db2)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	adapter2 := NewRevocationAnchoringAdapter(&mockNotarizer{}, store2)
	receipt, found, err := adapter2.GetReceipt(hash)
	if err != nil {
		t.Errorf("GetReceipt failed: %v", err)
	}
	if !found {
		t.Error("Receipt not persisted")
	}
	if receipt.Hash != hash {
		t.Errorf("Hash mismatch: %s != %s", receipt.Hash, hash)
	}
}

func TestRevocationAnchoringAdapter_ThreadSafety(t *testing.T) {
	adapter, db := createTestAdapter(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db close: %v", err)
		}
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			hash := fmt.Sprintf("sha256:hash-%d", id)
			if err := adapter.Anchor(hash); err != nil {
				t.Errorf("Anchor failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	stats, err := adapter.GetStats()
	if err != nil {
		t.Errorf("GetStats failed: %v", err)
	}
	if stats.TotalReceipts != 10 {
		t.Errorf("Expected 10 receipts, got %d", stats.TotalReceipts)
	}
}

func TestReceiptStore_Operations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "store.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	store, err := NewReceiptStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Store a receipt
	receipt := notary.Receipt{
		Hash:           "sha256:test",
		Timestamp:      time.Now().Format(time.RFC3339Nano),
		Provider:       "Test",
		Version:        1,
		Success:        true,
		LatencySeconds: 0.1,
	}

	if err2 := store.Store(receipt.Hash, receipt); err2 != nil {
		t.Fatalf("Store failed: %v", err2)
	}

	// Retrieve receipt
	retrieved, found, err := store.Get(receipt.Hash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Receipt not found")
	}
	if retrieved.Hash != receipt.Hash {
		t.Errorf("Hash mismatch: %s != %s", retrieved.Hash, receipt.Hash)
	}

	// List receipts
	receipts, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(receipts) != 1 {
		t.Errorf("Expected 1 receipt, got %d", len(receipts))
	}
}

func BenchmarkAnchor(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		b.Fatalf("Failed to create DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	store, err := NewReceiptStore(db)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}

	adapter := NewRevocationAnchoringAdapter(&mockNotarizer{}, store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := fmt.Sprintf("sha256:bench-%d", i)
		if err := adapter.Anchor(hash); err != nil {
			b.Fatalf("Anchor failed: %v", err)
		}
	}
}

func BenchmarkComputeRevocationHash(b *testing.B) {
	poaID := "poa-bench"
	revoker := "bench@test.com"
	ts := time.Now()
	reason := "benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ComputeRevocationHash(poaID, revoker, ts, reason)
	}
}
