package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TestAttestationTrustAnchorMetricsExposure verifies that the three granular
// attestation trust anchor failure counters are registered and incremented.
func TestAttestationTrustAnchorMetricsExposure(t *testing.T) {
	reg := prom.NewRegistry()
	pm := NewPrometheusMetrics(PrometheusAdapterOptions{Registry: reg})

	// Increment each counter once to ensure a non-zero sample value.
	pm.IncAttestationProofTrustAnchorMissing()
	pm.IncAttestationProofTrustAnchorAlgorithmMismatch()
	pm.IncAttestationProofTrustAnchorKeyMismatch()
	pm.IncCryptoSignatureMissing()

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	wanted := map[string]float64{
		"agentauth_aap001_attestation_proof_trust_anchor_missing_total":            0,
		"agentauth_aap001_attestation_proof_trust_anchor_algorithm_mismatch_total": 0,
		"agentauth_aap001_attestation_proof_trust_anchor_key_mismatch_total":       0,
		"agentauth_aap001_crypto_signature_missing_total":                          0,
	}

	seen := map[string]bool{}
	for _, mf := range mfs {
		name := mf.GetName()
		if _, ok := wanted[name]; ok {
			var total float64
			for _, m := range mf.GetMetric() {
				if c := m.GetCounter(); c != nil {
					total += c.GetValue()
				}
			}
			wanted[name] = total
			seen[name] = true
		}
	}

	for name, val := range wanted {
		if !seen[name] {
			t.Errorf("expected metric %s to be registered", name)
			continue
		}
		if val < 1 {
			t.Errorf("expected metric %s value >= 1 after increments, got %v", name, val)
		}
	}
}

// TestPrometheusAdapterBasic verifies counters increment & exposition contains expected names.
func TestPrometheusAdapterBasic(t *testing.T) {
	reg := prom.NewRegistry()
	pm := NewPrometheusMetrics(PrometheusAdapterOptions{Namespace: "agentauth", Subsystem: "test", Registry: reg})
	pm.IncDelegationsCreated()
	pm.IncDelegationsCreated()
	pm.IncSignaturesIssued()
	pm.IncSignatureIssueFailures()
	pm.IncSignatureVerifications()
	pm.IncSignatureVerificationFailures()
	pm.IncRevocationIntegrityFailures()
	pm.IncSignaturePublicKeyMissing()
	pm.ObserveValidationLatency(1234) // very small duration sample

	// Scrape exposition
	rr := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", nil))
	body := rr.Body.String()
	expected := []string{
		"agentauth_test_delegations_created_total",
		"agentauth_test_signatures_issued_total",
		"agentauth_test_signature_issue_failures_total",
		"agentauth_test_validation_latency_seconds_bucket",
		"agentauth_test_signature_public_key_missing_total",
	}
	for _, name := range expected {
		if !strings.Contains(body, name) {
			t.Fatalf("expected to find metric %s in exposition\nBody:\n%s", name, body)
		}
	}
}
