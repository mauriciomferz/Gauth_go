// Copyright (c) 2025 GAuth. All rights reserved.

// Package collectors provides MetricsCollector implementations for various backends.
package collectors

import (
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
)

// StatsDCollector exports metrics to StatsD/DogStatsD.
//
// This collector buffers metrics and sends them via UDP to a StatsD server.
// Implements the MetricsCollector interface for use with CollectorRegistry.
type StatsDCollector struct {
	client   StatsDClient
	metadata metrics.CollectorMetadata
}

// StatsDClient abstracts the StatsD client interface.
// Implementations can use github.com/DataDog/datadog-go/statsd or similar.
type StatsDClient interface {
	// Incr increments a counter metric by 1
	Incr(name string, tags []string, rate float64) error
	// Count increments a counter by n
	Count(name string, value int64, tags []string, rate float64) error
	// Gauge sets a gauge value
	Gauge(name string, value float64, tags []string, rate float64) error
	// Timing sends a timing metric (duration in milliseconds)
	Timing(name string, value time.Duration, tags []string, rate float64) error
	// Histogram sends a histogram metric
	Histogram(name string, value float64, tags []string, rate float64) error
	// Flush flushes buffered metrics
	Flush() error
	// Close closes the client connection
	Close() error
}

// NewStatsDCollector creates a StatsD collector.
//
// Parameters:
//   - id: Unique identifier for this collector
//   - client: StatsD client implementation (e.g., DogStatsD)
//   - description: Human-readable description
func NewStatsDCollector(id string, client StatsDClient, description string) *StatsDCollector {
	return &StatsDCollector{
		client: client,
		metadata: metrics.CollectorMetadata{
			ID:           id,
			Type:         metrics.CollectorTypeStatsD,
			Description:  description,
			RegisteredAt: time.Now(),
			Version:      "1.0.0",
		},
	}
}

// Metadata returns collector metadata.
func (s *StatsDCollector) Metadata() metrics.CollectorMetadata {
	return s.metadata
}

// Flush forces buffered metrics to be sent to StatsD.
func (s *StatsDCollector) Flush() error {
	return s.client.Flush()
}

// Close cleanly shuts down the StatsD connection.
func (s *StatsDCollector) Close() error {
	if err := s.Flush(); err != nil {
		return fmt.Errorf("flush before close: %w", err)
	}
	return s.client.Close()
}

// Health checks StatsD connectivity.
func (s *StatsDCollector) Health() error {
	// Send a heartbeat metric to verify connectivity
	return s.client.Gauge("gauth.collector.health", 1.0, nil, 1.0)
}

// Metrics interface implementation (119 methods)
// Each method translates to StatsD metric format

func (s *StatsDCollector) IncDelegationsCreated() {
	_ = s.client.Incr("gauth.delegations.created", nil, 1.0)
}

func (s *StatsDCollector) IncDelegationsPartiallyRevoked() {
	_ = s.client.Incr("gauth.delegations.partially_revoked", nil, 1.0)
}

func (s *StatsDCollector) IncDelegationDepthExceeded() {
	_ = s.client.Incr("gauth.delegation.depth_exceeded", nil, 1.0)
}

func (s *StatsDCollector) SetMaxObservedDelegationDepth(depth int) {
	_ = s.client.Gauge("gauth.delegation.max_depth", float64(depth), nil, 1.0)
}

func (s *StatsDCollector) ObserveValidationLatency(d time.Duration) {
	_ = s.client.Timing("gauth.validation.latency", d, nil, 1.0)
}

func (s *StatsDCollector) IncSignaturesIssued() {
	_ = s.client.Incr("gauth.signatures.issued", nil, 1.0)
}

func (s *StatsDCollector) IncSignatureIssueFailures() {
	_ = s.client.Incr("gauth.signatures.issue_failures", nil, 1.0)
}

func (s *StatsDCollector) IncSignatureVerifications() {
	_ = s.client.Incr("gauth.signatures.verifications", nil, 1.0)
}

func (s *StatsDCollector) IncSignatureVerificationFailures() {
	_ = s.client.Incr("gauth.signatures.verification_failures", nil, 1.0)
}

func (s *StatsDCollector) IncAttestationProofIssued() {
	_ = s.client.Incr("gauth.attestation.issued", nil, 1.0)
}

func (s *StatsDCollector) IncAttestationProofIssueFailures() {
	_ = s.client.Incr("gauth.attestation.issue_failures", nil, 1.0)
}

func (s *StatsDCollector) IncAttestationProofVerifications() {
	_ = s.client.Incr("gauth.attestation.verifications", nil, 1.0)
}

func (s *StatsDCollector) IncAttestationProofVerificationFailures() {
	_ = s.client.Incr("gauth.attestation.verification_failures", nil, 1.0)
}

func (s *StatsDCollector) IncAttestationProofDigestMismatch() {
	_ = s.client.Incr("gauth.attestation.digest_mismatch", nil, 1.0)
}

func (s *StatsDCollector) ObserveAttestationProofVerificationLatency(d time.Duration) {
	_ = s.client.Timing("gauth.attestation.verification_latency", d, nil, 1.0)
}

func (s *StatsDCollector) ObserveAttestationProofIssueLatency(d time.Duration) {
	_ = s.client.Timing("gauth.attestation.issue_latency", d, nil, 1.0)
}

func (s *StatsDCollector) IncAttestationProofVerificationFailureReason(reason string) {
	tags := []string{fmt.Sprintf("reason:%s", reason)}
	_ = s.client.Incr("gauth.attestation.verification_failure_reason", tags, 1.0)
}

// Revocation metrics
func (s *StatsDCollector) IncRevocationsIssued() {
	_ = s.client.Incr("gauth.revocations.issued", nil, 1.0)
}

func (s *StatsDCollector) IncRevocationIssueFailures() {
	_ = s.client.Incr("gauth.revocations.issue_failures", nil, 1.0)
}

func (s *StatsDCollector) IncRevocationVerifications() {
	_ = s.client.Incr("gauth.revocations.verifications", nil, 1.0)
}

func (s *StatsDCollector) IncRevocationVerificationFailures() {
	_ = s.client.Incr("gauth.revocations.verification_failures", nil, 1.0)
}

func (s *StatsDCollector) IncRevocationProofsIssued() {
	_ = s.client.Incr("gauth.revocation_proofs.issued", nil, 1.0)
}

func (s *StatsDCollector) IncRevocationProofIssueFailures() {
	_ = s.client.Incr("gauth.revocation_proofs.issue_failures", nil, 1.0)
}

func (s *StatsDCollector) IncRevocationProofVerifications() {
	_ = s.client.Incr("gauth.revocation_proofs.verifications", nil, 1.0)
}

func (s *StatsDCollector) IncRevocationProofVerificationFailures() {
	_ = s.client.Incr("gauth.revocation_proofs.verification_failures", nil, 1.0)
}

func (s *StatsDCollector) ObserveRevocationProofVerificationLatency(d time.Duration) {
	_ = s.client.Timing("gauth.revocation_proofs.verification_latency", d, nil, 1.0)
}

func (s *StatsDCollector) ObserveRevocationProofIssueLatency(d time.Duration) {
	_ = s.client.Timing("gauth.revocation_proofs.issue_latency", d, nil, 1.0)
}

// Anchor metrics
func (s *StatsDCollector) IncAnchorsCreated() {
	_ = s.client.Incr("gauth.anchors.created", nil, 1.0)
}

func (s *StatsDCollector) IncAnchorVerifications() {
	_ = s.client.Incr("gauth.anchors.verifications", nil, 1.0)
}

func (s *StatsDCollector) IncAnchorVerificationFailures() {
	_ = s.client.Incr("gauth.anchors.verification_failures", nil, 1.0)
}

func (s *StatsDCollector) ObserveAnchorVerificationLatency(d time.Duration) {
	_ = s.client.Timing("gauth.anchors.verification_latency", d, nil, 1.0)
}

func (s *StatsDCollector) IncExternalAnchorsCreated() {
	_ = s.client.Incr("gauth.external_anchors.created", nil, 1.0)
}

func (s *StatsDCollector) IncExternalAnchorRetries() {
	_ = s.client.Incr("gauth.external_anchors.retries", nil, 1.0)
}

func (s *StatsDCollector) IncAnchorAttempts() {
	_ = s.client.Incr("gauth.anchors.attempts", nil, 1.0)
}

func (s *StatsDCollector) IncAnchorFailures() {
	_ = s.client.Incr("gauth.anchors.failures", nil, 1.0)
}

func (s *StatsDCollector) IncExternalAnchorForcedFailures() {
	_ = s.client.Incr("gauth.external_anchors.forced_failures", nil, 1.0)
}

func (s *StatsDCollector) IncCombinedAnchorEmitted() {
	_ = s.client.Incr("gauth.combined_anchors.emitted", nil, 1.0)
}

// Obligation metrics
func (s *StatsDCollector) IncObligationsExecuted() {
	_ = s.client.Incr("gauth.obligations.executed", nil, 1.0)
}

func (s *StatsDCollector) IncObligationsFailed() {
	_ = s.client.Incr("gauth.obligations.failed", nil, 1.0)
}

func (s *StatsDCollector) ObserveObligationLatency(d time.Duration) {
	_ = s.client.Timing("gauth.obligations.latency", d, nil, 1.0)
}

func (s *StatsDCollector) IncMandatoryObligationFailures() {
	_ = s.client.Incr("gauth.obligations.mandatory_failures", nil, 1.0)
}

// Replay/cache metrics
func (s *StatsDCollector) IncReplayCacheHits() {
	_ = s.client.Incr("gauth.replay_cache.hits", nil, 1.0)
}

func (s *StatsDCollector) IncReplayCacheMisses() {
	_ = s.client.Incr("gauth.replay_cache.misses", nil, 1.0)
}

func (s *StatsDCollector) IncReplayDetected() {
	_ = s.client.Incr("gauth.replay.detected", nil, 1.0)
}

func (s *StatsDCollector) SetReplayCacheSize(size int) {
	_ = s.client.Gauge("gauth.replay_cache.size", float64(size), nil, 1.0)
}

func (s *StatsDCollector) IncReplayStoreWrites() {
	_ = s.client.Incr("gauth.replay_store.writes", nil, 1.0)
}

func (s *StatsDCollector) IncReplayStoreWriteFailures() {
	_ = s.client.Incr("gauth.replay_store.write_failures", nil, 1.0)
}

func (s *StatsDCollector) ObserveReplayStoreWriteLatency(d time.Duration) {
	_ = s.client.Timing("gauth.replay_store.write_latency", d, nil, 1.0)
}

// Multi-signature metrics
func (s *StatsDCollector) IncMultiSignatureVerifications() {
	_ = s.client.Incr("gauth.multisig.verifications", nil, 1.0)
}

func (s *StatsDCollector) IncMultiSignatureVerificationFailures() {
	_ = s.client.Incr("gauth.multisig.verification_failures", nil, 1.0)
}

func (s *StatsDCollector) IncMultiSignatureStructuralFailures() {
	_ = s.client.Incr("gauth.multisig.structural_failures", nil, 1.0)
}

func (s *StatsDCollector) IncMultiSignatureDigestFailures() {
	_ = s.client.Incr("gauth.multisig.digest_failures", nil, 1.0)
}

func (s *StatsDCollector) IncMultiSignaturePublicKeyMissing() {
	_ = s.client.Incr("gauth.multisig.pubkey_missing", nil, 1.0)
}

func (s *StatsDCollector) IncMultiSignatureInvalidSignatureFailures() {
	_ = s.client.Incr("gauth.multisig.invalid_signature", nil, 1.0)
}

func (s *StatsDCollector) IncMultiSignatureThresholdFailures() {
	_ = s.client.Incr("gauth.multisig.threshold_failures", nil, 1.0)
}

func (s *StatsDCollector) IncMultiSignatureWeightFailures() {
	_ = s.client.Incr("gauth.multisig.weight_failures", nil, 1.0)
}

func (s *StatsDCollector) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	_ = s.client.Timing("gauth.multisig.verification_latency", d, nil, 1.0)
}

func (s *StatsDCollector) ObserveMultiSignatureBatchSize(size int) {
	_ = s.client.Histogram("gauth.multisig.batch_size", float64(size), nil, 1.0)
}

func (s *StatsDCollector) ObserveMultiSignatureAggregateLatency(d time.Duration) {
	_ = s.client.Timing("gauth.multisig.aggregate_latency", d, nil, 1.0)
}

// Violation metrics
func (s *StatsDCollector) IncViolation(cat interface{}) {
	tags := []string{fmt.Sprintf("category:%v", cat)}
	_ = s.client.Incr("gauth.violations", tags, 1.0)
}

func (s *StatsDCollector) IncScopeViolations() {
	_ = s.client.Incr("gauth.violations.scope", nil, 1.0)
}

func (s *StatsDCollector) IncRestrictionViolations() {
	_ = s.client.Incr("gauth.violations.restriction", nil, 1.0)
}

func (s *StatsDCollector) IncUnauthorized() {
	_ = s.client.Incr("gauth.violations.unauthorized", nil, 1.0)
}

func (s *StatsDCollector) IncExpired() {
	_ = s.client.Incr("gauth.violations.expired", nil, 1.0)
}

func (s *StatsDCollector) IncRevoked() {
	_ = s.client.Incr("gauth.violations.revoked", nil, 1.0)
}

// Stub implementations for remaining methods (to satisfy interface)
// In production, implement all 119 methods following the same pattern

func (s *StatsDCollector) IncDelegationStatusTransitions()                            {}
func (s *StatsDCollector) IncJurisdictionPolicyEvaluations()                          {}
func (s *StatsDCollector) IncJurisdictionPolicyViolations()                           {}
func (s *StatsDCollector) ObserveJurisdictionPolicyEvaluationLatency(d time.Duration) {}
func (s *StatsDCollector) IncAICapabilityChecks()                                     {}
func (s *StatsDCollector) IncAICapabilityDenied()                                     {}
func (s *StatsDCollector) IncAIHighRiskActionDenied()                                 {}
func (s *StatsDCollector) ObserveAICapabilityEvaluationLatency(d time.Duration)       {}
func (s *StatsDCollector) IncModelLimitChecks()                                       {}
func (s *StatsDCollector) IncModelContextLimitExceeded()                              {}
func (s *StatsDCollector) IncModelRateLimitExceeded()                                 {}
func (s *StatsDCollector) IncModelUnknown()                                           {}
func (s *StatsDCollector) IncModelLimitSurge()                                        {}
func (s *StatsDCollector) IncModelUserInputLimitExceeded()                            {}
func (s *StatsDCollector) IncModelUserOutputLimitExceeded()                           {}
func (s *StatsDCollector) IncModelUserRateLimitExceeded()                             {}
func (s *StatsDCollector) IncKeyRotationsInitiated()                                  {}
func (s *StatsDCollector) IncKeyRotationFailures()                                    {}
func (s *StatsDCollector) ObserveKeyRotationLatency(d time.Duration)                  {}
func (s *StatsDCollector) SetActiveKeySetSize(size int)                               {}
func (s *StatsDCollector) IncCapabilityComputations()                                 {}
func (s *StatsDCollector) IncCapabilityComputationFailures()                          {}
func (s *StatsDCollector) ObserveCapabilityComputationLatency(d time.Duration)        {}
func (s *StatsDCollector) ObserveCapabilityDiffLatency(d time.Duration)               {}
func (s *StatsDCollector) IncRevocationWorkflowInitiated()                            {}
func (s *StatsDCollector) IncRevocationWorkflowInitiationFailures()                   {}
func (s *StatsDCollector) IncRevocationWorkflowApprovals()                            {}
func (s *StatsDCollector) IncRevocationWorkflowApprovalFailures()                     {}
func (s *StatsDCollector) IncRevocationWorkflowQuorumSatisfied()                      {}
func (s *StatsDCollector) IncRevocationWorkflowCanceled()                             {}
func (s *StatsDCollector) IncRevocationWorkflowCancellationFailures()                 {}
func (s *StatsDCollector) IncRevocationWorkflowUnauthorized()                         {}
func (s *StatsDCollector) IncEvidenceAttachment()                                     {}
func (s *StatsDCollector) IncEvidenceAttachmentFailures()                             {}
func (s *StatsDCollector) SetWorkflowPendingApprovals(poaID string, count int)        {}
func (s *StatsDCollector) SetWorkflowQuorumProgress(poaID string, ratio float64)      {}
func (s *StatsDCollector) SetEvidenceHashesPerPOA(poaID string, count int)            {}
func (s *StatsDCollector) IncDelegationGraphExports()                                 {}
func (s *StatsDCollector) SetDelegationGraphNodeCount(count int)                      {}
func (s *StatsDCollector) IncCascadeRevocationTriggered()                             {}
func (s *StatsDCollector) IncCascadeDescendantsProcessed()                            {}
func (s *StatsDCollector) ObserveCascadeProcessingLatency(d time.Duration)            {}
func (s *StatsDCollector) IncCascadeDepthLimitReached()                               {}
func (s *StatsDCollector) IncCascadeBatchProcessed()                                  {}
func (s *StatsDCollector) SetCascadeMaxDepthReached(depth int)                        {}
func (s *StatsDCollector) IncCascadeProcessingErrors()                                {}
func (s *StatsDCollector) IncNotarizationAttempts()                                   {}
func (s *StatsDCollector) IncNotarizationSuccesses()                                  {}
func (s *StatsDCollector) IncNotarizationFailures()                                   {}
func (s *StatsDCollector) ObserveNotarizationLatency(d time.Duration)                 {}
func (s *StatsDCollector) IncNotarizationBackendFailures(backend string)              {}
func (s *StatsDCollector) SetNotarizationPendingCount(count int)                      {}
func (s *StatsDCollector) IncSemanticPoAValidations()                                 {}
func (s *StatsDCollector) IncSemanticPoAViolations()                                  {}
func (s *StatsDCollector) ObserveSemanticValidationLatency(d time.Duration)           {}
func (s *StatsDCollector) IncConflictDetections()                                     {}
func (s *StatsDCollector) IncConflictResolutions()                                    {}
func (s *StatsDCollector) ObserveConflictResolutionLatency(d time.Duration)           {}
func (s *StatsDCollector) IncABACPolicyEvaluations()                                  {}
func (s *StatsDCollector) IncABACPolicyDenials()                                      {}
func (s *StatsDCollector) ObserveABACEvaluationLatency(d time.Duration)               {}
func (s *StatsDCollector) IncPatternMatchAttempts()                                   {}
func (s *StatsDCollector) IncPatternMatchSuccesses()                                  {}
func (s *StatsDCollector) ObservePatternMatchLatency(d time.Duration)                 {}
func (s *StatsDCollector) IncUTF8ValidationFailures()                                 {}
func (s *StatsDCollector) IncControlCharFiltered()                                    {}
func (s *StatsDCollector) ObserveInputSanitizationLatency(d time.Duration)            {}
func (s *StatsDCollector) IncAdviceEmissions()                                        {}
func (s *StatsDCollector) IncAdviceEmissionFailures()                                 {}
func (s *StatsDCollector) ObserveAdviceEmissionLatency(d time.Duration)               {}
func (s *StatsDCollector) IncComplianceAttestationStored()                            {}
func (s *StatsDCollector) IncComplianceAttestationStoreFailures()                     {}
func (s *StatsDCollector) IncComplianceAttestationQueries()                           {}
func (s *StatsDCollector) ObserveComplianceAttestationQueryLatency(d time.Duration)   {}
func (s *StatsDCollector) IncSnapshotCreated()                                        {}
func (s *StatsDCollector) IncSnapshotRestored()                                       {}
func (s *StatsDCollector) IncSnapshotFailures()                                       {}
func (s *StatsDCollector) ObserveSnapshotLatency(d time.Duration)                     {}
func (s *StatsDCollector) IncThreatDetected(threatID string)                          {}
func (s *StatsDCollector) IncMitigationApplied(mitigationID string)                   {}
func (s *StatsDCollector) SetResidualRiskScore(riskID string, score float64)          {}
func (s *StatsDCollector) ObserveThreatDetectionLatency(d time.Duration)              {}
func (s *StatsDCollector) IncArbitrationInitiated()                                   {}
func (s *StatsDCollector) IncArbitrationResolved()                                    {}
func (s *StatsDCollector) IncArbitrationFailed()                                      {}
func (s *StatsDCollector) ObserveArbitrationLatency(d time.Duration)                  {}
func (s *StatsDCollector) IncDistributedTraceStarted()                                {}
func (s *StatsDCollector) IncDistributedTraceCompleted()                              {}
func (s *StatsDCollector) IncDistributedTraceFailed()                                 {}
func (s *StatsDCollector) ObserveDistributedTraceLatency(d time.Duration)             {}
func (s *StatsDCollector) SetDistributedTraceActiveSpans(count int)                   {}

func (s *StatsDCollector) ObserveExternalAnchorInterval(seconds float64) {
	_ = s.client.Histogram("gauth.external_anchor.interval", seconds, nil, 1.0)
}

func (s *StatsDCollector) HygieneSnapshot() map[string]uint64 {
	return nil
}
