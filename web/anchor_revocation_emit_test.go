package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/anchor"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
	"github.com/gin-gonic/gin"
)

// buildTestServerMinimal constructs a BetaServer with routes initialized for testing new endpoints.
func buildTestServerMinimal() *BetaServer {
	gin.SetMode(gin.TestMode)
	s := NewBetaServer("")
	// Routes for new endpoints live under the beta group; ensure router initialized.
	// NewBetaServer already registers all routes, including /api/v1/anchor/revocation/emit
	return s
}

func TestRevocationAnchorEmit_EmptyChain(t *testing.T) {
	s := buildTestServerMinimal()
	t.Cleanup(func() { s.Shutdown() })
	// Attach anchor client to avoid client_unavailable error, keep chain empty to trigger revocation_chain_empty
	s.anchorClient = anchor.NewMemoryAnchor()
	if len(s.revocationChain.Events()) != 0 {
		t.Fatalf("expected empty revocation chain for test precondition")
	}
	w := httptest.NewRecorder()
	// Use stable path alias per OpenAPI spec
	req, _ := http.NewRequest("POST", "/api/v1/anchor/revocation/emit", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for empty chain got %d body=%s", w.Code, w.Body.String())
	}
	var body struct{ Code string }
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != "revocation_chain_empty" {
		t.Fatalf("expected code revocation_chain_empty got %s", body.Code)
	}
}

func TestRevocationAnchorEmit_Success(t *testing.T) {
	s := buildTestServerMinimal()
	t.Cleanup(func() { s.Shutdown() })
	// Populate chain with one event
	ev := delegation.RevocationEvent{ID: "rev-1", DelegationID: "del-1", Reason: string(delegation.RevocationReasonUserRequest)}
	if _, err := s.revocationChain.Append(ev); err != nil {
		t.Fatalf("append revocation: %v", err)
	}
	// Attach memory anchor client
	s.anchorClient = anchor.NewMemoryAnchor()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/anchor/revocation/emit", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Success     bool   `json:"success"`
		Hash        string `json:"hash"`
		MerkleRoot  string `json:"merkle_root"`
		ChainLength int    `json:"chain_length"`
		Type        string `json:"type"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Success || body.Type != "revocation_root" || body.ChainLength != 1 || body.Hash == "" || body.MerkleRoot == "" {
		t.Fatalf("unexpected response: %+v", body)
	}
	// Idempotency: second call should return same hash
	time.Sleep(5 * time.Millisecond)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/anchor/revocation/emit", nil)
	s.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 idempotent got %d", w2.Code)
	}
	var body2 struct{ Hash string }
	_ = json.Unmarshal(w2.Body.Bytes(), &body2)
	if body2.Hash != body.Hash {
		t.Fatalf("expected same anchor hash on idempotent second call: first=%s second=%s", body.Hash, body2.Hash)
	}
}

func TestRevocationAnchorEmit_NoAnchorClient(t *testing.T) {
	s := buildTestServerMinimal()
	t.Cleanup(func() { s.Shutdown() })
	// Populate chain but do not set anchor client
	ev := delegation.RevocationEvent{ID: "rev-2", DelegationID: "del-2", Reason: string(delegation.RevocationReasonUserRequest)}
	_, _ = s.revocationChain.Append(ev)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/anchor/revocation/emit", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d body=%s", w.Code, w.Body.String())
	}
	var body struct{ Code string }
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code != "revocation_anchor_client_unavailable" {
		t.Fatalf("expected code revocation_anchor_client_unavailable got %s", body.Code)
	}
}
