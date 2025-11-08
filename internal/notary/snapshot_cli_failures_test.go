package notary

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSnapshotCLIFailureExitCodes verifies CLI returns non-zero exit codes for invalid verification scenarios.
func TestSnapshotCLIFailureExitCodes(t *testing.T) {
	tempDir := t.TempDir()
	receiptsPath := filepath.Join(tempDir, "receipts.json")
	rs := NewReceiptStore(receiptsPath)
	if err := rs.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	os.Setenv("GAUTH_NOTARY_MERKLE_ENABLED", "1")
	// Append two receipts
	for _, h := range []string{"sha256:a1", "sha256:a2"} {
		r := Receipt{Hash: h, Timestamp: "2025-10-20T00:00:00Z", Provider: "memory", Version: 1, Success: true}
		if _, err := rs.Append(r); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	// Locate repo root
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}
	walk := filepath.Dir(file)
	repoRoot := ""
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(walk, "go.mod")); err == nil {
			repoRoot = walk
			break
		}
		parent := filepath.Dir(walk)
		if parent == walk {
			break
		}
		walk = parent
	}
	if repoRoot == "" {
		t.Fatalf("repo root not found")
	}
	// Generate snapshot
	snapshotPath := filepath.Join(tempDir, "snapshot.json")
	gen := exec.Command("go", "run", "./cmd/snapshot", "-receipts", receiptsPath, "-out", snapshotPath)
	gen.Dir = repoRoot
	gen.Env = os.Environ()
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generation failed: %v output=%s", err, string(out))
	}
	// Read snapshot JSON
	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	// Case 1: Tamper merkle root
	snap.MerkleRoot = tamperHexShort // invalid hex but CLI just string compares
	tamperedMerklePath := filepath.Join(tempDir, "snap_bad_merkle.json")
	b, _ := json.Marshal(snap)
	if err := os.WriteFile(tamperedMerklePath, b, 0o600); err != nil {
		t.Fatalf("write tampered merkle: %v", err)
	}
	verify1 := exec.Command("go", "run", "./cmd/snapshot", "-verify", "-receipts", receiptsPath, "-snapshot", tamperedMerklePath)
	verify1.Dir = repoRoot
	verify1.Env = os.Environ()
	if out, err := verify1.CombinedOutput(); err == nil {
		t.Fatalf("expected verification failure for merkle tamper; output=%s", string(out))
	}
	// Case 2: Tamper chain head
	var snap2 Snapshot
	if err := json.Unmarshal(raw, &snap2); err != nil {
		t.Fatalf("unmarshal snapshot2: %v", err)
	}
	snap2.ChainHead = "sha256:ffffffff" // mismatch
	tamperedChainPath := filepath.Join(tempDir, "snap_bad_chain.json")
	b2, _ := json.Marshal(snap2)
	if err := os.WriteFile(tamperedChainPath, b2, 0o600); err != nil {
		t.Fatalf("write tampered chain: %v", err)
	}
	verify2 := exec.Command("go", "run", "./cmd/snapshot", "-verify", "-receipts", receiptsPath, "-snapshot", tamperedChainPath)
	verify2.Dir = repoRoot
	verify2.Env = os.Environ()
	if out, err := verify2.CombinedOutput(); err == nil {
		t.Fatalf("expected verification failure for chain head tamper; output=%s", string(out))
	}
}
