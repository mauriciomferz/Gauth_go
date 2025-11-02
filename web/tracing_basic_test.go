package web

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

// TestTracingEnabledEmitsSpans ensures token.issue and token.validate spans appear when tracing enabled.
func TestTracingEnabledEmitsSpans(t *testing.T) {
	os.Setenv("GAUTH_TRACING_ENABLED", "1")
	os.Setenv("GAUTH_TRACING_SAMPLE_RATIO", "1")
	defer func() {
		os.Unsetenv("GAUTH_TRACING_ENABLED")
		os.Unsetenv("GAUTH_TRACING_SAMPLE_RATIO")
	}()
	srv := NewBetaServer("0")
	// Token create
	body := bytes.NewBufferString(`{"ttl_seconds":120}`)
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, httptest.NewRequest("POST", "/api/v1/token/create", body))
	if w1.Code != 201 { t.Fatalf("token create status %d (expected 201)", w1.Code) }
	var createResp struct { Success bool `json:"success"`; Token struct { ID string `json:"id"` } `json:"token"` }
	if err := json.Unmarshal(w1.Body.Bytes(), &createResp); err != nil || !createResp.Success || createResp.Token.ID == "" {
		 t.Fatalf("token create parse failed: %v body=%s", err, w1.Body.String())
	}
	// Token validate
	valBody := bytes.NewBufferString(`{"token_id":"` + createResp.Token.ID + `"}`)
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, httptest.NewRequest("POST", "/api/v1/token/validate", valBody))
	if w2.Code != 200 { t.Fatalf("token validate status %d", w2.Code) }
	if srv.tracerProvider == nil { t.Fatalf("tracerProvider nil when tracing enabled") }
	ops := map[string]bool{}
	for _, sp := range srv.tracerProvider.Spans() { ops[sp.Operation] = true }
	if !ops["token.issue"] { t.Fatalf("missing token.issue span; ops=%v", ops) }
	if !ops["token.validate"] { t.Fatalf("missing token.validate span; ops=%v", ops) }
}

// TestTracingDisabledNoSpans ensures spans absent when tracing not enabled.
func TestTracingDisabledNoSpans(t *testing.T) {
	os.Unsetenv("GAUTH_TRACING_ENABLED")
	os.Unsetenv("GAUTH_OTEL_ENABLE")
	os.Unsetenv("GAUTH_TRACING_SAMPLE_RATIO")
	srv := NewBetaServer("0")
	// Token create
	body := bytes.NewBufferString(`{"ttl_seconds":60}`)
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, httptest.NewRequest("POST", "/api/v1/token/create", body))
	if w1.Code != 201 { t.Fatalf("token create status %d (expected 201)", w1.Code) }
	// Token validate
	var createResp struct { Success bool `json:"success"`; Token struct { ID string `json:"id"` } `json:"token"` }
	if err := json.Unmarshal(w1.Body.Bytes(), &createResp); err != nil || createResp.Token.ID == "" { t.Fatalf("create parse failed: %v %s", err, w1.Body.String()) }
	valBody := bytes.NewBufferString(`{"token_id":"` + createResp.Token.ID + `"}`)
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, httptest.NewRequest("POST", "/api/v1/token/validate", valBody))
	if w2.Code != 200 { t.Fatalf("validate status %d", w2.Code) }
	if srv.tracerProvider != nil {
		if len(srv.tracerProvider.Spans()) > 0 {
			 t.Fatalf("expected no spans when tracing disabled; got %d", len(srv.tracerProvider.Spans()))
		}
	}
}

// TestTracingSampleRatioZeroEmitsSpans ensures ratio=0 (always sample by implementation) still records spans.
func TestTracingSampleRatioZeroEmitsSpans(t *testing.T) {
	os.Setenv("GAUTH_TRACING_ENABLED", "1")
	os.Setenv("GAUTH_TRACING_SAMPLE_RATIO", "0")
	defer func() {
		os.Unsetenv("GAUTH_TRACING_ENABLED")
		os.Unsetenv("GAUTH_TRACING_SAMPLE_RATIO")
	}()
	srv := NewBetaServer("0")
	body := bytes.NewBufferString(`{"ttl_seconds":45}`)
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, httptest.NewRequest("POST", "/api/v1/token/create", body))
	if w1.Code != 201 { t.Fatalf("token create status %d (expected 201)", w1.Code) }
	var createResp struct { Success bool `json:"success"`; Token struct { ID string `json:"id"` } `json:"token"` }
	if err := json.Unmarshal(w1.Body.Bytes(), &createResp); err != nil || createResp.Token.ID == "" { t.Fatalf("create parse failed: %v %s", err, w1.Body.String()) }
	valBody := bytes.NewBufferString(`{"token_id":"` + createResp.Token.ID + `"}`)
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, httptest.NewRequest("POST", "/api/v1/token/validate", valBody))
	if w2.Code != 200 { t.Fatalf("validate status %d", w2.Code) }
	if srv.tracerProvider == nil { t.Fatalf("tracerProvider nil when tracing enabled ratio=0") }
	ops := map[string]bool{}
	for _, sp := range srv.tracerProvider.Spans() { ops[sp.Operation] = true }
	if !ops["token.issue"] || !ops["token.validate"] { t.Fatalf("expected spans with ratio=0; ops=%v", ops) }
}

// TestTracingSampleRatioMidApproximatelySamples verifies probabilistic sampling at ratio=0.5.
// It asserts that some (not all) operations generate spans, reducing flakiness by using bounds.
func TestTracingSampleRatioMidApproximatelySamples(t *testing.T) {
	os.Setenv("GAUTH_TRACING_ENABLED", "1")
	os.Setenv("GAUTH_TRACING_SAMPLE_RATIO", "0.5")
	defer func() {
		os.Unsetenv("GAUTH_TRACING_ENABLED")
		os.Unsetenv("GAUTH_TRACING_SAMPLE_RATIO")
	}()
	srv := NewBetaServer("0")
	iterations := 40
	for i := 0; i < iterations; i++ {
		// create
		body := bytes.NewBufferString(`{"ttl_seconds":30}`)
		w1 := httptest.NewRecorder()
		srv.router.ServeHTTP(w1, httptest.NewRequest("POST", "/api/v1/token/create", body))
		if w1.Code != 201 { t.Fatalf("token create status %d (expected 201)", w1.Code) }
		var createResp struct { Success bool `json:"success"`; Token struct { ID string `json:"id"` } `json:"token"` }
		if err := json.Unmarshal(w1.Body.Bytes(), &createResp); err != nil || createResp.Token.ID == "" { t.Fatalf("create parse failed: %v %s", err, w1.Body.String()) }
		// validate
		valBody := bytes.NewBufferString(`{"token_id":"` + createResp.Token.ID + `"}`)
		w2 := httptest.NewRecorder()
		srv.router.ServeHTTP(w2, httptest.NewRequest("POST", "/api/v1/token/validate", valBody))
		if w2.Code != 200 { t.Fatalf("validate status %d", w2.Code) }
	}
	if srv.tracerProvider == nil { t.Fatalf("tracerProvider nil for mid-ratio test") }
	issueCount := 0
	validateCount := 0
	for _, sp := range srv.tracerProvider.Spans() {
		switch sp.Operation {
		case "token.issue":
			issueCount++
		case "token.validate":
			validateCount++
		}
	}
	// With ratio=0.5 and 40 attempts each, expect counts well within (5, 38) bounds.
	if issueCount <= 5 || issueCount >= 38 { t.Fatalf("token.issue spans out of expected range: %d", issueCount) }
	if validateCount <= 5 || validateCount >= 38 { t.Fatalf("token.validate spans out of expected range: %d", validateCount) }
}
