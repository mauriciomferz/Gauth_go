package web

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/mauriciomferz/AgentAuth/web/testutil"
)

// TestCapabilityPersistenceInvalidMapping ensures reload with invalid action mapping keeps previous state.
func TestCapabilityPersistenceInvalidMapping(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "caps-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(testutil.CapAlphaV1); err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	t.Setenv("GAUTH_CAPABILITIES_PATH", tmp.Name())
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	// modify file referencing unknown capability id
	//nolint:gosec // G306: test file permissions
	if err := os.WriteFile(tmp.Name(), []byte(testutil.CapAlphaUnknownMapping), 0o644); err != nil {
		t.Fatal(err)
	}
	w := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if w.Code != 500 {
		t.Fatalf("expected 500 on invalid reload got %d", w.Code)
	}

	// Discovery should still show original capability and original mapping (transaction:execute)
	disc := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if disc.Code != 200 {
		t.Fatalf("discovery status=%d", disc.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(disc.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	caps := doc["capability_registry"].([]any)
	if len(caps) != 1 {
		t.Fatalf("expected 1 capability stayed got %d", len(caps))
	}
	acts := doc["action_capabilities"].([]any)
	foundExecute := false
	for _, raw := range acts {
		obj := raw.(map[string]any)
		if obj["action"] == "transaction:execute" {
			foundExecute = true
		}
	}
	if !foundExecute {
		t.Fatalf("expected transaction:execute mapping preserved")
	}
}

// TestCapabilityPersistenceDuplicateID ensures duplicate capability IDs in file cause reload failure and preserve existing state.
func TestCapabilityPersistenceDuplicateID(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "caps-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(testutil.CapAlphaV1); err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	t.Setenv("GAUTH_CAPABILITIES_PATH", tmp.Name())
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	// write duplicate IDs
	//nolint:gosec // G306: test file permissions
	if err := os.WriteFile(tmp.Name(), []byte(testutil.CapAlphaDuplicateIDs), 0o644); err != nil {
		t.Fatal(err)
	}
	w := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if w.Code != 500 {
		t.Fatalf("expected 500 on invalid duplicate id reload got %d", w.Code)
	}

	disc := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if disc.Code != 200 {
		t.Fatalf("discovery status=%d", disc.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(disc.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	caps := doc["capability_registry"].([]any)
	if len(caps) != 1 {
		t.Fatalf("expected 1 capability preserved got %d", len(caps))
	}
}
