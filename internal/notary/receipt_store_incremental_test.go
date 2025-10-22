package notary

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeReceipt(i int) Receipt {
	return Receipt{Hash: hashString(i), Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Provider: "memory", Version: 1, Success: true, LatencySeconds: 0.0001}
}

func hashString(i int) string { return string(rune('a'+(i%26))) + "_hash" }

// TestVerifyIncremental ensures incremental verification only re-hashes new tail segments and detects corruption.
func TestVerifyIncremental(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "receipts.json")
	rs := NewReceiptStore(path)
	if err := rs.Load(); err != nil {
		t.Fatalf("load empty: %v", err)
	}
	// Append 3 receipts
	for i := 0; i < 3; i++ {
		if _, err := rs.Append(makeReceipt(i)); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	// Full verify (state cold)
	status, idx, head := rs.VerifyIncremental()
	if status != receiptStatusOK || idx != -1 || head == "" {
		t.Fatalf("expected %s initial verify, got %s idx=%d head=%s", receiptStatusOK, status, idx, head)
	}
	// Append 2 more receipts (incremental path should verify only tail)
	for i := 3; i < 5; i++ {
		if _, err := rs.Append(makeReceipt(i)); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	status2, idx2, head2 := rs.VerifyIncremental()
	if status2 != receiptStatusOK || idx2 != -1 || head2 == head {
		t.Fatalf("expected %s incremental with new head, got %s idx=%d oldHeadSame=%v", receiptStatusOK, status2, idx2, head2 == head)
	}
	// Corrupt file on disk: Modify prev_hash of last entry then reload and expect mismatch.
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Corrupt by flipping a byte in the last chain_hash value to break hash continuity.
	pat := []byte("\"chain_hash\":\"")
	if off := lastIndex(b, pat); off > 0 {
		// locate start of hash characters
		pos := off + len(pat)
		if pos < len(b) {
			if b[pos] == 'a' {
				b[pos] = 'b'
			} else {
				b[pos] = 'a'
			}
		}
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatalf("rewrite: %v", err)
		}
	} else {
		t.Fatalf("pattern chain_hash not found for corruption")
	}
	// Reload to reflect corruption
	rs2 := NewReceiptStore(path)
	if err := rs2.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	s3, idx3, _ := rs2.VerifyIncremental()
	if s3 != "mismatch" || idx3 == -1 {
		t.Fatalf("expected mismatch after corruption, got %s idx=%d", s3, idx3)
	}
}

// lastIndex returns last index of sub slice in b or -1.
func lastIndex(b, sub []byte) int {
	for i := len(b) - len(sub); i >= 0; i-- {
		if string(b[i:i+len(sub)]) == string(sub) {
			return i
		}
	}
	return -1
}
