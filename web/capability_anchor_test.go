package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCapabilityAnchoringLifecycle validates POST anchor and GET latest endpoints.
func TestCapabilityAnchoringLifecycle(t *testing.T) {
	t.Setenv("AGENTAUTH_ANCHOR_PROVIDER", "memory")
	t.Setenv("AGENTAUTH_CAPABILITY_ANCHOR_ENABLE", "1")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Wait for async startup anchor to complete
	time.Sleep(100 * time.Millisecond)
	// Initial latest should show anchored=false
	w0 := httptest.NewRecorder()
	req0 := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/latest", nil)
	srv.router.ServeHTTP(w0, req0)
	if w0.Code != 200 {
		t.Fatalf("latest status %d body=%s", w0.Code, w0.Body.String())
	}
	var latest0 map[string]any
	if err := json.Unmarshal(w0.Body.Bytes(), &latest0); err != nil {
		t.Fatalf("json latest0: %v", err)
	}
	anchored0, ok := latest0["anchored"].(bool)
	if !ok {
		t.Fatalf("latest0.anchored wrong type %#v", latest0["anchored"])
	}
	if !anchored0 {
		t.Fatalf("expected anchored=true (auto-seeded on startup)")
	}

	// POST anchor current registry hash
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/api/v1/beta/capabilities/anchor", nil)
	srv.router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("anchor post status %d body=%s", w1.Code, w1.Body.String())
	}
	var post1 map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &post1); err != nil {
		t.Fatalf("json post1: %v", err)
	}
	hash, ok := post1["hash"].(string)
	if !ok || hash == "" {
		t.Fatalf("expected hash string in post response got %#v", post1["hash"])
	}

	// Idempotent re-anchor (same hash)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/beta/capabilities/anchor", nil)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("anchor post2 status %d body=%s", w2.Code, w2.Body.String())
	}
	var post2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &post2); err != nil {
		t.Fatalf("json post2: %v", err)
	}
	hash2, ok := post2["hash"].(string)
	if !ok || hash2 == "" {
		t.Fatalf("post2 hash invalid %#v", post2["hash"])
	}
	if hash2 != hash {
		t.Fatalf("idempotent hash mismatch: %s vs %s", hash2, hash)
	}

	// Latest should now be anchored=true
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/latest", nil)
	srv.router.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("latest after status %d body=%s", w3.Code, w3.Body.String())
	}
	var latest1 map[string]any
	if err := json.Unmarshal(w3.Body.Bytes(), &latest1); err != nil {
		t.Fatalf("json latest1: %v", err)
	}
	anchored1, ok := latest1["anchored"].(bool)
	if !ok {
		t.Fatalf("latest1.anchored wrong type %#v", latest1["anchored"])
	}
	if !anchored1 {
		t.Fatalf("expected anchored=true after POSTs")
	}
	latestRaw, ok := latest1["latest"].(map[string]any)
	if !ok || latestRaw == nil {
		t.Fatalf("expected latest object present got %#v", latest1["latest"])
	}
	lh, ok := latestRaw["hash"].(string)
	if !ok || lh == "" {
		t.Fatalf("latest.hash invalid %#v", latestRaw["hash"])
	}
	if lh != hash {
		t.Fatalf("latest hash mismatch expected %s got %s", hash, lh)
	}
}

// TestCapabilityAnchorDisabled ensures POST returns 403 when enable flag not set.
func TestCapabilityAnchorDisabled(t *testing.T) {
	t.Setenv("AGENTAUTH_CAPABILITY_ANCHOR_ENABLE", "")
	t.Setenv("AGENTAUTH_ANCHOR_PROVIDER", "memory")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/beta/capabilities/anchor", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("expected 403 got %d body=%s", w.Code, w.Body.String())
	}
}
