package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// buildTestServer constructs a minimal BetaServer with revocation chain and key manager.
func buildTestServer(t *testing.T) *BetaServer {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// Attach a fresh key manager (1h TTL) for signatures.
	km, err := crypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	s := NewBetaServer("", WithKeyProvider(km))
	// Initialize revocation chain with two events.
	rc := delegation.NewRevocationChain(delegation.WithKeyProvider(km))
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

// ControllableKeyProvider wraps a KeyProvider to simulate key loss.
type ControllableKeyProvider struct {
	Delegate crypto.KeyProvider
	Enabled  bool
}

func (c *ControllableKeyProvider) ActiveSigner() (crypto.Signer, error) {
	if !c.Enabled {
		return nil, errors.New("key provider disabled")
	}
	return c.Delegate.ActiveSigner()
}

func (c *ControllableKeyProvider) PublicKey(keyID string) ([]byte, string, error) {
	if !c.Enabled {
		return nil, "", errors.New("key provider disabled")
	}
	return c.Delegate.PublicKey(keyID)
}

func (c *ControllableKeyProvider) VerifyWith(msg, sig []byte, keyID string) error {
	if !c.Enabled {
		return errors.New("key provider disabled")
	}
	return c.Delegate.VerifyWith(msg, sig, keyID)
}

func TestRevocationVerifyEndpointAfterKeyLoss(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Init real key manager
	km, err := crypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	// Wrap it
	ckp := &ControllableKeyProvider{Delegate: km, Enabled: true}

	// Create server with wrapper
	s := NewBetaServer("", WithKeyProvider(ckp))

	// Create chain with wrapper (so events get signed)
	rc := delegation.NewRevocationChain(delegation.WithKeyProvider(ckp))
	_, _ = rc.Append(delegation.RevocationEvent{ID: "rev-1", DelegationID: "del-1"})
	_, _ = rc.Append(delegation.RevocationEvent{ID: "rev-2", DelegationID: "del-2"})
	s.revocationChain = rc

	// Now simulate key loss
	ckp.Enabled = false

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
	req, _ := http.NewRequest("GET", "/.well-known/agentauth-configuration", nil)
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
