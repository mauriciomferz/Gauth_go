package semantic

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/anchor"
	"github.com/mauriciomferz/AgentAuth/pkg/ledger"
)

func TestHandler_Archival(t *testing.T) {
	h := NewHandler(nil, nil, "")
	l := ledger.NewMemoryStore()
	h.Ledger = l
	h.ArchiveInterval = 10 * time.Millisecond // Short interval for test

	// build some state
	h.mu.Lock()
	h.ewma["test"] = &anomalyPersist{Mean: 10.5, Count: 5}
	h.mu.Unlock()

	// first update should archive
	h.Update()

	// Verify ledger has 1 entry
	count := func() int {
		res, _ := l.VerifyChain(context.Background())
		return int(res.Count)
	}
	if c := count(); c != 1 {
		t.Fatalf("expected 1 entry in ledger after first update, got %d", c)
	}

	// Wait a bit to ensure next update archives again (due to interval)
	time.Sleep(20 * time.Millisecond)
	h.Update()

	// Verify ledger now has 2 entries
	if c := count(); c != 2 {
		t.Errorf("expected 2 entries in ledger after second update, got %d", c)
	}

	// Verify metadata content
	entries, _ := l.QueryBySubject(context.Background(), "ewma_stats")
	if len(entries) < 2 {
		t.Fatalf("failed to retrieve entries by subject")
	}
	meta := entries[1].Metadata
	if meta == nil {
		t.Fatal("metadata is nil")
	}

	// Complex check for map contents since it's interface{}
	testVal, ok := meta["test"]
	if !ok {
		t.Errorf("expected 'test' key in metadata")
	} else {
		data := testVal.(*anomalyPersist)
		if data.Mean != 10.5 {
			t.Errorf("expected Mean 10.5, got %v", data.Mean)
		}
	}
}

func TestHandler_Anchoring(t *testing.T) {
	h := NewHandler(nil, nil, "")
	l := ledger.NewMemoryStore()
	h.Ledger = l

	ap := anchor.NewMemoryProvider()
	h.AnchorProvider = ap
	h.AnchorInterval = 10 * time.Millisecond

	// build state and archive
	h.mu.Lock()
	h.ewma["test"] = &anomalyPersist{Mean: 10.5, Count: 5}
	h.mu.Unlock()

	h.archiveToLedger() // tip is now in ledger

	// Update should now trigger anchoring
	h.Update()

	h.mu.Lock()
	receipt := h.LastAnchorReceipt
	h.mu.Unlock()

	if receipt.Hash == "" {
		t.Errorf("expected anchor receipt hash to be populated")
	}

	// verify it matches ledger tip
	tip := ledger.ChainTip(l)
	if receipt.Hash != tip {
		t.Errorf("expected receipt hash %s to match ledger tip %s", receipt.Hash, tip)
	}
}
