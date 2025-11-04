package web

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// TestDiscoveryCapabilityRegistry ensures dynamic capability registry and ordered action capabilities are exposed.
func TestDiscoveryCapabilityRegistry(t *testing.T) {
	srv := NewBetaServer(":0")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	// capability_registry should be slice of objects with id/version/stable
	regRaw, ok := body["capability_registry"].([]any)
	if !ok || len(regRaw) == 0 {
		t.Fatalf("capability_registry missing or empty: %#v", body["capability_registry"])
	}
	// Verify deterministic ordering by id (ascending)
	prev := ""
	for i, raw := range regRaw {
		objMap, okMap := raw.(map[string]any)
		if !okMap {
			t.Fatalf("registry entry %d wrong type %#v", i, raw)
		}
		id, idOK := objMap["id"].(string)
		if !idOK || id == "" {
			t.Fatalf("registry entry %d missing id", i)
		}
		if prev != "" && prev > id {
			t.Fatalf("registry not sorted: %s before %s", prev, id)
		}
		prev = id
		if _, vok := objMap["version"].(string); !vok {
			t.Fatalf("registry entry %s missing version", id)
		}
	}
	// action_capabilities should be ordered slice with action + required array
	actsRaw, ok := body["action_capabilities"].([]any)
	if !ok || len(actsRaw) == 0 {
		t.Fatalf("action_capabilities missing or empty: %#v", body["action_capabilities"])
	}
	prevAct := ""
	for i, raw := range actsRaw {
		objMap, okMap := raw.(map[string]any)
		if !okMap {
			t.Fatalf("action entry %d wrong type %#v", i, raw)
		}
		act, actOK := objMap["action"].(string)
		if !actOK || act == "" {
			t.Fatalf("action entry %d missing action", i)
		}
		if prevAct != "" && prevAct > act {
			t.Fatalf("actions not sorted: %s before %s", prevAct, act)
		}
		prevAct = act
		reqCaps, rok := objMap["required"].([]any)
		if !rok || len(reqCaps) == 0 {
			t.Fatalf("action %s required capabilities invalid %#v", act, objMap["required"])
		}
	}
	// enforcement flag present (defaults false normally)
	enfRaw, ok := body["capability_enforcement"].(map[string]any)
	if !ok {
		t.Fatalf("capability_enforcement missing")
	}
	if _, ok := enfRaw["enabled"].(bool); !ok {
		t.Fatalf("capability_enforcement.enabled missing or wrong type")
	}
}

// TestDiscoveryCapabilityEnforcementFlag ensures enabled flips when env var set.
func TestDiscoveryCapabilityEnforcementFlag(t *testing.T) {
	t.Setenv("GAUTH_CAPABILITY_ENFORCE", "1")
	srv := NewBetaServer(":0")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	enfRaw, ok := body["capability_enforcement"].(map[string]any)
	if !ok {
		t.Fatalf("capability_enforcement missing")
	}
	v, ok := enfRaw["enabled"].(bool)
	if !ok || !v {
		t.Fatalf("expected capability enforcement enabled true")
	}
	// Basic sanity: ensure create/revoke actions appear when enforcement active
	actsRaw, ok := body["action_capabilities"].([]any)
	if !ok {
		t.Fatalf("action_capabilities missing")
	}
	foundCreate := false
	foundRevoke := false
	for _, raw := range actsRaw {
		capMap, capOK := raw.(map[string]any)
		if !capOK {
			t.Fatalf("action capability wrong type %#v", raw)
		}
		act, actOK := capMap["action"].(string)
		if !actOK || act == "" {
			t.Fatalf("action capability missing action %#v", capMap)
		}
		switch act {
		case "delegation:create":
			foundCreate = true
		case "delegation:revoke":
			foundRevoke = true
		}
	}
	if !foundCreate || !foundRevoke {
		t.Fatalf("delegation actions not found in action_capabilities")
	}
}

// TestAuditCapabilitiesPagination generates capability-related audit entries then pages through them.
//nolint:gocyclo // Capability audit pagination test
//nolint:gocyclo // Capability audit pagination test
func TestAuditCapabilitiesPagination(t *testing.T) {
	t.Setenv("GAUTH_CAPABILITY_ENFORCE", "1")
	srv := NewBetaServer(":0")
	// Generate > limit entries (use capability_enforce denials by triggering missing capability for issue action)
	// We invoke token issuance without required capability claims repeatedly.
	limit := 5
	total := 13
	for i := 0; i < total; i++ {
		// Craft request to delegation revoke with missing required capabilities to produce denial audit (simpler than token path if mapping set)
		// Use transaction:issue mapping which requires a capability; simulate by calling create without caps to enforce denial.
		// We can hit /api/v1/delegation/revoke with missing capability mapping when requiredActionCaps contains delegation:revoke once enforcement is active.
		// Fallback: directly append audit entry if enforcement path evolves.
		srv.audit.Append(&AuditEntry{ID: strconv.Itoa(i), At: srv.start.Add(time.Duration(i) * time.Millisecond), Actor: "tester", Action: "capability_enforce", Resource: "delegation", Outcome: "denied", Meta: map[string]any{"seq": i}})
	}
	// Page 1
	w1 := performRequest(srv.router, "GET", "/api/v1/audit/capabilities?limit=5")
	if w1.Code != 200 {
		t.Fatalf("page1 status=%d", w1.Code)
	}
	var p1 map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &p1); err != nil {
		t.Fatal(err)
	}
	cnt1, ok := p1["count"].(float64)
	if !ok || int(cnt1) != limit {
		t.Fatalf("expected count %d got %#v", limit, p1["count"])
	}
	hm1, ok := p1["has_more"].(bool)
	if !ok || !hm1 {
		t.Fatalf("expected has_more true on first page got %#v", p1["has_more"])
	}
	next1, ok := p1["next_cursor"].(string)
	if !ok || next1 == "" {
		t.Fatalf("expected next_cursor non-empty got %#v", p1["next_cursor"])
	}
	// Page 2
	w2 := performRequest(srv.router, "GET", "/api/v1/audit/capabilities?limit=5&cursor="+next1)
	if w2.Code != 200 {
		t.Fatalf("page2 status=%d", w2.Code)
	}
	var p2 map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &p2); err != nil {
		t.Fatal(err)
	}
	hm2, ok := p2["has_more"].(bool)
	if !ok || !hm2 {
		t.Fatalf("expected has_more true on second page got %#v", p2["has_more"])
	}
	next2, ok := p2["next_cursor"].(string)
	if !ok || next2 == "" {
		t.Fatalf("expected next_cursor for third page got %#v", p2["next_cursor"])
	}
	// Page 3 (final partial)
	w3 := performRequest(srv.router, "GET", "/api/v1/audit/capabilities?limit=5&cursor="+next2)
	if w3.Code != 200 {
		t.Fatalf("page3 status=%d", w3.Code)
	}
	var p3 map[string]any
	if err := json.Unmarshal(w3.Body.Bytes(), &p3); err != nil {
		t.Fatal(err)
	}
	hm3, ok := p3["has_more"].(bool)
	if !ok {
		t.Fatalf("has_more final page wrong type %#v", p3["has_more"])
	}
	if hm3 {
		t.Fatalf("expected has_more false on final page")
	}
	cnt2, ok := p2["count"].(float64)
	if !ok {
		t.Fatalf("p2.count wrong type %#v", p2["count"])
	}
	cnt3, ok := p3["count"].(float64)
	if !ok {
		t.Fatalf("p3.count wrong type %#v", p3["count"])
	}
	gotTotal := int(cnt1) + int(cnt2) + int(cnt3)
	if gotTotal != total {
		t.Fatalf("expected total %d accumulated got %d", total, gotTotal)
	}
	// Verify deterministic ordering: IDs increasing by sequence across pages
	seqs := []int{}
	for _, page := range []map[string]any{p1, p2, p3} {
		entries, entriesOK := page["entries"].([]any)
		if !entriesOK {
			t.Fatalf("entries wrong type %#v", page["entries"])
		}
		for _, raw := range entries {
			eMap, eOK := raw.(map[string]any)
			if !eOK {
				t.Fatalf("entry wrong type %#v", raw)
			}
			meta, metaOK := eMap["meta"].(map[string]any)
			if !metaOK {
				continue
			}
			if sVal, seqOK := meta["seq"].(float64); seqOK {
				seqs = append(seqs, int(sVal))
			}
		}
	}
	for i := 1; i < len(seqs); i++ {
		if seqs[i-1] > seqs[i] {
			t.Fatalf("sequence ordering violated at %d>%d", seqs[i-1], seqs[i])
		}
	}
}
