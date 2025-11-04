package audit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	webpkg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"
)

func perform(server *webpkg.BetaServer, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)
	return w
}

func TestAuditVerifyUnconfigured(t *testing.T) {
	// No GAUTH_CAP_AUDIT_PERSIST_PATH set -> configured=false
	srv := webpkg.NewBetaServer(":0")
	w := perform(srv, http.MethodGet, "/api/v1/beta/capabilities/audit/verify")
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool `json:"success"`
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || resp.Configured {
		t.Fatalf("expected configured=false got %+v", resp)
	}
}

func TestAuditAnchorDisabled(t *testing.T) {
	// Ensure anchoring disabled (env unset)
	srv := webpkg.NewBetaServer(":0")
	w := perform(srv, http.MethodPost, "/api/v1/beta/capabilities/audit/anchor")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Success || resp.Code != "anchoring_disabled" || resp.Error != "capability_anchor_disabled" {
		t.Fatalf("unexpected error payload %+v", resp)
	}
}
