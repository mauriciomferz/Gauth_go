package crypto

import (
	"os"
	"testing"

	prom "github.com/prometheus/client_golang/prometheus"
)

// TestKMSMetricsEmission ensures that when GAUTH_KMS_METRICS=1 is set the mock KMS
// emits Prometheus metrics for operations. We do a minimal sanity check on counter > 0
// and histogram sample presence. This is intentionally coarse to avoid flakiness from
// precise bucket counts.
func TestKMSMetricsEmission(t *testing.T) {
	t.Setenv("GAUTH_KMS_METRICS", "1")
	// Re-enable metrics in case another test already initialized them with different state.
	// (EnableKMSPrometheusMetrics is idempotent; safe to call.)
	EnableKMSPrometheusMetrics("gauth", "crypto")

	kms, err := NewMockKMS()
	if err != nil {
		t.Fatalf("new mock kms: %v", err)
	}

	// Exercise operations
	if _, err2 := kms.ActiveSigner(); err2 != nil {
		t.Fatalf("active signer: %v", err2)
	}
	if _, _, err2 := kms.PublicKey(kms.active.keyID); err2 != nil {
		t.Fatalf("public key: %v", err2)
	}
	if _, err2 := kms.Rotate(); err2 != nil {
		t.Fatalf("rotate: %v", err2)
	}
	if _, err2 := kms.ListKeys(); err2 != nil {
		t.Fatalf("list keys: %v", err2)
	}

	mfs, err := prom.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}

	// Helper to find metric family
	find := func(name string) *float64 {
		for _, mf := range mfs {
			if mf.GetName() == name {
				// Use first counter value (single provider label expected)
				if len(mf.Metric) > 0 {
					switch {
					case mf.GetType().String() == "COUNTER":
						return mf.Metric[0].Counter.Value
					case mf.GetType().String() == "HISTOGRAM":
						// Return sample count
						if mf.Metric[0].Histogram != nil {
							v := float64(mf.Metric[0].Histogram.GetSampleCount())
							return &v
						}
					}
				}
			}
		}
		return nil
	}

	if v := find("gauth_crypto_kms_active_signer_requests_total"); v == nil || *v == 0 {
		t.Fatalf("expected active signer counter >0, got %v", v)
	}
	if v := find("gauth_crypto_kms_rotate_total"); v == nil || *v == 0 {
		t.Fatalf("expected rotate counter >0, got %v", v)
	}
	if v := find("gauth_crypto_kms_list_keys_total"); v == nil || *v == 0 {
		t.Fatalf("expected list keys counter >0, got %v", v)
	}
	if v := find("gauth_crypto_kms_operation_latency_seconds"); v == nil || *v == 0 {
		t.Fatalf("expected latency histogram sample count >0, got %v", v)
	}

	// Suppress unused import warning if optimization removes something.
	_ = os.Getenv
}
