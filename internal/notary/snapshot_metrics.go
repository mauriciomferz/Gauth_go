package notary

import (
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Snapshot metrics (demo scope): latency histograms + outcome counters.
// Labels/outcomes: success | failure (verification invalid) | error (internal computation error)
var (
	snapshotGenerationLatency = promauto.NewHistogram(prom.HistogramOpts{
		Name:    "agentauth_snapshot_generation_latency_seconds",
		Help:    "Latency of snapshot generation operations",
		Buckets: prom.DefBuckets,
	})
	snapshotGenerationCounter = promauto.NewCounterVec(prom.CounterOpts{
		Name: "agentauth_snapshot_generation_total",
		Help: "Total snapshot generation attempts labeled by outcome (success|error)",
	}, []string{"outcome"})
	snapshotVerificationLatency = promauto.NewHistogram(prom.HistogramOpts{
		Name:    "agentauth_snapshot_verification_latency_seconds",
		Help:    "Latency of snapshot verification operations",
		Buckets: prom.DefBuckets,
	})
	snapshotVerificationCounter = promauto.NewCounterVec(prom.CounterOpts{
		Name: "agentauth_snapshot_verification_total",
		Help: "Total snapshot verification attempts labeled by outcome (success|failure|error)",
	}, []string{"outcome"})
)

func recordSnapshotGeneration(start time.Time, err error) {
	snapshotGenerationLatency.Observe(time.Since(start).Seconds())
	if err != nil {
		snapshotGenerationCounter.WithLabelValues("error").Inc()
	} else {
		snapshotGenerationCounter.WithLabelValues("success").Inc()
	}
}

func recordSnapshotVerification(start time.Time, valid bool, err error) {
	snapshotVerificationLatency.Observe(time.Since(start).Seconds())
	if err != nil {
		snapshotVerificationCounter.WithLabelValues("error").Inc()
		return
	}
	if valid {
		snapshotVerificationCounter.WithLabelValues("success").Inc()
	} else {
		snapshotVerificationCounter.WithLabelValues("failure").Inc()
	}
}
