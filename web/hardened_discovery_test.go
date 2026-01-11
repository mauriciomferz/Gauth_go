package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHardenedDiscoveryFields(t *testing.T) {
	t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", "eddsa")
	t.Setenv("AGENTAUTH_JWKS_SIGNING_KEY_ENABLED", "1")
	t.Setenv("AGENTAUTH_JWKS_SIGNING_KEY", "test-signing-key")

	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	// 1. Trigger JWKS generation to update server state
	wJWKS := httptest.NewRecorder()
	reqJWKS := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	srv.router.ServeHTTP(wJWKS, reqJWKS)

	if wJWKS.Code != 200 {
		t.Fatalf("JWKS failed: %d", wJWKS.Code)
	}

	sig := wJWKS.Header().Get("X-JWKS-Signature")
	if sig == "" {
		t.Fatal("missing X-JWKS-Signature header in JWKS response")
	}

	// 2. Add a deprecated key if we can access the key manager
	// Since NewBetaServer might not have keys yet in some test configs,
	// we just rely on the fact that UpdateJWKSSignature should have been called.

	// 3. Verify Discovery Configuration
	wDisc := httptest.NewRecorder()
	reqDisc := httptest.NewRequest("GET", "/.well-known/agentauth-configuration", nil)
	srv.router.ServeHTTP(wDisc, reqDisc)

	if wDisc.Code != 200 {
		t.Fatalf("discovery failed: %d", wDisc.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(wDisc.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal discovery payload: %v", err)
	}

	if payload["jwks_signature"] != sig {
		t.Errorf("expected jwks_signature %q, got %q", sig, payload["jwks_signature"])
	}

	// deprecation_schedule is harder to test without injecting keys,
	// but we can at least check it exists as a field (it might be nil if no keys)
	if _, ok := payload["deprecation_schedule"]; !ok {
		t.Error("missing deprecation_schedule in discovery payload")
	}
}

func TestDiscoveryDeprecationSchedule(t *testing.T) {
	t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", "eddsa")

	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })

	// Manually inject a schedule to test the server handler
	future := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	schedule := map[string]time.Time{
		"key:test-key": future,
	}
	srv.UpdateDeprecationSchedule(schedule)

	wDisc := httptest.NewRecorder()
	reqDisc := httptest.NewRequest("GET", "/.well-known/agentauth-configuration", nil)
	srv.router.ServeHTTP(wDisc, reqDisc)

	var payload map[string]any
	if err := json.Unmarshal(wDisc.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal discovery response: %v", err)
	}

	sched, ok := payload["deprecation_schedule"].(map[string]any)
	if !ok {
		t.Fatal("deprecation_schedule not found or wrong type")
	}

	if sched["key:test-key"] != future.Format(time.RFC3339) {
		t.Errorf("expected %q, got %q", future.Format(time.RFC3339), sched["key:test-key"])
	}
}
