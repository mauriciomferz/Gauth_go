package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestSemanticIntegrityMismatch simulates tampering by altering persisted previous hash causing integrity_status=mismatch.
func TestSemanticIntegrityMismatch(t *testing.T) {
    tmp := t.TempDir()
    persistPath := tmp + "/semantic_prev_hash.txt"
    os.Setenv("GAUTH_SEMANTIC_INTEGRITY_PERSIST_PATH", persistPath)
    s := NewBetaServer("")
    // First call establishes baseline and writes current hash.
    w1 := httptest.NewRecorder()
    req1, _ := http.NewRequest("GET", "/api/v1/diagnostics/semantic", http.NoBody)
    s.router.ServeHTTP(w1, req1)
    if w1.Code != http.StatusOK { t.Fatalf("first call status %d body=%s", w1.Code, w1.Body.String()) }
    var first struct { Integrity string `json:"integrity_status"`; Current string `json:"current_hash"` }
    if err := json.Unmarshal(w1.Body.Bytes(), &first); err != nil { t.Fatalf("unmarshal first: %v", err) }
    if first.Current == "" { t.Fatalf("expected non-empty current hash on first call") }
    // Tamper: overwrite persisted file with a different hash than prevHash that server will expect next time.
    if err := os.WriteFile(persistPath, []byte("sha256:tampered"), 0o600); err != nil { t.Fatalf("tamper write: %v", err) }
    time.Sleep(5 * time.Millisecond) // ensure timestamp difference not affecting logic
    // Second call should detect mismatch.
    w2 := httptest.NewRecorder()
    req2, _ := http.NewRequest("GET", "/api/v1/diagnostics/semantic", http.NoBody)
    s.router.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK { t.Fatalf("second call status %d body=%s", w2.Code, w2.Body.String()) }
    var second struct { Integrity string `json:"integrity_status"`; Prev string `json:"prev_hash"`; Current string `json:"current_hash"` }
    if err := json.Unmarshal(w2.Body.Bytes(), &second); err != nil { t.Fatalf("unmarshal second: %v", err) }
    if second.Integrity != "mismatch" { t.Fatalf("expected integrity_status mismatch got %s prev=%s cur=%s", second.Integrity, second.Prev, second.Current) }
}
