package metrics

import (
	"log"
	"os"
	"time"
)

// LoggingMetrics is a simple metrics collector that logs significant events.
// It is intended as a failover backend when the primary collector (e.g. Prometheus) is unavailable.
type LoggingMetrics struct {
	noop
	logger *log.Logger
}

// NewLoggingMetrics creates a new LoggingMetrics collector.
// If logger is nil, it defaults to logging to os.Stderr.
func NewLoggingMetrics(logger *log.Logger) *LoggingMetrics {
	if logger == nil {
		logger = log.New(os.Stderr, "[METRICS-FAILOVER] ", log.LstdFlags)
	}
	return &LoggingMetrics{
		logger: logger,
	}
}

// RecordDecision logs access decisions.
func (m *LoggingMetrics) RecordDecision(action, resource, outcome string, d time.Duration) {
	m.logger.Printf("Decision: action=%s resource=%s outcome=%s latency=%v", action, resource, outcome, d)
}

// RecordDecisionWithReason logs access decisions with reason.
func (m *LoggingMetrics) RecordDecisionWithReason(action, resource, outcome, reason string) {
	m.logger.Printf("Decision: action=%s resource=%s outcome=%s reason=%s", action, resource, outcome, reason)
}

// IncViolation logs policy violations.
func (m *LoggingMetrics) IncViolation(cat interface{}) {
	m.logger.Printf("Violation: category=%v", cat)
}

// IncUnauthorized logs unauthorized access attempts.
func (m *LoggingMetrics) IncUnauthorized() {
	m.logger.Println("Unauthorized access attempt")
}

// IncSignatureVerificationFailures logs signature verification failures.
func (m *LoggingMetrics) IncSignatureVerificationFailures() {
	m.logger.Println("Signature verification failed")
}

// IncAttestationProofVerificationFailures logs attestation failures.
func (m *LoggingMetrics) IncAttestationProofVerificationFailures() {
	m.logger.Println("Attestation proof verification failed")
}
