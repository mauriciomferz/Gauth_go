package agentauth_aap_001

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	detachedOnce   sync.Once
	detachedIssued = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "agentauth", Subsystem: "token", Name: "detached_signature_issued_total",
		Help: "Count of envelopes issued with detached signature.",
	})
	detachedVerify = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agentauth", Subsystem: "token", Name: "detached_signature_verify_total",
		Help: "Verification outcomes for detached signatures (success|invalid_signature|pubkey_missing|digest_mismatch).",
	}, []string{"outcome"})
)

func registerDetachedMetrics() {
	detachedOnce.Do(func() {
		prometheus.MustRegister(detachedIssued, detachedVerify)
	})
}

// helper wrappers used inside issuance / verification paths (kept small to avoid import churn in core file)
//
//nolint:lll,unused // reserved for future detached PoA feature
func incDetachedIssued() { registerDetachedMetrics(); detachedIssued.Inc() }
func incDetachedVerify(outcome string) {
	registerDetachedMetrics()
	detachedVerify.WithLabelValues(outcome).Inc()
}
