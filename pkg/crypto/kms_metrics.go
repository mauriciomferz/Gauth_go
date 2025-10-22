package crypto

// kms_metrics.go
// Prometheus instrumentation for KMS operations (mock or real). Kept separate from internal/metrics
// to avoid widening the core Metrics interface prematurely. Enabled automatically when
// GAUTH_KMS_METRICS=1 is set (or explicitly via EnableKMSPrometheusMetrics()).

import (
	"os"
	"sync"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	kmsActiveSignerRequests *prom.CounterVec
	kmsRotateTotal          *prom.CounterVec
	kmsListKeysTotal        *prom.CounterVec
	kmsLatencyHistogram     *prom.HistogramVec
	registerKMSMetricsOnce  sync.Once
	kmsMetricsEnabled       bool
)

// EnableKMSPrometheusMetrics registers KMS Prometheus collectors (idempotent).
// Namespace/subsystem kept consistent with other metrics: namespace=gauth subsystem=crypto unless overridden.
func EnableKMSPrometheusMetrics(namespace, subsystem string) {
	if namespace == "" {
		namespace = "gauth"
	}
	if subsystem == "" {
		subsystem = "crypto"
	}
	registerKMSMetricsOnce.Do(func() {
		kmsActiveSignerRequests = prom.NewCounterVec(prom.CounterOpts{Namespace: namespace, Subsystem: subsystem, Name: "kms_active_signer_requests_total", Help: "Number of ActiveSigner() retrievals from KMS"}, []string{"provider"})
		kmsRotateTotal = prom.NewCounterVec(prom.CounterOpts{Namespace: namespace, Subsystem: subsystem, Name: "kms_rotate_total", Help: "Number of key rotations performed by KMS"}, []string{"provider"})
		kmsListKeysTotal = prom.NewCounterVec(prom.CounterOpts{Namespace: namespace, Subsystem: subsystem, Name: "kms_list_keys_total", Help: "Number of ListKeys() calls"}, []string{"provider"})
		kmsLatencyHistogram = prom.NewHistogramVec(prom.HistogramOpts{Namespace: namespace, Subsystem: subsystem, Name: "kms_operation_latency_seconds", Help: "Latency of KMS operations in seconds", Buckets: prom.DefBuckets}, []string{"provider", "op"})
		// Register (ignore duplicate registration errors silently)
		prom.MustRegister(kmsActiveSignerRequests, kmsRotateTotal, kmsListKeysTotal, kmsLatencyHistogram)
		kmsMetricsEnabled = true
	})
}

// maybeEnableKMSMetrics checks env flag.
func maybeEnableKMSMetrics() {
	if os.Getenv("GAUTH_KMS_METRICS") == "1" {
		EnableKMSPrometheusMetrics("gauth", "crypto")
	}
}

// recordKMSMetric utility: wraps in latency measurement.
func recordKMSMetric(provider, op string, fn func()) {
	start := time.Now()
	fn()
	if !kmsMetricsEnabled {
		return
	}
	switch op {
	case "active_signer":
		kmsActiveSignerRequests.WithLabelValues(provider).Inc()
	case "rotate":
		kmsRotateTotal.WithLabelValues(provider).Inc()
	case "list_keys":
		kmsListKeysTotal.WithLabelValues(provider).Inc()
	}
	kmsLatencyHistogram.WithLabelValues(provider, op).Observe(time.Since(start).Seconds())
}
