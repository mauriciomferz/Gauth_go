package web

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"
)

// TestCapabilityPersistenceMissingSchemaVersion ensures that removing schema_version causes reload failure and preserves previous state.
func TestCapabilityPersistenceMissingSchemaVersion(t *testing.T) {
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

	// Confirm initial load succeeded (source should be file, schema version present)
	disc1 := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if disc1.Code != 200 {
		t.Fatalf("discovery1 status=%d", disc1.Code)
	}
	var doc1 map[string]any
	if err := json.Unmarshal(disc1.Body.Bytes(), &doc1); err != nil {
		t.Fatal(err)
	}
	if doc1["capability_registry_source"] != capSourceFile {
		t.Fatalf("expected source file got %v", doc1["capability_registry_source"])
	}
	if doc1["capability_registry_schema_version"].(float64) != 1 {
		t.Fatalf("expected schema_version 1 got %v", doc1["capability_registry_schema_version"])
	}

	// Overwrite file removing schema_version field
	if err := os.WriteFile(tmp.Name(), []byte(testutil.CapAlphaMissingSchemaVersion), 0o644); err != nil {
		t.Fatal(err)
	}
	w := performRequest(srv.router, "POST", "/api/v1/beta/capabilities/reload")
	if w.Code != 500 {
		t.Fatalf("expected 500 reload failure got %d", w.Code)
	}

	// Discovery should still reflect previous schema_version and capability
	disc2 := performRequest(srv.router, "GET", "/.well-known/gauth-configuration")
	if disc2.Code != 200 {
		t.Fatalf("discovery2 status=%d", disc2.Code)
	}
	var doc2 map[string]any
	if err := json.Unmarshal(disc2.Body.Bytes(), &doc2); err != nil {
		t.Fatal(err)
	}
	if doc2["capability_registry_schema_version"].(float64) != 1 {
		t.Fatalf("schema_version changed unexpectedly %v", doc2["capability_registry_schema_version"])
	}
	caps := doc2["capability_registry"].([]any)
	if len(caps) != 1 {
		t.Fatalf("expected 1 capability preserved got %d", len(caps))
	}
}
