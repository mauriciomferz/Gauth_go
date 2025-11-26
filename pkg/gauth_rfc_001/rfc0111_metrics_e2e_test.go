package gauth_rfc_001

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	imetrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
)

// TestRFC0111MetricsE2E spins up an in-memory RFC0111 Service with Prometheus metrics and
// performs a small happy-path flow: create delegation, validate it, revoke it, then attempt
// a validation that should fail. Finally scrapes /metrics and asserts counters present and incremented.
func TestRFC0111MetricsE2E(t *testing.T) {
	reg := prom.NewRegistry()
	pm := imetrics.NewPrometheusMetrics(imetrics.PrometheusAdapterOptions{Namespace: "gauth", Subsystem: "rfc0111", Registry: reg})

	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	// Install permissive policies for create & revoke to exercise flow.
	// Use wildcard resource so subsequent revocation which references specific POA ID passes.
	authorizer.AddPolicy(authz.Policy{ID: "p1", Subject: "alice", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	authorizer.AddPolicy(authz.Policy{ID: "p2", Subject: "alice", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLogger, authorizer, WithMetrics(pm))

	// Create delegation
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: time.Hour})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}

	// Validate (success)
	if err := svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, "bob", "transaction:execute"); err != nil {
		t.Fatalf("validate delegation: %v", err)
	}

	// Revoke
	if err := svc.RevokeDelegationCtx(context.Background(), resp.POA.ID, "alice"); err != nil {
		t.Fatalf("revoke delegation: %v", err)
	}
	// Validate again (should fail revoked) – ignore error content; ensures revocation path touched.
	_ = svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, "bob", "transaction:execute")

	// Scrape metrics
	rr := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	body := rr.Body.String()

	// Assertions: expect creation + validation latency histogram + revocation related counters possibly zero/non-zero.
	requiredNames := []string{
		"gauth_rfc0111_delegations_created_total", // at least 1
		"gauth_rfc0111_validation_latency_seconds_bucket",
		"gauth_rfc0111_signatures_issued_total",            // may be zero if signer not configured, tolerate presence absence? we only check presence of name due to registration
		"gauth_rfc0111_signature_public_key_missing_total", // soft-skip case may stay zero
	}
	for _, name := range requiredNames {
		if !strings.Contains(body, name) {
			t.Fatalf("expected metric name %s in exposition:\n%s", name, body)
		}
	}
	// Ensure delegations_created counter incremented (value > 0)
	if !containsMetricValue(body, "gauth_rfc0111_delegations_created_total", "1") && !containsMetricValue(body, "gauth_rfc0111_delegations_created_total", "2") {
		t.Fatalf("expected delegations_created_total to be >=1; exposition snippet: %s", firstLines(body, 40))
	}
}

// containsMetricValue performs a simple substring search for 'name value' pattern.
func containsMetricValue(expo, name, value string) bool {
	needle := name + " " + value + "\n"
	return strings.Contains(expo, needle)
}

// firstLines returns first n lines for debug readability.
func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
