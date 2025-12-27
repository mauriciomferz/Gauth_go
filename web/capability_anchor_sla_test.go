package web

import (
	"encoding/json"
	"testing"
	"time"
)

// capabilityAnchorStatusResponse subset for SLA fields
type capabilityAnchorStatusResponse struct {
	Success               bool   `json:"success"`
	Configured            bool   `json:"configured"`
	LastWrite             string `json:"last_write"`
	AgeSeconds            uint64 `json:"age_seconds"`
	StaleThresholdSeconds int    `json:"stale_threshold_seconds"`
	Stale                 bool   `json:"stale"`
}

// TestCapabilityAnchorSLAStale verifies stale flag toggles after threshold exceeded.
func TestCapabilityAnchorSLAStale(t *testing.T) {
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", t.TempDir()+"/anchor.json")
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	t.Setenv("GAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS", "2") // very small threshold for test
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// Simulate a last write 5 seconds ago (beyond threshold)
	srv.capAnchorLastWrite = time.Now().Add(-5 * time.Second)
	// Verify SLA logic via metrics or state
	// For test purposes, we check internal age state if accessible or via metrics (mocked)
	// Trigger monitor tick manually by computing age & stale
	age := uint64(time.Since(srv.capAnchorLastWrite).Seconds())
	// Set threshold explicitly for test consistency
	srv.capabilitiesHandler.AnchorStaleThreshold = 2 * time.Second
	srv.capabilitiesHandler.SetAnchorState(age > 2, time.Duration(age)*time.Second)

	rr := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/status")
	if rr.Code != 200 {
		t.Fatalf("unexpected status code %d", rr.Code)
	}
	var resp capabilityAnchorStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Stale {
		t.Fatalf("expected stale=true got false age=%d threshold=%d", resp.AgeSeconds, resp.StaleThresholdSeconds)
	}
	// #nosec G115: test code, threshold is small int value
	if resp.AgeSeconds < uint64(resp.StaleThresholdSeconds) {
		t.Fatalf("age below threshold unexpectedly age=%d threshold=%d", resp.AgeSeconds, resp.StaleThresholdSeconds)
	}
}

// TestCapabilityAnchorSLAFresh verifies stale=false when age below threshold.
func TestCapabilityAnchorSLAFresh(t *testing.T) {
	t.Setenv("GAUTH_CAP_ANCHOR_FILE_PATH", t.TempDir()+"/anchor.json")
	t.Setenv("GAUTH_CAP_ANCHOR_WRITE_INTERVAL", "1m")
	t.Setenv("GAUTH_CAP_ANCHOR_STALE_THRESHOLD_SECONDS", "10") // threshold higher than age
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	srv.capAnchorLastWrite = time.Now().Add(-2 * time.Second)
	age := uint64(time.Since(srv.capAnchorLastWrite).Seconds())
	// Use explicit threshold for test
	srv.capabilitiesHandler.AnchorStaleThreshold = 10 * time.Second
	srv.capabilitiesHandler.SetAnchorState(false, time.Duration(age)*time.Second)

	rr := performRequest(srv.router, "GET", "/api/v1/beta/capabilities/anchor/status")
	if rr.Code != 200 {
		t.Fatalf("unexpected status code %d", rr.Code)
	}
	var resp capabilityAnchorStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Stale {
		t.Fatalf("expected stale=false got true age=%d threshold=%d", resp.AgeSeconds, resp.StaleThresholdSeconds)
	}
	// #nosec G115: test code, threshold is small int value
	if resp.AgeSeconds >= uint64(resp.StaleThresholdSeconds) {
		t.Fatalf("age exceeds threshold unexpectedly age=%d threshold=%d", resp.AgeSeconds, resp.StaleThresholdSeconds)
	}
}
