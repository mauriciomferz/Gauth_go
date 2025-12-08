package web

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
)

// TestValidationFailureCounters exercises token & delegation status update failure paths
// and asserts the dedicated validation failure counters increment.
func TestValidationFailureCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := imetrics.NewMemory()
	// Use NewBetaServerWithMetrics to properly inject metrics at construction time
	srv := NewBetaServerWithMetrics("", m)
	t.Cleanup(func() { srv.Shutdown() })
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
	wCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":60}`))
	reqCreate.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wCreate, reqCreate)
	var resp struct {
		Token struct {
			ID string
		} `json:"token"`
		JWT string `json:"jwt"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &resp)

	id := resp.Token.ID
	if id == "" {
		// Maybe JWT mode?
		if resp.JWT != "" {
			// JWT ID extraction not easy without parsing.
			// But default config might use opaque tokens.
			// If JWT, we can't easily get ID unless we parse it.
			// Let's assume non-JWT for this test or that Create returns ID in token object.
			// NewBetaServer might use Tokens which are opaque by default if JWT env not set.
			// But NewBetaServer might default to JWT?
			// Default BetaServer uses JWT if GAUTH_USE_JWT_LIB=1.
			// Test doesn't set it. So it uses opaque tokens.
		}
	}

	// Terminate it
	post("/api/v1/token/status/update", `{"token_id":"`+id+`","new_status":"terminated"}`)

	// Try to activate
	code = post("/api/v1/token/status/update", `{"token_id":"`+id+`","new_status":"active"}`)
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
	// Initialize to terminated via API
	post("/api/v1/delegation/status/update", `{"delegation_id":"del_term","new_status":"terminated"}`)
	code = post("/api/v1/delegation/status/update", `{"delegation_id":"del_term","new_status":"active"}`)
	if code != 409 {
		t.Fatalf("expected 409 invalid transition delegation got %d", code)
	}
	if got := m.InvalidTransitionFailures(); got != 2 {
		t.Fatalf("expected invalid_transition_failures=2 got %d", got)
	}
}
