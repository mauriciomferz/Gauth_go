package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestRevocationConsistencySizesTrivial verifies trivial proof returned when older==newer==current length.
func TestRevocationConsistencySizesTrivial(t *testing.T) {
	srv := NewBetaServer("")
	// Empty chain length expected => trivial when requesting 0,0
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/token/revocation/consistency_sizes?older=0&newer=0", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 trivial consistency, got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	proof, ok := body["proof"].(map[string]any)
	if !ok {
		t.Fatalf("missing proof object body=%v", body)
	}
	if proof["trivial"] != true {
		t.Fatalf("expected trivial=true proof=%v", proof)
	}
	if path, ok := proof["path"].([]any); ok && len(path) != 0 {
		t.Fatalf("expected empty path proof=%v", proof)
	}
}

// TestRevocationConsistencySizesUnavailable verifies differing sizes produce 501 proof_unavailable.
func TestRevocationConsistencySizesUnavailable(t *testing.T) {
	srv := NewBetaServer("")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/token/revocation/consistency_sizes?older=0&newer=1", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 501 {
		t.Fatalf("expected 501 proof unavailable got %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != "consistency_proof_unavailable" {
		t.Fatalf("expected code consistency_proof_unavailable body=%v", body)
	}
}
