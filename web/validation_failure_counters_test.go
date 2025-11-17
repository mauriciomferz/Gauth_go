package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/gin-gonic/gin"
)

// TestValidationFailureCounters exercises token & delegation status update failure paths
// and asserts the dedicated validation failure counters increment.
func TestValidationFailureCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := imetrics.NewMemory()
	// NewBetaServer signature currently expects config string (observed in other tests). We will set metrics manually.
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	srv.metrics = m
	r := srv.router

	// Helper to POST JSON and decode response
	post := func(path, body string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w.Code
	}

	// 1. invalid payload (missing fields) token
	code := post("/api/v1/token/status/update", `{}`)
	if code != 400 {
		t.Fatalf("expected 400 invalid payload token, got %d", code)
	}
	if got := m.InvalidPayloadFailures(); got != 1 {
		t.Fatalf("expected invalid_payload_failures=1 got %d", got)
	}

	// 2. unsupported status token
	code = post("/api/v1/token/status/update", `{"token_id":"t1","new_status":"bogus"}`)
	if code != 400 {
		t.Fatalf("expected 400 unsupported status token got %d", code)
	}
	if got := m.UnsupportedStatusFailures(); got != 1 {
		t.Fatalf("expected unsupported_status_failures=1 got %d", got)
	}

	// 3. not found token
	code = post("/api/v1/token/status/update", `{"token_id":"does_not_exist","new_status":"active"}`)
	if code != 404 {
		t.Fatalf("expected 404 not found token got %d", code)
	}
	if got := m.NotFoundFailures(); got != 1 {
		t.Fatalf("expected not_found_failures=1 got %d", got)
	}

	// 4. invalid transition token (terminated -> active)
	// First create token implicitly by success path: status update success requires existing token creation logic? If not available, we simulate by inserting into map directly.
	srv.tokens.mu.Lock()
	srv.tokens.tokens["tok_term"] = &Token{ID: "tok_term", Status: "terminated"}
	srv.tokens.mu.Unlock()
	code = post("/api/v1/token/status/update", `{"token_id":"tok_term","new_status":"active"}`)
	if code != 409 {
		t.Fatalf("expected 409 invalid transition token got %d", code)
	}
	if got := m.InvalidTransitionFailures(); got != 1 {
		t.Fatalf("expected invalid_transition_failures=1 got %d", got)
	}

	// Delegation failure paths
	// 5. invalid payload delegation
	code = post("/api/v1/delegation/status/update", `{}`)
	if code != 400 {
		t.Fatalf("expected 400 invalid payload delegation got %d", code)
	}
	if got := m.InvalidPayloadFailures(); got != 2 {
		t.Fatalf("expected invalid_payload_failures=2 got %d", got)
	}

	// 6. unsupported status delegation
	code = post("/api/v1/delegation/status/update", `{"delegation_id":"d1","new_status":"zzz"}`)
	if code != 400 {
		t.Fatalf("expected 400 unsupported status delegation got %d", code)
	}
	if got := m.UnsupportedStatusFailures(); got != 2 {
		t.Fatalf("expected unsupported_status_failures=2 got %d", got)
	}

	// 7. invalid transition delegation: terminated -> active
	// initialize terminated
	srv.delegationStatusMu.Lock()
	srv.delegationStatus["del_term"] = "terminated"
	srv.delegationStatusMu.Unlock()
	code = post("/api/v1/delegation/status/update", `{"delegation_id":"del_term","new_status":"active"}`)
	if code != 409 {
		t.Fatalf("expected 409 invalid transition delegation got %d", code)
	}
	if got := m.InvalidTransitionFailures(); got != 2 {
		t.Fatalf("expected invalid_transition_failures=2 got %d", got)
	}
}
