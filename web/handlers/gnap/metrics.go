// Package gnap provides Prometheus metrics for GNAP operations.
package gnap

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// GrantRequestsTotal counts grant requests by result
	GrantRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gnap_grant_requests_total",
			Help: "Total number of GNAP grant requests",
		},
		[]string{"result", "interaction_required"},
	)

	// GrantContinuationsTotal counts continuation requests by result
	GrantContinuationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gnap_grant_continuations_total",
			Help: "Total number of GNAP grant continuation requests",
		},
		[]string{"result"},
	)

	// TokenOperationsTotal counts token operations by type
	TokenOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gnap_token_operations_total",
			Help: "Total number of GNAP token operations",
		},
		[]string{"operation"}, // issue, rotate, revoke
	)

	// GrantDurationSeconds tracks grant processing time
	GrantDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gnap_grant_duration_seconds",
			Help:    "Time taken to process grant requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
	)

	// ActiveGrantsGauge tracks currently active grants
	ActiveGrantsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gnap_active_grants",
			Help: "Number of currently active (non-terminal) grants",
		},
	)

	// SignatureVerificationsTotal counts signature verification results
	SignatureVerificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gnap_signature_verifications_total",
			Help: "Total number of HTTP signature verifications",
		},
		[]string{"result"}, // success, failure, skipped
	)
)

// RecordGrantRequest records a grant request metric.
func RecordGrantRequest(success bool, interactionRequired bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	interaction := "false"
	if interactionRequired {
		interaction = "true"
	}
	GrantRequestsTotal.WithLabelValues(result, interaction).Inc()
}

// RecordGrantContinuation records a continuation metric.
func RecordGrantContinuation(success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	GrantContinuationsTotal.WithLabelValues(result).Inc()
}

// RecordTokenOperation records a token operation metric.
func RecordTokenOperation(operation string) {
	TokenOperationsTotal.WithLabelValues(operation).Inc()
}

// RecordSignatureVerification records signature verification result.
func RecordSignatureVerification(result string) {
	SignatureVerificationsTotal.WithLabelValues(result).Inc()
}
