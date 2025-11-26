package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
)

// doPost helper for JSON POST
func doPostMetrics(bs *BetaServer, path string, body any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	return w
}

// TestCapabilityEnforcementMetrics verifies allowed/denied counters increment.
func TestCapabilityEnforcementMetrics(t *testing.T) {
	t.Setenv("GAUTH_CAPABILITY_ENFORCE", "1")
	mem := imetrics.NewMemory()
	bs := NewBetaServerWithMetrics("", mem)
	t.Cleanup(func() { bs.Shutdown() })
	// Denied create (missing capability)
	resp := doPostMetrics(bs, "/api/v1/delegation/create", map[string]any{"delegation_id": "c1", "subject": "s1", "delegate": "dlt"})
	if resp.Code != 403 {
		// Ensure enforcement denial path hit
		b := resp.Body.String()
		if !bytes.Contains(resp.Body.Bytes(), []byte("capability_denied")) {
			t.Fatalf("expected denial (403 capability_denied) code=%d body=%s", resp.Code, b)
		}
	}
	// Allowed create (with capability)
	resp = doPostMetrics(bs, "/api/v1/delegation/create", map[string]any{"delegation_id": "c2", "subject": "s1", "delegate": "dlt", "claims": map[string]any{"cap": []string{"cap.delegation.create"}}})
	if resp.Code != 200 {
		b := resp.Body.String()
		if bytes.Contains(resp.Body.Bytes(), []byte("capability_denied")) {
			t.Fatalf("unexpected denial for allowed path code=%d body=%s", resp.Code, b)
		}
		// Fail explicitly if not success
		t.Fatalf("expected success code=200 got=%d body=%s", resp.Code, b)
	}
	// Snapshot counters
	allowed := mem.CapabilityEnforceAllowed()
	denied := mem.CapabilityEnforceDenied()
	if allowed != 1 {
		// Provide diagnostic snapshot of metrics memory struct
		b := resp.Body.String()
		t.Fatalf("expected allowed counter=1 got=%d last body=%s", allowed, b)
	}
	if denied != 1 {
		t.Fatalf("expected denied counter=1 got=%d", denied)
	}
}
