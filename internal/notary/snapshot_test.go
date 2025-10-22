package notary

import (
	"os"
	"testing"
)

func TestGenerateSnapshot(t *testing.T) {
	path := t.TempDir() + "/receipts.json"
	rs := NewReceiptStore(path)
	if err := rs.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	// Append a few receipts
	r1 := Receipt{Hash: "sha256:one", Timestamp: "2025-10-20T00:00:00Z", Provider: "memory", Version: 1, Success: true}
	if _, err := rs.Append(r1); err != nil {
		t.Fatalf("append1: %v", err)
	}
	r2 := Receipt{Hash: "sha256:two", Timestamp: "2025-10-20T00:00:01Z", Provider: "memory", Version: 1, Success: true}
	if _, err := rs.Append(r2); err != nil {
		t.Fatalf("append2: %v", err)
	}
	// Disabled merkle
	os.Unsetenv("GAUTH_NOTARY_MERKLE_ENABLED")
	snap1, err := GenerateSnapshot(rs, "")
	if err != nil {
		t.Fatalf("snapshot1 failed: %v", err)
	}
	if snap1.ReceiptCount != 2 || snap1.ChainHead == "" {
		t.Fatalf("unexpected snapshot1 counts/head: %+v", snap1)
	}
	if snap1.MerkleRoot != "" {
		t.Fatalf("expected empty merkle root when disabled")
	}
	if snap1.Hash == "" {
		t.Fatalf("expected snapshot hash")
	}
	// Enabled merkle
	os.Setenv("GAUTH_NOTARY_MERKLE_ENABLED", "1")
	r3 := Receipt{Hash: "sha256:three", Timestamp: "2025-10-20T00:00:02Z", Provider: "memory", Version: 1, Success: true}
	if _, appendErr := rs.Append(r3); appendErr != nil {
		t.Fatalf("append3: %v", appendErr)
	}
	snap2, err := GenerateSnapshot(rs, snap1.Hash)
	if err != nil {
		t.Fatalf("snapshot2 failed: %v", err)
	}
	if snap2.ReceiptCount != 3 || snap2.ChainHead == "" {
		t.Fatalf("unexpected snapshot2 counts/head: %+v", snap2)
	}
	if snap2.MerkleRoot == "" {
		t.Fatalf("expected merkle root when enabled")
	}
	if snap2.PreviousHash != snap1.Hash {
		t.Fatalf("expected previous hash chain linkage")
	}
}
