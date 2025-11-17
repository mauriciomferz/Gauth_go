package web

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDelegationLifecycleMetrics validates delegation status transition & failure counters.
func TestDelegationLifecycleMetrics(t *testing.T) {
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// Initialize delegation active
	rrInit := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d1","new_status":"active"}`)
	if rrInit.Code != 200 {
		t.Fatalf("init expected 200 got %d", rrInit.Code)
	}

	// Snapshot baseline metrics
	m0 := doGET(t, srv, "/api/v1/beta/metrics/lifecycle")
	if m0.Code != 200 {
		t.Fatalf("metrics baseline code=%d", m0.Code)
	}
	var snap0 struct {
		Success bool           `json:"success"`
		Metrics map[string]any `json:"metrics"`
	}
	_ = json.Unmarshal(m0.Body.Bytes(), &snap0)
	if !snap0.Success {
		t.Fatalf("baseline metrics not success")
	}
	baseTransitions := toUint64(snap0.Metrics["delegation_status_transitions"])
	baseFailures := toUint64(snap0.Metrics["delegation_status_transition_failures"])

	// Perform valid transitions: active->suspended->active->partially_revoked->terminated
	paths := []string{"suspended", "active", "partially_revoked", "terminated"}
	for _, newStatus := range paths {
		rr := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d1","new_status":"`+newStatus+`"}`)
		if rr.Code != 200 {
			t.Fatalf("transition %s expected 200 got %d", newStatus, rr.Code)
		}
	}
	// Invalid transition terminated->active (should fail)
	rrBad := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d1","new_status":"active"}`)
	if rrBad.Code != 409 {
		t.Fatalf("expected 409 invalid transition terminated->active got %d", rrBad.Code)
	}
	// Invalid transition partially_revoked->active: need fresh delegation that is partially_revoked
	rrInit2 := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d2","new_status":"active"}`)
	if rrInit2.Code != 200 {
		t.Fatalf("init second delegation expected 200 got %d", rrInit2.Code)
	}
	rrPR := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d2","new_status":"partially_revoked"}`)
	if rrPR.Code != 200 {
		t.Fatalf("partial_revoked expected 200 got %d", rrPR.Code)
	}
	rrPRBad := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d2","new_status":"active"}`)
	if rrPRBad.Code != 409 {
		t.Fatalf("expected 409 invalid transition partially_revoked->active got %d", rrPRBad.Code)
	}

	// Fetch metrics again
	m1 := doGET(t, srv, "/api/v1/beta/metrics/lifecycle")
	if m1.Code != 200 {
		t.Fatalf("metrics post code=%d", m1.Code)
	}
	var snap1 struct {
		Success bool           `json:"success"`
		Metrics map[string]any `json:"metrics"`
	}
	_ = json.Unmarshal(m1.Body.Bytes(), &snap1)
	if !snap1.Success {
		t.Fatalf("post metrics not success")
	}
	t1 := toUint64(snap1.Metrics["delegation_status_transitions"])
	f1 := toUint64(snap1.Metrics["delegation_status_transition_failures"])

	// Expect at least 6 successful transitions after baseline (d1: suspended, active, partially_revoked, terminated; d2: active init + partially_revoked) => +5 from baseline init of d1
	if t1 < baseTransitions+5 {
		t.Fatalf("expected at least 5 new transitions base=%d final=%d", baseTransitions, t1)
	}
	// Failures: terminated->active, partially_revoked->active
	if f1 < baseFailures+2 {
		t.Fatalf("expected at least 2 new failures base=%d final=%d", baseFailures, f1)
	}

	// Breakdown assertions
	lbAny, ok := snap1.Metrics["lifecycle_breakdown"].(map[string]any)
	if !ok {
		t.Fatalf("missing lifecycle_breakdown map")
	}
	if toUint64(lbAny["delegation|_|active|success"]) == 0 {
		t.Fatalf("expected initialization breakdown key delegation|_|active|success >0")
	}
	// Success transitions keys
	successKeys := []string{
		"delegation|active|suspended|success",
		"delegation|suspended|active|success",
		"delegation|active|partially_revoked|success",
		"delegation|partially_revoked|terminated|success",
		"delegation|_|active|success", // second delegation init
	}
	for _, k := range successKeys {
		if toUint64(lbAny[k]) == 0 {
			t.Fatalf("expected lifecycle_breakdown key %s >0", k)
		}
	}
	if toUint64(lbAny["delegation|terminated|active|failure"]) == 0 {
		t.Fatalf("expected failure breakdown key delegation|terminated|active|failure >0")
	}
	if toUint64(lbAny["delegation|partially_revoked|active|failure"]) == 0 {
		t.Fatalf("expected failure breakdown key delegation|partially_revoked|active|failure >0")
	}

	// Latency aggregates present
	if latCounts, ok2 := snap1.Metrics["lifecycle_latency_counts"].(map[string]any); ok2 {
		var total uint64
		for k, v := range latCounts {
			if strings.HasPrefix(k, "delegation|") {
				total += toUint64(v)
			}
		}
		if total == 0 {
			t.Fatalf("expected at least one lifecycle latency count for delegation entity (delegation|*)")
		}
	} else {
		t.Fatalf("missing lifecycle_latency_counts map")
	}
}
