// Copyright (c) 2025 GAuth. All rights reserved.

// Package metrics provides a pluggable metrics collector registry for GAuth.
//
// P3.2 (sec7.item2): Metrics collector registration framework for extensible observability.
package metrics

import (
	"fmt"
	"sync"
	"time"
)

// CollectorType identifies the type of metrics collector.
type CollectorType string

const (
	// CollectorTypePrometheus identifies Prometheus exporters.
	CollectorTypePrometheus CollectorType = "prometheus"
	// CollectorTypeStatsD identifies StatsD/DogStatsD exporters.
	CollectorTypeStatsD CollectorType = "statsd"
	// CollectorTypeJSON identifies JSON file exporters.
	CollectorTypeJSON CollectorType = "json"
	// CollectorTypeOpenTelemetry identifies OpenTelemetry exporters.
	CollectorTypeOpenTelemetry CollectorType = "opentelemetry"
	// CollectorTypeCustom identifies custom/third-party exporters.
	CollectorTypeCustom CollectorType = "custom"
)

// CollectorMetadata provides information about a registered collector.
type CollectorMetadata struct {
	// ID is a unique identifier for the collector instance.
	ID string
	// Type categorizes the collector (prometheus, statsd, json, etc.).
	Type CollectorType
	// Description provides human-readable information about the collector.
	Description string
	// RegisteredAt records when the collector was registered.
	RegisteredAt time.Time
	// Version identifies the collector implementation version.
	Version string
}

// MetricsCollector defines the interface for pluggable metrics exporters.
//
// Implementations must be thread-safe and handle concurrent metric updates.
// Collectors should be lightweight adapters that forward metrics to external
// systems (Prometheus, StatsD, CloudWatch, etc.) without blocking.
type MetricsCollector interface {
	// Metrics embeds the core metrics interface, allowing collectors to
	// receive all standard metric events.
	Metrics

	// Metadata returns information about this collector instance.
	Metadata() CollectorMetadata

	// Flush forces any buffered metrics to be exported immediately.
	// Returns error if flush fails (e.g., network unavailable).
	Flush() error

	// Close cleanly shuts down the collector, flushing pending metrics
	// and releasing resources. After Close(), the collector should not
	// be used.
	Close() error

	// Health returns nil if the collector is healthy and able to export
	// metrics. Returns error describing health issues otherwise.
	Health() error
}

// CollectorRegistry manages a collection of metrics collectors.
//
// The registry dispatches metric events to all registered collectors in parallel,
// allowing simultaneous export to multiple backends (e.g., Prometheus + StatsD).
//
// Thread-safe for concurrent registration/deregistration and metric updates.
type CollectorRegistry struct {
	mu         sync.RWMutex
	collectors map[string]MetricsCollector
	// Dispatch metrics in parallel to avoid slow collectors blocking others
	dispatchConcurrent bool
}

// NewCollectorRegistry creates a new metrics collector registry.
//
// If concurrent=true, metric events are dispatched to collectors in parallel,
// preventing slow collectors from blocking fast ones. If false, collectors
// are called sequentially (useful for testing/debugging).
func NewCollectorRegistry(concurrent bool) *CollectorRegistry {
	return &CollectorRegistry{
		collectors:         make(map[string]MetricsCollector),
		dispatchConcurrent: concurrent,
	}
}

// Register adds a collector to the registry.
//
// Returns error if a collector with the same ID is already registered.
// The collector will immediately start receiving metric events.
func (reg *CollectorRegistry) Register(collector MetricsCollector) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	meta := collector.Metadata()
	if _, exists := reg.collectors[meta.ID]; exists {
		return fmt.Errorf("collector %q already registered", meta.ID)
	}

	reg.collectors[meta.ID] = collector
	return nil
}

// Deregister removes a collector from the registry.
//
// The collector is flushed and closed before removal. Returns error if
// collector not found or if flush/close fails.
func (reg *CollectorRegistry) Deregister(id string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	collector, exists := reg.collectors[id]
	if !exists {
		return fmt.Errorf("collector %q not found", id)
	}

	// Flush pending metrics before removal
	if err := collector.Flush(); err != nil {
		return fmt.Errorf("flush failed for %q: %w", id, err)
	}

	// Close collector resources
	if err := collector.Close(); err != nil {
		return fmt.Errorf("close failed for %q: %w", id, err)
	}

	delete(reg.collectors, id)
	return nil
}

// Get retrieves a collector by ID.
//
// Returns nil if collector not found.
func (reg *CollectorRegistry) Get(id string) MetricsCollector {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	return reg.collectors[id]
}

// List returns metadata for all registered collectors.
func (reg *CollectorRegistry) List() []CollectorMetadata {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	result := make([]CollectorMetadata, 0, len(reg.collectors))
	for _, collector := range reg.collectors {
		result = append(result, collector.Metadata())
	}
	return result
}

// FlushAll flushes all registered collectors.
//
// Returns a map of collector ID -> error for any collectors that failed to flush.
// If all collectors flush successfully, returns empty map.
func (reg *CollectorRegistry) FlushAll() map[string]error {
	reg.mu.RLock()
	collectors := make([]MetricsCollector, 0, len(reg.collectors))
	for _, c := range reg.collectors {
		collectors = append(collectors, c)
	}
	reg.mu.RUnlock()

	errors := make(map[string]error)
	for _, collector := range collectors {
		if err := collector.Flush(); err != nil {
			errors[collector.Metadata().ID] = err
		}
	}
	return errors
}

// CloseAll closes all registered collectors and clears the registry.
//
// Returns a map of collector ID -> error for any collectors that failed to close.
func (reg *CollectorRegistry) CloseAll() map[string]error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	errors := make(map[string]error)
	for id, collector := range reg.collectors {
		if err := collector.Flush(); err != nil {
			errors[id] = fmt.Errorf("flush: %w", err)
		}
		if err := collector.Close(); err != nil {
			if existing, ok := errors[id]; ok {
				errors[id] = fmt.Errorf("%v; close: %w", existing, err)
			} else {
				errors[id] = err
			}
		}
	}

	// Clear registry
	reg.collectors = make(map[string]MetricsCollector)
	return errors
}

// HealthCheck checks the health of all registered collectors.
//
// Returns a map of collector ID -> error for any unhealthy collectors.
// If all collectors are healthy, returns empty map.
func (reg *CollectorRegistry) HealthCheck() map[string]error {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	errors := make(map[string]error)
	for id, collector := range reg.collectors {
		if err := collector.Health(); err != nil {
			errors[id] = err
		}
	}
	return errors
}

// dispatch calls a function on all registered collectors.
//
// If concurrent dispatch is enabled, collectors are called in parallel.
// Otherwise, collectors are called sequentially.
func (reg *CollectorRegistry) dispatch(fn func(MetricsCollector)) {
	reg.mu.RLock()
	collectors := make([]MetricsCollector, 0, len(reg.collectors))
	for _, c := range reg.collectors {
		collectors = append(collectors, c)
	}
	concurrent := reg.dispatchConcurrent
	reg.mu.RUnlock()

	if concurrent && len(collectors) > 1 {
		// Parallel dispatch to avoid slow collectors blocking fast ones
		var wg sync.WaitGroup
		for _, collector := range collectors {
			wg.Add(1)
			go func(c MetricsCollector) {
				defer wg.Done()
				fn(c)
			}(collector)
		}
		wg.Wait()
	} else {
		// Sequential dispatch (simpler, useful for testing)
		for _, collector := range collectors {
			fn(collector)
		}
	}
}

// Implement Metrics interface by dispatching to all registered collectors.
// Auto-generated from noop implementation (119 methods total).

//go:generate python3 ../../scripts/generate_registry_methods.py
// Auto-generated registry dispatch methods (119 total)

func (reg *CollectorRegistry) IncDelegationsCreated() {
	reg.dispatch(func(c MetricsCollector) { c.IncDelegationsCreated() })
}

func (reg *CollectorRegistry) IncDelegationsPartiallyRevoked() {
	reg.dispatch(func(c MetricsCollector) { c.IncDelegationsPartiallyRevoked() })
}

func (reg *CollectorRegistry) IncDelegationDepthExceeded() {
	reg.dispatch(func(c MetricsCollector) { c.IncDelegationDepthExceeded() })
}

func (reg *CollectorRegistry) SetMaxObservedDelegationDepth(depth int) {
	reg.dispatch(func(c MetricsCollector) { c.SetMaxObservedDelegationDepth(depth) })
}

func (reg *CollectorRegistry) ObserveValidationLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveValidationLatency(d) })
}

func (reg *CollectorRegistry) IncSignaturesIssued() {
	reg.dispatch(func(c MetricsCollector) { c.IncSignaturesIssued() })
}

func (reg *CollectorRegistry) IncSignatureIssueFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncSignatureIssueFailures() })
}

func (reg *CollectorRegistry) IncSignatureVerifications() {
	reg.dispatch(func(c MetricsCollector) { c.IncSignatureVerifications() })
}

func (reg *CollectorRegistry) IncSignatureVerificationFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncSignatureVerificationFailures() })
}

func (reg *CollectorRegistry) IncAttestationProofIssued() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofIssued() })
}

func (reg *CollectorRegistry) IncAttestationProofIssueFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofIssueFailures() })
}

func (reg *CollectorRegistry) IncAttestationProofVerifications() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofVerifications() })
}

func (reg *CollectorRegistry) IncAttestationProofVerificationFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofVerificationFailures() })
}

func (reg *CollectorRegistry) IncAttestationProofDigestMismatch() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofDigestMismatch() })
}

func (reg *CollectorRegistry) ObserveAttestationProofVerificationLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveAttestationProofVerificationLatency(d) })
}

func (reg *CollectorRegistry) ObserveAttestationProofIssueLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveAttestationProofIssueLatency(d) })
}

func (reg *CollectorRegistry) IncAttestationProofVerificationFailureReason(reason string) {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofVerificationFailureReason(reason) })
}

func (reg *CollectorRegistry) IncBLSPoPChallengesIssued() {
	reg.dispatch(func(c MetricsCollector) { c.IncBLSPoPChallengesIssued() })
}

func (reg *CollectorRegistry) IncBLSPoPVerifications() {
	reg.dispatch(func(c MetricsCollector) { c.IncBLSPoPVerifications() })
}

func (reg *CollectorRegistry) IncBLSPoPVerificationFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncBLSPoPVerificationFailures() })
}

func (reg *CollectorRegistry) IncAttestationProofTrustAnchorMissing() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofTrustAnchorMissing() })
}

func (reg *CollectorRegistry) IncAttestationProofTrustAnchorAlgorithmMismatch() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofTrustAnchorAlgorithmMismatch() })
}

func (reg *CollectorRegistry) IncAttestationProofTrustAnchorKeyMismatch() {
	reg.dispatch(func(c MetricsCollector) { c.IncAttestationProofTrustAnchorKeyMismatch() })
}

func (reg *CollectorRegistry) IncMultiSignatureVerifications() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignatureVerifications() })
}

func (reg *CollectorRegistry) IncMultiSignatureVerificationFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignatureVerificationFailures() })
}

func (reg *CollectorRegistry) IncEnvelopeV1Issued() {
	reg.dispatch(func(c MetricsCollector) { c.IncEnvelopeV1Issued() })
}

func (reg *CollectorRegistry) IncEnvelopeV2Issued() {
	reg.dispatch(func(c MetricsCollector) { c.IncEnvelopeV2Issued() })
}

func (reg *CollectorRegistry) SetEnvelopeV2AdoptionRatio(r2 float64) {
	reg.dispatch(func(c MetricsCollector) { c.SetEnvelopeV2AdoptionRatio(r2) })
}

func (reg *CollectorRegistry) IncEnvelopeDigestMismatch() {
	reg.dispatch(func(c MetricsCollector) { c.IncEnvelopeDigestMismatch() })
}

func (reg *CollectorRegistry) IncEnvelopeDigestMismatchReason(reason string) {
	reg.dispatch(func(c MetricsCollector) { c.IncEnvelopeDigestMismatchReason(reason) })
}

func (reg *CollectorRegistry) ObserveEnvelopeIssuanceCadence(seconds float64) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveEnvelopeIssuanceCadence(seconds) })
}

func (reg *CollectorRegistry) SetEnvelopeV1SunsetPhase(phase int) {
	reg.dispatch(func(c MetricsCollector) { c.SetEnvelopeV1SunsetPhase(phase) })
}

func (reg *CollectorRegistry) SetSunsetPhaseSatisfactionProgress(p float64) {
	reg.dispatch(func(c MetricsCollector) { c.SetSunsetPhaseSatisfactionProgress(p) })
}

func (reg *CollectorRegistry) IncEnvelopeRawPOAEmbedded() {
	reg.dispatch(func(c MetricsCollector) { c.IncEnvelopeRawPOAEmbedded() })
}

func (reg *CollectorRegistry) IncEnvelopeRawPOATooLarge() {
	reg.dispatch(func(c MetricsCollector) { c.IncEnvelopeRawPOATooLarge() })
}

func (reg *CollectorRegistry) IncMultiSignatureStructuralFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignatureStructuralFailures() })
}

func (reg *CollectorRegistry) IncMultiSignatureDigestFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignatureDigestFailures() })
}

func (reg *CollectorRegistry) IncMultiSignaturePublicKeyMissing() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignaturePublicKeyMissing() })
}

func (reg *CollectorRegistry) IncHierDigestIssued() {
	reg.dispatch(func(c MetricsCollector) { c.IncHierDigestIssued() })
}

func (reg *CollectorRegistry) IncHierDigestParentDigestMissing() {
	reg.dispatch(func(c MetricsCollector) { c.IncHierDigestParentDigestMissing() })
}

func (reg *CollectorRegistry) IncHierDigestVersionMismatch() {
	reg.dispatch(func(c MetricsCollector) { c.IncHierDigestVersionMismatch() })
}

func (reg *CollectorRegistry) IncMultiSignatureInvalidSignatureFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignatureInvalidSignatureFailures() })
}

func (reg *CollectorRegistry) IncMultiSignatureThresholdFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignatureThresholdFailures() })
}

func (reg *CollectorRegistry) IncViolation(cat interface{}) {
	reg.dispatch(func(c MetricsCollector) { c.IncViolation(cat) })
}

func (reg *CollectorRegistry) IncMultiSignatureWeightFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncMultiSignatureWeightFailures() })
}

func (reg *CollectorRegistry) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveMultiSignatureVerificationLatency(d) })
}

func (reg *CollectorRegistry) ObserveMultiSignatureBatchSize(size int) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveMultiSignatureBatchSize(size) })
}

func (reg *CollectorRegistry) ObserveMultiSignatureAggregateLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveMultiSignatureAggregateLatency(d) })
}

func (reg *CollectorRegistry) IncRevocationIntegrityFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationIntegrityFailures() })
}

func (reg *CollectorRegistry) IncSignaturePublicKeyMissing() {
	reg.dispatch(func(c MetricsCollector) { c.IncSignaturePublicKeyMissing() })
}

func (reg *CollectorRegistry) IncCryptoSignatureMissing() {
	reg.dispatch(func(c MetricsCollector) { c.IncCryptoSignatureMissing() })
}

func (reg *CollectorRegistry) IncAnchorAttempts() {
	reg.dispatch(func(c MetricsCollector) { c.IncAnchorAttempts() })
}

func (reg *CollectorRegistry) IncCombinedAnchorEmitted() {
	reg.dispatch(func(c MetricsCollector) { c.IncCombinedAnchorEmitted() })
}

func (reg *CollectorRegistry) IncCombinedAnchorFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncCombinedAnchorFailures() })
}

func (reg *CollectorRegistry) IncAnchorFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncAnchorFailures() })
}

func (reg *CollectorRegistry) IncExternalAnchorForcedFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncExternalAnchorForcedFailures() })
}

func (reg *CollectorRegistry) IncObligationsExecuted() {
	reg.dispatch(func(c MetricsCollector) { c.IncObligationsExecuted() })
}

func (reg *CollectorRegistry) IncObligationsFailed() {
	reg.dispatch(func(c MetricsCollector) { c.IncObligationsFailed() })
}

func (reg *CollectorRegistry) ObserveObligationLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveObligationLatency(d) })
}

func (reg *CollectorRegistry) IncMandatoryObligationFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncMandatoryObligationFailures() })
}

func (reg *CollectorRegistry) IncReplayHits() {
	reg.dispatch(func(c MetricsCollector) { c.IncReplayHits() })
}

func (reg *CollectorRegistry) IncReplayMisses() {
	reg.dispatch(func(c MetricsCollector) { c.IncReplayMisses() })
}

func (reg *CollectorRegistry) IncReplayStoreErrors() {
	reg.dispatch(func(c MetricsCollector) { c.IncReplayStoreErrors() })
}

func (reg *CollectorRegistry) ObserveReplayStoreLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveReplayStoreLatency(d) })
}

func (reg *CollectorRegistry) SetReplayWALPending(p int) {
	reg.dispatch(func(c MetricsCollector) { c.SetReplayWALPending(p) })
}

func (reg *CollectorRegistry) ObserveReplayWALFlushLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveReplayWALFlushLatency(d) })
}

func (reg *CollectorRegistry) ObserveReplayWALSnapshotDuration(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveReplayWALSnapshotDuration(d) })
}

func (reg *CollectorRegistry) IncCapabilityAnchorEmitted() {
	reg.dispatch(func(c MetricsCollector) { c.IncCapabilityAnchorEmitted() })
}

func (reg *CollectorRegistry) IncCapabilityAnchorSkipped() {
	reg.dispatch(func(c MetricsCollector) { c.IncCapabilityAnchorSkipped() })
}

func (reg *CollectorRegistry) IncCapabilityRegistryHashChanged() {
	reg.dispatch(func(c MetricsCollector) { c.IncCapabilityRegistryHashChanged() })
}

func (reg *CollectorRegistry) IncCapabilityAnchorAlgorithm(algo string) {
	reg.dispatch(func(c MetricsCollector) { c.IncCapabilityAnchorAlgorithm(algo) })
}

func (reg *CollectorRegistry) SetCapabilityAnchorAlgorithmRatio(algo string, ratio float64) {
	reg.dispatch(func(c MetricsCollector) { c.SetCapabilityAnchorAlgorithmRatio(algo, ratio) })
}

func (reg *CollectorRegistry) SetCapabilityAnchorLastWriteUnix(ts uint64) {
	reg.dispatch(func(c MetricsCollector) { c.SetCapabilityAnchorLastWriteUnix(ts) })
}

func (reg *CollectorRegistry) IncCapabilityEnforceAllowed() {
	reg.dispatch(func(c MetricsCollector) { c.IncCapabilityEnforceAllowed() })
}

func (reg *CollectorRegistry) IncCapabilityEnforceDenied() {
	reg.dispatch(func(c MetricsCollector) { c.IncCapabilityEnforceDenied() })
}

func (reg *CollectorRegistry) IncPEPEnforcements(allowed bool, actionType string) {
	reg.dispatch(func(c MetricsCollector) { c.IncPEPEnforcements(allowed, actionType) })
}

func (reg *CollectorRegistry) IncPEPViolations(violationType, severity string) {
	reg.dispatch(func(c MetricsCollector) { c.IncPEPViolations(violationType, severity) })
}

func (reg *CollectorRegistry) ObservePEPEnforcementLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObservePEPEnforcementLatency(d) })
}

func (reg *CollectorRegistry) SetPEPAuditBufferSize(enforcement, violation int) {
	reg.dispatch(func(c MetricsCollector) { c.SetPEPAuditBufferSize(enforcement, violation) })
}

func (reg *CollectorRegistry) IncModelLimitExceeded() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelLimitExceeded() })
}

func (reg *CollectorRegistry) IncModelOutputLimitExceeded() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelOutputLimitExceeded() })
}

func (reg *CollectorRegistry) IncModelRateLimitExceeded() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelRateLimitExceeded() })
}

func (reg *CollectorRegistry) IncModelUnknown() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelUnknown() })
}

func (reg *CollectorRegistry) IncModelLimitSurge() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelLimitSurge() })
}

func (reg *CollectorRegistry) IncModelUserInputLimitExceeded() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelUserInputLimitExceeded() })
}

func (reg *CollectorRegistry) IncModelUserOutputLimitExceeded() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelUserOutputLimitExceeded() })
}

func (reg *CollectorRegistry) IncModelUserRateLimitExceeded() {
	reg.dispatch(func(c MetricsCollector) { c.IncModelUserRateLimitExceeded() })
}

func (reg *CollectorRegistry) IncScopeViolations() {
	reg.dispatch(func(c MetricsCollector) { c.IncScopeViolations() })
}

func (reg *CollectorRegistry) IncRestrictionViolations() {
	reg.dispatch(func(c MetricsCollector) { c.IncRestrictionViolations() })
}

func (reg *CollectorRegistry) IncUnauthorized() {
	reg.dispatch(func(c MetricsCollector) { c.IncUnauthorized() })
}

func (reg *CollectorRegistry) IncExpired() {
	reg.dispatch(func(c MetricsCollector) { c.IncExpired() })
}

func (reg *CollectorRegistry) IncRevoked() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevoked() })
}

func (reg *CollectorRegistry) IncDelegationStatusTransitions() {
	reg.dispatch(func(c MetricsCollector) { c.IncDelegationStatusTransitions() })
}

func (reg *CollectorRegistry) IncDelegationStatusTransitionFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncDelegationStatusTransitionFailures() })
}

func (reg *CollectorRegistry) IncTokenStatusTransitions() {
	reg.dispatch(func(c MetricsCollector) { c.IncTokenStatusTransitions() })
}

func (reg *CollectorRegistry) IncTokenStatusTransitionFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncTokenStatusTransitionFailures() })
}

func (reg *CollectorRegistry) RecordDecision(action string, resource string, outcome string) {
	reg.dispatch(func(c MetricsCollector) { c.RecordDecision(action, resource, outcome) })
}

func (reg *CollectorRegistry) RecordDecisionWithReason(action string, resource string, outcome string, reason string) {
	reg.dispatch(func(c MetricsCollector) { c.RecordDecisionWithReason(action, resource, outcome, reason) })
}

func (reg *CollectorRegistry) RecordLifecycleTransition(entityType string, oldStatus string, newStatus string, outcome string) {
	reg.dispatch(func(c MetricsCollector) { c.RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome) })
}

func (reg *CollectorRegistry) ObserveLifecycleTransitionLatency(entityType string, outcome string, d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveLifecycleTransitionLatency(entityType, outcome, d) })
}

func (reg *CollectorRegistry) SetLifecycleTransitionLatencyQuantile(entityType string, outcome string, quantile string, value float64) {
	reg.dispatch(func(c MetricsCollector) {
		c.SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile, value)
	})
}

func (reg *CollectorRegistry) IncCapabilityDiffRequests() {
	reg.dispatch(func(c MetricsCollector) { c.IncCapabilityDiffRequests() })
}

func (reg *CollectorRegistry) ObserveCapabilityDiffLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveCapabilityDiffLatency(d) })
}

func (reg *CollectorRegistry) IncRevocationWorkflowInitiated() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowInitiated() })
}

func (reg *CollectorRegistry) IncRevocationWorkflowInitiationFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowInitiationFailures() })
}

func (reg *CollectorRegistry) IncRevocationWorkflowApprovals() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowApprovals() })
}

func (reg *CollectorRegistry) IncRevocationWorkflowApprovalFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowApprovalFailures() })
}

func (reg *CollectorRegistry) IncRevocationWorkflowQuorumSatisfied() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowQuorumSatisfied() })
}

func (reg *CollectorRegistry) IncRevocationWorkflowCanceled() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowCanceled() })
}

func (reg *CollectorRegistry) IncRevocationWorkflowCancellationFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowCancellationFailures() })
}

func (reg *CollectorRegistry) IncRevocationWorkflowUnauthorized() {
	reg.dispatch(func(c MetricsCollector) { c.IncRevocationWorkflowUnauthorized() })
}

func (reg *CollectorRegistry) IncEvidenceAttachment() {
	reg.dispatch(func(c MetricsCollector) { c.IncEvidenceAttachment() })
}

func (reg *CollectorRegistry) IncEvidenceAttachmentFailures() {
	reg.dispatch(func(c MetricsCollector) { c.IncEvidenceAttachmentFailures() })
}

func (reg *CollectorRegistry) SetEvidenceHashesPerPOA(poaID string, count int) {
	reg.dispatch(func(c MetricsCollector) { c.SetEvidenceHashesPerPOA(poaID, count) })
}

func (reg *CollectorRegistry) IncDelegationGraphExports() {
	reg.dispatch(func(c MetricsCollector) { c.IncDelegationGraphExports() })
}

func (reg *CollectorRegistry) SetDelegationGraphNodeCount(count int) {
	reg.dispatch(func(c MetricsCollector) { c.SetDelegationGraphNodeCount(count) })
}

func (reg *CollectorRegistry) IncCascadeRevocationTriggered() {
	reg.dispatch(func(c MetricsCollector) { c.IncCascadeRevocationTriggered() })
}

func (reg *CollectorRegistry) IncCascadeDescendantsProcessed() {
	reg.dispatch(func(c MetricsCollector) { c.IncCascadeDescendantsProcessed() })
}

func (reg *CollectorRegistry) ObserveCascadeProcessingLatency(d time.Duration) {
	reg.dispatch(func(c MetricsCollector) { c.ObserveCascadeProcessingLatency(d) })
}

func (reg *CollectorRegistry) IncCascadeDepthLimitReached() {
	reg.dispatch(func(c MetricsCollector) { c.IncCascadeDepthLimitReached() })
}

func (reg *CollectorRegistry) IncCascadeBatchProcessed() {
	reg.dispatch(func(c MetricsCollector) { c.IncCascadeBatchProcessed() })
}

func (reg *CollectorRegistry) SetCascadeMaxDepthReached(depth int) {
	reg.dispatch(func(c MetricsCollector) { c.SetCascadeMaxDepthReached(depth) })
}

func (reg *CollectorRegistry) IncCascadeProcessingErrors() {
	reg.dispatch(func(c MetricsCollector) { c.IncCascadeProcessingErrors() })
}
