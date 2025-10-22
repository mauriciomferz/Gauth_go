package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDiscoveryDeprecationMetadata(t *testing.T) {
	os.Setenv("GAUTH_DEPRECATED_AFTER", "2025-12-31T00:00:00Z")
	os.Setenv("GAUTH_SUNSET_AFTER", "2026-06-30T00:00:00Z")
	s := NewBetaServer("0")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["deprecated_after"] != "2025-12-31T00:00:00Z" {
		t.Fatalf("deprecated_after mismatch: %v", body["deprecated_after"])
	}
	if body["sunset_after"] != "2026-06-30T00:00:00Z" {
		t.Fatalf("sunset_after mismatch: %v", body["sunset_after"])
	}
}
