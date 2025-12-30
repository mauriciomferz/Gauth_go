package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	testutil "github.com/mauriciomferz/AgentAuth/web/testutil"
)

// TestNotarizationReceiptVerification exercises the /verify endpoint for normal and tampered chains.
func TestNotarizationReceiptVerification(t *testing.T) {
	dir := t.TempDir()
	capFile := filepath.Join(dir, "caps.json")
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferV1), 0o600); err != nil {
		t.Fatalf("write caps: %v", err)
	}
	anchorPath := filepath.Join(dir, "anchor.json")
	receiptPath := filepath.Join(dir, "receipts.json")
	t.Setenv("GAUTH_CAPABILITIES_PATH", capFile)
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", anchorPath)
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1s")
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARIZE", "1")
	t.Setenv("GAUTH_CAP_ANCHOR_NOTARY_PROVIDER", "memory")
	t.Setenv("GAUTH_NOTARY_RECEIPT_PERSIST_PATH", receiptPath)
	srv := NewBetaServer("0")
	t.Cleanup(func() { srv.Shutdown() })
	// Trigger second emission to ensure persistence exists
	time.Sleep(1100 * time.Millisecond)
	if err := os.WriteFile(capFile, []byte(testutil.CapTransferIssueV1), 0o600); err != nil {
		t.Fatalf("write caps2: %v", err)
	}
	if err := srv.capabilitiesHandler.LoadFromFile(capFile); err != nil {
		t.Fatalf("reload caps2: %v", err)
	}

	// Call verify endpoint (should be ok)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/notarization/receipts/verify", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("verify status=%d body=%s", w.Code, w.Body.String())
	}
	if !containsAll(w.Body.String(), []string{"\"integrity\":\"ok\""}) {
		t.Fatalf("expected integrity ok body=%s", w.Body.String())
	}

	// Tamper the persistence file: modify first entry's hash field to induce mismatch.
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	var file struct {
		Entries []map[string]interface{} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("unmarshal receipts: %v", err)
	}
	if len(file.Entries) == 0 {
		t.Fatalf("expected entries")
	}
	file.Entries[0]["hash"] = "tampered" // change input affecting chain hash recomputation
	tampered, _ := json.Marshal(file)
	// Minimal rewrite (ignore chain_head for mismatch simulation) - this will break recomputation for entry 0.
	if err := os.WriteFile(receiptPath, tampered, 0o600); err != nil {
		t.Fatalf("write tampered: %v", err)
	}
	// Reload store so server sees tampering (optional)
	_ = srv.receiptStore.(interface{ Load() error }).Load()
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/beta/notarization/receipts/verify", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("verify tampered status=%d body=%s", w2.Code, w2.Body.String())
	}
	if !containsAll(w2.Body.String(), []string{"\"integrity\":\"mismatch\""}) {
		t.Fatalf("expected integrity mismatch body=%s", w2.Body.String())
	}
}
