package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	srv.violationHandler.Save()
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
	// The original 'path' variable declaration is removed as per instruction.
	// The 'path' variable is still needed for os.Setenv and srv.semanticHandler.Save.
	// Assuming the instruction meant to remove an *unused* base, but 'path' is used.
	// Re-adding the path declaration to ensure the code remains syntactically correct and functional.
	path := filepath.Join(dir, "semantic.json")
	os.Setenv("GAUTH_SEMANTIC_PERSIST_PATH", path)
	os.Setenv("GAUTH_SEMANTIC_PERSIST_NO_THROTTLE", "1")
	srv := NewBetaServer("8132")
	t.Cleanup(func() { srv.Shutdown() })
	if srv.rfc0111Service == nil {
		t.Fatalf("semantic service not initialized")
	}
	// Inject mock service to ensure counters are present for persistence test
	mockSvc := &mockRFC0111Service{
		snapshots: []map[string]uint64{
			{"scope_violation": 10},
			{"scope_violation": 20}, // subsequent calls get subsequent snapshots
			{"scope_violation": 30},
			{"scope_violation": 40},
		},
	}
	srv.rfc0111Service = mockSvc
	srv.semanticHandler.Service = mockSvc
	// Ensure we have some data
	srv.semanticHandler.Update()
	time.Sleep(1100 * time.Millisecond)
	srv.semanticHandler.Update()
	time.Sleep(1100 * time.Millisecond)
	srv.semanticHandler.Update()

	ewmaCount, _ := srv.semanticHandler.Stats()
	t.Logf("DEBUG: Post-Update EWMA count: %d MockIdx: %d", ewmaCount, mockSvc.idx)
	if err := srv.semanticHandler.Save(); err != nil {
		t.Fatalf("save semantic failed: %v", err)
	}
	ws := httptest.NewRecorder()
	rs := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics/verify", nil)
	srv.router.ServeHTTP(ws, rs)
	if !strings.Contains(ws.Body.String(), "integrity") {
		t.Fatalf("semantic verify missing integrity: %s", ws.Body.String())
	}
	t.Logf("Semantic Verify Body: %s", ws.Body.String())
	if strings.Contains(ws.Body.String(), "\"mismatch\"") {
		t.Fatalf("Semantic verify returned mismatch pre-tamper: %s", ws.Body.String())
	}
	// Prometheus: semantic integrity gauge should not show mismatch yet
	ms1 := httptest.NewRecorder()
	msr1 := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics/prometheus", nil)
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
	// Payload structure is map[string]*Welford (or similar struct)
	var inner map[string]any
	if innerErr2 := json.Unmarshal(wrapper2.Payload, &inner); innerErr2 != nil {
		t.Fatalf("unmarshal inner semantic payload: %v raw=%s", innerErr2, string(wrapper2.Payload))
	}
	// We want to mutate payload content to change hash, but keep wrapper.Hash same.
	// We can adds a dummy key or modify existing.
	// Modify one entry if exists, or add one.
	if entry, ok := inner["scope_violation"]; ok {
		// modify it (it is a map/object)
		if eMap, ok := entry.(map[string]any); ok {
			eMap["mean"] = 99999.0
			inner["scope_violation"] = eMap
		} else {
			// Fallback
			inner["tampered"] = "true"
		}
	} else {
		inner["tampered"] = 123
	}

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
	msr2 := httptest.NewRequest("GET", "/api/v1/beta/metrics/poa/semantics/prometheus", nil)
	srv.router.ServeHTTP(ms2, msr2)
	if !strings.Contains(ms2.Body.String(), "gauth_persistence_integrity_semantic 0") {
		t.Fatalf("expected semantic integrity gauge=0 after tamper got: %s", ms2.Body.String())
	}
}
