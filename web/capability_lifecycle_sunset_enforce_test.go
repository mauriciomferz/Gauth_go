package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/capability"
)

// TestCapabilitySunsetEnforcement ensures usage is denied after sunset when enforcement enabled.
func TestCapabilitySunsetEnforcement(t *testing.T) {
	t.Log("starting TestCapabilitySunsetEnforcement")
	t.Setenv("GAUTH_CAPABILITY_ENFORCE", "1")
	t.Setenv("GAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE", "1")
	t.Setenv("GAUTH_SKIP_SMOKETEST", "1")
	// Reset registry to avoid cross-test contamination
	capability.Reset([]capability.Capability{})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Register a capability already sunset (timestamp in past)
	past := "2025-01-01T00:00:00Z"
	capability.Register(capability.Capability{ID: "cap.sunset.demo", Version: "1.0", Stable: false, SunsetAfter: past})

	// Attempt delegation create requiring a capability that is sunset (simulate mapping)
	srv.requiredActionCaps["delegation:create"] = []string{"cap.sunset.demo"}
	body := map[string]any{"delegation_id": "d1", "subject": "alice", "delegate": "bob", "claims": map[string]any{"cap": []string{"cap.sunset.demo"}}}
	// Local inline POST helper with timeout context to avoid potential hangs
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/delegation/create", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	resp := rr
	t.Logf("delegation create response code=%d body=%s", resp.Code, resp.Body.String())
	if resp.Code != 403 {
		t.Fatalf("expected 403 denial after sunset got %d body=%s", resp.Code, resp.Body.String())
	}
	// Verify audit entry includes sunset phase
	// List capability audit logs and search for phase=sunset
	logs := doGET(t, srv, "/api/v1/audit/capabilities")
	if logs.Code != 200 {
		t.Fatalf("expected 200 listing audit logs got %d", logs.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(logs.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	entries, _ := doc["entries"].([]any)
	found := false
	for _, e := range entries {
		m, _ := e.(map[string]any)
		if m["action"] == "capability_enforce" {
			meta, _ := m["meta"].(map[string]any)
			if meta != nil {
				if lc, ok := meta["lifecycle"].([]any); ok {
					for _, item := range lc {
						im, _ := item.(map[string]any)
						if im["phase"] == "sunset" {
							found = true
						}
					}
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected lifecycle phase sunset in audit capability_enforce entry")
	}
	// Graceful shutdown to stop background loops
	srv.Shutdown()
	t.Log("completed TestCapabilitySunsetEnforcement (server shutdown invoked)")
}
