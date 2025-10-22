package web

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDelegationLifecycleMetrics validates delegation status transition & failure counters.
func TestDelegationLifecycleMetrics(t *testing.T) {
	srv := NewBetaServer("")

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

	// Perform valid transitions: active->suspended->active->terminated
	paths := []string{"suspended", "active", "terminated"}
	for _, newStatus := range paths {
		rr := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d1","new_status":"`+newStatus+`"}`)
		if rr.Code != 200 {
			t.Fatalf("transition %s expected 200 got %d", newStatus, rr.Code)
		}
	}
	// Invalid transition terminated->active (should fail)
	rrBad := doPOST(t, srv, "/api/v1/delegation/status/update", `{"delegation_id":"d1","new_status":"active"}`)
	if rrBad.Code != 409 {
		t.Fatalf("expected 409 invalid transition got %d", rrBad.Code)
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

	// We expect exactly +4 successful transitions after baseline snapshot:
	// suspended, active, terminated, plus initialization already counted at baseline.
	// Accept >= to be resilient to future additional transitions.
	if t1 < baseTransitions+3 { // +3 after baseline (suspended, active, terminated)
		t.Fatalf("expected at least 3 new transitions base=%d final=%d", baseTransitions, t1)
	}
	if f1 < baseFailures+1 {
		t.Fatalf("expected at least 1 new failure base=%d final=%d", baseFailures, f1)
	}

	// Breakdown assertions
	lbAny, ok := snap1.Metrics["lifecycle_breakdown"].(map[string]any)
	if !ok {
		t.Fatalf("missing lifecycle_breakdown map")
	}
	// Initialization now normalized to underscore old status.
	if toUint64(lbAny["delegation|_|active|success"]) == 0 {
		t.Fatalf("expected initialization breakdown key delegation|_|active|success >0")
	}
	// Success transitions
	successKeys := []string{"delegation|active|suspended|success", "delegation|suspended|active|success", "delegation|active|terminated|success"}
	for _, k := range successKeys {
		if toUint64(lbAny[k]) == 0 {
			t.Fatalf("expected lifecycle_breakdown key %s >0", k)
		}
	}
	// Failure terminated->active
	if toUint64(lbAny["delegation|terminated|active|failure"]) == 0 {
		t.Fatalf("expected failure breakdown key delegation|terminated|active|failure >0")
	}

	// Latency aggregates present for delegation entity (outcome-labeled e.g. delegation|success)
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
