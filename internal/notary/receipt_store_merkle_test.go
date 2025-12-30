package notary

import (
	"os"
	"testing"
)

// TestMerkleRootFeatureFlag verifies that MerkleRoot is populated only when AGENTAUTH_NOTARY_MERKLE_ENABLED=1
// and that root changes deterministically with appended entries.
func TestMerkleRootFeatureFlag(t *testing.T) {
	path := t.TempDir() + "/receipts.json"
	// Disabled case
	os.Unsetenv("AGENTAUTH_NOTARY_MERKLE_ENABLED")
	rs := NewReceiptStore(path)
	if err := rs.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	r1 := Receipt{Hash: "sha256:aaa", Timestamp: "2025-10-20T00:00:00Z", Provider: "memory", Version: 1, Success: true, LatencySeconds: 0}
	sr1, err := rs.Append(r1)
	if err != nil {
		t.Fatalf("append1 failed: %v", err)
	}
	if sr1.MerkleRoot != "" {
		t.Fatalf("expected empty MerkleRoot when disabled, got %q", sr1.MerkleRoot)
	}

	// Enabled case
	os.Setenv("AGENTAUTH_NOTARY_MERKLE_ENABLED", "1")
	r2 := Receipt{Hash: "sha256:bbb", Timestamp: "2025-10-20T00:00:01Z", Provider: "memory", Version: 1, Success: true, LatencySeconds: 0}
	sr2, err := rs.Append(r2)
	if err != nil {
		t.Fatalf("append2 failed: %v", err)
	}
	if sr2.MerkleRoot == "" {
		t.Fatalf("expected MerkleRoot populated when enabled")
	}

	// Capture root after two entries
	root2 := sr2.MerkleRoot

	// Add third entry; root must change
	r3 := Receipt{Hash: "sha256:ccc", Timestamp: "2025-10-20T00:00:02Z", Provider: "memory", Version: 1, Success: true, LatencySeconds: 0}
	sr3, err := rs.Append(r3)
	if err != nil {
		t.Fatalf("append3 failed: %v", err)
	}
	if sr3.MerkleRoot == "" {
		t.Fatalf("expected MerkleRoot for third entry")
	}
	if sr3.MerkleRoot == root2 {
		t.Fatalf("expected MerkleRoot to change after new entry; unchanged=%q", sr3.MerkleRoot)
	}
}
