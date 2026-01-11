package notary

import (
	"fmt"
	"testing"
)

func TestVerifySnapshot(t *testing.T) {
	path := t.TempDir() + "/receipts.json"
	rs := NewReceiptStore(path)
	if err := rs.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	t.Setenv("AGENTAUTH_NOTARY_MERKLE_ENABLED", "1")
	// Append receipts
	for i, h := range []string{"sha256:a", "sha256:b", "sha256:c"} {
		ts := fmt.Sprintf("2025-10-20T00:00:0%dZ", i)
		r := Receipt{Hash: h, Timestamp: ts, Provider: "memory", Version: 1, Success: true}
		if _, err := rs.Append(r); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	snap, err := GenerateSnapshot(rs, "")
	if err != nil {
		t.Fatalf("generate snapshot: %v", err)
	}
	res, err := VerifySnapshot(rs, snap)
	if err != nil {
		t.Fatalf("verify snapshot: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid snapshot; got %+v", res)
	}
	// Tamper 1: change chain head -> expect chain_head_mismatch or hash_mismatch (hash mismatch primary)
	snapBad := snap
	snapBad.ChainHead = "deadbeef"
	resBad, err := VerifySnapshot(rs, snapBad)
	if err != nil {
		t.Fatalf("verify tampered chain head: %v", err)
	}
	if resBad.Valid {
		t.Fatalf("expected invalid snapshot on chain head tamper")
	}
	if resBad.Reason != "hash_mismatch" && resBad.Reason != "chain_head_mismatch" {
		t.Fatalf("unexpected reason: %s", resBad.Reason)
	}
	// Tamper 2: modify merkle root only (simulate corruption) keeping chain head same
	if snap.MerkleRoot != "" {
		snapBad2 := snap
		snapBad2.MerkleRoot = "cafecafe"
		resBad2, err := VerifySnapshot(rs, snapBad2)
		if err != nil {
			t.Fatalf("verify tampered merkle: %v", err)
		}
		if resBad2.Valid || resBad2.Reason != "merkle_mismatch" {
			t.Fatalf("expected merkle_mismatch; got %+v", resBad2)
		}
	}
}
