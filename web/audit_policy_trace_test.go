package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// TestPolicyDecisionTraceability_RFC111_C4 verifies that audit logs capture
// the correct bundle hash and chain head for policy evaluations, ensuring traceability
// across policy updates (multi-bundle replay).
func TestPolicyDecisionTraceability_RFC111_C4(t *testing.T) {
	// 1. Setup Server
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	// Helper to add a bundle
	addBundle := func(id, content string) string {
		// Matching Policy struct in pkg/policy/engine.go
		p := policy.Policy{
			ID:       "pol-" + id,
			Subjects: []string{"*"}, // match all for simplicity
			Rules: []policy.Rule{
				{
					Actions:   []string{"read"},
					Resources: []string{"res-" + content},
					Effect:    policy.Allow,
				},
			},
		}
		reqBody, _ := json.Marshal(map[string]any{
			"id":       "bundle-" + id,
			"policies": []policy.Policy{p},
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/policy/bundles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("AddBundle %s failed: %d %s", id, w.Code, w.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		return resp["bundle_hash"].(string)
	}

	// Helper to evaluate
	evaluate := func(res string) {
		reqBody := fmt.Sprintf(`{"subject":"alice","action":"read","resource":"%s"}`, res)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/policy/evaluate", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("Evaluate failed: %d", w.Code)
		}
	}

	// 2. Add Bundle V1 & Evaluate
	hashV1 := addBundle("v1", "v1")
	evaluate("res-v1")

	// 3. Add Bundle V2 & Evaluate
	hashV2 := addBundle("v2", "v2")
	evaluate("res-v2")

	// 4. Verify Audit Log
	// Give a moment for async logs
	time.Sleep(250 * time.Millisecond)

	// Fetch more entries to be safe against startup noise
	entries := srv.audit.List(100)

	var evalEntries []map[string]any
	for _, ae := range entries {
		// Debug print
		// t.Logf("Entry: Actor=%s Action=%s Res=%s Meta=%v", ae.Actor, ae.Action, ae.Resource, ae.Meta)
		if ae.Action == "evaluate" {
			// Meta is any, need to cast to map[string]interfaceIfPossible
			// The JSON marshalling/unmarshalling usually makes it map[string]interface{}
			// But here it's in-memory, so it stays as whatever it was set to.
			// api.go sets it as map[string]string.

			// Let's copy it to generic map for test
			m := map[string]any{}
			if metaMap, ok := ae.Meta.(map[string]string); ok {
				for k, v := range metaMap {
					m[k] = v
				}
			} else if metaMap, ok := ae.Meta.(map[string]any); ok {
				for k, v := range metaMap {
					m[k] = v
				}
			}
			evalEntries = append(evalEntries, map[string]any{
				"meta":     m,
				"resource": ae.Resource, // Use top-level resource too
			})
		}
	}

	if len(evalEntries) < 2 {
		t.Logf("Found %d evaluate entries:", len(evalEntries))
		for _, e := range entries {
			t.Logf("- %s on %s (%v)", e.Action, e.Resource, e.Meta)
		}
		t.Fatalf("Expected at least 2 evaluation audit entries")
	}

	findEntry := func(res string) map[string]any {
		for _, e := range evalEntries {
			// check top level resource first
			if e["resource"] == res {
				return e["meta"].(map[string]any)
			}
			// fall back to meta resource if needed
			if meta := e["meta"].(map[string]any); meta["resource"] == res {
				return meta
			}
		}
		return nil
	}

	entryV1 := findEntry("res-v1")
	if entryV1 == nil {
		t.Fatal("Audit entry for res-v1 not found")
	}
	if got := entryV1["bundle_hash"]; got != hashV1 {
		t.Errorf("V1 Audit: expected bundle_hash %s, got %v", hashV1, got)
	}

	entryV2 := findEntry("res-v2")
	if entryV2 == nil {
		t.Fatal("Audit entry for res-v2 not found")
	}
	if got := entryV2["bundle_hash"]; got != hashV2 {
		t.Errorf("V2 Audit: expected bundle_hash %s, got %v", hashV2, got)
	}
}
