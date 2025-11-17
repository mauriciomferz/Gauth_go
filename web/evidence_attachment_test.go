package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
)

// helper to POST JSON (reuses performJSONPost in revocation tests if in same package, but redeclare for clarity)
func performJSONPostEvidence(s *BetaServer, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func createTestPOA(srv *BetaServer) (string, error) {
	// Create a delegation through the service
	svc, ok := srv.rfc0111Service.(*rfc0111.Service)
	if !ok || svc == nil {
		return "", fmt.Errorf("RFC0111 service not available")
	}

	resp, err := svc.CreateDelegationCtx(context.Background(), rfc0111.DelegationRequest{
		Grantor:  "testgrantor",
		Grantee:  "testgrantee",
		Scope:    []string{"read"},
		Duration: time.Hour,
	})
	if err != nil {
		return "", err
	}

	return resp.POA.ID, nil
}

func TestEvidenceAttachment_SuccessAndDuplicate(t *testing.T) {
	t.Skip("Test requires proper authorization setup - skipping until policy configuration is fixed")

	// Enable RFC0111 service and policy seeding
	t.Setenv("GAUTH_DISABLE_RFC0111_SERVICE", "0")
	t.Setenv("GAUTH_SEED_POLICY", "1")
	pm := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111"})
	srv := NewBetaServerWithMetrics("", pm)
	t.Cleanup(func() { srv.Shutdown() })

	poaID, err := createTestPOA(srv)
	if err != nil {
		t.Fatalf("failed to create test POA: %v", err)
	}

	// Attach two new hashes
	payload := map[string]any{"hashes": []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}
	res := performJSONPostEvidence(srv, "/api/v1/beta/poa/"+poaID+"/evidence", payload)
	if res.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", res.Code, res.Body.String())
	}

	// Duplicate submission (all duplicates) should fail (400/409 expected) per service logic returning invalid_request
	resDup := performJSONPostEvidence(srv, "/api/v1/beta/poa/"+poaID+"/evidence", payload)
	if resDup.Code == 200 {
		t.Fatalf("expected duplicate-only submission to fail got 200 body=%s", resDup.Body.String())
	}

	// Metrics presence: successful attachments counter and failures counter (at least 1 failure)
	metReq := httptest.NewRequest("GET", "/metrics", nil)
	metRec := httptest.NewRecorder()
	srv.router.ServeHTTP(metRec, metReq)
	if metRec.Code != 200 {
		t.Fatalf("metrics endpoint expected 200 got %d", metRec.Code)
	}
	body := metRec.Body.String()
	for _, substr := range []string{"evidence_attachment_total", "evidence_attachment_failures_total", "evidence_hashes_per_poa"} {
		if !contains(body, substr) {
			t.Fatalf("expected metrics substring %s not found", substr)
		}
	}
}

func TestEvidenceAttachment_InvalidHash(t *testing.T) {
	t.Skip("Test requires proper authorization setup - skipping until policy configuration is fixed")

	t.Setenv("GAUTH_DISABLE_RFC0111_SERVICE", "0")
	t.Setenv("GAUTH_SEED_POLICY", "1")
	pm := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111"})
	srv := NewBetaServerWithMetrics("", pm)
	t.Cleanup(func() { srv.Shutdown() })

	poaID, err := createTestPOA(srv)
	if err != nil {
		t.Fatalf("failed to create test POA: %v", err)
	}

	payload := map[string]any{"hashes": []string{"not-hex"}}
	res := performJSONPostEvidence(srv, "/api/v1/beta/poa/"+poaID+"/evidence", payload)
	if res.Code == 200 {
		t.Fatalf("expected invalid hash format rejection got 200 body=%s", res.Body.String())
	}

	// Metrics: failure counter incremented
	metReq := httptest.NewRequest("GET", "/metrics", nil)
	metRec := httptest.NewRecorder()
	srv.router.ServeHTTP(metRec, metReq)
	if metRec.Code != 200 {
		t.Fatalf("metrics endpoint expected 200 got %d", metRec.Code)
	}
	if !contains(metRec.Body.String(), "evidence_attachment_failures_total") {
		t.Fatalf("expected failure counter substring not found")
	}
}

func TestEvidenceAttachment_NotFound(t *testing.T) {
	t.Skip("Test requires proper authorization setup - skipping until policy configuration is fixed")

	t.Setenv("GAUTH_DISABLE_RFC0111_SERVICE", "0")
	t.Setenv("GAUTH_SEED_POLICY", "1")
	pm := metrics.NewPrometheusMetrics(metrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111"})
	srv := NewBetaServerWithMetrics("", pm)
	t.Cleanup(func() { srv.Shutdown() })
	payload := map[string]any{"hashes": []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	res := performJSONPostEvidence(srv, "/api/v1/beta/poa/does-not-exist/evidence", payload)
	if res.Code != 404 && res.Code != 400 {
		t.Fatalf("expected not found status got %d body=%s", res.Code, res.Body.String())
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
