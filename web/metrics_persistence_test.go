package web

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
)

// TestMetricsPersistence ensures labeled lifecycle & decision counts survive save/load cycle.
func TestMetricsPersistence(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "metrics_persist_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	path := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	t.Setenv("AGENTAUTH_METRICS_PERSIST_PATH", path)
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })

	// Perform token lifecycle transitions
	rrCreate := doPOST(t, srv, "/api/v1/token/create", "{}")
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
	// active -> suspended -> active
	for _, st := range []string{"suspended", "active"} {
		rr := doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+createResp.Token.ID+`","new_status":"`+st+`"}`)
		if rr.Code != 200 {
			t.Fatalf("transition %s code=%d", st, rr.Code)
		}
	}

	// Trigger validation failure counters prior to save (invalid payload + unsupported + invalid transition)
	_ = doPOST(t, srv, "/api/v1/token/status/update", `{}`)                                                          // invalid payload
	_ = doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+createResp.Token.ID+`","new_status":"bogus"}`) // unsupported
	// terminated transition invalid: set status then attempt invalid change
	// terminated transition invalid: set status then attempt invalid change
	_ = doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+createResp.Token.ID+`","new_status":"terminated"}`)
	_ = doPOST(t, srv, "/api/v1/token/status/update", `{"token_id":"`+createResp.Token.ID+`","new_status":"active"}`) // invalid transition
	// Persist explicitly (simulate shutdown)
	if mm, ok := srv.metrics.(*metrics.Memory); ok {
		if err := mm.Save(); err != nil {
			t.Fatalf("save error: %v", err)
		}
		// Sanity: ensure counters >0 before reload
		if mm.InvalidPayloadFailures() == 0 || mm.UnsupportedStatusFailures() == 0 || mm.InvalidTransitionFailures() == 0 {
			t.Fatalf("expected validation failure counters >0 before reload")
		}
	} else {
		t.Fatalf("memory metrics expected")
	}

	// Create new server loading same file
	srv2 := NewBetaServer("")
	t.Cleanup(func() { srv2.Shutdown() })
	// Force same path again
	t.Setenv("AGENTAUTH_METRICS_PERSIST_PATH", path)
	// Manually enable persistence load (since env read occurred earlier in constructor)
	if mm2, ok2 := srv2.metrics.(*metrics.Memory); ok2 {
		if err := mm2.EnablePersistence(path); err != nil {
			t.Fatalf("reload error: %v", err)
		}
	}

	// Inspect metrics snapshot from second server
	m := doGET(t, srv2, "/api/v1/beta/metrics/lifecycle")
	if m.Code != 200 {
		t.Fatalf("metrics load code=%d", m.Code)
	}
	var snap struct {
		Success bool           `json:"success"`
		Metrics map[string]any `json:"metrics"`
	}
	_ = json.Unmarshal(m.Body.Bytes(), &snap)
	if !snap.Success {
		t.Fatalf("metrics endpoint not success")
	}
	// Verify at least one lifecycle labeled counter persisted
	lb, ok := snap.Metrics["lifecycle_breakdown"].(map[string]any)
	if !ok || len(lb) == 0 {
		t.Fatalf("expected non-empty lifecycle_breakdown after reload")
	}
	// Expect token|active|suspended|success present
	if toUint64(lb["token|active|suspended|success"]) == 0 {
		t.Fatalf("persisted lifecycle key missing")
	}
	// Validation failure counters persisted?
	if mm2, ok2 := srv2.metrics.(*metrics.Memory); ok2 {
		if mm2.InvalidPayloadFailures() == 0 || mm2.UnsupportedStatusFailures() == 0 || mm2.InvalidTransitionFailures() == 0 {
			t.Fatalf("expected validation failure counters persisted & reloaded")
		}
	} else {
		t.Fatalf("memory metrics expected on reload")
	}
}
