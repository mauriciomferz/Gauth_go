package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper copied (avoid import cycle) - minimal GET request
func doGET(t *testing.T, srv *BetaServer, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	return rr
}

// TestLifecycleMetricsEndpoint ensures lifecycle counters change after transitions
// and are exposed via /api/v1/beta/metrics/lifecycle.
func TestLifecycleMetricsEndpoint(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// Create token
	rrCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/token/create", nil)
	srv.router.ServeHTTP(rrCreate, reqCreate)
	if rrCreate.Code != 201 {
		t.Fatalf("create token code=%d", rrCreate.Code)
	}
	var createResp struct {
		Token struct {
			ID string `json:"id"`
		} `json:"token"`
	}
	_ = json.Unmarshal(rrCreate.Body.Bytes(), &createResp)
	if createResp.Token.ID == "" {
		t.Fatalf("missing token id")
	}

	// Initial metrics snapshot
	m0 := doGET(t, srv, "/api/v1/beta/metrics/lifecycle")
	if m0.Code != 200 {
		t.Fatalf("metrics initial code=%d", m0.Code)
	}
	// Use raw map to access breakdown sub-maps
	var snap0 struct {
		Success bool           `json:"success"`
		Metrics map[string]any `json:"metrics"`
	}
	_ = json.Unmarshal(m0.Body.Bytes(), &snap0)
	if !snap0.Success {
		t.Fatalf("metrics initial not success")
	}

	// Perform two valid transitions: active->suspended, suspended->active
	body1 := `{"token_id":"` + createResp.Token.ID + `","new_status":"suspended"}`
	rr1 := doPOST(t, srv, "/api/v1/token/status/update", body1)
	if rr1.Code != 200 {
		t.Fatalf("first transition code=%d", rr1.Code)
	}

	body2 := `{"token_id":"` + createResp.Token.ID + `","new_status":"active"}`
	rr2 := doPOST(t, srv, "/api/v1/token/status/update", body2)
	if rr2.Code != 200 {
		t.Fatalf("second transition code=%d", rr2.Code)
	}

	// Invalid transition: terminated->active to produce failure counter
	rrTerm := doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+createResp.Token.ID+`","new_status":"terminated"}`)
	if rrTerm.Code != 200 {
		t.Fatalf("terminate code=%d", rrTerm.Code)
	}
	rrInvalid := doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+createResp.Token.ID+`","new_status":"active"}`)
	if rrInvalid.Code != 409 {
		t.Fatalf("expected 409 invalid transition got %d", rrInvalid.Code)
	}

	// Fetch metrics again
	m1 := doGET(t, srv, "/api/v1/beta/metrics/lifecycle")
	if m1.Code != 200 {
		t.Fatalf("metrics second code=%d", m1.Code)
	}
	var snap1 struct {
		Success bool           `json:"success"`
		Metrics map[string]any `json:"metrics"`
	}
	_ = json.Unmarshal(m1.Body.Bytes(), &snap1)
	if !snap1.Success {
		t.Fatalf("metrics second not success")
	}

	// Basic assertions: transitions increased, failures >=1
	// Extract counters
	t0 := toUint64(snap0.Metrics["token_status_transitions"])
	t1 := toUint64(snap1.Metrics["token_status_transitions"])
	if t1 < t0+3 {
		t.Fatalf("expected at least 3 transitions (including terminate) got initial=%d final=%d", t0, t1)
	}
	f1 := toUint64(snap1.Metrics["token_status_transition_failures"])
	if f1 < 1 {
		t.Fatalf("expected at least 1 failure counter after invalid transition, got %d", f1)
	}

	// Breakdown assertions
	lbAny, ok := snap1.Metrics["lifecycle_breakdown"].(map[string]any)
	if !ok {
		t.Fatalf("missing lifecycle_breakdown map")
	}
	// Expect labeled success entries for active->suspended, suspended->active, active->terminated
	successTransitions := []string{"token|active|suspended|success", "token|suspended|active|success", "token|active|terminated|success"}
	for _, key := range successTransitions {
		if toUint64(lbAny[key]) == 0 {
			t.Fatalf("expected lifecycle_breakdown key %s >0", key)
		}
	}
	// Failure entry for terminated->active
	if toUint64(lbAny["token|terminated|active|failure"]) == 0 {
		t.Fatalf("expected failure breakdown key token|terminated|active|failure >0")
	}
	// No-op not exercised here (optional check if present)

	// Latency aggregates present for token entity (outcome-labeled e.g. token|success)
	if latCounts, ok2 := snap1.Metrics["lifecycle_latency_counts"].(map[string]any); ok2 {
		var total uint64
		for k, v := range latCounts {
			if strings.HasPrefix(k, "token|") {
				total += toUint64(v)
			}
		}
		if total == 0 {
			t.Fatalf("expected at least one lifecycle latency count for token entity (token|*)")
		}
	} else {
		t.Fatalf("missing lifecycle_latency_counts map")
	}
}

// toUint64 safely converts interface{} to uint64 for test metrics.
func toUint64(v any) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case float64:
		return uint64(x)
	case int:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case json.Number:
		if iv, err := x.Int64(); err == nil && iv >= 0 {
			return uint64(iv)
		}
	}
	return 0
}
