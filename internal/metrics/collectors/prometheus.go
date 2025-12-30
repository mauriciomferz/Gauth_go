// Copyright (c) 2025 AgentAuth. All rights reserved.

// Package collectors provides example MetricsCollector implementations.
package collectors

import (
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
)

// PrometheusCollector wraps an existing PrometheusMetrics implementation
// to conform to the MetricsCollector interface.
//
// This adapter allows PrometheusMetrics to be used with the CollectorRegistry,
// enabling side-by-side operation with other collector types (StatsD, JSON, etc.).
type PrometheusCollector struct {
	impl     metrics.Metrics
	metadata metrics.CollectorMetadata
}

// NewPrometheusCollector creates a PrometheusCollector wrapping an existing
// Prometheus metrics implementation.
//
// Parameters:
// - id: Unique identifier for this collector instance
// - impl: Existing PrometheusMetrics instance (or any Metrics implementation)
// - description: Human-readable collector description
func NewPrometheusCollector(id string, impl metrics.Metrics, description string) *PrometheusCollector {
	return &PrometheusCollector{
		impl: impl,
		metadata: metrics.CollectorMetadata{
			ID:           id,
			Type:         metrics.CollectorTypePrometheus,
			Description:  description,
			RegisteredAt: time.Now(),
			Version:      "1.0.0",
		},
	}
}

// Metadata returns collector metadata.
func (p *PrometheusCollector) Metadata() metrics.CollectorMetadata {
	return p.metadata
}

// Flush is a no-op for Prometheus (pull model, no buffering).
func (p *PrometheusCollector) Flush() error {
	return nil
}

// Close is a no-op for Prometheus (metrics persist in registry).
func (p *PrometheusCollector) Close() error {
	return nil
}

// Health returns nil (Prometheus collector is always healthy if constructed).
func (p *PrometheusCollector) Health() error {
	return nil
}

// All Metrics interface methods delegate to the wrapped implementation.
// Auto-generated delegation (119 methods).

func (p *PrometheusCollector) IncDelegationsCreated() {
	p.impl.IncDelegationsCreated()
}

func (p *PrometheusCollector) IncDelegationsPartiallyRevoked() {
	p.impl.IncDelegationsPartiallyRevoked()
}

func (p *PrometheusCollector) IncDelegationDepthExceeded() {
	p.impl.IncDelegationDepthExceeded()
}

func (p *PrometheusCollector) SetMaxObservedDelegationDepth(depth int) {
	p.impl.SetMaxObservedDelegationDepth(depth)
}

func (p *PrometheusCollector) ObserveValidationLatency(d time.Duration) {
	p.impl.ObserveValidationLatency(d)
}

func (p *PrometheusCollector) IncSignaturesIssued() {
	p.impl.IncSignaturesIssued()
}

func (p *PrometheusCollector) IncSignatureIssueFailures() {
	p.impl.IncSignatureIssueFailures()
}

func (p *PrometheusCollector) IncSignatureVerifications() {
	p.impl.IncSignatureVerifications()
}

func (p *PrometheusCollector) IncSignatureVerificationFailures() {
	p.impl.IncSignatureVerificationFailures()
}

func (p *PrometheusCollector) IncAttestationProofIssued() {
	p.impl.IncAttestationProofIssued()
}

func (p *PrometheusCollector) IncAttestationProofIssueFailures() {
	p.impl.IncAttestationProofIssueFailures()
}

func (p *PrometheusCollector) IncAttestationProofVerifications() {
	p.impl.IncAttestationProofVerifications()
}

func (p *PrometheusCollector) IncAttestationProofVerificationFailures() {
	p.impl.IncAttestationProofVerificationFailures()
}

func (p *PrometheusCollector) IncAttestationProofDigestMismatch() {
	p.impl.IncAttestationProofDigestMismatch()
}

func (p *PrometheusCollector) ObserveAttestationProofVerificationLatency(d time.Duration) {
	p.impl.ObserveAttestationProofVerificationLatency(d)
}

func (p *PrometheusCollector) ObserveAttestationProofIssueLatency(d time.Duration) {
	p.impl.ObserveAttestationProofIssueLatency(d)
}

func (p *PrometheusCollector) IncAttestationProofVerificationFailureReason(reason string) {
	p.impl.IncAttestationProofVerificationFailureReason(reason)
}

func (p *PrometheusCollector) IncBLSPoPChallengesIssued() {
	p.impl.IncBLSPoPChallengesIssued()
}

func (p *PrometheusCollector) IncBLSPoPVerifications() {
	p.impl.IncBLSPoPVerifications()
}

func (p *PrometheusCollector) IncBLSPoPVerificationFailures() {
	p.impl.IncBLSPoPVerificationFailures()
}

// ... (Remaining 99 methods follow the same pattern)
// For brevity, showing representative sample. Full implementation would include
// all 119 methods from the Metrics interface, delegating to p.impl.
//
// To generate complete implementation:
// $ python3 scripts/generate_collector_wrapper.py prometheus > internal/metrics/collectors/prometheus.go

func (p *PrometheusCollector) IncAttestationProofTrustAnchorMissing() {
	p.impl.IncAttestationProofTrustAnchorMissing()
}

func (p *PrometheusCollector) IncAttestationProofTrustAnchorAlgorithmMismatch() {
	p.impl.IncAttestationProofTrustAnchorAlgorithmMismatch()
}

func (p *PrometheusCollector) IncAttestationProofTrustAnchorKeyMismatch() {
	p.impl.IncAttestationProofTrustAnchorKeyMismatch()
}

func (p *PrometheusCollector) IncRevocationIntegrityFailures() {
	p.impl.IncRevocationIntegrityFailures()
}

func (p *PrometheusCollector) IncSignaturePublicKeyMissing() {
	p.impl.IncSignaturePublicKeyMissing()
}

func (p *PrometheusCollector) IncCryptoSignatureMissing() {
	p.impl.IncCryptoSignatureMissing()
}

func (p *PrometheusCollector) IncEnvelopeV1Issued() {
	p.impl.IncEnvelopeV1Issued()
}

func (p *PrometheusCollector) IncEnvelopeV2Issued() {
	p.impl.IncEnvelopeV2Issued()
}

func (p *PrometheusCollector) IncHierDigestIssued() {
	p.impl.IncHierDigestIssued()
}

func (p *PrometheusCollector) IncHierDigestParentDigestMissing() {
	p.impl.IncHierDigestParentDigestMissing()
}

func (p *PrometheusCollector) IncHierDigestVersionMismatch() {
	p.impl.IncHierDigestVersionMismatch()
}

func (p *PrometheusCollector) SetEnvelopeV2AdoptionRatio(r2 float64) {
	p.impl.SetEnvelopeV2AdoptionRatio(r2)
}

func (p *PrometheusCollector) IncEnvelopeDigestMismatch() {
	p.impl.IncEnvelopeDigestMismatch()
}

func (p *PrometheusCollector) IncEnvelopeDigestMismatchReason(reason string) {
	p.impl.IncEnvelopeDigestMismatchReason(reason)
}

func (p *PrometheusCollector) ObserveEnvelopeIssuanceCadence(seconds float64) {
	p.impl.ObserveEnvelopeIssuanceCadence(seconds)
}

func (p *PrometheusCollector) SetEnvelopeV1SunsetPhase(phase int) {
	p.impl.SetEnvelopeV1SunsetPhase(phase)
}

func (p *PrometheusCollector) SetSunsetPhaseSatisfactionProgress(p2 float64) {
	p.impl.SetSunsetPhaseSatisfactionProgress(p2)
}

func (p *PrometheusCollector) IncEnvelopeRawPOAEmbedded() {
	p.impl.IncEnvelopeRawPOAEmbedded()
}

func (p *PrometheusCollector) IncEnvelopeRawPOATooLarge() {
	p.impl.IncEnvelopeRawPOATooLarge()
}

func (p *PrometheusCollector) IncMultiSignatureVerifications() {
	p.impl.IncMultiSignatureVerifications()
}

func (p *PrometheusCollector) IncMultiSignatureVerificationFailures() {
	p.impl.IncMultiSignatureVerificationFailures()
}

func (p *PrometheusCollector) IncMultiSignatureStructuralFailures() {
	p.impl.IncMultiSignatureStructuralFailures()
}

func (p *PrometheusCollector) IncMultiSignatureDigestFailures() {
	p.impl.IncMultiSignatureDigestFailures()
}

func (p *PrometheusCollector) IncMultiSignaturePublicKeyMissing() {
	p.impl.IncMultiSignaturePublicKeyMissing()
}

func (p *PrometheusCollector) IncMultiSignatureInvalidSignatureFailures() {
	p.impl.IncMultiSignatureInvalidSignatureFailures()
}

func (p *PrometheusCollector) IncMultiSignatureThresholdFailures() {
	p.impl.IncMultiSignatureThresholdFailures()
}

func (p *PrometheusCollector) IncViolation(cat interface{}) {
	p.impl.IncViolation(cat)
}

func (p *PrometheusCollector) IncMultiSignatureWeightFailures() {
	p.impl.IncMultiSignatureWeightFailures()
}

func (p *PrometheusCollector) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	p.impl.ObserveMultiSignatureVerificationLatency(d)
}

func (p *PrometheusCollector) ObserveMultiSignatureBatchSize(size int) {
	p.impl.ObserveMultiSignatureBatchSize(size)
}

func (p *PrometheusCollector) ObserveMultiSignatureAggregateLatency(d time.Duration) {
	p.impl.ObserveMultiSignatureAggregateLatency(d)
}

func (p *PrometheusCollector) IncAnchorAttempts() {
	p.impl.IncAnchorAttempts()
}

func (p *PrometheusCollector) IncCombinedAnchorEmitted() {
	p.impl.IncCombinedAnchorEmitted()
}

func (p *PrometheusCollector) IncCombinedAnchorFailures() {
	p.impl.IncCombinedAnchorFailures()
}

func (p *PrometheusCollector) IncAnchorFailures() {
	p.impl.IncAnchorFailures()
}

func (p *PrometheusCollector) IncExternalAnchorForcedFailures() {
	p.impl.IncExternalAnchorForcedFailures()
}

func (p *PrometheusCollector) IncObligationsExecuted() {
	p.impl.IncObligationsExecuted()
}

func (p *PrometheusCollector) IncObligationsFailed() {
	p.impl.IncObligationsFailed()
}

func (p *PrometheusCollector) ObserveObligationLatency(d time.Duration) {
	p.impl.ObserveObligationLatency(d)
}

func (p *PrometheusCollector) IncMandatoryObligationFailures() {
	p.impl.IncMandatoryObligationFailures()
}

func (p *PrometheusCollector) IncReplayHits() {
	p.impl.IncReplayHits()
}

func (p *PrometheusCollector) IncReplayMisses() {
	p.impl.IncReplayMisses()
}

func (p *PrometheusCollector) IncReplayStoreErrors() {
	p.impl.IncReplayStoreErrors()
}

func (p *PrometheusCollector) ObserveReplayStoreLatency(d time.Duration) {
	p.impl.ObserveReplayStoreLatency(d)
}

func (p *PrometheusCollector) SetReplayWALPending(n int) {
	p.impl.SetReplayWALPending(n)
}

func (p *PrometheusCollector) ObserveReplayWALFlushLatency(d time.Duration) {
	p.impl.ObserveReplayWALFlushLatency(d)
}

func (p *PrometheusCollector) ObserveReplayWALSnapshotDuration(d time.Duration) {
	p.impl.ObserveReplayWALSnapshotDuration(d)
}

func (p *PrometheusCollector) IncCapabilityDiffRequests() {
	p.impl.IncCapabilityDiffRequests()
}

func (p *PrometheusCollector) ObserveCapabilityDiffLatency(d time.Duration) {
	p.impl.ObserveCapabilityDiffLatency(d)
}

func (p *PrometheusCollector) IncCapabilityAnchorEmitted() {
	p.impl.IncCapabilityAnchorEmitted()
}

func (p *PrometheusCollector) IncCapabilityAnchorSkipped() {
	p.impl.IncCapabilityAnchorSkipped()
}

func (p *PrometheusCollector) IncCapabilityRegistryHashChanged() {
	p.impl.IncCapabilityRegistryHashChanged()
}

func (p *PrometheusCollector) SetCapabilityAnchorLastWriteUnix(ts uint64) {
	p.impl.SetCapabilityAnchorLastWriteUnix(ts)
}

func (p *PrometheusCollector) IncCapabilityAnchorAlgorithm(algo string) {
	p.impl.IncCapabilityAnchorAlgorithm(algo)
}

func (p *PrometheusCollector) SetCapabilityAnchorAlgorithmRatio(algo string, ratio float64) {
	p.impl.SetCapabilityAnchorAlgorithmRatio(algo, ratio)
}

func (p *PrometheusCollector) IncCapabilityEnforceAllowed() {
	p.impl.IncCapabilityEnforceAllowed()
}

func (p *PrometheusCollector) IncCapabilityEnforceDenied() {
	p.impl.IncCapabilityEnforceDenied()
}

func (p *PrometheusCollector) IncModelLimitExceeded() {
	p.impl.IncModelLimitExceeded()
}

func (p *PrometheusCollector) IncModelOutputLimitExceeded() {
	p.impl.IncModelOutputLimitExceeded()
}

func (p *PrometheusCollector) IncModelRateLimitExceeded() {
	p.impl.IncModelRateLimitExceeded()
}

func (p *PrometheusCollector) IncModelUnknown() {
	p.impl.IncModelUnknown()
}

func (p *PrometheusCollector) IncModelLimitSurge() {
	p.impl.IncModelLimitSurge()
}

func (p *PrometheusCollector) IncModelUserInputLimitExceeded() {
	p.impl.IncModelUserInputLimitExceeded()
}

func (p *PrometheusCollector) IncModelUserOutputLimitExceeded() {
	p.impl.IncModelUserOutputLimitExceeded()
}

func (p *PrometheusCollector) IncModelUserRateLimitExceeded() {
	p.impl.IncModelUserRateLimitExceeded()
}

func (p *PrometheusCollector) IncScopeViolations() {
	p.impl.IncScopeViolations()
}

func (p *PrometheusCollector) IncRestrictionViolations() {
	p.impl.IncRestrictionViolations()
}

func (p *PrometheusCollector) IncUnauthorized() {
	p.impl.IncUnauthorized()
}

func (p *PrometheusCollector) IncExpired() {
	p.impl.IncExpired()
}

func (p *PrometheusCollector) IncRevoked() {
	p.impl.IncRevoked()
}

func (p *PrometheusCollector) IncDelegationStatusTransitions() {
	p.impl.IncDelegationStatusTransitions()
}

func (p *PrometheusCollector) IncDelegationStatusTransitionFailures() {
	p.impl.IncDelegationStatusTransitionFailures()
}

func (p *PrometheusCollector) IncTokenStatusTransitions() {
	p.impl.IncTokenStatusTransitions()
}

func (p *PrometheusCollector) IncTokenStatusTransitionFailures() {
	p.impl.IncTokenStatusTransitionFailures()
}

func (p *PrometheusCollector) IncRevocationWorkflowInitiated() {
	p.impl.IncRevocationWorkflowInitiated()
}

func (p *PrometheusCollector) IncRevocationWorkflowInitiationFailures() {
	p.impl.IncRevocationWorkflowInitiationFailures()
}

func (p *PrometheusCollector) IncRevocationWorkflowApprovals() {
	p.impl.IncRevocationWorkflowApprovals()
}

func (p *PrometheusCollector) IncRevocationWorkflowApprovalFailures() {
	p.impl.IncRevocationWorkflowApprovalFailures()
}

func (p *PrometheusCollector) IncRevocationWorkflowQuorumSatisfied() {
	p.impl.IncRevocationWorkflowQuorumSatisfied()
}

func (p *PrometheusCollector) IncRevocationWorkflowCanceled() {
	p.impl.IncRevocationWorkflowCanceled()
}

func (p *PrometheusCollector) IncRevocationWorkflowCancellationFailures() {
	p.impl.IncRevocationWorkflowCancellationFailures()
}

func (p *PrometheusCollector) IncRevocationWorkflowUnauthorized() {
	p.impl.IncRevocationWorkflowUnauthorized()
}

func (p *PrometheusCollector) IncEvidenceAttachment() {
	p.impl.IncEvidenceAttachment()
}

func (p *PrometheusCollector) IncEvidenceAttachmentFailures() {
	p.impl.IncEvidenceAttachmentFailures()
}

func (p *PrometheusCollector) SetEvidenceHashesPerPOA(poaID string, n int) {
	p.impl.SetEvidenceHashesPerPOA(poaID, n)
}

func (p *PrometheusCollector) IncDelegationGraphExports() {
	p.impl.IncDelegationGraphExports()
}

func (p *PrometheusCollector) SetDelegationGraphNodeCount(n int) {
	p.impl.SetDelegationGraphNodeCount(n)
}

func (p *PrometheusCollector) RecordDecision(action string, resource string, outcome string, d time.Duration) {
	p.impl.RecordDecision(action, resource, outcome, d)
}

func (p *PrometheusCollector) RecordDecisionWithReason(action string, resource string, outcome string, reason string) {
	p.impl.RecordDecisionWithReason(action, resource, outcome, reason)
}

func (p *PrometheusCollector) RecordLifecycleTransition(entityType string, oldStatus string, newStatus string, outcome string) {
	p.impl.RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome)
}

func (p *PrometheusCollector) ObserveLifecycleTransitionLatency(entityType string, outcome string, d time.Duration) {
	p.impl.ObserveLifecycleTransitionLatency(entityType, outcome, d)
}

func (p *PrometheusCollector) SetLifecycleTransitionLatencyQuantile(entityType string, outcome string, quantile string, value float64) {
	p.impl.SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile, value)
}

func (p *PrometheusCollector) IncCascadeRevocationTriggered() {
	p.impl.IncCascadeRevocationTriggered()
}

func (p *PrometheusCollector) IncCascadeDescendantsProcessed() {
	p.impl.IncCascadeDescendantsProcessed()
}

func (p *PrometheusCollector) ObserveCascadeProcessingLatency(d time.Duration) {
	p.impl.ObserveCascadeProcessingLatency(d)
}

func (p *PrometheusCollector) IncCascadeDepthLimitReached() {
	p.impl.IncCascadeDepthLimitReached()
}

func (p *PrometheusCollector) IncCascadeBatchProcessed() {
	p.impl.IncCascadeBatchProcessed()
}

func (p *PrometheusCollector) SetCascadeMaxDepthReached(depth int) {
	p.impl.SetCascadeMaxDepthReached(depth)
}

func (p *PrometheusCollector) IncCascadeProcessingErrors() {
	p.impl.IncCascadeProcessingErrors()
}
