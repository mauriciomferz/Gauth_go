package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestViolationPersistence ensures counters are saved and restored when persistence path is configured.
func TestViolationPersistence(t *testing.T) {
	dir := t.TempDir()
	persistFile := filepath.Join(dir, "violations.json")
	os.Setenv("GAUTH_VIOLATION_PERSIST_PATH", persistFile)
	os.Setenv("GAUTH_VIOLATION_AUTOSAVE_SEC", "0") // disable autosave loop for test
	os.Setenv("GAUTH_VIOLATION_PERSIST_NO_THROTTLE", "1")
	srv := NewBetaServer("8091")
	// Trigger some violations (empty token)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/token/validate", strings.NewReader(`{"token":""}`))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
	}
	// Force save
	srv.saveViolationPersistence()
	// Read file
	data, err := os.ReadFile(persistFile)
	if err != nil {
		t.Fatalf("read persistence file: %v", err)
	}
	// Support new hash-chain wrapper or legacy direct format.
	var wrapper struct {
		Payload   json.RawMessage `json:"payload"`
		PrevHash  string          `json:"prev_hash"`
		Hash      string          `json:"hash"`
		Timestamp string          `json:"timestamp"`
	}
	raw := data
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Payload) > 0 {
		raw = wrapper.Payload
		if wrapper.Hash == "" {
			t.Fatalf("expected hash in wrapper")
		}
	}
	var payload struct {
		Counters map[string]uint64 `json:"counters"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Counters["missing_claim"] < 5 {
		t.Fatalf("expected missing_claim >=5, got %d", payload.Counters["missing_claim"])
	}
	// New server should restore counters
	srv2 := NewBetaServer("8092")
	snap := srv2.primaryAuthService.ViolationSnapshot()
	if snap["missing_claim"] < 5 {
		t.Fatalf("restored missing_claim expected >=5 got %d", snap["missing_claim"])
	}
}
