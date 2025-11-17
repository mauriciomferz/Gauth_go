package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// (Removed unused readJSONBody helper; parsing done inline in tests.)

func TestViolationPersistenceVerify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "violations.json")
	os.Setenv("GAUTH_VIOLATION_PERSIST_PATH", path)
	os.Setenv("GAUTH_VIOLATION_PERSIST_NO_THROTTLE", "1")
	srv := NewBetaServer("8131")
	t.Cleanup(func() { srv.Shutdown() })
	// Trigger some violations
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/token/validate", strings.NewReader(`{"token":""}`))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
	}
	srv.saveViolationPersistence()
	// Verify OK
	wv := httptest.NewRecorder()
	rv := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations/verify", nil)
	srv.router.ServeHTTP(wv, rv)
	if !strings.Contains(wv.Body.String(), "integrity") {
		t.Fatalf("verify response missing integrity: %s", wv.Body.String())
	}
	// Check Prometheus metrics integrity gauge is unconfigured (not yet set) or ok after first verify
	mw := httptest.NewRecorder()
	mr := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations/prometheus", nil)
	srv.router.ServeHTTP(mw, mr)
	promBody := mw.Body.String()
	if !strings.Contains(promBody, "gauth_persistence_integrity_violation") {
		t.Fatalf("prometheus output missing violation integrity gauge: %s", promBody)
	}
	if strings.Contains(promBody, "gauth_persistence_integrity_violation 0") {
		t.Fatalf("expected non-mismatch state initially got body=%s", promBody)
	}
	// Tamper file by modifying a counter value inside the payload (ensures hash mismatch)
	contents, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read persisted: %v", readErr)
	}
	var wrapper struct {
		Payload   json.RawMessage `json:"payload"`
		PrevHash  string          `json:"prev_hash"`
		Hash      string          `json:"hash"`
		Timestamp string          `json:"timestamp"`
	}
	if wrapErr := json.Unmarshal(contents, &wrapper); wrapErr != nil {
		t.Fatalf("unmarshal wrapper: %v", wrapErr)
	}
	if len(wrapper.Payload) == 0 {
		t.Fatalf("empty payload in wrapper (unexpected)")
	}
	// Decode payload to mutate a counter
	var payload struct {
		Counters map[string]uint64 `json:"counters"`
		History  []struct {
			At    string `json:"at"`
			Total uint64 `json:"total"`
		} `json:"history"`
	}
	if innerErr := json.Unmarshal(wrapper.Payload, &payload); innerErr != nil {
		t.Fatalf("unmarshal inner payload: %v raw=%s", innerErr, string(wrapper.Payload))
	}
	if payload.Counters == nil {
		payload.Counters = map[string]uint64{}
	}
	payload.Counters["sig_invalid"]++ // increment one counter
	// Re-marshal mutated payload but keep original wrapper.Hash (expected to mismatch now)
	mutatedPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal mutated payload: %v", err)
	}
	wrapper.Payload = mutatedPayload
	mutatedWrapperBytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("marshal mutated wrapper: %v", err)
	}
	if writeErr := os.WriteFile(path, mutatedWrapperBytes, 0o600); writeErr != nil {
		t.Fatalf("write tampered: %v", writeErr)
	}
	wv2 := httptest.NewRecorder()
	rv2 := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations/verify", nil)
	srv.router.ServeHTTP(wv2, rv2)
	if !strings.Contains(wv2.Body.String(), "mismatch") {
		t.Fatalf("expected mismatch after tamper got: %s", wv2.Body.String())
	}
	// Prometheus gauge should now reflect mismatch (0)
	mw2 := httptest.NewRecorder()
	mr2 := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations/prometheus", nil)
	srv.router.ServeHTTP(mw2, mr2)
	if !strings.Contains(mw2.Body.String(), "gauth_persistence_integrity_violation 0") {
		t.Fatalf("expected violation integrity gauge=0 after tamper got: %s", mw2.Body.String())
	}
}

func TestSemanticPersistenceVerify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "semantic.json")
	os.Setenv("GAUTH_SEMANTIC_PERSIST_PATH", path)
	os.Setenv("GAUTH_SEMANTIC_PERSIST_NO_THROTTLE", "1")
	srv := NewBetaServer("8132")
	t.Cleanup(func() { srv.Shutdown() })
	if srv.rfc0111Service == nil {
		t.Fatalf("semantic service not initialized")
	}
	// Trigger semantic counters via authorize endpoint (empty payload leads to some defaults). Provide mismatching scopes.
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/poa/authorize", strings.NewReader(`{"delegation":{"scope":["a"],"requested_scope":["b"],"amount_limit":1,"requested_amount":5}}`))
		req.Header.Set("Content-Type", "application/json")
		srv.router.ServeHTTP(w, req)
	}
	srv.saveSemanticPersistence()
	ws := httptest.NewRecorder()
	rs := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics/verify", nil)
	srv.router.ServeHTTP(ws, rs)
	if !strings.Contains(ws.Body.String(), "integrity") {
		t.Fatalf("semantic verify missing integrity: %s", ws.Body.String())
	}
	// Prometheus: semantic integrity gauge should not show mismatch yet
	ms1 := httptest.NewRecorder()
	msr1 := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations/prometheus", nil)
	srv.router.ServeHTTP(ms1, msr1)
	if !strings.Contains(ms1.Body.String(), "gauth_persistence_integrity_semantic") {
		t.Fatalf("prometheus missing semantic integrity gauge: %s", ms1.Body.String())
	}
	if strings.Contains(ms1.Body.String(), "gauth_persistence_integrity_semantic 0") {
		t.Fatalf("unexpected semantic mismatch gauge pre-tamper: %s", ms1.Body.String())
	}
	// Tamper semantic file by modifying a counter value (scope_violation) to force hash mismatch while keeping stored hash.
	contents, readErr2 := os.ReadFile(path)
	if readErr2 != nil {
		t.Fatalf("read semantic persisted: %v", readErr2)
	}
	var wrapper2 struct {
		Payload   json.RawMessage `json:"payload"`
		PrevHash  string          `json:"prev_hash"`
		Hash      string          `json:"hash"`
		Timestamp string          `json:"timestamp"`
	}
	if wrapErr2 := json.Unmarshal(contents, &wrapper2); wrapErr2 != nil {
		t.Fatalf("unmarshal wrapper: %v", wrapErr2)
	}
	if len(wrapper2.Payload) == 0 {
		t.Fatalf("empty payload in semantic wrapper")
	}
	var inner struct {
		Counters  map[string]uint64 `json:"counters"`
		Timestamp string            `json:"timestamp"`
	}
	if innerErr2 := json.Unmarshal(wrapper2.Payload, &inner); innerErr2 != nil {
		t.Fatalf("unmarshal inner semantic payload: %v raw=%s", innerErr2, string(wrapper2.Payload))
	}
	if inner.Counters == nil {
		inner.Counters = map[string]uint64{}
	}
	inner.Counters["scope_violation"]++
	mutatedPayload, marshalErr2 := json.Marshal(inner)
	if marshalErr2 != nil {
		t.Fatalf("marshal mutated inner semantic payload: %v", marshalErr2)
	}
	wrapper2.Payload = mutatedPayload
	mutatedWrapperBytes2, marshalErr3 := json.Marshal(wrapper2)
	if marshalErr3 != nil {
		t.Fatalf("marshal mutated semantic wrapper: %v", marshalErr3)
	}
	if writeErr2 := os.WriteFile(path, mutatedWrapperBytes2, 0o600); writeErr2 != nil {
		t.Fatalf("write tampered semantic: %v", writeErr2)
	}
	ws2 := httptest.NewRecorder()
	rs2 := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics/verify", nil)
	srv.router.ServeHTTP(ws2, rs2)
	if !strings.Contains(ws2.Body.String(), "mismatch") {
		t.Fatalf("expected mismatch after semantic tamper got: %s", ws2.Body.String())
	}
	// Prometheus gauge should now reflect mismatch
	ms2 := httptest.NewRecorder()
	msr2 := httptest.NewRequest("GET", "/api/v1/beta/metrics/violations/prometheus", nil)
	srv.router.ServeHTTP(ms2, msr2)
	if !strings.Contains(ms2.Body.String(), "gauth_persistence_integrity_semantic 0") {
		t.Fatalf("expected semantic integrity gauge=0 after tamper got: %s", ms2.Body.String())
	}
}
