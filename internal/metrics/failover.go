package metrics

import (
	"log"
	"sync"
	"time"
)

// FailoverMetricsCollector wraps a primary and secondary collector.
// It delegates to the primary by default, but switches to secondary
// if the primary is marked unhealthy.
type FailoverMetricsCollector struct {
	primary   Metrics
	secondary Metrics

	mu             sync.RWMutex
	primaryHealthy bool
}

// NewFailoverMetricsCollector creates a new failover collector.
func NewFailoverMetricsCollector(primary, secondary Metrics) *FailoverMetricsCollector {
	return &FailoverMetricsCollector{
		primary:        primary,
		secondary:      secondary,
		primaryHealthy: true,
	}
}

// NewFailoverWithLogging creates a failover collector with a LoggingMetrics secondary backend.
// This is a convenient default configuration for redundancy (RR-012).
func NewFailoverWithLogging(primary Metrics, logger *log.Logger) *FailoverMetricsCollector {
	secondary := NewLoggingMetrics(logger)
	return NewFailoverMetricsCollector(primary, secondary)
}

// active returns the currently active collector.
func (f *FailoverMetricsCollector) active() Metrics {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.primaryHealthy {
		return f.primary
	}
	return f.secondary
}

// MarkPrimaryUnhealthy forces a switch to the secondary collector.
func (f *FailoverMetricsCollector) MarkPrimaryUnhealthy() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.primaryHealthy = false
}

// MarkPrimaryHealthy switches back to the primary collector.
func (f *FailoverMetricsCollector) MarkPrimaryHealthy() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.primaryHealthy = true
}

// IsPrimaryHealthy returns true if currently using primary.
func (f *FailoverMetricsCollector) IsPrimaryHealthy() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.primaryHealthy
}

// Delegate methods

func (f *FailoverMetricsCollector) IncDelegationsCreated() { f.active().IncDelegationsCreated() }
func (f *FailoverMetricsCollector) IncDelegationsPartiallyRevoked() {
	f.active().IncDelegationsPartiallyRevoked()
}
func (f *FailoverMetricsCollector) IncDelegationDepthExceeded() {
	f.active().IncDelegationDepthExceeded()
}
func (f *FailoverMetricsCollector) SetMaxObservedDelegationDepth(depth int) {
	f.active().SetMaxObservedDelegationDepth(depth)
}
func (f *FailoverMetricsCollector) ObserveValidationLatency(d time.Duration) {
	f.active().ObserveValidationLatency(d)
}
func (f *FailoverMetricsCollector) IncSignaturesIssued() { f.active().IncSignaturesIssued() }
func (f *FailoverMetricsCollector) IncSignatureIssueFailures() {
	f.active().IncSignatureIssueFailures()
}
func (f *FailoverMetricsCollector) IncSignatureVerifications() {
	f.active().IncSignatureVerifications()
}
func (f *FailoverMetricsCollector) IncSignatureVerificationFailures() {
	f.active().IncSignatureVerificationFailures()
}
func (f *FailoverMetricsCollector) IncAttestationProofIssued() {
	f.active().IncAttestationProofIssued()
}
func (f *FailoverMetricsCollector) IncAttestationProofIssueFailures() {
	f.active().IncAttestationProofIssueFailures()
}
func (f *FailoverMetricsCollector) IncAttestationProofVerifications() {
	f.active().IncAttestationProofVerifications()
}
func (f *FailoverMetricsCollector) IncAttestationProofVerificationFailures() {
	f.active().IncAttestationProofVerificationFailures()
}
func (f *FailoverMetricsCollector) IncAttestationProofDigestMismatch() {
	f.active().IncAttestationProofDigestMismatch()
}
func (f *FailoverMetricsCollector) ObserveAttestationProofVerificationLatency(d time.Duration) {
	f.active().ObserveAttestationProofVerificationLatency(d)
}
func (f *FailoverMetricsCollector) ObserveAttestationProofIssueLatency(d time.Duration) {
	f.active().ObserveAttestationProofIssueLatency(d)
}
func (f *FailoverMetricsCollector) IncAttestationProofVerificationFailureReason(reason string) {
	f.active().IncAttestationProofVerificationFailureReason(reason)
}
func (f *FailoverMetricsCollector) IncBLSPoPChallengesIssued() {
	f.active().IncBLSPoPChallengesIssued()
}
func (f *FailoverMetricsCollector) IncBLSPoPVerifications() { f.active().IncBLSPoPVerifications() }
func (f *FailoverMetricsCollector) IncBLSPoPVerificationFailures() {
	f.active().IncBLSPoPVerificationFailures()
}
func (f *FailoverMetricsCollector) IncAttestationProofTrustAnchorMissing() {
	f.active().IncAttestationProofTrustAnchorMissing()
}
func (f *FailoverMetricsCollector) IncAttestationProofTrustAnchorAlgorithmMismatch() {
	f.active().IncAttestationProofTrustAnchorAlgorithmMismatch()
}
func (f *FailoverMetricsCollector) IncAttestationProofTrustAnchorKeyMismatch() {
	f.active().IncAttestationProofTrustAnchorKeyMismatch()
}
func (f *FailoverMetricsCollector) IncRevocationIntegrityFailures() {
	f.active().IncRevocationIntegrityFailures()
}
func (f *FailoverMetricsCollector) IncSignaturePublicKeyMissing() {
	f.active().IncSignaturePublicKeyMissing()
}
func (f *FailoverMetricsCollector) IncCryptoSignatureMissing() {
	f.active().IncCryptoSignatureMissing()
}
func (f *FailoverMetricsCollector) IncEnvelopeV1Issued() { f.active().IncEnvelopeV1Issued() }
func (f *FailoverMetricsCollector) IncEnvelopeV2Issued() { f.active().IncEnvelopeV2Issued() }
func (f *FailoverMetricsCollector) IncHierDigestIssued() { f.active().IncHierDigestIssued() }
func (f *FailoverMetricsCollector) IncHierDigestParentDigestMissing() {
	f.active().IncHierDigestParentDigestMissing()
}
func (f *FailoverMetricsCollector) IncHierDigestVersionMismatch() {
	f.active().IncHierDigestVersionMismatch()
}
func (f *FailoverMetricsCollector) SetEnvelopeV2AdoptionRatio(r float64) {
	f.active().SetEnvelopeV2AdoptionRatio(r)
}
func (f *FailoverMetricsCollector) IncEnvelopeDigestMismatch() {
	f.active().IncEnvelopeDigestMismatch()
}
func (f *FailoverMetricsCollector) IncEnvelopeDigestMismatchReason(reason string) {
	f.active().IncEnvelopeDigestMismatchReason(reason)
}
func (f *FailoverMetricsCollector) ObserveEnvelopeIssuanceCadence(seconds float64) {
	f.active().ObserveEnvelopeIssuanceCadence(seconds)
}
func (f *FailoverMetricsCollector) SetEnvelopeV1SunsetPhase(phase int) {
	f.active().SetEnvelopeV1SunsetPhase(phase)
}
func (f *FailoverMetricsCollector) SetSunsetPhaseSatisfactionProgress(p float64) {
	f.active().SetSunsetPhaseSatisfactionProgress(p)
}
func (f *FailoverMetricsCollector) IncEnvelopeRawPOAEmbedded() {
	f.active().IncEnvelopeRawPOAEmbedded()
}
func (f *FailoverMetricsCollector) IncEnvelopeRawPOATooLarge() {
	f.active().IncEnvelopeRawPOATooLarge()
}
func (f *FailoverMetricsCollector) IncMultiSignatureVerifications() {
	f.active().IncMultiSignatureVerifications()
}
func (f *FailoverMetricsCollector) IncMultiSignatureSuccess() { f.active().IncMultiSignatureSuccess() }
func (f *FailoverMetricsCollector) IncMultiSignatureVerificationFailures() {
	f.active().IncMultiSignatureVerificationFailures()
}
func (f *FailoverMetricsCollector) IncMultiSignatureIssued()  { f.active().IncMultiSignatureIssued() }
func (f *FailoverMetricsCollector) IncSingleSignatureIssued() { f.active().IncSingleSignatureIssued() }
func (f *FailoverMetricsCollector) SetMultiSignatureAdoptionRatio(r float64) {
	f.active().SetMultiSignatureAdoptionRatio(r)
}
func (f *FailoverMetricsCollector) IncMultiSignatureStructuralFailures() {
	f.active().IncMultiSignatureStructuralFailures()
}
func (f *FailoverMetricsCollector) IncMultiSignatureDigestFailures() {
	f.active().IncMultiSignatureDigestFailures()
}
func (f *FailoverMetricsCollector) IncMultiSignaturePublicKeyMissing() {
	f.active().IncMultiSignaturePublicKeyMissing()
}
func (f *FailoverMetricsCollector) IncMultiSignatureInvalidSignatureFailures() {
	f.active().IncMultiSignatureInvalidSignatureFailures()
}
func (f *FailoverMetricsCollector) IncMultiSignatureThresholdFailures() {
	f.active().IncMultiSignatureThresholdFailures()
}
func (f *FailoverMetricsCollector) IncViolation(cat interface{}) { f.active().IncViolation(cat) }
func (f *FailoverMetricsCollector) IncMultiSignatureWeightFailures() {
	f.active().IncMultiSignatureWeightFailures()
}
func (f *FailoverMetricsCollector) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	f.active().ObserveMultiSignatureVerificationLatency(d)
}
func (f *FailoverMetricsCollector) ObserveMultiSignatureBatchSize(size int) {
	f.active().ObserveMultiSignatureBatchSize(size)
}
func (f *FailoverMetricsCollector) ObserveMultiSignatureAggregateLatency(d time.Duration) {
	f.active().ObserveMultiSignatureAggregateLatency(d)
}
func (f *FailoverMetricsCollector) IncAnchorAttempts()        { f.active().IncAnchorAttempts() }
func (f *FailoverMetricsCollector) IncCombinedAnchorEmitted() { f.active().IncCombinedAnchorEmitted() }
func (f *FailoverMetricsCollector) IncCombinedAnchorFailures() {
	f.active().IncCombinedAnchorFailures()
}
func (f *FailoverMetricsCollector) IncAnchorFailures() { f.active().IncAnchorFailures() }
func (f *FailoverMetricsCollector) IncExternalAnchorAttempts(provider string) {
	f.active().IncExternalAnchorAttempts(provider)
}
func (f *FailoverMetricsCollector) IncExternalAnchorFailures(provider string) {
	f.active().IncExternalAnchorFailures(provider)
}
func (f *FailoverMetricsCollector) IncExternalAnchorForcedFailures() {
	f.active().IncExternalAnchorForcedFailures()
}
func (f *FailoverMetricsCollector) IncExternalAnchorForcedFailuresProvider(provider string) {
	f.active().IncExternalAnchorForcedFailuresProvider(provider)
}
func (f *FailoverMetricsCollector) ObserveExternalAnchorLatency(provider string, d time.Duration) {
	f.active().ObserveExternalAnchorLatency(provider, d)
}
func (f *FailoverMetricsCollector) SetExternalAnchorLastHashLen(n int) {
	f.active().SetExternalAnchorLastHashLen(n)
}
func (f *FailoverMetricsCollector) SetExternalAnchorAgeSeconds(age uint64) {
	f.active().SetExternalAnchorAgeSeconds(age)
}
func (f *FailoverMetricsCollector) ObserveExternalAnchorInterval(seconds float64) {
	f.active().ObserveExternalAnchorInterval(seconds)
}
func (f *FailoverMetricsCollector) HygieneSnapshot() map[string]uint64 {
	return f.active().HygieneSnapshot()
}
func (f *FailoverMetricsCollector) IncObligationsExecuted() { f.active().IncObligationsExecuted() }
func (f *FailoverMetricsCollector) IncObligationsFailed()   { f.active().IncObligationsFailed() }
func (f *FailoverMetricsCollector) ObserveObligationLatency(d time.Duration) {
	f.active().ObserveObligationLatency(d)
}
func (f *FailoverMetricsCollector) IncMandatoryObligationFailures() {
	f.active().IncMandatoryObligationFailures()
}
func (f *FailoverMetricsCollector) IncReplayHits()        { f.active().IncReplayHits() }
func (f *FailoverMetricsCollector) IncReplayMisses()      { f.active().IncReplayMisses() }
func (f *FailoverMetricsCollector) IncReplayStoreErrors() { f.active().IncReplayStoreErrors() }
func (f *FailoverMetricsCollector) IncMalformedJTI(reason string) {
	f.active().IncMalformedJTI(reason)
}
func (f *FailoverMetricsCollector) IncReplayStoreEvictions() { f.active().IncReplayStoreEvictions() }
func (f *FailoverMetricsCollector) ObserveReplayStoreLatency(d time.Duration) {
	f.active().ObserveReplayStoreLatency(d)
}
func (f *FailoverMetricsCollector) SetReplayWALPending(n int) { f.active().SetReplayWALPending(n) }
func (f *FailoverMetricsCollector) ObserveReplayWALFlushLatency(d time.Duration) {
	f.active().ObserveReplayWALFlushLatency(d)
}
func (f *FailoverMetricsCollector) ObserveReplayWALSnapshotDuration(d time.Duration) {
	f.active().ObserveReplayWALSnapshotDuration(d)
}
func (f *FailoverMetricsCollector) IncReplayStoreAvailabilityImpact() {
	f.active().IncReplayStoreAvailabilityImpact()
}
func (f *FailoverMetricsCollector) IncCapabilityDiffRequests() {
	f.active().IncCapabilityDiffRequests()
}
func (f *FailoverMetricsCollector) ObserveCapabilityDiffLatency(d time.Duration) {
	f.active().ObserveCapabilityDiffLatency(d)
}
func (f *FailoverMetricsCollector) IncCapabilityAnchorEmitted() {
	f.active().IncCapabilityAnchorEmitted()
}
func (f *FailoverMetricsCollector) IncCapabilityAnchorSkipped() {
	f.active().IncCapabilityAnchorSkipped()
}
func (f *FailoverMetricsCollector) IncCapabilityRegistryHashChanged() {
	f.active().IncCapabilityRegistryHashChanged()
}
func (f *FailoverMetricsCollector) SetCapabilityAnchorLastWriteUnix(ts uint64) {
	f.active().SetCapabilityAnchorLastWriteUnix(ts)
}
func (f *FailoverMetricsCollector) IncCapabilityAnchorAlgorithm(algo string) {
	f.active().IncCapabilityAnchorAlgorithm(algo)
}
func (f *FailoverMetricsCollector) SetCapabilityAnchorAlgorithmRatio(algo string, ratio float64) {
	f.active().SetCapabilityAnchorAlgorithmRatio(algo, ratio)
}
func (f *FailoverMetricsCollector) ObserveCapabilityAnchorInterval(d time.Duration) {
	f.active().ObserveCapabilityAnchorInterval(d)
}
func (f *FailoverMetricsCollector) IncCapabilityEnforceAllowed() {
	f.active().IncCapabilityEnforceAllowed()
}
func (f *FailoverMetricsCollector) IncCapabilityEnforceDenied() {
	f.active().IncCapabilityEnforceDenied()
}
func (f *FailoverMetricsCollector) IncPEPEnforcements(allowed bool, actionType string) {
	f.active().IncPEPEnforcements(allowed, actionType)
}
func (f *FailoverMetricsCollector) IncPEPViolations(violationType, severity string) {
	f.active().IncPEPViolations(violationType, severity)
}
func (f *FailoverMetricsCollector) ObservePEPEnforcementLatency(d time.Duration) {
	f.active().ObservePEPEnforcementLatency(d)
}
func (f *FailoverMetricsCollector) SetPEPAuditBufferSize(enforcement, violation int) {
	f.active().SetPEPAuditBufferSize(enforcement, violation)
}
func (f *FailoverMetricsCollector) IncModelLimitExceeded() { f.active().IncModelLimitExceeded() }
func (f *FailoverMetricsCollector) IncModelOutputLimitExceeded() {
	f.active().IncModelOutputLimitExceeded()
}
func (f *FailoverMetricsCollector) IncModelRateLimitExceeded() {
	f.active().IncModelRateLimitExceeded()
}
func (f *FailoverMetricsCollector) IncModelUnknown()    { f.active().IncModelUnknown() }
func (f *FailoverMetricsCollector) IncModelLimitSurge() { f.active().IncModelLimitSurge() }
func (f *FailoverMetricsCollector) IncModelUserInputLimitExceeded() {
	f.active().IncModelUserInputLimitExceeded()
}
func (f *FailoverMetricsCollector) IncModelUserOutputLimitExceeded() {
	f.active().IncModelUserOutputLimitExceeded()
}
func (f *FailoverMetricsCollector) IncModelUserRateLimitExceeded() {
	f.active().IncModelUserRateLimitExceeded()
}
func (f *FailoverMetricsCollector) IncScopeViolations()       { f.active().IncScopeViolations() }
func (f *FailoverMetricsCollector) IncRestrictionViolations() { f.active().IncRestrictionViolations() }
func (f *FailoverMetricsCollector) IncUnauthorized()          { f.active().IncUnauthorized() }
func (f *FailoverMetricsCollector) IncExpired()               { f.active().IncExpired() }
func (f *FailoverMetricsCollector) IncRevoked()               { f.active().IncRevoked() }
func (f *FailoverMetricsCollector) IncDelegationStatusTransitions() {
	f.active().IncDelegationStatusTransitions()
}
func (f *FailoverMetricsCollector) IncDelegationStatusTransitionFailures() {
	f.active().IncDelegationStatusTransitionFailures()
}
func (f *FailoverMetricsCollector) IncTokenStatusTransitions() {
	f.active().IncTokenStatusTransitions()
}
func (f *FailoverMetricsCollector) IncTokenStatusTransitionFailures() {
	f.active().IncTokenStatusTransitionFailures()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowInitiated() {
	f.active().IncRevocationWorkflowInitiated()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowInitiationFailures() {
	f.active().IncRevocationWorkflowInitiationFailures()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowApprovals() {
	f.active().IncRevocationWorkflowApprovals()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowApprovalFailures() {
	f.active().IncRevocationWorkflowApprovalFailures()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowQuorumSatisfied() {
	f.active().IncRevocationWorkflowQuorumSatisfied()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowCanceled() {
	f.active().IncRevocationWorkflowCanceled()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowCancellationFailures() {
	f.active().IncRevocationWorkflowCancellationFailures()
}
func (f *FailoverMetricsCollector) IncRevocationWorkflowUnauthorized() {
	f.active().IncRevocationWorkflowUnauthorized()
}
func (f *FailoverMetricsCollector) IncEvidenceAttachment() { f.active().IncEvidenceAttachment() }
func (f *FailoverMetricsCollector) IncEvidenceAttachmentFailures() {
	f.active().IncEvidenceAttachmentFailures()
}
func (f *FailoverMetricsCollector) SetEvidenceHashesPerPOA(poaID string, n int) {
	f.active().SetEvidenceHashesPerPOA(poaID, n)
}
func (f *FailoverMetricsCollector) IncDelegationGraphExports() {
	f.active().IncDelegationGraphExports()
}
func (f *FailoverMetricsCollector) SetDelegationGraphNodeCount(n int) {
	f.active().SetDelegationGraphNodeCount(n)
}
func (f *FailoverMetricsCollector) RecordDecision(action, resource, outcome string, d time.Duration) {
	f.active().RecordDecision(action, resource, outcome, d)
}
func (f *FailoverMetricsCollector) RecordDecisionWithReason(action, resource, outcome, reason string) {
	f.active().RecordDecisionWithReason(action, resource, outcome, reason)
}
func (f *FailoverMetricsCollector) RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome string) {
	f.active().RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome)
}
func (f *FailoverMetricsCollector) ObserveLifecycleTransitionLatency(entityType, outcome string, d time.Duration) {
	f.active().ObserveLifecycleTransitionLatency(entityType, outcome, d)
}
func (f *FailoverMetricsCollector) SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile string, value float64) {
	f.active().SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile, value)
}
func (f *FailoverMetricsCollector) IncCascadeRevocationTriggered() {
	f.active().IncCascadeRevocationTriggered()
}
func (f *FailoverMetricsCollector) IncCascadeDescendantsProcessed() {
	f.active().IncCascadeDescendantsProcessed()
}
func (f *FailoverMetricsCollector) ObserveCascadeProcessingLatency(d time.Duration) {
	f.active().ObserveCascadeProcessingLatency(d)
}
func (f *FailoverMetricsCollector) IncCascadeDepthLimitReached() {
	f.active().IncCascadeDepthLimitReached()
}
func (f *FailoverMetricsCollector) IncCascadeBatchProcessed() { f.active().IncCascadeBatchProcessed() }
func (f *FailoverMetricsCollector) SetCascadeMaxDepthReached(depth int) {
	f.active().SetCascadeMaxDepthReached(depth)
}
func (f *FailoverMetricsCollector) IncCascadeProcessingErrors() {
	f.active().IncCascadeProcessingErrors()
}
func (f *FailoverMetricsCollector) IncJurisdictionEnforcementErrors() {
	f.active().IncJurisdictionEnforcementErrors()
}
func (f *FailoverMetricsCollector) IncJurisdictionEnforcementDenials() {
	f.active().IncJurisdictionEnforcementDenials()
}
func (f *FailoverMetricsCollector) IncJurisdictionEnforcementAllows() {
	f.active().IncJurisdictionEnforcementAllows()
}
func (f *FailoverMetricsCollector) SetSystemClockSkew(seconds float64) {
	f.active().SetSystemClockSkew(seconds)
}
