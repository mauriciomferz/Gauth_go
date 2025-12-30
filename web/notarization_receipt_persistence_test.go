package web

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/web/testutil"
)

// TestNotarizationReceiptPersistence verifies that successful capability anchor notarization receipts
// are persisted with a hash-chain and exposed via the latest and chain endpoints.
// Flow:
// 1. Create a temporary capabilities file and configure anchor emission + notarization + receipt persistence.
// 2. Instantiate server which loads capabilities (triggering anchor emission + notarization).
// 3. Validate receipt persistence file structure and hash-chain for first entry.
// 4. Invoke receipt latest and chain endpoints and assert expected JSON fields.
// 5. (Optional) Force a second emission by waiting > interval and modifying capabilities file; ensure chain grows.
func TestNotarizationReceiptPersistence(t *testing.T) {
	dir := t.TempDir()
	capFile := filepath.Join(dir, "caps.json")
	// Minimal valid capabilities file (schema_version required)
	// action_mappings include one mapping to exercise hashing determinism.
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err != nil {
		t.Fatalf("write capabilities file: %v", err)
	}

	anchorPath := filepath.Join(dir, "anchor.json")
	receiptPath := filepath.Join(dir, "receipts.json")
	// Use t.Setenv to ensure environment modifications are reverted after this test, preventing cross-test interference.
	t.Setenv("AGENTAUTH_CAPABILITIES_PATH", capFile)
	t.Setenv("AGENTAUTH_CAP_ANCHOR_FILE_PATH", anchorPath)
	t.Setenv("AGENTAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1s") // speed up second emission (optional)
	t.Setenv("AGENTAUTH_CAP_ANCHOR_NOTARIZE", "1")
	t.Setenv("AGENTAUTH_CAP_ANCHOR_NOTARY_PROVIDER", "memory") // deterministic success, near-zero latency
	t.Setenv("AGENTAUTH_NOTARY_RECEIPT_PERSIST_PATH", receiptPath)

	srv := NewBetaServer("0")
	t.Cleanup(func() { srv.Shutdown() })
	// NOTE: During NewBetaServer initialization the capability file is loaded BEFORE the notarizer & receiptStore are initialized.
	// Therefore the first anchor emission + notarization occurs without persistence (receiptStore still nil).
	// We trigger a second capability reload (after waiting >= write interval and mutating the file) to ensure persistence occurs.
	if srv.GetCapabilityRegistryHash() == "" {
		t.Fatalf("expected capabilityRegistryHash to be set")
	}
	// First attempt: file SHOULD exist because Initial Load occurs AFTER receiptStore initialization,
	// so startup anchor (via OnReload) persists the first receipt.
	if _, err := os.ReadFile(receiptPath); err != nil {
		t.Fatalf("expected receipts file to be present after initialization: %v", err)
	}
	// Wait > interval then modify capabilities to force hash change and second emission.
	time.Sleep(1100 * time.Millisecond)
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferIssueV1), 0o600); err != nil {
		t.Fatalf("update capabilities file: %v", err)
	}
	if err := srv.capabilitiesHandler.LoadFromFile(capFile); err != nil {
		t.Fatalf("reload capabilities for second emission: %v", err)
	}
	// Now receipts file should exist.
	b, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("read receipt persistence file after second emission: %v", err)
	}
	var file struct {
		Entries []struct {
			Hash           string  `json:"hash"`
			Timestamp      string  `json:"timestamp"`
			Provider       string  `json:"provider"`
			Version        int     `json:"version"`
			Success        bool    `json:"success"`
			LatencySeconds float64 `json:"latency_seconds"`
			PrevHash       string  `json:"prev_hash"`
			ChainHash      string  `json:"chain_hash"`
		} `json:"entries"`
		ChainHead string `json:"chain_head"`
	}
	if err2 := json.Unmarshal(b, &file); err2 != nil {
		t.Fatalf("unmarshal receipt file: %v raw=%s", err2, string(b))
	}
	if len(file.Entries) == 0 {
		t.Fatalf("expected at least one receipt entry persisted")
	}
	// Verify hash-chain for each entry
	for i, e := range file.Entries {
		// Reconstruct base struct used for hashing (excluding chain_hash)
		tmp := struct {
			Hash           string  `json:"hash"`
			Timestamp      string  `json:"timestamp"`
			Provider       string  `json:"provider"`
			Version        int     `json:"version"`
			Success        bool    `json:"success"`
			LatencySeconds float64 `json:"latency_seconds"`
			PrevHash       string  `json:"prev_hash"`
		}{Hash: e.Hash, Timestamp: e.Timestamp, Provider: e.Provider, Version: e.Version, Success: e.Success, LatencySeconds: e.LatencySeconds, PrevHash: e.PrevHash}
		enc, mErr := json.Marshal(tmp)
		if mErr != nil {
			t.Fatalf("marshal base receipt %d: %v", i, mErr)
		}
		expected := fmt.Sprintf("%x", sha256.Sum256(append([]byte(e.PrevHash), enc...)))
		if expected != e.ChainHash {
			t.Fatalf("chain hash mismatch entry=%d expected=%s got=%s prev=%s", i, expected, e.ChainHash, e.PrevHash)
		}
	}
	// ChainHead must equal last entry's chain_hash
	if file.ChainHead != file.Entries[len(file.Entries)-1].ChainHash {
		t.Fatalf("chain_head mismatch expected=%s got=%s", file.Entries[len(file.Entries)-1].ChainHash, file.ChainHead)
	}

	// Exercise latest endpoint
	wLatest := httptest.NewRecorder()
	reqLatest := httptest.NewRequest("GET", "/api/v1/beta/notarization/receipts/latest", nil)
	srv.router.ServeHTTP(wLatest, reqLatest)
	if wLatest.Code != 200 {
		t.Fatalf("latest endpoint status=%d body=%s", wLatest.Code, wLatest.Body.String())
	}
	if !containsAll(wLatest.Body.String(), []string{"chain_hash", "prev_hash", "provider"}) {
		t.Fatalf("latest endpoint missing expected fields body=%s", wLatest.Body.String())
	}

	// Exercise chain endpoint
	wChain := httptest.NewRecorder()
	reqChain := httptest.NewRequest("GET", "/api/v1/beta/notarization/receipts", nil)
	srv.router.ServeHTTP(wChain, reqChain)
	if wChain.Code != 200 {
		t.Fatalf("chain endpoint status=%d body=%s", wChain.Code, wChain.Body.String())
	}
	if !containsAll(wChain.Body.String(), []string{"entries", "total", "chain_hash", "prev_hash"}) {
		t.Fatalf("chain endpoint response missing expected markers body=%s", wChain.Body.String())
	}

	// Trigger a third emission to test append-only growth.
	time.Sleep(1100 * time.Millisecond)
	if err2 := os.WriteFile(capFile, []byte(testutil.CapTransferIssueDelegationCreateV1), 0o600); err2 != nil {
		t.Fatalf("update capabilities file third emission: %v", err2)
	}
	if err := srv.capabilitiesHandler.LoadFromFile(capFile); err != nil {
		t.Fatalf("reload capabilities third emission: %v", err)
	}
	b3, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("read receipt persistence file after third emission: %v", err)
	}
	var file3 struct {
		Entries []struct {
			ChainHash string `json:"chain_hash"`
		} `json:"entries"`
	}
	if uErr := json.Unmarshal(b3, &file3); uErr != nil {
		t.Fatalf("unmarshal third receipt file: %v", uErr)
	}
	if len(file3.Entries) < len(file.Entries) {
		t.Fatalf("expected append-only receipts growth len_third=%d len_second=%d", len(file3.Entries), len(file.Entries))
	}
}

// containsAll helper used for lightweight response body assertions without full JSON parsing overhead.
func containsAll(body string, needles []string) bool {
	for _, n := range needles {
		if !strings.Contains(body, n) {
			return false
		}
	}
	return true
}
