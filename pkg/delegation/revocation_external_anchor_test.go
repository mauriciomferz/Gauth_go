package delegation

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/anchor"
)

func TestExternalRevocationAnchor(t *testing.T) {
	// Create memory provider for testing
	provider := anchor.NewMemoryProvider()

	// Create temporary receipt store
	tmpDir := t.TempDir()
	receiptPath := tmpDir + "/revocation_receipts.jsonl"

	// Create anchor observer
	anchorObs, err := NewExternalRevocationAnchor(provider, receiptPath)
	if err != nil {
		t.Fatalf("failed to create anchor observer: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := anchorObs.Close(); closeErr != nil {
			t.Errorf("anchor observer close: %v", closeErr)
		}
	})

	// Create a mock SignedTreeHead
	sth := &SignedTreeHead{
		Version:       1,
		ChainLength:   10,
		Timestamp:     time.Now(),
		MerkleRoot:    "test-merkle-root-abc123",
		AggregateHash: "test-aggregate-hash",
		Signatures:    []TreeHeadSignature{},
	}

	// Test anchoring
	err = anchorObs.OnRevocationAnchor(sth)
	if err != nil {
		t.Fatalf("OnRevocationAnchor failed: %v", err)
	}

	// Verify status
	status := anchorObs.Status()
	if status["anchor_count"].(uint64) != 1 {
		t.Errorf("expected anchor_count=1, got %v", status["anchor_count"])
	}

	// Verify latest receipt
	receipt := anchorObs.LatestReceipt()
	if receipt.Hash != sth.MerkleRoot {
		t.Errorf("expected receipt hash %s, got %s", sth.MerkleRoot, receipt.Hash)
	}
	if receipt.Provider != "memory" {
		t.Errorf("expected provider 'memory', got %s", receipt.Provider)
	}

	// Verify receipt validation
	err = anchorObs.VerifyReceipt(receipt)
	if err != nil {
		t.Errorf("receipt verification failed: %v", err)
	}
}

func TestExternalRevocationAnchor_MultipleAnchors(t *testing.T) {
	provider := anchor.NewMemoryProvider()
	tmpDir := t.TempDir()
	receiptPath := tmpDir + "/revocation_receipts.jsonl"

	anchorObs, err := NewExternalRevocationAnchor(provider, receiptPath)
	if err != nil {
		t.Fatalf("failed to create anchor observer: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := anchorObs.Close(); closeErr != nil {
			t.Errorf("anchor observer close: %v", closeErr)
		}
	})

	// Anchor multiple tree heads
	for i := 0; i < 5; i++ {
		sth := &SignedTreeHead{
			Version:       1,
			ChainLength:   i + 1,
			Timestamp:     time.Now(),
			MerkleRoot:    "root-hash-" + string(rune('a'+i)),
			AggregateHash: "agg-hash-" + string(rune('a'+i)),
			Signatures:    []TreeHeadSignature{},
		}

		err = anchorObs.OnRevocationAnchor(sth)
		if err != nil {
			t.Fatalf("OnRevocationAnchor failed on iteration %d: %v", i, err)
		}
	}

	// Verify count
	status := anchorObs.Status()
	if status["anchor_count"].(uint64) != 5 {
		t.Errorf("expected anchor_count=5, got %v", status["anchor_count"])
	}
}

func TestExternalRevocationAnchor_FallbackToAggregateHash(t *testing.T) {
	provider := anchor.NewMemoryProvider()
	tmpDir := t.TempDir()
	receiptPath := tmpDir + "/revocation_receipts.jsonl"

	anchorObs, err := NewExternalRevocationAnchor(provider, receiptPath)
	if err != nil {
		t.Fatalf("failed to create anchor observer: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := anchorObs.Close(); closeErr != nil {
			t.Errorf("anchor observer close: %v", closeErr)
		}
	})

	// Create STH without MerkleRoot (should fall back to AggregateHash)
	sth := &SignedTreeHead{
		Version:       1,
		ChainLength:   1,
		Timestamp:     time.Now(),
		MerkleRoot:    "", // Empty
		AggregateHash: "fallback-aggregate-hash",
		Signatures:    []TreeHeadSignature{},
	}

	err = anchorObs.OnRevocationAnchor(sth)
	if err != nil {
		t.Fatalf("OnRevocationAnchor failed: %v", err)
	}

	// Verify the receipt used the AggregateHash
	receipt := anchorObs.LatestReceipt()
	if receipt.Hash != sth.AggregateHash {
		t.Errorf("expected receipt to use AggregateHash %s, got %s", sth.AggregateHash, receipt.Hash)
	}
}
