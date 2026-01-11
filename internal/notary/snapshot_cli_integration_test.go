package notary

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// Integration test invoking the snapshot CLI for generation and verification.
func TestSnapshotCLIIntegration(t *testing.T) {
	// Build receipts file via ReceiptStore
	tempDir := t.TempDir()
	receiptsPath := filepath.Join(tempDir, "receipts.json")
	rs := NewReceiptStore(receiptsPath)
	if err := rs.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	t.Setenv("AGENTAUTH_NOTARY_MERKLE_ENABLED", "1")
	for _, h := range []string{"sha256:x1", "sha256:x2"} {
		r := Receipt{Hash: h, Timestamp: "2025-10-20T00:00:00Z", Provider: "memory", Version: 1, Success: true}
		if _, err := rs.Append(r); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	snapshotPath := filepath.Join(tempDir, "snapshot.json")
	// Generate snapshot
	// Determine module root (two directories up from this file's directory if needed)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}
	walkDir := filepath.Dir(file) // initial directory for upward traversal
	repoRoot := ""
	for i := 0; i < 10; i++ { // limit traversal depth
		if _, err := os.Stat(filepath.Join(walkDir, "go.mod")); err == nil {
			repoRoot = walkDir
			break
		}
		parent := filepath.Dir(walkDir)
		if parent == walkDir {
			break
		}
		walkDir = parent
	}
	if repoRoot == "" {
		t.Fatalf("could not locate repo root (go.mod) from %s", file)
	}
	cmdGen := exec.Command("go", "run", "./cmd/snapshot", "-receipts", receiptsPath, "-out", snapshotPath)
	cmdGen.Env = os.Environ()
	cmdGen.Dir = repoRoot
	outGen, err := cmdGen.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot generation failed: %v output=%s", err, string(outGen))
	}
	// Verify snapshot
	cmdVerify := exec.Command("go", "run", "./cmd/snapshot", "-verify", "-receipts", receiptsPath, "-snapshot", snapshotPath)
	cmdVerify.Env = os.Environ()
	cmdVerify.Dir = repoRoot
	outVerify, err := cmdVerify.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot verification command failed: %v output=%s", err, string(outVerify))
	}
	// Parse JSON result and assert valid
	var parsed struct {
		Result struct {
			Valid bool `json:"valid"`
		} `json:"result"`
	}
	if err := json.Unmarshal(outVerify, &parsed); err != nil {
		t.Fatalf("unmarshal verify output: %v output=%s", err, string(outVerify))
	}
	if !parsed.Result.Valid {
		t.Fatalf("expected valid snapshot; output=%s", string(outVerify))
	}
}
