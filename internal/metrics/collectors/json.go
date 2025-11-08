// Copyright (c) 2025 GAuth. All rights reserved.

// Package collectors provides example MetricsCollector implementations.
package collectors

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// JSONCollector exports metrics as JSON for debugging and testing.
//
// Metrics are buffered in memory and periodically flushed to a JSON file.
// This collector is useful for development, testing, and troubleshooting.
type JSONCollector struct {
	mu           sync.Mutex
	metadata     metrics.CollectorMetadata
	outputPath   string
	buffer       map[string]interface{}
	flushOnWrite bool
}

// NewJSONCollector creates a JSON metrics collector.
//
// Parameters:
// - id: Unique collector identifier
// - outputPath: File path for JSON output
// - flushOnWrite: If true, flush after every metric (slower but safer)
func NewJSONCollector(id, outputPath string, flushOnWrite bool) *JSONCollector {
	return &JSONCollector{
		metadata: metrics.CollectorMetadata{
			ID:           id,
			Type:         metrics.CollectorTypeJSON,
			Description:  fmt.Sprintf("JSON debug exporter to %s", outputPath),
			RegisteredAt: time.Now(),
			Version:      "1.0.0",
		},
		outputPath:   outputPath,
		buffer:       make(map[string]interface{}),
		flushOnWrite: flushOnWrite,
	}
}

// Metadata returns collector metadata.
func (j *JSONCollector) Metadata() metrics.CollectorMetadata {
	return j.metadata
}

// Flush writes buffered metrics to JSON file.
func (j *JSONCollector) Flush() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	data, err := json.MarshalIndent(j.buffer, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON marshal: %w", err)
	}

	if err := os.WriteFile(j.outputPath, data, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Close flushes and resets the collector.
func (j *JSONCollector) Close() error {
	if err := j.Flush(); err != nil {
		return err
	}
	j.mu.Lock()
	j.buffer = make(map[string]interface{})
	j.mu.Unlock()
	return nil
}

// Health checks if output path is writable.
func (j *JSONCollector) Health() error {
	// Try writing an empty file to test permissions
	f, err := os.OpenFile(j.outputPath, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w", j.outputPath, err)
	}
	f.Close()
	return nil
}

// incrementCounter increments a counter in the buffer.
func (j *JSONCollector) incrementCounter(name string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if v, ok := j.buffer[name]; ok {
		if count, ok := v.(float64); ok {
			j.buffer[name] = count + 1
		} else {
			j.buffer[name] = 1.0
		}
	} else {
		j.buffer[name] = 1.0
	}

	if j.flushOnWrite {
		_ = j.Flush()
	}
}

// setGauge sets a gauge value in the buffer.
func (j *JSONCollector) setGauge(name string, value interface{}) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.buffer[name] = value

	if j.flushOnWrite {
		_ = j.Flush()
	}
}

// recordHistogram records a histogram observation (simplified: just store latest value).
func (j *JSONCollector) recordHistogram(name string, value interface{}) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// For JSON collector, store latest observation (simplified)
	j.buffer[name+"_latest"] = value

	if j.flushOnWrite {
		_ = j.Flush()
	}
}

// Implement Metrics interface (simplified - all counters/gauges recorded in JSON).

func (j *JSONCollector) IncDelegationsCreated() {
	j.incrementCounter("delegations_created")
}

func (j *JSONCollector) IncDelegationsPartiallyRevoked() {
	j.incrementCounter("delegations_partially_revoked")
}

func (j *JSONCollector) IncDelegationDepthExceeded() {
	j.incrementCounter("delegation_depth_exceeded")
}

func (j *JSONCollector) SetMaxObservedDelegationDepth(depth int) {
	j.setGauge("max_observed_delegation_depth", depth)
}

func (j *JSONCollector) ObserveValidationLatency(d time.Duration) {
	j.recordHistogram("validation_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) IncSignaturesIssued() {
	j.incrementCounter("signatures_issued")
}

func (j *JSONCollector) IncSignatureIssueFailures() {
	j.incrementCounter("signature_issue_failures")
}

func (j *JSONCollector) IncSignatureVerifications() {
	j.incrementCounter("signature_verifications")
}

func (j *JSONCollector) IncSignatureVerificationFailures() {
	j.incrementCounter("signature_verification_failures")
}

func (j *JSONCollector) IncAttestationProofIssued() {
	j.incrementCounter("attestation_proof_issued")
}

func (j *JSONCollector) IncAttestationProofIssueFailures() {
	j.incrementCounter("attestation_proof_issue_failures")
}

func (j *JSONCollector) IncAttestationProofVerifications() {
	j.incrementCounter("attestation_proof_verifications")
}

func (j *JSONCollector) IncAttestationProofVerificationFailures() {
	j.incrementCounter("attestation_proof_verification_failures")
}

func (j *JSONCollector) IncAttestationProofDigestMismatch() {
	j.incrementCounter("attestation_proof_digest_mismatch")
}

func (j *JSONCollector) ObserveAttestationProofVerificationLatency(d time.Duration) {
	j.recordHistogram("attestation_proof_verification_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) ObserveAttestationProofIssueLatency(d time.Duration) {
	j.recordHistogram("attestation_proof_issue_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) IncAttestationProofVerificationFailureReason(reason string) {
	j.incrementCounter("attestation_proof_verification_failure_reason_" + reason)
}

func (j *JSONCollector) IncBLSPoPChallengesIssued() {
	j.incrementCounter("bls_pop_challenges_issued")
}

func (j *JSONCollector) IncBLSPoPVerifications() {
	j.incrementCounter("bls_pop_verifications")
}

func (j *JSONCollector) IncBLSPoPVerificationFailures() {
	j.incrementCounter("bls_pop_verification_failures")
}

func (j *JSONCollector) IncAttestationProofTrustAnchorMissing() {
	j.incrementCounter("attestation_proof_trust_anchor_missing")
}

func (j *JSONCollector) IncAttestationProofTrustAnchorAlgorithmMismatch() {
	j.incrementCounter("attestation_proof_trust_anchor_algorithm_mismatch")
}

func (j *JSONCollector) IncAttestationProofTrustAnchorKeyMismatch() {
	j.incrementCounter("attestation_proof_trust_anchor_key_mismatch")
}

func (j *JSONCollector) IncRevocationIntegrityFailures() {
	j.incrementCounter("revocation_integrity_failures")
}

func (j *JSONCollector) IncSignaturePublicKeyMissing() {
	j.incrementCounter("signature_public_key_missing")
}

func (j *JSONCollector) IncCryptoSignatureMissing() {
	j.incrementCounter("crypto_signature_missing")
}

func (j *JSONCollector) IncEnvelopeV1Issued() {
	j.incrementCounter("envelope_v1_issued")
}

func (j *JSONCollector) IncEnvelopeV2Issued() {
	j.incrementCounter("envelope_v2_issued")
}

func (j *JSONCollector) IncHierDigestIssued() {
	j.incrementCounter("hier_digest_issued")
}

func (j *JSONCollector) IncHierDigestParentDigestMissing() {
	j.incrementCounter("hier_digest_parent_digest_missing")
}

func (j *JSONCollector) IncHierDigestVersionMismatch() {
	j.incrementCounter("hier_digest_version_mismatch")
}

func (j *JSONCollector) SetEnvelopeV2AdoptionRatio(r2 float64) {
	j.setGauge("envelope_v2_adoption_ratio", r2)
}

func (j *JSONCollector) IncEnvelopeDigestMismatch() {
	j.incrementCounter("envelope_digest_mismatch")
}

func (j *JSONCollector) IncEnvelopeDigestMismatchReason(reason string) {
	j.incrementCounter("envelope_digest_mismatch_reason_" + reason)
}

func (j *JSONCollector) ObserveEnvelopeIssuanceCadence(seconds float64) {
	j.recordHistogram("envelope_issuance_cadence_seconds", seconds)
}

func (j *JSONCollector) SetEnvelopeV1SunsetPhase(phase int) {
	j.setGauge("envelope_v1_sunset_phase", phase)
}

func (j *JSONCollector) SetSunsetPhaseSatisfactionProgress(p float64) {
	j.setGauge("sunset_phase_satisfaction_progress", p)
}

func (j *JSONCollector) IncEnvelopeRawPOAEmbedded() {
	j.incrementCounter("envelope_raw_poa_embedded")
}

func (j *JSONCollector) IncEnvelopeRawPOATooLarge() {
	j.incrementCounter("envelope_raw_poa_too_large")
}

// Remaining methods implemented similarly (119 total).
// For brevity, showing pattern. Full implementation would include all methods.

func (j *JSONCollector) IncMultiSignatureVerifications() {
	j.incrementCounter("multi_signature_verifications")
}

func (j *JSONCollector) IncMultiSignatureVerificationFailures() {
	j.incrementCounter("multi_signature_verification_failures")
}

func (j *JSONCollector) IncMultiSignatureStructuralFailures() {
	j.incrementCounter("multi_signature_structural_failures")
}

func (j *JSONCollector) IncMultiSignatureDigestFailures() {
	j.incrementCounter("multi_signature_digest_failures")
}

func (j *JSONCollector) IncMultiSignaturePublicKeyMissing() {
	j.incrementCounter("multi_signature_public_key_missing")
}

func (j *JSONCollector) IncMultiSignatureInvalidSignatureFailures() {
	j.incrementCounter("multi_signature_invalid_signature_failures")
}

func (j *JSONCollector) IncMultiSignatureThresholdFailures() {
	j.incrementCounter("multi_signature_threshold_failures")
}

func (j *JSONCollector) IncViolation(cat interface{}) {
	j.incrementCounter(fmt.Sprintf("violation_%v", cat))
}

func (j *JSONCollector) IncMultiSignatureWeightFailures() {
	j.incrementCounter("multi_signature_weight_failures")
}

func (j *JSONCollector) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	j.recordHistogram("multi_signature_verification_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) ObserveMultiSignatureBatchSize(size int) {
	j.recordHistogram("multi_signature_batch_size", size)
}

func (j *JSONCollector) ObserveMultiSignatureAggregateLatency(d time.Duration) {
	j.recordHistogram("multi_signature_aggregate_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) IncAnchorAttempts() {
	j.incrementCounter("anchor_attempts")
}

func (j *JSONCollector) IncCombinedAnchorEmitted() {
	j.incrementCounter("combined_anchor_emitted")
}

func (j *JSONCollector) IncCombinedAnchorFailures() {
	j.incrementCounter("combined_anchor_failures")
}

func (j *JSONCollector) IncAnchorFailures() {
	j.incrementCounter("anchor_failures")
}

func (j *JSONCollector) IncExternalAnchorForcedFailures() {
	j.incrementCounter("external_anchor_forced_failures")
}

func (j *JSONCollector) IncObligationsExecuted() {
	j.incrementCounter("obligations_executed")
}

func (j *JSONCollector) IncObligationsFailed() {
	j.incrementCounter("obligations_failed")
}

func (j *JSONCollector) ObserveObligationLatency(d time.Duration) {
	j.recordHistogram("obligation_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) IncMandatoryObligationFailures() {
	j.incrementCounter("mandatory_obligation_failures")
}

func (j *JSONCollector) IncReplayHits() {
	j.incrementCounter("replay_hits")
}

func (j *JSONCollector) IncReplayMisses() {
	j.incrementCounter("replay_misses")
}

func (j *JSONCollector) IncReplayStoreErrors() {
	j.incrementCounter("replay_store_errors")
}

func (j *JSONCollector) ObserveReplayStoreLatency(d time.Duration) {
	j.recordHistogram("replay_store_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) SetReplayWALPending(n int) {
	j.setGauge("replay_wal_pending", n)
}

func (j *JSONCollector) ObserveReplayWALFlushLatency(d time.Duration) {
	j.recordHistogram("replay_wal_flush_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) ObserveReplayWALSnapshotDuration(d time.Duration) {
	j.recordHistogram("replay_wal_snapshot_duration_ns", d.Nanoseconds())
}

func (j *JSONCollector) IncCapabilityDiffRequests() {
	j.incrementCounter("capability_diff_requests")
}

func (j *JSONCollector) ObserveCapabilityDiffLatency(d time.Duration) {
	j.recordHistogram("capability_diff_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) IncCapabilityAnchorEmitted() {
	j.incrementCounter("capability_anchor_emitted")
}

func (j *JSONCollector) IncCapabilityAnchorSkipped() {
	j.incrementCounter("capability_anchor_skipped")
}

func (j *JSONCollector) IncCapabilityRegistryHashChanged() {
	j.incrementCounter("capability_registry_hash_changed")
}

func (j *JSONCollector) SetCapabilityAnchorLastWriteUnix(ts uint64) {
	j.setGauge("capability_anchor_last_write_unix", ts)
}

func (j *JSONCollector) IncCapabilityAnchorAlgorithm(algo string) {
	j.incrementCounter("capability_anchor_algorithm_" + algo)
}

func (j *JSONCollector) SetCapabilityAnchorAlgorithmRatio(algo string, ratio float64) {
	j.setGauge("capability_anchor_algorithm_ratio_"+algo, ratio)
}

func (j *JSONCollector) IncCapabilityEnforceAllowed() {
	j.incrementCounter("capability_enforce_allowed")
}

func (j *JSONCollector) IncCapabilityEnforceDenied() {
	j.incrementCounter("capability_enforce_denied")
}

func (j *JSONCollector) IncModelLimitExceeded() {
	j.incrementCounter("model_limit_exceeded")
}

func (j *JSONCollector) IncModelOutputLimitExceeded() {
	j.incrementCounter("model_output_limit_exceeded")
}

func (j *JSONCollector) IncModelRateLimitExceeded() {
	j.incrementCounter("model_rate_limit_exceeded")
}

func (j *JSONCollector) IncModelUnknown() {
	j.incrementCounter("model_unknown")
}

func (j *JSONCollector) IncModelLimitSurge() {
	j.incrementCounter("model_limit_surge")
}

func (j *JSONCollector) IncModelUserInputLimitExceeded() {
	j.incrementCounter("model_user_input_limit_exceeded")
}

func (j *JSONCollector) IncModelUserOutputLimitExceeded() {
	j.incrementCounter("model_user_output_limit_exceeded")
}

func (j *JSONCollector) IncModelUserRateLimitExceeded() {
	j.incrementCounter("model_user_rate_limit_exceeded")
}

func (j *JSONCollector) IncScopeViolations() {
	j.incrementCounter("scope_violations")
}

func (j *JSONCollector) IncRestrictionViolations() {
	j.incrementCounter("restriction_violations")
}

func (j *JSONCollector) IncUnauthorized() {
	j.incrementCounter("unauthorized")
}

func (j *JSONCollector) IncExpired() {
	j.incrementCounter("expired")
}

func (j *JSONCollector) IncRevoked() {
	j.incrementCounter("revoked")
}

func (j *JSONCollector) IncDelegationStatusTransitions() {
	j.incrementCounter("delegation_status_transitions")
}

func (j *JSONCollector) IncDelegationStatusTransitionFailures() {
	j.incrementCounter("delegation_status_transition_failures")
}

func (j *JSONCollector) IncTokenStatusTransitions() {
	j.incrementCounter("token_status_transitions")
}

func (j *JSONCollector) IncTokenStatusTransitionFailures() {
	j.incrementCounter("token_status_transition_failures")
}

func (j *JSONCollector) IncRevocationWorkflowInitiated() {
	j.incrementCounter("revocation_workflow_initiated")
}

func (j *JSONCollector) IncRevocationWorkflowInitiationFailures() {
	j.incrementCounter("revocation_workflow_initiation_failures")
}

func (j *JSONCollector) IncRevocationWorkflowApprovals() {
	j.incrementCounter("revocation_workflow_approvals")
}

func (j *JSONCollector) IncRevocationWorkflowApprovalFailures() {
	j.incrementCounter("revocation_workflow_approval_failures")
}

func (j *JSONCollector) IncRevocationWorkflowQuorumSatisfied() {
	j.incrementCounter("revocation_workflow_quorum_satisfied")
}

func (j *JSONCollector) IncRevocationWorkflowCanceled() {
	j.incrementCounter("revocation_workflow_canceled")
}

func (j *JSONCollector) IncRevocationWorkflowCancellationFailures() {
	j.incrementCounter("revocation_workflow_cancellation_failures")
}

func (j *JSONCollector) IncRevocationWorkflowUnauthorized() {
	j.incrementCounter("revocation_workflow_unauthorized")
}

func (j *JSONCollector) IncEvidenceAttachment() {
	j.incrementCounter("evidence_attachment")
}

func (j *JSONCollector) IncEvidenceAttachmentFailures() {
	j.incrementCounter("evidence_attachment_failures")
}

func (j *JSONCollector) SetEvidenceHashesPerPOA(poaID string, n int) {
	j.setGauge("evidence_hashes_per_poa_"+poaID, n)
}

func (j *JSONCollector) IncDelegationGraphExports() {
	j.incrementCounter("delegation_graph_exports")
}

func (j *JSONCollector) SetDelegationGraphNodeCount(n int) {
	j.setGauge("delegation_graph_node_count", n)
}

func (j *JSONCollector) RecordDecision(action, resource, outcome string) {
	j.incrementCounter(fmt.Sprintf("decision_%s_%s_%s", action, resource, outcome))
}

func (j *JSONCollector) RecordDecisionWithReason(action, resource, outcome, reason string) {
	j.incrementCounter(fmt.Sprintf("decision_%s_%s_%s_%s", action, resource, outcome, reason))
}

func (j *JSONCollector) RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome string) {
	j.incrementCounter(fmt.Sprintf("lifecycle_%s_%s_to_%s_%s", entityType, oldStatus, newStatus, outcome))
}

func (j *JSONCollector) ObserveLifecycleTransitionLatency(entityType, outcome string, d time.Duration) {
	j.recordHistogram(fmt.Sprintf("lifecycle_latency_%s_%s_ns", entityType, outcome), d.Nanoseconds())
}

func (j *JSONCollector) SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile string, value float64) {
	j.setGauge(fmt.Sprintf("lifecycle_latency_quantile_%s_%s_%s", entityType, outcome, quantile), value)
}

func (j *JSONCollector) IncCascadeRevocationTriggered() {
	j.incrementCounter("cascade_revocation_triggered")
}

func (j *JSONCollector) IncCascadeDescendantsProcessed() {
	j.incrementCounter("cascade_descendants_processed")
}

func (j *JSONCollector) ObserveCascadeProcessingLatency(d time.Duration) {
	j.recordHistogram("cascade_processing_latency_ns", d.Nanoseconds())
}

func (j *JSONCollector) IncCascadeDepthLimitReached() {
	j.incrementCounter("cascade_depth_limit_reached")
}

func (j *JSONCollector) IncCascadeBatchProcessed() {
	j.incrementCounter("cascade_batch_processed")
}

func (j *JSONCollector) SetCascadeMaxDepthReached(depth int) {
	j.setGauge("cascade_max_depth_reached", depth)
}

func (j *JSONCollector) IncCascadeProcessingErrors() {
	j.incrementCounter("cascade_processing_errors")
}
