package capabilities_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webpkg "github.com/mauriciomferz/AgentAuth/web"
)

// helper to perform request against BetaServer
func perform(server *webpkg.BetaServer, method, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)
	return w
}

func TestCapabilitiesListBasic(t *testing.T) {
	srv := webpkg.NewBetaServer("")
	w := perform(srv, http.MethodGet, "/api/v1/beta/capabilities", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success      bool                     `json:"success"`
		Capabilities []map[string]interface{} `json:"capabilities"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success false")
	}
	if len(resp.Capabilities) == 0 {
		t.Fatalf("expected >=1 capability")
	}
	// Basic shape assertions
	first := resp.Capabilities[0]
	if _, ok := first["id"]; !ok {
		t.Fatalf("missing id field")
	}
	if _, ok := first["version"]; !ok {
		t.Fatalf("missing version field")
	}
}

func TestCapabilitiesNegotiateHappy(t *testing.T) {
	srv := webpkg.NewBetaServer("")
	// Discover a capability id + version from list first
	wList := perform(srv, http.MethodGet, "/api/v1/beta/capabilities", "")
	var list struct {
		Success      bool `json:"success"`
		Capabilities []struct {
			ID      string `json:"id"`
			Version string `json:"version"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(wList.Body.Bytes(), &list); err != nil || len(list.Capabilities) == 0 {
		t.Fatalf("list decode failed err=%v len=%d", err, len(list.Capabilities))
	}
	cid := list.Capabilities[0].ID
	ver := list.Capabilities[0].Version
	payload := `{"client_versions": {"` + cid + `": ["` + ver + `", "9.9"], "unknown.cap": ["0.1"]}}`
	w := perform(srv, http.MethodPost, "/api/v1/beta/capabilities/negotiate", payload)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success         bool                `json:"success"`
		Agreed          map[string]string   `json:"agreed"`
		Unsupported     map[string][]string `json:"unsupported"`
		LifecycleStrict bool                `json:"lifecycle_strict"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success false")
	}
	if resp.Agreed[cid] != ver {
		t.Fatalf("expected agreed %s=%s got %v", cid, ver, resp.Agreed)
	}
	if _, ok := resp.Unsupported["unknown.cap"]; !ok {
		t.Fatalf("expected unknown.cap unsupported list present")
	}
}

func TestCapabilitiesNegotiateInvalidPayload(t *testing.T) {
	srv := webpkg.NewBetaServer("")
	// Empty object should trigger invalid payload error
	w := perform(srv, http.MethodPost, "/api/v1/beta/capabilities/negotiate", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
	var resp struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected success=false")
	}
	if resp.Code != "capabilities_negotiate_invalid_payload" {
		t.Fatalf("unexpected code %s", resp.Code)
	}
	if resp.Error != "invalid_payload" {
		t.Fatalf("unexpected error %s", resp.Error)
	}
}
