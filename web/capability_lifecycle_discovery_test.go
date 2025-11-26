package web

import (
	"encoding/json"
	"testing"

	"github.com/mauriciomferz/Gauth_go/internal/capability"
)

// TestCapabilityLifecycleDiscovery ensures lifecycle summary object present in /info.
func TestCapabilityLifecycleDiscovery(t *testing.T) {
	t.Setenv("GAUTH_CAP_LIFECYCLE_STRICT", "1")
	t.Setenv("GAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE", "1")
	capability.Reset([]capability.Capability{})
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	resp := doGET(t, srv, "/api/v1/beta/info")
	if resp.Code != 200 {
		t.Fatalf("expected 200 got %d", resp.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	lc, ok := doc["capability_lifecycle"].(map[string]any)
	if !ok {
		t.Fatalf("capability_lifecycle object missing")
	}
	if v, ok := lc["strict_enabled"].(bool); !ok || !v {
		t.Fatalf("strict_enabled flag missing or false")
	}
	if v, ok := lc["sunset_enforce_enabled"].(bool); !ok || !v {
		t.Fatalf("sunset_enforce_enabled flag missing or false")
	}
}
