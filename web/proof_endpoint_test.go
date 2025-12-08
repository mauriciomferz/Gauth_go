package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	delegation "github.com/mauriciomferz/Gauth_go/pkg/delegation"
)

// helper to create beta server with revocation events appended
func newTestServerWithRevocations(t *testing.T, n int) *BetaServer {
	t.Helper()
	gin.SetMode(gin.TestMode)
	srv := NewBetaServer("")
	t.Cleanup(func() {
		srv.Shutdown()
	})
	// ensure chain exists
	if srv.revocationChain == nil {
		srv.revocationChain = delegation.NewRevocationChain()
	}
	// append deterministic dummy events
	for i := 0; i < n; i++ {
		ev := delegation.RevocationEvent{ID: fmt.Sprintf("rev-%d", i), DelegationID: fmt.Sprintf("del-%d", i), Reason: "user_request"}
		if _, err := srv.revocationChain.Append(ev); err != nil {
			t.Fatalf("append revocation: %v", err)
		}
	}
	// Sign a tree head snapshot to populate sth_latest discovery (optional)
	if _, err := srv.revocationChain.SignTreeHead(); err != nil {
		t.Fatalf("sign tree head: %v", err)
	}
	return srv
}

func TestProofEndpointByID(t *testing.T) {
	srv := newTestServerWithRevocations(t, 3)
	events := srv.revocationChain.Events()
	targetID := events[0].ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/proof?id="+targetID, nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), targetID) {
		t.Fatalf("response missing target id; body=%s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "merkle_root") {
		t.Fatalf("missing merkle_root in response")
	}
}

func TestProofEndpointByIndex(t *testing.T) {
	srv := newTestServerWithRevocations(t, 5)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/proof?index=2", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "index:2") {
		t.Fatalf("expected index identifier in response; body=%s", w.Body.String())
	}
}

func TestProofEndpointByHash(t *testing.T) {
	srv := newTestServerWithRevocations(t, 4)
	events := srv.revocationChain.Events()
	targetHash := events[2].Hash
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/proof?hash="+targetHash, nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), targetHash) {
		t.Fatalf("expected hash in response; body=%s", w.Body.String())
	}
}

func TestProofEndpointMissingParams(t *testing.T) {
	srv := newTestServerWithRevocations(t, 2)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/proof", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for missing params, got %d", w.Code)
	}
}

func TestProofEndpointBadIndex(t *testing.T) {
	srv := newTestServerWithRevocations(t, 2)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/proof?index=notanumber", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for bad index, got %d", w.Code)
	}
}
