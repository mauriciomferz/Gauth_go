package web

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestTokenCreateClientNonceReplay verifies strict nonce reuse rejection.
func TestTokenCreateClientNonceReplay(t *testing.T) {
	t.Setenv("GAUTH_REPLAY_STRICT", "1")
	gin.SetMode(gin.TestMode)
	s := NewBetaServer(":0")
	t.Cleanup(func() { s.Shutdown() })
	nonce := "demo-nonce-1"
	body1, _ := json.Marshal(map[string]interface{}{"ttl_seconds": 60, "nonce": nonce})
	req1 := httptest.NewRequest("POST", "/api/v1/token/create", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	s.router.ServeHTTP(w1, req1)
	// Endpoint returns 201 (Created) on success; accept 200 or 201 for backward compatibility.
	if w1.Code != 200 && w1.Code != 201 {
		t.Fatalf("expected 200/201 first create got %d body=%s", w1.Code, w1.Body.String())
	}
	req2 := httptest.NewRequest("POST", "/api/v1/token/create", bytes.NewReader(body1))
	w2 := httptest.NewRecorder()
	s.router.ServeHTTP(w2, req2)
	if w2.Code != 409 {
		t.Fatalf("expected 409 replay detection got %d body=%s", w2.Code, w2.Body.String())
	}
	if !bytes.Contains(w2.Body.Bytes(), []byte("nonce_reused")) {
		t.Fatalf("expected nonce_reused code in body: %s", w2.Body.String())
	}
}
