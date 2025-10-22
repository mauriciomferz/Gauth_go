package web

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDecisionMetricsEndpoint validates /api/v1/beta/metrics/decisions returns deterministic labeled counts.
func TestDecisionMetricsEndpoint(t *testing.T) {
	srv := NewBetaServer(":0")
	// Record several lifecycle decisions via token status update and delegation status update endpoints.
	// Use token create to get a token id.
	reqCreate := httptest.NewRequest("POST", "/api/v1/token/create", strings.NewReader(`{"subject":"alice@example.com","scopes":["demo"],"audience":["beta"],"ttl_seconds":10}`))
	reqCreate.Header.Set("Content-Type", "application/json")
	create := httptest.NewRecorder()
	srv.router.ServeHTTP(create, reqCreate)
	// Accept 200 OK or 201 Created from token create endpoint.
	if create.Code != 200 && create.Code != 201 {
		t.Fatalf("token create unexpected status=%d body=%s", create.Code, create.Body.String())
	}
	var cdoc map[string]any
	if err := json.Unmarshal(create.Body.Bytes(), &cdoc); err != nil {
		t.Fatal(err)
	}
	// token create response embeds token object under "token" key.
	tokObj, _ := cdoc["token"].(map[string]any)
	tokID, _ := tokObj["id"].(string)
	if tokID == "" {
		t.Fatalf("missing token id in response: %v", cdoc)
	}

	// Transition token status to suspended then suspended again (noop) to generate reason entries.
	reqSuspend := httptest.NewRequest("POST", "/api/v1/token/status/update", strings.NewReader(`{"token_id":"`+tokID+`","new_status":"suspended"}`))
	reqSuspend.Header.Set("Content-Type", "application/json")
	suspend := httptest.NewRecorder()
	srv.router.ServeHTTP(suspend, reqSuspend)
	if suspend.Code != 200 {
		t.Fatalf("suspend status=%d body=%s", suspend.Code, suspend.Body.String())
	}
	// No-op update (same status) to create reason=noop entry.
	reqNoop := httptest.NewRequest("POST", "/api/v1/token/status/update", strings.NewReader(`{"token_id":"`+tokID+`","new_status":"suspended"}`))
	reqNoop.Header.Set("Content-Type", "application/json")
	noop := httptest.NewRecorder()
	srv.router.ServeHTTP(noop, reqNoop)
	if noop.Code != 200 {
		t.Fatalf("noop status=%d body=%s", noop.Code, noop.Body.String())
	}

	// Fetch decision metrics.
	dm := performRequest(srv.router, "GET", "/api/v1/beta/metrics/decisions")
	if dm.Code != 200 {
		t.Fatalf("decision metrics status=%d body=%s", dm.Code, dm.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(dm.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	decisions, ok := doc["decisions"].(map[string]any)
	if !ok {
		t.Fatalf("missing decisions object")
	}
	counts, ok := decisions["counts"].([]any)
	if !ok {
		t.Fatalf("missing counts array")
	}
	reasons, ok := decisions["reasons"].([]any)
	if !ok {
		t.Fatalf("missing reasons array")
	}
	if len(counts) == 0 {
		t.Fatalf("expected at least one decision count entry")
	}
	// Ensure we have at least one reason entry (noop or status_change)
	if len(reasons) == 0 {
		t.Fatalf("expected at least one decision reason entry")
	}
	// Basic structural check of an entry
	e0, _ := counts[0].(map[string]any)
	if e0["action"] == nil || e0["resource"] == nil || e0["outcome"] == nil || e0["count"] == nil {
		t.Fatalf("malformed counts entry: %v", e0)
	}
	r0, _ := reasons[0].(map[string]any)
	if r0["reason"] == nil {
		t.Fatalf("malformed reasons entry: %v", r0)
	}
}
