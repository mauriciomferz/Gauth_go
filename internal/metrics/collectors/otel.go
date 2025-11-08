// Copyright (c) 2025 GAuth. All rights reserved.

// Package collectors provides MetricsCollector implementations for various backends.
package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OpenTelemetryCollector exports metrics using OpenTelemetry.
//
// This collector uses OpenTelemetry's metrics SDK to export metrics to
// various backends (Prometheus, OTLP, etc.) via configured exporters.
type OpenTelemetryCollector struct {
	meter    metric.Meter
	counters map[string]metric.Int64Counter
	gauges   map[string]metric.Int64ObservableGauge
	histos   map[string]metric.Float64Histogram
	metadata metrics.CollectorMetadata
	ctx      context.Context
}

// NewOpenTelemetryCollector creates an OpenTelemetry metrics collector.
//
// Parameters:
//   - id: Unique identifier
//   - meter: Configured OpenTelemetry meter (from MeterProvider)
//   - ctx: Context for meter operations
//   - description: Human-readable description
func NewOpenTelemetryCollector(id string, meter metric.Meter, ctx context.Context, description string) *OpenTelemetryCollector {
	return &OpenTelemetryCollector{
		meter:    meter,
		counters: make(map[string]metric.Int64Counter),
		gauges:   make(map[string]metric.Int64ObservableGauge),
		histos:   make(map[string]metric.Float64Histogram),
		ctx:      ctx,
		metadata: metrics.CollectorMetadata{
			ID:           id,
			Type:         metrics.CollectorTypeOpenTelemetry,
			Description:  description,
			RegisteredAt: time.Now(),
			Version:      "1.0.0",
		},
	}
}

// Metadata returns collector metadata.
func (o *OpenTelemetryCollector) Metadata() metrics.CollectorMetadata {
	return o.metadata
}

// Flush forces metrics export (implementation depends on exporter configuration).
func (o *OpenTelemetryCollector) Flush() error {
	// OpenTelemetry SDK handles batching/flushing via MeterProvider
	// This is typically a no-op unless using a custom exporter
	return nil
}

// Close cleanly shuts down the OpenTelemetry resources.
func (o *OpenTelemetryCollector) Close() error {
	// Cleanup is typically handled by MeterProvider.Shutdown()
	return nil
}

// Health checks OpenTelemetry meter availability.
func (o *OpenTelemetryCollector) Health() error {
	// Simple health check: try to get/create a test counter
	if _, err := o.getOrCreateCounter("gauth.health.check"); err != nil {
		return fmt.Errorf("otel health check failed: %w", err)
	}
	return nil
}

// Helper methods for metric instrument creation

func (o *OpenTelemetryCollector) getOrCreateCounter(name string) (metric.Int64Counter, error) {
	if counter, exists := o.counters[name]; exists {
		return counter, nil
	}

	counter, err := o.meter.Int64Counter(
		name,
		metric.WithDescription(fmt.Sprintf("GAuth counter: %s", name)),
	)
	if err != nil {
		return nil, err
	}

	o.counters[name] = counter
	return counter, nil
}

func (o *OpenTelemetryCollector) getOrCreateHistogram(name string) (metric.Float64Histogram, error) {
	if histo, exists := o.histos[name]; exists {
		return histo, nil
	}

	histo, err := o.meter.Float64Histogram(
		name,
		metric.WithDescription(fmt.Sprintf("GAuth histogram: %s", name)),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	o.histos[name] = histo
	return histo, nil
}

func (o *OpenTelemetryCollector) incCounter(name string, attrs ...attribute.KeyValue) {
	counter, err := o.getOrCreateCounter(name)
	if err != nil {
		return // Silent failure to avoid breaking metrics pipeline
	}
	counter.Add(o.ctx, 1, metric.WithAttributes(attrs...))
}

func (o *OpenTelemetryCollector) recordDuration(name string, d time.Duration, attrs ...attribute.KeyValue) {
	histo, err := o.getOrCreateHistogram(name)
	if err != nil {
		return
	}
	histo.Record(o.ctx, float64(d.Milliseconds()), metric.WithAttributes(attrs...))
}

// Metrics interface implementation (119 methods)

func (o *OpenTelemetryCollector) IncDelegationsCreated() {
	o.incCounter("gauth.delegations.created")
}

func (o *OpenTelemetryCollector) IncDelegationsPartiallyRevoked() {
	o.incCounter("gauth.delegations.partially_revoked")
}

func (o *OpenTelemetryCollector) IncDelegationDepthExceeded() {
	o.incCounter("gauth.delegation.depth_exceeded")
}

func (o *OpenTelemetryCollector) SetMaxObservedDelegationDepth(depth int) {
	// For gauges in OTel, use ObservableGauge with callbacks
	// Simplified: store in counter for now (production would use proper gauge)
	counter, _ := o.getOrCreateCounter("gauth.delegation.max_depth")
	if counter != nil {
		counter.Add(o.ctx, int64(depth))
	}
}

func (o *OpenTelemetryCollector) ObserveValidationLatency(d time.Duration) {
	o.recordDuration("gauth.validation.latency", d)
}

func (o *OpenTelemetryCollector) IncSignaturesIssued() {
	o.incCounter("gauth.signatures.issued")
}

func (o *OpenTelemetryCollector) IncSignatureIssueFailures() {
	o.incCounter("gauth.signatures.issue_failures")
}

func (o *OpenTelemetryCollector) IncSignatureVerifications() {
	o.incCounter("gauth.signatures.verifications")
}

func (o *OpenTelemetryCollector) IncSignatureVerificationFailures() {
	o.incCounter("gauth.signatures.verification_failures")
}

func (o *OpenTelemetryCollector) IncAttestationProofIssued() {
	o.incCounter("gauth.attestation.issued")
}

func (o *OpenTelemetryCollector) IncAttestationProofIssueFailures() {
	o.incCounter("gauth.attestation.issue_failures")
}

func (o *OpenTelemetryCollector) IncAttestationProofVerifications() {
	o.incCounter("gauth.attestation.verifications")
}

func (o *OpenTelemetryCollector) IncAttestationProofVerificationFailures() {
	o.incCounter("gauth.attestation.verification_failures")
}

func (o *OpenTelemetryCollector) IncAttestationProofDigestMismatch() {
	o.incCounter("gauth.attestation.digest_mismatch")
}

func (o *OpenTelemetryCollector) ObserveAttestationProofVerificationLatency(d time.Duration) {
	o.recordDuration("gauth.attestation.verification_latency", d)
}

func (o *OpenTelemetryCollector) ObserveAttestationProofIssueLatency(d time.Duration) {
	o.recordDuration("gauth.attestation.issue_latency", d)
}

func (o *OpenTelemetryCollector) IncAttestationProofVerificationFailureReason(reason string) {
	o.incCounter("gauth.attestation.verification_failure_reason",
		attribute.String("reason", reason))
}

// Revocation metrics
func (o *OpenTelemetryCollector) IncRevocationsIssued() {
	o.incCounter("gauth.revocations.issued")
}

func (o *OpenTelemetryCollector) IncRevocationIssueFailures() {
	o.incCounter("gauth.revocations.issue_failures")
}

func (o *OpenTelemetryCollector) IncRevocationVerifications() {
	o.incCounter("gauth.revocations.verifications")
}

func (o *OpenTelemetryCollector) IncRevocationVerificationFailures() {
	o.incCounter("gauth.revocations.verification_failures")
}

func (o *OpenTelemetryCollector) IncRevocationProofsIssued() {
	o.incCounter("gauth.revocation_proofs.issued")
}

func (o *OpenTelemetryCollector) IncRevocationProofIssueFailures() {
	o.incCounter("gauth.revocation_proofs.issue_failures")
}

func (o *OpenTelemetryCollector) IncRevocationProofVerifications() {
	o.incCounter("gauth.revocation_proofs.verifications")
}

func (o *OpenTelemetryCollector) IncRevocationProofVerificationFailures() {
	o.incCounter("gauth.revocation_proofs.verification_failures")
}

func (o *OpenTelemetryCollector) ObserveRevocationProofVerificationLatency(d time.Duration) {
	o.recordDuration("gauth.revocation_proofs.verification_latency", d)
}

func (o *OpenTelemetryCollector) ObserveRevocationProofIssueLatency(d time.Duration) {
	o.recordDuration("gauth.revocation_proofs.issue_latency", d)
}

// Anchor metrics
func (o *OpenTelemetryCollector) IncAnchorsCreated() {
	o.incCounter("gauth.anchors.created")
}

func (o *OpenTelemetryCollector) IncAnchorVerifications() {
	o.incCounter("gauth.anchors.verifications")
}

func (o *OpenTelemetryCollector) IncAnchorVerificationFailures() {
	o.incCounter("gauth.anchors.verification_failures")
}

func (o *OpenTelemetryCollector) ObserveAnchorVerificationLatency(d time.Duration) {
	o.recordDuration("gauth.anchors.verification_latency", d)
}

func (o *OpenTelemetryCollector) IncExternalAnchorsCreated() {
	o.incCounter("gauth.external_anchors.created")
}

func (o *OpenTelemetryCollector) IncExternalAnchorRetries() {
	o.incCounter("gauth.external_anchors.retries")
}

func (o *OpenTelemetryCollector) IncAnchorAttempts() {
	o.incCounter("gauth.anchors.attempts")
}

func (o *OpenTelemetryCollector) IncAnchorFailures() {
	o.incCounter("gauth.anchors.failures")
}

func (o *OpenTelemetryCollector) IncExternalAnchorForcedFailures() {
	o.incCounter("gauth.external_anchors.forced_failures")
}

func (o *OpenTelemetryCollector) IncCombinedAnchorEmitted() {
	o.incCounter("gauth.combined_anchors.emitted")
}

// Obligation metrics
func (o *OpenTelemetryCollector) IncObligationsExecuted() {
	o.incCounter("gauth.obligations.executed")
}

func (o *OpenTelemetryCollector) IncObligationsFailed() {
	o.incCounter("gauth.obligations.failed")
}

func (o *OpenTelemetryCollector) ObserveObligationLatency(d time.Duration) {
	o.recordDuration("gauth.obligations.latency", d)
}

func (o *OpenTelemetryCollector) IncMandatoryObligationFailures() {
	o.incCounter("gauth.obligations.mandatory_failures")
}

// Replay/cache metrics
func (o *OpenTelemetryCollector) IncReplayCacheHits() {
	o.incCounter("gauth.replay_cache.hits")
}

func (o *OpenTelemetryCollector) IncReplayCacheMisses() {
	o.incCounter("gauth.replay_cache.misses")
}

func (o *OpenTelemetryCollector) IncReplayDetected() {
	o.incCounter("gauth.replay.detected")
}

func (o *OpenTelemetryCollector) SetReplayCacheSize(size int) {
	counter, _ := o.getOrCreateCounter("gauth.replay_cache.size")
	if counter != nil {
		counter.Add(o.ctx, int64(size))
	}
}

func (o *OpenTelemetryCollector) IncReplayStoreWrites() {
	o.incCounter("gauth.replay_store.writes")
}

func (o *OpenTelemetryCollector) IncReplayStoreWriteFailures() {
	o.incCounter("gauth.replay_store.write_failures")
}

func (o *OpenTelemetryCollector) ObserveReplayStoreWriteLatency(d time.Duration) {
	o.recordDuration("gauth.replay_store.write_latency", d)
}

// Multi-signature metrics
func (o *OpenTelemetryCollector) IncMultiSignatureVerifications() {
	o.incCounter("gauth.multisig.verifications")
}

func (o *OpenTelemetryCollector) IncMultiSignatureVerificationFailures() {
	o.incCounter("gauth.multisig.verification_failures")
}

func (o *OpenTelemetryCollector) IncMultiSignatureStructuralFailures() {
	o.incCounter("gauth.multisig.structural_failures")
}

func (o *OpenTelemetryCollector) IncMultiSignatureDigestFailures() {
	o.incCounter("gauth.multisig.digest_failures")
}

func (o *OpenTelemetryCollector) IncMultiSignaturePublicKeyMissing() {
	o.incCounter("gauth.multisig.pubkey_missing")
}

func (o *OpenTelemetryCollector) IncMultiSignatureInvalidSignatureFailures() {
	o.incCounter("gauth.multisig.invalid_signature")
}

func (o *OpenTelemetryCollector) IncMultiSignatureThresholdFailures() {
	o.incCounter("gauth.multisig.threshold_failures")
}

func (o *OpenTelemetryCollector) IncMultiSignatureWeightFailures() {
	o.incCounter("gauth.multisig.weight_failures")
}

func (o *OpenTelemetryCollector) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	o.recordDuration("gauth.multisig.verification_latency", d)
}

func (o *OpenTelemetryCollector) ObserveMultiSignatureBatchSize(size int) {
	histo, _ := o.getOrCreateHistogram("gauth.multisig.batch_size")
	if histo != nil {
		histo.Record(o.ctx, float64(size))
	}
}

func (o *OpenTelemetryCollector) ObserveMultiSignatureAggregateLatency(d time.Duration) {
	o.recordDuration("gauth.multisig.aggregate_latency", d)
}

// Violation metrics
func (o *OpenTelemetryCollector) IncViolation(cat interface{}) {
	o.incCounter("gauth.violations",
		attribute.String("category", fmt.Sprintf("%v", cat)))
}

func (o *OpenTelemetryCollector) IncScopeViolations() {
	o.incCounter("gauth.violations.scope")
}

func (o *OpenTelemetryCollector) IncRestrictionViolations() {
	o.incCounter("gauth.violations.restriction")
}

func (o *OpenTelemetryCollector) IncUnauthorized() {
	o.incCounter("gauth.violations.unauthorized")
}

func (o *OpenTelemetryCollector) IncExpired() {
	o.incCounter("gauth.violations.expired")
}

func (o *OpenTelemetryCollector) IncRevoked() {
	o.incCounter("gauth.violations.revoked")
}

// Stub implementations for remaining methods
// Production implementation would include all 119 methods

func (o *OpenTelemetryCollector) IncDelegationStatusTransitions()                            {}
func (o *OpenTelemetryCollector) IncJurisdictionPolicyEvaluations()                          {}
func (o *OpenTelemetryCollector) IncJurisdictionPolicyViolations()                           {}
func (o *OpenTelemetryCollector) ObserveJurisdictionPolicyEvaluationLatency(d time.Duration) {}
func (o *OpenTelemetryCollector) IncAICapabilityChecks()                                     {}
func (o *OpenTelemetryCollector) IncAICapabilityDenied()                                     {}
func (o *OpenTelemetryCollector) IncAIHighRiskActionDenied()                                 {}
func (o *OpenTelemetryCollector) ObserveAICapabilityEvaluationLatency(d time.Duration)       {}
func (o *OpenTelemetryCollector) IncModelLimitChecks()                                       {}
func (o *OpenTelemetryCollector) IncModelContextLimitExceeded()                              {}
func (o *OpenTelemetryCollector) IncModelRateLimitExceeded()                                 {}
func (o *OpenTelemetryCollector) IncModelUnknown()                                           {}
func (o *OpenTelemetryCollector) IncModelLimitSurge()                                        {}
func (o *OpenTelemetryCollector) IncModelUserInputLimitExceeded()                            {}
func (o *OpenTelemetryCollector) IncModelUserOutputLimitExceeded()                           {}
func (o *OpenTelemetryCollector) IncModelUserRateLimitExceeded()                             {}
func (o *OpenTelemetryCollector) IncKeyRotationsInitiated()                                  {}
func (o *OpenTelemetryCollector) IncKeyRotationFailures()                                    {}
func (o *OpenTelemetryCollector) ObserveKeyRotationLatency(d time.Duration)                  {}
func (o *OpenTelemetryCollector) SetActiveKeySetSize(size int)                               {}
func (o *OpenTelemetryCollector) IncCapabilityComputations()                                 {}
func (o *OpenTelemetryCollector) IncCapabilityComputationFailures()                          {}
func (o *OpenTelemetryCollector) ObserveCapabilityComputationLatency(d time.Duration)        {}
func (o *OpenTelemetryCollector) ObserveCapabilityDiffLatency(d time.Duration)               {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowInitiated()                            {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowInitiationFailures()                   {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowApprovals()                            {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowApprovalFailures()                     {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowQuorumSatisfied()                      {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowCanceled()                             {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowCancellationFailures()                 {}
func (o *OpenTelemetryCollector) IncRevocationWorkflowUnauthorized()                         {}
func (o *OpenTelemetryCollector) IncEvidenceAttachment()                                     {}
func (o *OpenTelemetryCollector) IncEvidenceAttachmentFailures()                             {}
func (o *OpenTelemetryCollector) SetWorkflowPendingApprovals(poaID string, count int)        {}
func (o *OpenTelemetryCollector) SetWorkflowQuorumProgress(poaID string, ratio float64)      {}
func (o *OpenTelemetryCollector) SetEvidenceHashesPerPOA(poaID string, count int)            {}
func (o *OpenTelemetryCollector) IncDelegationGraphExports()                                 {}
func (o *OpenTelemetryCollector) SetDelegationGraphNodeCount(count int)                      {}
func (o *OpenTelemetryCollector) IncCascadeRevocationTriggered()                             {}
func (o *OpenTelemetryCollector) IncCascadeDescendantsProcessed()                            {}
func (o *OpenTelemetryCollector) ObserveCascadeProcessingLatency(d time.Duration)            {}
func (o *OpenTelemetryCollector) IncCascadeDepthLimitReached()                               {}
func (o *OpenTelemetryCollector) IncCascadeBatchProcessed()                                  {}
func (o *OpenTelemetryCollector) SetCascadeMaxDepthReached(depth int)                        {}
func (o *OpenTelemetryCollector) IncCascadeProcessingErrors()                                {}
func (o *OpenTelemetryCollector) IncNotarizationAttempts()                                   {}
func (o *OpenTelemetryCollector) IncNotarizationSuccesses()                                  {}
func (o *OpenTelemetryCollector) IncNotarizationFailures()                                   {}
func (o *OpenTelemetryCollector) ObserveNotarizationLatency(d time.Duration)                 {}
func (o *OpenTelemetryCollector) IncNotarizationBackendFailures(backend string)              {}
func (o *OpenTelemetryCollector) SetNotarizationPendingCount(count int)                      {}
func (o *OpenTelemetryCollector) IncSemanticPoAValidations()                                 {}
func (o *OpenTelemetryCollector) IncSemanticPoAViolations()                                  {}
func (o *OpenTelemetryCollector) ObserveSemanticValidationLatency(d time.Duration)           {}
func (o *OpenTelemetryCollector) IncConflictDetections()                                     {}
func (o *OpenTelemetryCollector) IncConflictResolutions()                                    {}
func (o *OpenTelemetryCollector) ObserveConflictResolutionLatency(d time.Duration)           {}
func (o *OpenTelemetryCollector) IncABACPolicyEvaluations()                                  {}
func (o *OpenTelemetryCollector) IncABACPolicyDenials()                                      {}
func (o *OpenTelemetryCollector) ObserveABACEvaluationLatency(d time.Duration)               {}
func (o *OpenTelemetryCollector) IncPatternMatchAttempts()                                   {}
func (o *OpenTelemetryCollector) IncPatternMatchSuccesses()                                  {}
func (o *OpenTelemetryCollector) ObservePatternMatchLatency(d time.Duration)                 {}
func (o *OpenTelemetryCollector) IncUTF8ValidationFailures()                                 {}
func (o *OpenTelemetryCollector) IncControlCharFiltered()                                    {}
func (o *OpenTelemetryCollector) ObserveInputSanitizationLatency(d time.Duration)            {}
func (o *OpenTelemetryCollector) IncAdviceEmissions()                                        {}
func (o *OpenTelemetryCollector) IncAdviceEmissionFailures()                                 {}
func (o *OpenTelemetryCollector) ObserveAdviceEmissionLatency(d time.Duration)               {}
func (o *OpenTelemetryCollector) IncComplianceAttestationStored()                            {}
func (o *OpenTelemetryCollector) IncComplianceAttestationStoreFailures()                     {}
func (o *OpenTelemetryCollector) IncComplianceAttestationQueries()                           {}
func (o *OpenTelemetryCollector) ObserveComplianceAttestationQueryLatency(d time.Duration)   {}
func (o *OpenTelemetryCollector) IncSnapshotCreated()                                        {}
func (o *OpenTelemetryCollector) IncSnapshotRestored()                                       {}
func (o *OpenTelemetryCollector) IncSnapshotFailures()                                       {}
func (o *OpenTelemetryCollector) ObserveSnapshotLatency(d time.Duration)                     {}
func (o *OpenTelemetryCollector) IncThreatDetected(threatID string)                          {}
func (o *OpenTelemetryCollector) IncMitigationApplied(mitigationID string)                   {}
func (o *OpenTelemetryCollector) SetResidualRiskScore(riskID string, score float64)          {}
func (o *OpenTelemetryCollector) ObserveThreatDetectionLatency(d time.Duration)              {}
func (o *OpenTelemetryCollector) IncArbitrationInitiated()                                   {}
func (o *OpenTelemetryCollector) IncArbitrationResolved()                                    {}
func (o *OpenTelemetryCollector) IncArbitrationFailed()                                      {}
func (o *OpenTelemetryCollector) ObserveArbitrationLatency(d time.Duration)                  {}
func (o *OpenTelemetryCollector) IncDistributedTraceStarted()                                {}
func (o *OpenTelemetryCollector) IncDistributedTraceCompleted()                              {}
func (o *OpenTelemetryCollector) IncDistributedTraceFailed()                                 {}
func (o *OpenTelemetryCollector) ObserveDistributedTraceLatency(d time.Duration)             {}
func (o *OpenTelemetryCollector) SetDistributedTraceActiveSpans(count int)                   {}
