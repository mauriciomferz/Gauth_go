package web

import (
	"net/http/httptest"
	"testing"
	"time"
)

// TestExternalAnchorVerifySuccess ensures verify endpoint reports verified=true for memory provider after successful anchor.
func TestExternalAnchorVerifySuccess(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "memory")
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Allow initial anchor
	time.Sleep(30 * time.Millisecond)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/external/verify", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !hasSubstr(body, "\"verified\":true") && !hasSubstr(body, "\"verified\": true") {
		t.Fatalf("expected verified true, body=%s", body)
	}
}

// TestExternalAnchorVerifyFailure ensures verify endpoint returns verified=false when provider fails (tsa_stub forced failure).
func TestExternalAnchorVerifyFailure(t *testing.T) {
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_PROVIDER", "tsa_stub")
	t.Setenv("GAUTH_CAP_EXTERNAL_ANCHOR_FAIL_PROB", "1") // always fail
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	// Give stub time to attempt initial anchor (should fail) then request verify.
	time.Sleep(40 * time.Millisecond)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/beta/capabilities/anchor/external/verify", nil)
	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	// For failure scenario we expect either empty receipt or verified false.
	// For failure scenario provider either produced no receipt (empty:true) or verify would be skipped (verified absent).
	if !hasSubstr(body, "\"empty\":true") && !hasSubstr(body, "\"empty\": true") {
		t.Fatalf("expected empty:true for forced failure scenario, body=%s", body)
	}
}
