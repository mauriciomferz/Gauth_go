package notary

import (
	"sync/atomic"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Rotation metrics outcome constants.
const (
	rotationOutcomeSuccess = "success"
	rotationOutcomeFailure = "failure"
	rotationOutcomeError   = "error"
)

var (
	rotationVerificationLatency = promauto.NewHistogram(prom.HistogramOpts{
		Name:    "agentauth_rotation_verification_latency_seconds",
		Help:    "Latency of key rotation chain verification operations",
		Buckets: prom.DefBuckets,
	})
	rotationVerificationCounter = promauto.NewCounterVec(prom.CounterOpts{
		Name: "agentauth_rotation_verification_total",
		Help: "Total rotation verification attempts labeled by outcome (success|failure|error)",
	}, []string{"outcome"})
	rotationVerificationFailureReason = promauto.NewCounterVec(prom.CounterOpts{
		Name: "agentauth_rotation_verification_failure_reason_total",
		Help: "Total rotation verification failures labeled by reason",
	}, []string{"reason"})
	rotationSummaryLatency = promauto.NewHistogram(prom.HistogramOpts{
		Name:    "agentauth_rotation_summary_latency_seconds",
		Help:    "Latency of rotation summary build (including optional signing)",
		Buckets: prom.DefBuckets,
	})
	rotationSummaryCounter = promauto.NewCounterVec(prom.CounterOpts{
		Name: "agentauth_rotation_summary_total",
		Help: "Total rotation summary requests labeled by outcome (success|error)",
	}, []string{"outcome"})
	rotationSummaryAnchors = promauto.NewCounterVec(prom.CounterOpts{
		Name: "agentauth_rotation_summary_anchor_total",
		Help: "Rotation summary anchoring attempts labeled by result (anchored|skipped|error)",
	}, []string{"result"})

	// rotationSummaryChainLength -> metrics.RotationChainLength
	// rotationSummaryHeadAgeSeconds -> metrics.RotationHeadAge
	// rotationSummaryLastAnchorAgeSeconds -> metrics.RotationLastAnchorAge
	// rotationSignatureVerifyLatency -> metrics.RotationSignatureVerifyLatency

	rotationSignatureVerifyFailures = promauto.NewCounterVec(prom.CounterOpts{
		Name: "agentauth_rotation_signature_verify_failures_total",
		Help: "Total failed rotation signature verifications labeled by reason",
	}, []string{"reason"})
	lastAnchorUnixNano atomic.Int64
)

func recordRotationVerification(start time.Time, summary RotationVerificationSummary) {
	rotationVerificationLatency.Observe(time.Since(start).Seconds())
	outcome := rotationOutcomeSuccess
	if summary.Failures > 0 {
		outcome = rotationOutcomeFailure
	}
	rotationVerificationCounter.WithLabelValues(outcome).Inc()
	if summary.Failures > 0 {
		for _, r := range summary.Results {
			if r.Reason != "" {
				rotationVerificationFailureReason.WithLabelValues(r.Reason).Inc()
			}
		}
	}
}

// RecordRotationSummary records metrics for the rotation summary endpoint.
func RecordRotationSummary(
	start time.Time,
	sum *RotationSummary,
	anchored bool,
	err error,
	anchorAttempted bool,
	anchorErr error,
) {
	rotationSummaryLatency.Observe(time.Since(start).Seconds())
	outcome := rotationOutcomeSuccess
	if err != nil {
		outcome = rotationOutcomeError
	}
	rotationSummaryCounter.WithLabelValues(outcome).Inc()
	if sum != nil {
		metrics.RotationChainLength.Set(float64(sum.ChainLength))
		if ts, perr := time.Parse(time.RFC3339Nano, sum.GeneratedAt); perr == nil {
			metrics.RotationHeadAge.Set(time.Since(ts).Seconds())
		}
	}
	if anchorAttempted {
		switch {
		case anchorErr != nil:
			rotationSummaryAnchors.WithLabelValues("error").Inc()
		case anchored:
			rotationSummaryAnchors.WithLabelValues("anchored").Inc()
			// record the successful anchor time
			lastAnchorUnixNano.Store(time.Now().UnixNano())
		default:
			rotationSummaryAnchors.WithLabelValues("skipped").Inc()
		}
	}
	// update last anchor age gauge if we have a recorded anchor
	if ts := lastAnchorUnixNano.Load(); ts > 0 {
		anchorTime := time.Unix(0, ts)
		metrics.RotationLastAnchorAge.Set(time.Since(anchorTime).Seconds())
	}
}
