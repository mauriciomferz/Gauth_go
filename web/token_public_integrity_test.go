package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestTokenIntegrityPublicRS256(t *testing.T) {
	os.Setenv("GAUTH_USE_JWT_LIB", "1")
	os.Setenv("GAUTH_JWT_ALG", "RS256")
	srv := NewBetaServer(":0")
	// Issue token
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", strings.NewReader(`{"ttl_seconds":10}`))
	req.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("issue status=%d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	jwtVal, _ := body["jwt"].(string)
	if jwtVal == "" {
		t.Fatalf("missing jwt field")
	}
	// Validate token (public path) – expects valid_jwt status
	w2 := httptest.NewRecorder()
	vreq := httptest.NewRequest(http.MethodPost, "/api/v1/token/validate", strings.NewReader(`{"token":"`+jwtVal+`"}`))
	vreq.Header.Set("Content-Type", "application/json")
	srv.router.ServeHTTP(w2, vreq)
	if w2.Code != 200 {
		t.Fatalf("validate status=%d body=%s", w2.Code, w2.Body.String())
	}
	var vbody map[string]any
	_ = json.Unmarshal(w2.Body.Bytes(), &vbody)
	if vbody["status"] != statusValidJWT {
			t.Fatalf("expected %s got %v", statusValidJWT, vbody["status"])
	}
}
