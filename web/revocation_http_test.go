package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
	"github.com/gin-gonic/gin"
)

// buildTestServer constructs a minimal BetaServer with revocation chain and key manager.
func buildTestServer(t *testing.T) *BetaServer {
	t.Helper()
	gin.SetMode(gin.TestMode)
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	// Attach a fresh key manager (1h TTL) for signatures.
	km, err := crypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	// Initialize revocation chain with two events.
	rc := delegation.NewRevocationChain()
	_, _ = rc.Append(delegation.RevocationEvent{ID: "rev-1", DelegationID: "del-1"})
	_, _ = rc.Append(delegation.RevocationEvent{ID: "rev-2", DelegationID: "del-2"})
	s.revocationChain = rc
	// Re-register routes (constructor does this already) but ensure using same router.
	// server_clean.go sets up routes in NewBetaServer; chain already assigned so endpoints will see it.
	return s
}

func TestRevocationVerifyEndpoint(t *testing.T) {
	s := buildTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/verify", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var body struct {
		Success   bool             `json:"success"`
		Length    int              `json:"length"`
		Verified  bool             `json:"verified"`
		Events    []map[string]any `json:"events"`
		Aggregate string           `json:"aggregate_hash"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Success || !body.Verified || body.Length != 2 {
		t.Fatalf("unexpected verify response %+v", body)
	}
	if body.Aggregate == "" {
		t.Fatalf("expected aggregate hash present")
	}
	// Ensure signature_present true for both
	for _, e := range body.Events {
		if v, ok := e["signature_present"].(bool); !ok || !v {
			t.Fatalf("expected signature_present true")
		}
		if v, ok := e["signature_valid"].(bool); !ok || !v {
			t.Fatalf("expected signature_valid true")
		}
	}
}

func TestRevocationVerifyEndpointAfterKeyLoss(t *testing.T) {
	s := buildTestServer(t)
	// Remove manager (simulate unknown kid scenario) and call endpoint again
	crypto.GlobalEdDSARegistry = nil
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/verify", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	// Global chain verification should fail; endpoint sets verified false and includes verification_error.
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v, ok := body["verified"].(bool); !ok || v {
		t.Fatalf("expected verified=false after key loss")
	}
	if _, ok := body["verification_error"]; !ok {
		t.Fatalf("expected verification_error field present")
	}
}

func TestDiscoveryIncludesSignatureMetadata(t *testing.T) {
	s := buildTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Metadata is nested under revocation_support
	rs, ok := body["revocation_support"].(map[string]any)
	if !ok {
		t.Fatalf("missing revocation_support")
	}
	if v, ok := rs["signatures_enabled"].(bool); !ok || !v {
		t.Fatalf("expected signatures_enabled=true")
	}
	if list, ok := rs["signing_kids"].([]any); !ok || len(list) == 0 {
		t.Fatalf("expected signing_kids non-empty")
	}
}
