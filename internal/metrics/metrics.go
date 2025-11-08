package metrics

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	observability "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/observability"
)

const (
	unknownOutcome = "unknown"
	otherReason    = "other"
)

// Metrics defines the minimal instrumentation surface for early Phase 2.
type Metrics interface {
	IncDelegationsCreated()
	// IncDelegationsPartiallyRevoked increments counter for delegations transitioned to partially_revoked state.
	IncDelegationsPartiallyRevoked()
	// IncDelegationDepthExceeded increments counter when a delegation append exceeds configured max depth.
	IncDelegationDepthExceeded()
	// SetMaxObservedDelegationDepth sets a gauge tracking the maximum delegation chain depth observed (root=1).
	SetMaxObservedDelegationDepth(depth int)
	ObserveValidationLatency(d time.Duration)
	IncSignaturesIssued()
	IncSignatureIssueFailures()
	IncSignatureVerifications()
	IncSignatureVerificationFailures()
	// Attestation proof counters (prototype Task 9)
	IncAttestationProofIssued()
	IncAttestationProofIssueFailures()
	IncAttestationProofVerifications()
	IncAttestationProofVerificationFailures()
	IncAttestationProofDigestMismatch()
	ObserveAttestationProofVerificationLatency(d time.Duration)
	// ObserveAttestationProofIssueLatency records issuance latency (distinct from verification latency)
	ObserveAttestationProofIssueLatency(d time.Duration)
	// IncAttestationProofVerificationFailureReason increments labeled failure reason counter (digest_mismatch|algo_mismatch|key_mismatch|missing_anchor|other)
	IncAttestationProofVerificationFailureReason(reason string)
	// BLS Proof-of-Possession metrics
	IncBLSPoPChallengesIssued()
	IncBLSPoPVerifications()
	IncBLSPoPVerificationFailures()
	// Trust anchor enforcement granular counters (attestation issuer binding failures)
	IncAttestationProofTrustAnchorMissing()
	IncAttestationProofTrustAnchorAlgorithmMismatch()
	IncAttestationProofTrustAnchorKeyMismatch()
	IncRevocationIntegrityFailures()
	IncSignaturePublicKeyMissing() // soft skip: signature present but key id not found in key ring
	// Detached signature enforcement counters
	IncCryptoSignatureMissing() // missing detached signature artifact when enforcement enabled
	// Envelope issuance version counters (migration observability)
	IncEnvelopeV1Issued()
	IncEnvelopeV2Issued()
	// Hierarchical digest V4 domain counters (Beta MVP)
	IncHierDigestIssued()
	IncHierDigestParentDigestMissing()
	IncHierDigestVersionMismatch()
	// SetEnvelopeV2AdoptionRatio sets a gauge expressing V2 issuance ratio (0.0-1.0) over sliding window or total counters.
	SetEnvelopeV2AdoptionRatio(r float64)
	// IncEnvelopeDigestMismatch increments counter when canonical digest recomputed at verification differs from stored digest.
	IncEnvelopeDigestMismatch()
	// IncEnvelopeDigestMismatchReason increments labeled mismatch reason counter (canonicalization_error|tamper_suspected|domain_conflict|other).
	IncEnvelopeDigestMismatchReason(reason string)
	// ObserveEnvelopeIssuanceCadence records seconds elapsed since previous envelope issuance (any version) for migration cadence analysis.
	ObserveEnvelopeIssuanceCadence(seconds float64)
	// SetEnvelopeV1SunsetPhase sets gauge enumerating sunset lifecycle phase (0 Pilot,1 Broad,2 Stabilization,3 SoftDep,4 Sunset,5 PostVerify).
	SetEnvelopeV1SunsetPhase(phase int)
	// SetSunsetPhaseSatisfactionProgress sets a gauge (0..1) representing fraction of current phase promotion window satisfied.
	SetSunsetPhaseSatisfactionProgress(p float64)
	// RawPOA embedding counters (RFC0115 sec3.item2 implementation progress)
	IncEnvelopeRawPOAEmbedded() // successful embedding of canonical RawPOA into EnvelopeV2
	IncEnvelopeRawPOATooLarge() // attempted embedding omitted due to size > GAUTH_MAX_RAW_POA_BYTES
	IncMultiSignatureVerifications()
	IncMultiSignatureVerificationFailures()
	// Granular multi-signature failure categorization (additive to generic failure counters)
	IncMultiSignatureStructuralFailures()       // structural preconditions failed (e.g. duplicate signer, insufficient signers, bad weight map)
	IncMultiSignatureDigestFailures()           // canonical digest computation failed
	IncMultiSignaturePublicKeyMissing()         // signer public key missing (strict mode may treat as hard failure)
	IncMultiSignatureInvalidSignatureFailures() // signature present but cryptographic verification failed
	IncMultiSignatureThresholdFailures()        // valid signatures < count-based threshold
	IncViolation(cat interface{})               // generic category increment (observability categories)
	// Multi-signature weighted threshold failures (valid signatures present but cumulative weight < threshold)
	IncMultiSignatureWeightFailures()
	// ObserveMultiSignatureVerificationLatency records latency of multi-signature verification path (distinct from generic validation latency)
	ObserveMultiSignatureVerificationLatency(d time.Duration)
	// ObserveMultiSignatureBatchSize records batch size of multi-signature operations (aggregation or verification sets).
	ObserveMultiSignatureBatchSize(size int)
	// ObserveMultiSignatureAggregateLatency records latency of aggregate signature computation (distinct from verification path).
	ObserveMultiSignatureAggregateLatency(d time.Duration)
	IncAnchorAttempts()
	IncCombinedAnchorEmitted()
	IncCombinedAnchorFailures()
	IncAnchorFailures()
	// IncExternalAnchorForcedFailures increments counter of failures explicitly forced via GAUTH_CAP_EXTERNAL_ANCHOR_FAILS_BEFORE_SUCCESS
	// distinguishing deterministic test harness failures from probabilistic model failures.
	IncExternalAnchorForcedFailures()
	// Obligations / advice execution metrics
	IncObligationsExecuted()
	IncObligationsFailed()
	// ObserveObligationLatency records per-obligation execution latency (nanoseconds -> reservoir/histogram).
	ObserveObligationLatency(d time.Duration)
	// IncMandatoryObligationFailures increments counter when a mandatory obligation failure flips an allow decision to deny.
	IncMandatoryObligationFailures()
	IncReplayHits()
	IncReplayMisses()
	IncReplayStoreErrors()
	ObserveReplayStoreLatency(d time.Duration)
	// WAL durability metrics (RB1 Phase): pending buffered entries and flush latency.
	SetReplayWALPending(n int)
	ObserveReplayWALFlushLatency(d time.Duration)
	// ObserveReplayWALSnapshotDuration measures the duration of creating a snapshot
	// of the replay WAL (copying state before compaction/rotation) exclusive of
	// the subsequent flush/rotation latency which is tracked separately.
	ObserveReplayWALSnapshotDuration(d time.Duration)
	// Capability diff endpoint metrics (RB13)
	// IncCapabilityDiffRequests increments the capability diff requests counter.
	IncCapabilityDiffRequests()
	// ObserveCapabilityDiffLatency records diff computation latency (hash + compare phase) excluding serialization.
	ObserveCapabilityDiffLatency(d time.Duration)
	// Capability registry anchoring metrics (Phase 2 governance integrity)
	IncCapabilityAnchorEmitted()
	IncCapabilityAnchorSkipped()
	IncCapabilityRegistryHashChanged()
	// SetCapabilityAnchorLastWriteUnix records the unix epoch seconds of the last successful capability anchor artifact emission.
	// Implementations MAY expose this as a gauge. Zero value indicates no emission yet.
	SetCapabilityAnchorLastWriteUnix(ts uint64)
	// IncCapabilityAnchorAlgorithm increments a labeled counter for each cryptographic algorithm
	// present during a capability anchor emission. This provides per-algorithm visibility
	// on anchor events (e.g. ed25519 vs ecdsa-p256 vs future bls aggregated paths).
	// The caller SHOULD pass a stable lowercase algorithm identifier. Empty values are
	// normalized to "_". Multiple algorithms may be incremented for a single anchor emission.
	IncCapabilityAnchorAlgorithm(algo string)
	// SetCapabilityAnchorAlgorithmRatio sets a gauge expressing per-algorithm anchor emission ratio (0..1) vs total across all algorithms.
	// Implementations MAY approximate over a sliding window; callers should pre-compute ratio externally.
	SetCapabilityAnchorAlgorithmRatio(algo string, ratio float64)
	// Capability enforcement decision counters (added after initial generic violation-only path).
	// These provide explicit allow/deny monotonic counters for capability matrix enforcement decisions
	// to enable ratio calculations and alerting without scraping audit logs.
	IncCapabilityEnforceAllowed()
	IncCapabilityEnforceDenied()
	// IncModelLimitExceeded increments dedicated model limit exceed counter (model governance).
	IncModelLimitExceeded()
	// IncModelOutputLimitExceeded increments output token exceed counter.
	IncModelOutputLimitExceeded()
	// IncModelRateLimitExceeded increments per-minute request rate exceed counter.
	IncModelRateLimitExceeded()
	// IncModelUnknown increments counter for strict-mode unknown model validation denials.
	IncModelUnknown()
	// IncModelLimitSurge increments counter when surge detection triggers (exceeds moving average threshold).
	IncModelLimitSurge()
	// Per-user (subject scoped) model limit exceed counters
	IncModelUserInputLimitExceeded()
	IncModelUserOutputLimitExceeded()
	IncModelUserRateLimitExceeded()
	// Extended instrumentation (Phase 2B): violation & lifecycle counters
	IncScopeViolations()
	IncRestrictionViolations()
	IncUnauthorized()
	IncExpired()
	IncRevoked()
	// Lifecycle status transition instrumentation
	IncDelegationStatusTransitions()
	IncDelegationStatusTransitionFailures()
	IncTokenStatusTransitions()
	IncTokenStatusTransitionFailures()
	// Dual-control revocation workflow metrics (hierarchical governance observability)
	// IncRevocationWorkflowInitiated increments counter when a pending dual-control revocation is successfully initiated.
	IncRevocationWorkflowInitiated()
	// IncRevocationWorkflowInitiationFailures increments counter when initiation is rejected (invalid state, already pending, etc.).
	IncRevocationWorkflowInitiationFailures()
	// IncRevocationWorkflowApprovals increments counter when a new unique approval is recorded (duplicate approvals ignored).
	IncRevocationWorkflowApprovals()
	// IncRevocationWorkflowApprovalFailures increments counter when an approval attempt fails (unauthorized, no pending revocation, finalized/canceled).
	IncRevocationWorkflowApprovalFailures()
	// IncRevocationWorkflowQuorumSatisfied increments counter when quorum (count or weight) is satisfied and revocation is finalized.
	IncRevocationWorkflowQuorumSatisfied()
	// IncRevocationWorkflowCanceled increments counter when a pending revocation is canceled.
	IncRevocationWorkflowCanceled()
	// IncRevocationWorkflowCancellationFailures increments counter when cancellation attempt fails (unauthorized, no pending, already finalized, etc.).
	IncRevocationWorkflowCancellationFailures()
	// IncRevocationWorkflowUnauthorized increments counter for any unauthorized dual-control revocation action attempt (initiate/approve/cancel).
	IncRevocationWorkflowUnauthorized()
	// Evidence attachment metrics (forensic evidentiary strengthening)
	// IncEvidenceAttachment increments counter when a valid new evidence hash is attached to a POA (unique additions only).
	IncEvidenceAttachment()
	// IncEvidenceAttachmentFailures increments counter when an evidence attachment attempt fails (invalid format, not found, duplicate-only submission, etc.).
	IncEvidenceAttachmentFailures()
	// SetEvidenceHashesPerPOA sets a gauge (labeled by poa_id in Prometheus implementation) with current evidence hash count for a POA.
	SetEvidenceHashesPerPOA(poaID string, n int)
	// Delegation graph export metrics
	IncDelegationGraphExports()        // count graph export requests
	SetDelegationGraphNodeCount(n int) // gauge of current nodes in exported graph
	// RecordDecision captures an authorization decision with action/resource/outcome labels.
	// outcome SHOULD be one of: allow, deny, error, success, failure, noop.
	RecordDecision(action, resource, outcome string)
	// RecordDecisionWithReason adds a reason label (e.g. status_change, invalid_transition, init, noop).
	RecordDecisionWithReason(action, resource, outcome, reason string)
	// RecordLifecycleTransition captures a lifecycle state change with labels.
	// entityType: token | delegation; outcome: success|noop|failure
	RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome string)
	// ObserveLifecycleTransitionLatency records the latency of a lifecycle transition (token/delegation status update)
	// outcome aligns with lifecycle transition outcome labels: success|failure|noop|unknown
	ObserveLifecycleTransitionLatency(entityType, outcome string, d time.Duration)
	// SetLifecycleTransitionLatencyQuantile sets a gauge labeled with entity,outcome,quantile (e.g. p50,p95,p99) for latency distribution.
	// Computation performed externally (e.g. using reservoir samples); implementations simply store/update gauge.
	SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile string, value float64)
	// Cascade revocation metrics (Phase 2b)
	// IncCascadeRevocationTriggered increments counter when a cascade revocation is initiated (parent POA revoked)
	IncCascadeRevocationTriggered()
	// IncCascadeDescendantsProcessed increments counter for each descendant POA processed during cascade
	IncCascadeDescendantsProcessed()
	// ObserveCascadeProcessingLatency records total time to process cascade revocation for a parent POA
	ObserveCascadeProcessingLatency(d time.Duration)
	// IncCascadeDepthLimitReached increments counter when cascade processing hits configured max depth limit
	IncCascadeDepthLimitReached()
	// IncCascadeBatchProcessed increments counter for each batch processed during cascade operation
	IncCascadeBatchProcessed()
	// SetCascadeMaxDepthReached sets gauge tracking deepest cascade depth processed in current session
	SetCascadeMaxDepthReached(depth int)
	// IncCascadeProcessingErrors increments counter for cascade processing errors (failures to update descendants)
	IncCascadeProcessingErrors()
	// Optional adapter-specific setters (Prometheus) for capability anchor SLA freshness. Memory implementation intentionally omits.
	// These are not required by all implementations; callers SHOULD use type assertions.
	// SetCapabilityAnchorAgeSeconds(age uint64)
	// SetCapabilityAnchorStale(stale bool)
}

// Noop provides a do-nothing implementation used when instrumentation is disabled.
var Noop Metrics = noop{}

type noop struct{}

func (n noop) IncDelegationsCreated()                                     {}
func (n noop) IncDelegationsPartiallyRevoked()                            {}
func (n noop) IncDelegationDepthExceeded()                                {}
func (n noop) SetMaxObservedDelegationDepth(depth int)                    {}
func (n noop) ObserveValidationLatency(d time.Duration)                   {}
func (n noop) IncSignaturesIssued()                                       {}
func (n noop) IncSignatureIssueFailures()                                 {}
func (n noop) IncSignatureVerifications()                                 {}
func (n noop) IncSignatureVerificationFailures()                          {}
func (n noop) IncAttestationProofIssued()                                 {}
func (n noop) IncAttestationProofIssueFailures()                          {}
func (n noop) IncAttestationProofVerifications()                          {}
func (n noop) IncAttestationProofVerificationFailures()                   {}
func (n noop) IncAttestationProofDigestMismatch()                         {}
func (n noop) ObserveAttestationProofVerificationLatency(d time.Duration) {}
func (n noop) ObserveAttestationProofIssueLatency(d time.Duration)        {}
func (n noop) IncAttestationProofVerificationFailureReason(reason string) {}
func (n noop) IncBLSPoPChallengesIssued()                                 {}
func (n noop) IncBLSPoPVerifications()                                    {}
func (n noop) IncBLSPoPVerificationFailures()                             {}
func (n noop) IncAttestationProofTrustAnchorMissing()                     {}
func (n noop) IncAttestationProofTrustAnchorAlgorithmMismatch()           {}
func (n noop) IncAttestationProofTrustAnchorKeyMismatch()                 {}
func (n noop) IncMultiSignatureVerifications()                            {}
func (n noop) IncMultiSignatureVerificationFailures()                     {}
func (n noop) IncEnvelopeV1Issued()                                       {}
func (n noop) IncEnvelopeV2Issued()                                       {}
func (n noop) SetEnvelopeV2AdoptionRatio(r float64)                       {}
func (n noop) IncEnvelopeDigestMismatch()                                 {}
func (n noop) IncEnvelopeDigestMismatchReason(reason string)              {}
func (n noop) ObserveEnvelopeIssuanceCadence(seconds float64)             {}
func (n noop) SetEnvelopeV1SunsetPhase(phase int)                         {}
func (n noop) SetSunsetPhaseSatisfactionProgress(p float64)               {}
func (n noop) IncEnvelopeRawPOAEmbedded()                                 {}
func (n noop) IncEnvelopeRawPOATooLarge()                                 {}
func (n noop) IncMultiSignatureStructuralFailures()                       {}
func (n noop) IncMultiSignatureDigestFailures()                           {}
func (n noop) IncMultiSignaturePublicKeyMissing()                         {}

// Hierarchical digest metrics (V4)
func (n noop) IncHierDigestIssued()                                                          {}
func (n noop) IncHierDigestParentDigestMissing()                                             {}
func (n noop) IncHierDigestVersionMismatch()                                                 {}
func (n noop) IncMultiSignatureInvalidSignatureFailures()                                    {}
func (n noop) IncMultiSignatureThresholdFailures()                                           {}
func (n noop) IncViolation(cat interface{})                                                  {}
func (n noop) IncMultiSignatureWeightFailures()                                              {}
func (n noop) ObserveMultiSignatureVerificationLatency(d time.Duration)                      {}
func (n noop) ObserveMultiSignatureBatchSize(size int)                                       {}
func (n noop) ObserveMultiSignatureAggregateLatency(d time.Duration)                         {}
func (n noop) IncRevocationIntegrityFailures()                                               {}
func (n noop) IncSignaturePublicKeyMissing()                                                 {}
func (n noop) IncCryptoSignatureMissing()                                                    {}
func (n noop) IncAnchorAttempts()                                                            {}
func (n noop) IncCombinedAnchorEmitted()                                                     {}
func (n noop) IncCombinedAnchorFailures()                                                    {}
func (n noop) IncAnchorFailures()                                                            {}
func (n noop) IncExternalAnchorForcedFailures()                                              {}
func (n noop) IncObligationsExecuted()                                                       {}
func (n noop) IncObligationsFailed()                                                         {}
func (n noop) ObserveObligationLatency(d time.Duration)                                      {}
func (n noop) IncMandatoryObligationFailures()                                               {}
func (n noop) IncReplayHits()                                                                {}
func (n noop) IncReplayMisses()                                                              {}
func (n noop) IncReplayStoreErrors()                                                         {}
func (n noop) ObserveReplayStoreLatency(d time.Duration)                                     {}
func (n noop) SetReplayWALPending(p int)                                                     {}
func (n noop) ObserveReplayWALFlushLatency(d time.Duration)                                  {}
func (n noop) ObserveReplayWALSnapshotDuration(d time.Duration)                              {}
func (n noop) IncCapabilityAnchorEmitted()                                                   {}
func (n noop) IncCapabilityAnchorSkipped()                                                   {}
func (n noop) IncCapabilityRegistryHashChanged()                                             {}
func (n noop) IncCapabilityAnchorAlgorithm(algo string)                                      {}
func (n noop) SetCapabilityAnchorAlgorithmRatio(algo string, ratio float64)                  {}
func (n noop) SetCapabilityAnchorLastWriteUnix(ts uint64)                                    {}
func (n noop) IncCapabilityEnforceAllowed()                                                  {}
func (n noop) IncCapabilityEnforceDenied()                                                   {}
func (n noop) IncModelLimitExceeded()                                                        {}
func (n noop) IncModelOutputLimitExceeded()                                                  {}
func (n noop) IncModelRateLimitExceeded()                                                    {}
func (n noop) IncModelUnknown()                                                              {}
func (n noop) IncModelLimitSurge()                                                           {}
func (n noop) IncModelUserInputLimitExceeded()                                               {}
func (n noop) IncModelUserOutputLimitExceeded()                                              {}
func (n noop) IncModelUserRateLimitExceeded()                                                {}
func (n noop) IncScopeViolations()                                                           {}
func (n noop) IncRestrictionViolations()                                                     {}
func (n noop) IncUnauthorized()                                                              {}
func (n noop) IncExpired()                                                                   {}
func (n noop) IncRevoked()                                                                   {}
func (n noop) IncDelegationStatusTransitions()                                               {}
func (n noop) IncDelegationStatusTransitionFailures()                                        {}
func (n noop) IncTokenStatusTransitions()                                                    {}
func (n noop) IncTokenStatusTransitionFailures()                                             {}
func (n noop) RecordDecision(action, resource, outcome string)                               {}
func (n noop) RecordDecisionWithReason(action, resource, outcome, reason string)             {}
func (n noop) RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome string)    {}
func (n noop) ObserveLifecycleTransitionLatency(entityType, outcome string, d time.Duration) {}
func (n noop) SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile string, value float64) {
}
func (n noop) IncCapabilityDiffRequests()                      {}
func (n noop) ObserveCapabilityDiffLatency(d time.Duration)    {}
func (n noop) IncRevocationWorkflowInitiated()                 {}
func (n noop) IncRevocationWorkflowInitiationFailures()        {}
func (n noop) IncRevocationWorkflowApprovals()                 {}
func (n noop) IncRevocationWorkflowApprovalFailures()          {}
func (n noop) IncRevocationWorkflowQuorumSatisfied()           {}
func (n noop) IncRevocationWorkflowCanceled()                  {}
func (n noop) IncRevocationWorkflowCancellationFailures()      {}
func (n noop) IncRevocationWorkflowUnauthorized()              {}
func (n noop) IncEvidenceAttachment()                          {}
func (n noop) IncEvidenceAttachmentFailures()                  {}
func (n noop) SetEvidenceHashesPerPOA(poaID string, count int) {}
func (n noop) IncDelegationGraphExports()                      {}
func (n noop) SetDelegationGraphNodeCount(count int)           {}

// Cascade revocation noop implementations
func (n noop) IncCascadeRevocationTriggered()                  {}
func (n noop) IncCascadeDescendantsProcessed()                 {}
func (n noop) ObserveCascadeProcessingLatency(d time.Duration) {}
func (n noop) IncCascadeDepthLimitReached()                    {}
func (n noop) IncCascadeBatchProcessed()                       {}
func (n noop) SetCascadeMaxDepthReached(depth int)             {}
func (n noop) IncCascadeProcessingErrors()                     {}

// Memory is a simple in-process metrics collector used for tests and benchmarks.
// It is intentionally minimal and lock-free for write paths using atomics.
type Memory struct {
	delegationsCreated uint64
	// validation count & total nanoseconds to derive average latency.
	validationCount   uint64
	validationTotalNS uint64
	// track max latency observed (nanoseconds) for quick regression signal.
	validationMaxNS uint64
	// track min latency observed
	validationMinNS uint64
	// simple fixed-size reservoir sample of validation latencies (nanoseconds) for percentile estimation.
	// We keep a ring buffer of the most recent N samples (power-of-two size for cheap masking) and an index.
	reservoir      [256]uint64
	reservoirIndex uint64 // monotonically increasing; position = idx & 255
	// signature counters
	signaturesIssued              uint64
	signatureIssueFailures        uint64
	signatureVerifications        uint64
	signatureVerificationFailures uint64
	envelopeV1Issued              uint64
	envelopeV2Issued              uint64
	envelopeDigestMismatch        uint64
	envelopeRawPOAEmbedded        uint64 // count of envelopes embedding RawPOA
	envelopeRawPOATooLarge        uint64 // count of embedding attempts omitted due to size
	// Attestation proof counters (Task 9)
	attestationProofIssued                     uint64
	attestationProofIssueFailures              uint64
	attestationProofVerifications              uint64
	attestationProofVerificationFailures       uint64
	attestationProofDigestMismatch             uint64
	attestationProofVerificationLatencyCount   uint64
	attestationProofVerificationLatencyTotalNS uint64
	attestationProofVerificationLatencyMaxNS   uint64
	// Attestation proof issuance latency summary
	attestationProofIssueLatencyCount   uint64
	attestationProofIssueLatencyTotalNS uint64
	attestationProofIssueLatencyMaxNS   uint64
	// Attestation proof verification failure reasons (labeled): reason -> count
	attestationProofVerificationFailureReasons   map[string]uint64
	attestationProofVerificationFailureReasonsMu sync.Mutex
	// BLS PoP counters
	blsPoPChallengesIssued     uint64
	blsPoPVerifications        uint64
	blsPoPVerificationFailures uint64
	// Trust anchor enforcement granular counters
	attestationProofTrustAnchorMissing           uint64
	attestationProofTrustAnchorAlgorithmMismatch uint64
	attestationProofTrustAnchorKeyMismatch       uint64
	// labeled digest mismatch reasons map (small cardinality, protected by mutex)
	envelopeDigestMismatchReasons          map[string]uint64
	envelopeDigestMismatchReasonsMu        sync.Mutex
	envelopeV2AdoptionRatioBits            uint64 // store math.Float64bits(ratio) atomically
	lastEnvelopeIssuanceUnix               uint64 // unix seconds of last issuance (either version) for cadence computation
	envelopeIssuanceCadenceCount           uint64 // number of cadence observations
	envelopeIssuanceCadenceTotal           uint64 // total floating seconds scaled by 1e9 (nanoseconds) for avg (approx)
	envelopeV1SunsetPhase                  uint64 // current sunset phase enum (0..5)
	sunsetPhaseSatisfactionBits            uint64 // float64 bits storing progress (0..1)
	multiSignatureVerifications            uint64
	multiSignatureVerificationFailures     uint64
	multiSignatureWeightFailures           uint64
	multiSignatureStructuralFailures       uint64
	multiSignatureDigestFailures           uint64
	multiSignaturePublicKeyMissingFailures uint64
	multiSignatureInvalidSignatureFailures uint64
	multiSignatureThresholdFailures        uint64
	multiSignatureBatchSizeCount           uint64
	multiSignatureBatchSizeTotal           uint64 // sum of batch sizes for avg
	multiSignatureBatchSizeMax             uint64
	multiSignatureAggregateLatencyCount    uint64
	multiSignatureAggregateLatencyTotalNS  uint64
	multiSignatureAggregateLatencyMaxNS    uint64
	// revocation integrity failures (revocation chain tamper detection)
	revocationIntegrityFailures      uint64
	signaturePublicKeyMissing        uint64
	cryptoSignatureMissing           uint64
	anchorAttempts                   uint64
	combinedAnchorEmitted            uint64
	combinedAnchorFailures           uint64
	anchorFailures                   uint64
	externalAnchorForcedFailures     uint64 // forced initial failures (deterministic override)
	obligationsExecuted              uint64 // successful obligation/advice executions
	obligationsFailed                uint64 // failed obligation/advice executions
	obligationLatencyCount           uint64 // number of obligation latency observations
	obligationLatencyTotalNS         uint64 // total latency nanoseconds
	obligationLatencyMaxNS           uint64 // max latency observed
	mandatoryObligationFailures      uint64 // count of mandatory failures that flipped decision
	replayHits                       uint64
	replayMisses                     uint64
	replayStoreErrors                uint64
	replayStoreLatencyCount          uint64
	replayStoreLatencyTotalNS        uint64
	replayStoreLatencyMaxNS          uint64
	replayWALPending                 uint64
	replayWALFlushLatencyCount       uint64
	replayWALFlushLatencyTotalNS     uint64
	replayWALFlushLatencyMaxNS       uint64
	replayWALSnapshotDurationCount   uint64
	replayWALSnapshotDurationTotalNS uint64
	replayWALSnapshotDurationMaxNS   uint64
	// Capability anchoring counters
	capabilityAnchorEmitted       uint64
	capabilityAnchorSkipped       uint64
	capabilityRegistryHashChanged uint64
	capabilityAnchorLastWriteUnix uint64 // unix seconds of last successful anchor artifact emission
	// Per-algorithm capability anchor emission counters (labeled by algorithm name)
	anchorAlgoCounts map[string]uint64
	anchorAlgoMu     sync.Mutex
	// Per-algorithm anchor emission ratios (gauge semantics) algorithm->float64 bits
	anchorAlgoRatioBits map[string]uint64
	anchorAlgoRatioMu   sync.Mutex
	// Extended counters
	scopeViolations       uint64
	restrictionViolations uint64
	unauthorized          uint64
	expiredDelegations    uint64
	revokedDelegations    uint64
	// partial revocation (delegation scope reduction without full termination)
	partiallyRevokedDelegations uint64
	delegationDepthExceeded     uint64
	maxObservedDelegationDepth  uint64 // stores max observed delegation chain depth (root=1)
	// Capability enforcement decision counters
	capabilityEnforceAllowed     uint64
	capabilityEnforceDenied      uint64
	modelLimitExceeded           uint64
	modelOutputLimitExceeded     uint64
	modelRateLimitExceeded       uint64
	modelUserInputLimitExceeded  uint64
	modelUserOutputLimitExceeded uint64
	modelUserRateLimitExceeded   uint64
	modelUnknown                 uint64
	modelLimitSurges             uint64
	// lifecycle transition counters
	delegationStatusTransitions        uint64
	delegationStatusTransitionFailures uint64
	tokenStatusTransitions             uint64
	tokenStatusTransitionFailures      uint64
	// Validation failure counters (by reason)
	invalidPayloadFailures    uint64
	unsupportedStatusFailures uint64
	invalidTransitionFailures uint64
	notFoundFailures          uint64
	// Labeled decision counts: key format action|resource|outcome
	decisionCounts map[string]uint64
	decisionMu     sync.Mutex
	// Labeled decision counts with reason: key format action|resource|outcome|reason
	decisionReasonCounts map[string]uint64
	decisionReasonMu     sync.Mutex
	// Labeled lifecycle transition counts: key format entity|old|new|outcome
	lifecycleCounts map[string]uint64
	lifecycleMu     sync.Mutex
	// Lifecycle transition latency summary (aggregate nanoseconds + count + max) per entity type
	// outcome dimension included in key entity|outcome for aggregates.
	lifecycleLatencyTotals map[string]uint64 // total nanoseconds per entity|outcome
	lifecycleLatencyCounts map[string]uint64 // count per entity|outcome
	lifecycleLatencyMax    map[string]uint64 // max nanoseconds per entity|outcome
	lifecycleLatencyMu     sync.Mutex
	// Lifecycle latency reservoir samples per entity|outcome for percentile estimation
	lifecycleLatencyRes map[string]*latencyReservoir
	// Lifecycle latency quantile gauges (entity|outcome|quantile -> float64 bits)
	lifecycleLatencyQuantiles   map[string]uint64
	lifecycleLatencyQuantilesMu sync.Mutex
	// persistence path (optional)
	persistPath string
	// lastPersistUnix stores the last successful persistence UNIX timestamp (seconds).
	lastPersistUnix uint64
	// RB4: policy manifest emitted counter
	policyManifestEmitted uint64
	// Capability diff metrics (RB13)
	capabilityDiffRequests       uint64
	capabilityDiffLatencyCount   uint64
	capabilityDiffLatencyTotalNS uint64
	capabilityDiffLatencyMaxNS   uint64
	// Dual-control revocation workflow counters
	revWorkflowInitiated            uint64
	revWorkflowInitiationFailures   uint64
	revWorkflowApprovals            uint64
	revWorkflowApprovalFailures     uint64
	revWorkflowQuorumSatisfied      uint64
	revWorkflowCanceled             uint64
	revWorkflowCancellationFailures uint64
	revWorkflowUnauthorized         uint64
	// Evidence attachment metrics
	evidenceAttachments        uint64
	evidenceAttachmentFailures uint64
	// per-POA evidence hash counts (only used for snapshot & tests; small cardinality assumption)
	evidenceCountsMu         sync.Mutex
	evidenceCounts           map[string]int
	delegationGraphExports   uint64
	delegationGraphNodeCount uint64
	// Hierarchical digest V4 counters
	hierDigestIssued              uint64
	hierDigestParentDigestMissing uint64
	hierDigestVersionMismatch     uint64
	// Cascade revocation metrics
	cascadeRevocationTriggered      uint64
	cascadeDescendantsProcessed     uint64
	cascadeProcessingLatencyCount   uint64
	cascadeProcessingLatencyTotalNS uint64
	cascadeProcessingLatencyMaxNS   uint64
	cascadeDepthLimitReached        uint64
	cascadeBatchProcessed           uint64
	cascadeMaxDepthReachedGauge     uint64
	cascadeProcessingErrors         uint64
}

// IncEvidenceAttachment increments successful evidence hash attachment counter.
func (m *Memory) IncEvidenceAttachment() { atomic.AddUint64(&m.evidenceAttachments, 1) }

// IncEvidenceAttachmentFailures increments failed evidence hash attachment attempts counter.
func (m *Memory) IncEvidenceAttachmentFailures() { atomic.AddUint64(&m.evidenceAttachmentFailures, 1) }

// SetEvidenceHashesPerPOA records current evidence hash count for a POA in memory map.
func (m *Memory) SetEvidenceHashesPerPOA(poaID string, n int) {
	if poaID == "" {
		return
	}
	m.evidenceCountsMu.Lock()
	defer m.evidenceCountsMu.Unlock()
	if m.evidenceCounts == nil {
		m.evidenceCounts = make(map[string]int)
	}
	m.evidenceCounts[poaID] = n
}

// Delegation graph metrics
func (m *Memory) IncDelegationGraphExports() { atomic.AddUint64(&m.delegationGraphExports, 1) }
func (m *Memory) SetDelegationGraphNodeCount(n int) {
	//nolint:gosec // G115: node count, always non-negative
	atomic.StoreUint64(&m.delegationGraphNodeCount, uint64(n))
}

// Hierarchical digest metrics
func (m *Memory) IncHierDigestIssued() { atomic.AddUint64(&m.hierDigestIssued, 1) }
func (m *Memory) IncHierDigestParentDigestMissing() {
	atomic.AddUint64(&m.hierDigestParentDigestMissing, 1)
}
func (m *Memory) IncHierDigestVersionMismatch() { atomic.AddUint64(&m.hierDigestVersionMismatch, 1) }

// Cascade revocation metrics
func (m *Memory) IncCascadeRevocationTriggered() { atomic.AddUint64(&m.cascadeRevocationTriggered, 1) }
func (m *Memory) IncCascadeDescendantsProcessed() {
	atomic.AddUint64(&m.cascadeDescendantsProcessed, 1)
}
func (m *Memory) ObserveCascadeProcessingLatency(d time.Duration) {
	if d < 0 {
		return
	}
	//nolint:gosec // G115: duration nanoseconds always non-negative
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.cascadeProcessingLatencyCount, 1)
	atomic.AddUint64(&m.cascadeProcessingLatencyTotalNS, ns)
	for {
		old := atomic.LoadUint64(&m.cascadeProcessingLatencyMaxNS)
		if ns <= old || atomic.CompareAndSwapUint64(&m.cascadeProcessingLatencyMaxNS, old, ns) {
			break
		}
	}
}
func (m *Memory) IncCascadeDepthLimitReached() { atomic.AddUint64(&m.cascadeDepthLimitReached, 1) }
func (m *Memory) IncCascadeBatchProcessed()    { atomic.AddUint64(&m.cascadeBatchProcessed, 1) }
func (m *Memory) SetCascadeMaxDepthReached(depth int) {
	if depth <= 0 {
		return
	}
	for {
		cur := atomic.LoadUint64(&m.cascadeMaxDepthReachedGauge)
		if uint64(depth) <= cur {
			return
		}
		if atomic.CompareAndSwapUint64(&m.cascadeMaxDepthReachedGauge, cur, uint64(depth)) {
			return
		}
	}
}
func (m *Memory) IncCascadeProcessingErrors() { atomic.AddUint64(&m.cascadeProcessingErrors, 1) }

// ValidationLatencyPercentiles returns approximate p50, p95, p99 validation latency using the reservoir.
// Falls back to zero durations when insufficient samples. Computation mirrors Snapshot() logic.
func (m *Memory) ValidationLatencyPercentiles() (p50, p95, p99 time.Duration) {
	vc := atomic.LoadUint64(&m.validationCount)
	if vc == 0 {
		return
	}
	// Determine sample size (cannot exceed reservoir size nor validation count)
	sampleSize := vc
	if sampleSize > uint64(len(m.reservoir)) {
		sampleSize = uint64(len(m.reservoir))
	}
	if sampleSize == 0 {
		return
	}
	buf := make([]uint64, 0, sampleSize)
	end := atomic.LoadUint64(&m.reservoirIndex)
	for i := uint64(0); i < sampleSize; i++ {
		pos := (end - 1 - i) & 255
		buf = append(buf, m.reservoir[pos])
	}
	sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
	pick := func(p float64) time.Duration {
		if len(buf) == 0 {
			return 0
		}
		rank := int(p*float64(len(buf)-1) + 0.5)
		if rank < 0 {
			rank = 0
		}
		if rank >= len(buf) {
			rank = len(buf) - 1
		}
		//nolint:gosec // G115: converting stored nanosecond sample, safe range
		return time.Duration(buf[rank])
	}
	p50 = pick(0.50)
	p95 = pick(0.95)
	p99 = pick(0.99)
	return
}

// latencyReservoir maintains a fixed-size ring buffer of recent latency samples (nanoseconds)
// for approximate percentile estimation.
type latencyReservoir struct {
	samples []uint64 // power-of-two length for cheap masking
	writes  uint64   // total writes (monotonic) used for indexing & sample size
}

// NewMemory constructs a fresh Memory metrics collector.
func NewMemory() *Memory { return &Memory{} }

// SetReplayWALPending sets current pending WAL entries gauge.
//nolint:gosec // G115: WAL pending count, always non-negative
func (m *Memory) SetReplayWALPending(n int) { atomic.StoreUint64(&m.replayWALPending, uint64(n)) }

// ObserveReplayWALFlushLatency records WAL flush latency.
func (m *Memory) ObserveReplayWALFlushLatency(d time.Duration) {
	if d < 0 {
		return
	}
	//nolint:gosec // G115: duration nanoseconds always non-negative
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.replayWALFlushLatencyCount, 1)
	atomic.AddUint64(&m.replayWALFlushLatencyTotalNS, ns)
	for {
		old := atomic.LoadUint64(&m.replayWALFlushLatencyMaxNS)
		if ns <= old || atomic.CompareAndSwapUint64(&m.replayWALFlushLatencyMaxNS, old, ns) {
			break
		}
	}
}

// ObserveReplayWALSnapshotDuration records snapshot write duration prior to WAL rotation.
func (m *Memory) ObserveReplayWALSnapshotDuration(d time.Duration) {
	if d < 0 {
		return
	}
	//nolint:gosec // G115: duration nanoseconds always non-negative
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.replayWALSnapshotDurationCount, 1)
	atomic.AddUint64(&m.replayWALSnapshotDurationTotalNS, ns)
	for {
		old := atomic.LoadUint64(&m.replayWALSnapshotDurationMaxNS)
		if ns <= old || atomic.CompareAndSwapUint64(&m.replayWALSnapshotDurationMaxNS, old, ns) {
			break
		}
	}
}

// IncCapabilityDiffRequests increments diff endpoint request counter.
func (m *Memory) IncCapabilityDiffRequests() { atomic.AddUint64(&m.capabilityDiffRequests, 1) }

// ObserveCapabilityDiffLatency records diff computation latency.
func (m *Memory) ObserveCapabilityDiffLatency(d time.Duration) {
	if d < 0 {
		return
	}
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.capabilityDiffLatencyCount, 1)
	atomic.AddUint64(&m.capabilityDiffLatencyTotalNS, ns)
	for {
		old := atomic.LoadUint64(&m.capabilityDiffLatencyMaxNS)
		if ns <= old || atomic.CompareAndSwapUint64(&m.capabilityDiffLatencyMaxNS, old, ns) {
			break
		}
	}
}

// EnablePersistence configures a file path for periodic / shutdown persistence.
// If the file exists it will be loaded immediately.
func (m *Memory) EnablePersistence(path string) error {
	if path == "" {
		return nil
	}
	m.persistPath = path
	// Attempt load if file exists.
	if _, err := os.Stat(path); err == nil {
		return m.loadFromFile(path)
	}
	return nil
}

// persistentState defines the JSON structure saved/loaded.
type persistentState struct {
	DelegationsCreated                 uint64            `json:"delegations_created"`
	DecisionCounts                     map[string]uint64 `json:"decision_counts"`
	DecisionReasonCounts               map[string]uint64 `json:"decision_reason_counts"`
	LifecycleCounts                    map[string]uint64 `json:"lifecycle_counts"`
	DelegationStatusTransitions        uint64            `json:"delegation_status_transitions"`
	DelegationStatusTransitionFailures uint64            `json:"delegation_status_transition_failures"`
	TokenStatusTransitions             uint64            `json:"token_status_transitions"`
	TokenStatusTransitionFailures      uint64            `json:"token_status_transition_failures"`
	InvalidPayloadFailures             uint64            `json:"invalid_payload_failures"`
	UnsupportedStatusFailures          uint64            `json:"unsupported_status_failures"`
	InvalidTransitionFailures          uint64            `json:"invalid_transition_failures"`
	NotFoundFailures                   uint64            `json:"not_found_failures"`
}

// Save writes current labeled counters & selected aggregates to disk.
func (m *Memory) Save() error {
	if m.persistPath == "" {
		return nil
	}
	st := persistentState{
		DelegationsCreated:                 atomic.LoadUint64(&m.delegationsCreated),
		DelegationStatusTransitions:        atomic.LoadUint64(&m.delegationStatusTransitions),
		DelegationStatusTransitionFailures: atomic.LoadUint64(&m.delegationStatusTransitionFailures),
		TokenStatusTransitions:             atomic.LoadUint64(&m.tokenStatusTransitions),
		TokenStatusTransitionFailures:      atomic.LoadUint64(&m.tokenStatusTransitionFailures),
		InvalidPayloadFailures:             atomic.LoadUint64(&m.invalidPayloadFailures),
		UnsupportedStatusFailures:          atomic.LoadUint64(&m.unsupportedStatusFailures),
		InvalidTransitionFailures:          atomic.LoadUint64(&m.invalidTransitionFailures),
		NotFoundFailures:                   atomic.LoadUint64(&m.notFoundFailures),
	}
	m.decisionMu.Lock()
	if m.decisionCounts != nil {
		st.DecisionCounts = make(map[string]uint64, len(m.decisionCounts))
		for k, v := range m.decisionCounts {
			st.DecisionCounts[k] = v
		}
	}
	m.decisionMu.Unlock()
	m.decisionReasonMu.Lock()
	if m.decisionReasonCounts != nil {
		st.DecisionReasonCounts = make(map[string]uint64, len(m.decisionReasonCounts))
		for k, v := range m.decisionReasonCounts {
			st.DecisionReasonCounts[k] = v
		}
	}
	m.decisionReasonMu.Unlock()
	m.lifecycleMu.Lock()
	if m.lifecycleCounts != nil {
		st.LifecycleCounts = make(map[string]uint64, len(m.lifecycleCounts))
		for k, v := range m.lifecycleCounts {
			st.LifecycleCounts[k] = v
		}
	}
	m.lifecycleMu.Unlock()
	b, err := json.MarshalIndent(&st, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.Create(m.persistPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(b); err != nil {
		return err
	}
	// Mark last successful persistence time.
	atomic.StoreUint64(&m.lastPersistUnix, uint64(time.Now().Unix()))
	return nil
}

// loadFromFile restores state from a persistence file.
func (m *Memory) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var st persistentState
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}
	atomic.StoreUint64(&m.delegationsCreated, st.DelegationsCreated)
	atomic.StoreUint64(&m.delegationStatusTransitions, st.DelegationStatusTransitions)
	atomic.StoreUint64(&m.delegationStatusTransitionFailures, st.DelegationStatusTransitionFailures)
	atomic.StoreUint64(&m.tokenStatusTransitions, st.TokenStatusTransitions)
	atomic.StoreUint64(&m.tokenStatusTransitionFailures, st.TokenStatusTransitionFailures)
	atomic.StoreUint64(&m.invalidPayloadFailures, st.InvalidPayloadFailures)
	atomic.StoreUint64(&m.unsupportedStatusFailures, st.UnsupportedStatusFailures)
	atomic.StoreUint64(&m.invalidTransitionFailures, st.InvalidTransitionFailures)
	atomic.StoreUint64(&m.notFoundFailures, st.NotFoundFailures)
	if st.DecisionCounts != nil {
		m.decisionMu.Lock()
		m.decisionCounts = st.DecisionCounts
		m.decisionMu.Unlock()
	}
	if st.DecisionReasonCounts != nil {
		m.decisionReasonMu.Lock()
		m.decisionReasonCounts = st.DecisionReasonCounts
		m.decisionReasonMu.Unlock()
	}
	if st.LifecycleCounts != nil {
		m.lifecycleMu.Lock()
		m.lifecycleCounts = st.LifecycleCounts
		m.lifecycleMu.Unlock()
	}
	return nil
}

func (m *Memory) IncInvalidPayloadFailure()    { atomic.AddUint64(&m.invalidPayloadFailures, 1) }
func (m *Memory) IncUnsupportedStatusFailure() { atomic.AddUint64(&m.unsupportedStatusFailures, 1) }
func (m *Memory) IncInvalidTransitionFailure() { atomic.AddUint64(&m.invalidTransitionFailures, 1) }
func (m *Memory) IncNotFoundFailure()          { atomic.AddUint64(&m.notFoundFailures, 1) }

// Dual-control revocation workflow metric increments
func (m *Memory) IncRevocationWorkflowInitiated() { atomic.AddUint64(&m.revWorkflowInitiated, 1) }
func (m *Memory) IncRevocationWorkflowInitiationFailures() {
	atomic.AddUint64(&m.revWorkflowInitiationFailures, 1)
}
func (m *Memory) IncRevocationWorkflowApprovals() { atomic.AddUint64(&m.revWorkflowApprovals, 1) }
func (m *Memory) IncRevocationWorkflowApprovalFailures() {
	atomic.AddUint64(&m.revWorkflowApprovalFailures, 1)
}
func (m *Memory) IncRevocationWorkflowQuorumSatisfied() {
	atomic.AddUint64(&m.revWorkflowQuorumSatisfied, 1)
}
func (m *Memory) IncRevocationWorkflowCanceled() { atomic.AddUint64(&m.revWorkflowCanceled, 1) }
func (m *Memory) IncRevocationWorkflowCancellationFailures() {
	atomic.AddUint64(&m.revWorkflowCancellationFailures, 1)
}
func (m *Memory) IncRevocationWorkflowUnauthorized() { atomic.AddUint64(&m.revWorkflowUnauthorized, 1) }

// Accessors for validation failure counters (read paths for Prometheus exposition without exporting raw fields)
func (m *Memory) InvalidPayloadFailures() uint64 { return atomic.LoadUint64(&m.invalidPayloadFailures) }

func (m *Memory) UnsupportedStatusFailures() uint64 {
	return atomic.LoadUint64(&m.unsupportedStatusFailures)
}

func (m *Memory) InvalidTransitionFailures() uint64 {
	return atomic.LoadUint64(&m.invalidTransitionFailures)
}
func (m *Memory) NotFoundFailures() uint64 { return atomic.LoadUint64(&m.notFoundFailures) }

// Violation counters (semantic/authorization hygiene)
func (m *Memory) ScopeViolations() uint64       { return atomic.LoadUint64(&m.scopeViolations) }
func (m *Memory) RestrictionViolations() uint64 { return atomic.LoadUint64(&m.restrictionViolations) }
func (m *Memory) UnauthorizedDecisions() uint64 { return atomic.LoadUint64(&m.unauthorized) }
func (m *Memory) ExpiredDelegations() uint64    { return atomic.LoadUint64(&m.expiredDelegations) }
func (m *Memory) RevokedDelegations() uint64    { return atomic.LoadUint64(&m.revokedDelegations) }

func (m *Memory) IncDelegationsCreated() {
	atomic.AddUint64(&m.delegationsCreated, 1)
}

// Replay metrics increments
func (m *Memory) IncReplayHits()        { atomic.AddUint64(&m.replayHits, 1) }
func (m *Memory) IncReplayMisses()      { atomic.AddUint64(&m.replayMisses, 1) }
func (m *Memory) IncReplayStoreErrors() { atomic.AddUint64(&m.replayStoreErrors, 1) }
func (m *Memory) ObserveReplayStoreLatency(d time.Duration) {
	if d < 0 {
		return
	}
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.replayStoreLatencyCount, 1)
	atomic.AddUint64(&m.replayStoreLatencyTotalNS, ns)
	for {
		old := atomic.LoadUint64(&m.replayStoreLatencyMaxNS)
		if ns <= old || atomic.CompareAndSwapUint64(&m.replayStoreLatencyMaxNS, old, ns) {
			break
		}
	}
}

func (m *Memory) ObserveValidationLatency(d time.Duration) {
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.validationCount, 1)
	atomic.AddUint64(&m.validationTotalNS, ns)
	// reservoir write (no CAS needed for approximate stats)
	idx := atomic.AddUint64(&m.reservoirIndex, 1) - 1
	m.reservoir[idx&255] = ns
	for {
		cur := atomic.LoadUint64(&m.validationMaxNS)
		if ns <= cur {
			break
		}
		if atomic.CompareAndSwapUint64(&m.validationMaxNS, cur, ns) {
			break
		}
	}
	for {
		cur := atomic.LoadUint64(&m.validationMinNS)
		if cur != 0 && ns >= cur {
			break
		}
		if atomic.CompareAndSwapUint64(&m.validationMinNS, cur, ns) {
			break
		}
	}
}

func (m *Memory) IncSignaturesIssued()       { atomic.AddUint64(&m.signaturesIssued, 1) }
func (m *Memory) IncSignatureIssueFailures() { atomic.AddUint64(&m.signatureIssueFailures, 1) }
func (m *Memory) IncSignatureVerifications() { atomic.AddUint64(&m.signatureVerifications, 1) }
func (m *Memory) IncSignatureVerificationFailures() {
	atomic.AddUint64(&m.signatureVerificationFailures, 1)
}

// Attestation proof counters
func (m *Memory) IncAttestationProofIssued() { atomic.AddUint64(&m.attestationProofIssued, 1) }
func (m *Memory) IncAttestationProofIssueFailures() {
	atomic.AddUint64(&m.attestationProofIssueFailures, 1)
}
func (m *Memory) IncAttestationProofVerifications() {
	atomic.AddUint64(&m.attestationProofVerifications, 1)
}
func (m *Memory) IncAttestationProofVerificationFailures() {
	atomic.AddUint64(&m.attestationProofVerificationFailures, 1)
}
func (m *Memory) IncAttestationProofDigestMismatch() {
	atomic.AddUint64(&m.attestationProofDigestMismatch, 1)
}
func (m *Memory) ObserveAttestationProofVerificationLatency(d time.Duration) {
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.attestationProofVerificationLatencyCount, 1)
	atomic.AddUint64(&m.attestationProofVerificationLatencyTotalNS, ns)
	for {
		cur := atomic.LoadUint64(&m.attestationProofVerificationLatencyMaxNS)
		if ns <= cur {
			break
		}
		if atomic.CompareAndSwapUint64(&m.attestationProofVerificationLatencyMaxNS, cur, ns) {
			break
		}
	}
}

// ObserveAttestationProofIssueLatency records issuance latency.
func (m *Memory) ObserveAttestationProofIssueLatency(d time.Duration) {
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.attestationProofIssueLatencyCount, 1)
	atomic.AddUint64(&m.attestationProofIssueLatencyTotalNS, ns)
	for {
		cur := atomic.LoadUint64(&m.attestationProofIssueLatencyMaxNS)
		if ns <= cur {
			break
		}
		if atomic.CompareAndSwapUint64(&m.attestationProofIssueLatencyMaxNS, cur, ns) {
			break
		}
	}
}

// IncAttestationProofVerificationFailureReason increments labeled verification failure reason counter.
func (m *Memory) IncAttestationProofVerificationFailureReason(reason string) {
	if reason == "" {
		reason = otherReason
	}
	m.attestationProofVerificationFailureReasonsMu.Lock()
	if m.attestationProofVerificationFailureReasons == nil {
		m.attestationProofVerificationFailureReasons = make(map[string]uint64, 8)
	}
	m.attestationProofVerificationFailureReasons[reason]++
	m.attestationProofVerificationFailureReasonsMu.Unlock()
}

// Trust anchor granular attestation failure counters
func (m *Memory) IncAttestationProofTrustAnchorMissing() {
	atomic.AddUint64(&m.attestationProofTrustAnchorMissing, 1)
}
func (m *Memory) IncAttestationProofTrustAnchorAlgorithmMismatch() {
	atomic.AddUint64(&m.attestationProofTrustAnchorAlgorithmMismatch, 1)
}
func (m *Memory) IncAttestationProofTrustAnchorKeyMismatch() {
	atomic.AddUint64(&m.attestationProofTrustAnchorKeyMismatch, 1)
}
func (m *Memory) IncEnvelopeV1Issued()       { atomic.AddUint64(&m.envelopeV1Issued, 1) }
func (m *Memory) IncEnvelopeV2Issued()       { atomic.AddUint64(&m.envelopeV2Issued, 1) }
func (m *Memory) IncEnvelopeDigestMismatch() { atomic.AddUint64(&m.envelopeDigestMismatch, 1) }
func (m *Memory) IncEnvelopeRawPOAEmbedded() { atomic.AddUint64(&m.envelopeRawPOAEmbedded, 1) }
func (m *Memory) IncEnvelopeRawPOATooLarge() { atomic.AddUint64(&m.envelopeRawPOATooLarge, 1) }
func (m *Memory) IncEnvelopeDigestMismatchReason(reason string) {
	if reason == "" {
		reason = otherReason
	}
	m.envelopeDigestMismatchReasonsMu.Lock()
	if m.envelopeDigestMismatchReasons == nil {
		m.envelopeDigestMismatchReasons = make(map[string]uint64, 4)
	}
	m.envelopeDigestMismatchReasons[reason]++
	m.envelopeDigestMismatchReasonsMu.Unlock()
}

// EnvelopeDigestMismatchReasonsSnapshot returns a copy of labeled mismatch reasons.
func (m *Memory) EnvelopeDigestMismatchReasonsSnapshot() map[string]uint64 {
	m.envelopeDigestMismatchReasonsMu.Lock()
	defer m.envelopeDigestMismatchReasonsMu.Unlock()
	if m.envelopeDigestMismatchReasons == nil {
		return map[string]uint64{}
	}
	out := make(map[string]uint64, len(m.envelopeDigestMismatchReasons))
	for k, v := range m.envelopeDigestMismatchReasons {
		out[k] = v
	}
	return out
}

// SetEnvelopeV2AdoptionRatio stores ratio using atomic uint64 conversion; caller responsible for computing ratio.
func (m *Memory) SetEnvelopeV2AdoptionRatio(r float64) {
	atomic.StoreUint64(&m.envelopeV2AdoptionRatioBits, math.Float64bits(r))
}

// ObserveEnvelopeIssuanceCadence records interval in seconds since previous issuance (if previous exists)
func (m *Memory) ObserveEnvelopeIssuanceCadence(seconds float64) {
	atomic.AddUint64(&m.envelopeIssuanceCadenceCount, 1)
	// Accumulate as nanoseconds for integer atomic add (seconds * 1e9)
	ns := uint64(seconds * 1e9)
	atomic.AddUint64(&m.envelopeIssuanceCadenceTotal, ns)
}

// SetEnvelopeV1SunsetPhase sets current sunset lifecycle phase
func (m *Memory) SetEnvelopeV1SunsetPhase(phase int) {
	if phase < 0 {
		phase = 0
	}
	//nolint:gosec // G115: phase validated non-negative, small enum value
	atomic.StoreUint64(&m.envelopeV1SunsetPhase, uint64(phase))
}

// SetSunsetPhaseSatisfactionProgress stores progress ratio (0..1) atomically.
func (m *Memory) SetSunsetPhaseSatisfactionProgress(p float64) {
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	atomic.StoreUint64(&m.sunsetPhaseSatisfactionBits, math.Float64bits(p))
}

func (m *Memory) IncMultiSignatureVerifications() {
	atomic.AddUint64(&m.multiSignatureVerifications, 1)
}

func (m *Memory) IncMultiSignatureVerificationFailures() {
	atomic.AddUint64(&m.multiSignatureVerificationFailures, 1)
}

func (m *Memory) IncMultiSignatureWeightFailures() {
	atomic.AddUint64(&m.multiSignatureWeightFailures, 1)
}

func (m *Memory) IncMultiSignatureStructuralFailures() {
	atomic.AddUint64(&m.multiSignatureStructuralFailures, 1)
}

func (m *Memory) IncMultiSignatureDigestFailures() {
	atomic.AddUint64(&m.multiSignatureDigestFailures, 1)
}

func (m *Memory) IncMultiSignaturePublicKeyMissing() {
	atomic.AddUint64(&m.multiSignaturePublicKeyMissingFailures, 1)
}

func (m *Memory) IncMultiSignatureInvalidSignatureFailures() {
	atomic.AddUint64(&m.multiSignatureInvalidSignatureFailures, 1)
}

func (m *Memory) IncMultiSignatureThresholdFailures() {
	atomic.AddUint64(&m.multiSignatureThresholdFailures, 1)
}

func (m *Memory) ObserveMultiSignatureBatchSize(size int) {
	if size <= 0 {
		return
	}
	s := uint64(size)
	atomic.AddUint64(&m.multiSignatureBatchSizeCount, 1)
	atomic.AddUint64(&m.multiSignatureBatchSizeTotal, s)
	for {
		cur := atomic.LoadUint64(&m.multiSignatureBatchSizeMax)
		if s <= cur {
			break
		}
		if atomic.CompareAndSwapUint64(&m.multiSignatureBatchSizeMax, cur, s) {
			break
		}
	}
}

func (m *Memory) ObserveMultiSignatureAggregateLatency(d time.Duration) {
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.multiSignatureAggregateLatencyCount, 1)
	atomic.AddUint64(&m.multiSignatureAggregateLatencyTotalNS, ns)
	for {
		cur := atomic.LoadUint64(&m.multiSignatureAggregateLatencyMaxNS)
		if ns <= cur {
			break
		}
		if atomic.CompareAndSwapUint64(&m.multiSignatureAggregateLatencyMaxNS, cur, ns) {
			break
		}
	}
}

// ObserveMultiSignatureVerificationLatency reuses generic validation reservoir for simplicity; could add dedicated reservoir later.
func (m *Memory) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	m.ObserveValidationLatency(d)
}

func (m *Memory) IncRevocationIntegrityFailures() {
	atomic.AddUint64(&m.revocationIntegrityFailures, 1)
}
func (m *Memory) IncSignaturePublicKeyMissing() { atomic.AddUint64(&m.signaturePublicKeyMissing, 1) }

// IncCryptoSignatureMissing increments missing detached signature counter.
func (m *Memory) IncCryptoSignatureMissing() { atomic.AddUint64(&m.cryptoSignatureMissing, 1) }
func (m *Memory) IncAnchorAttempts()         { atomic.AddUint64(&m.anchorAttempts, 1) }
func (m *Memory) IncCombinedAnchorEmitted()  { atomic.AddUint64(&m.combinedAnchorEmitted, 1) }
func (m *Memory) IncCombinedAnchorFailures() { atomic.AddUint64(&m.combinedAnchorFailures, 1) }
func (m *Memory) IncAnchorFailures()         { atomic.AddUint64(&m.anchorFailures, 1) }
func (m *Memory) IncExternalAnchorForcedFailures() {
	atomic.AddUint64(&m.externalAnchorForcedFailures, 1)
}
func (m *Memory) IncObligationsExecuted() { atomic.AddUint64(&m.obligationsExecuted, 1) }
func (m *Memory) IncObligationsFailed()   { atomic.AddUint64(&m.obligationsFailed, 1) }
func (m *Memory) ObserveObligationLatency(d time.Duration) {
	ns := uint64(d.Nanoseconds())
	atomic.AddUint64(&m.obligationLatencyCount, 1)
	atomic.AddUint64(&m.obligationLatencyTotalNS, ns)
	for {
		cur := atomic.LoadUint64(&m.obligationLatencyMaxNS)
		if ns <= cur {
			break
		}
		if atomic.CompareAndSwapUint64(&m.obligationLatencyMaxNS, cur, ns) {
			break
		}
	}
}
func (m *Memory) IncMandatoryObligationFailures() {
	atomic.AddUint64(&m.mandatoryObligationFailures, 1)
}
func (m *Memory) IncCapabilityAnchorEmitted() { atomic.AddUint64(&m.capabilityAnchorEmitted, 1) }
func (m *Memory) IncCapabilityAnchorSkipped() { atomic.AddUint64(&m.capabilityAnchorSkipped, 1) }
func (m *Memory) IncCapabilityRegistryHashChanged() {
	atomic.AddUint64(&m.capabilityRegistryHashChanged, 1)
}

// IncCapabilityAnchorAlgorithm increments per-algorithm anchor emission counter.
// Multiple algorithms may be recorded for a single emission event.
func (m *Memory) IncCapabilityAnchorAlgorithm(algo string) {
	if algo == "" {
		algo = "_"
	}
	m.anchorAlgoMu.Lock()
	if m.anchorAlgoCounts == nil {
		m.anchorAlgoCounts = make(map[string]uint64, 4)
	}
	m.anchorAlgoCounts[algo]++
	m.anchorAlgoMu.Unlock()
}

// SetCapabilityAnchorAlgorithmRatio stores per-algorithm ratio (0..1) as float64 bits.
func (m *Memory) SetCapabilityAnchorAlgorithmRatio(algo string, ratio float64) {
	if algo == "" {
		algo = "_"
	}
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	m.anchorAlgoRatioMu.Lock()
	if m.anchorAlgoRatioBits == nil {
		m.anchorAlgoRatioBits = make(map[string]uint64, 4)
	}
	m.anchorAlgoRatioBits[algo] = math.Float64bits(ratio)
	m.anchorAlgoRatioMu.Unlock()
}

// SetCapabilityAnchorLastWriteUnix sets unix epoch seconds for last emission (idempotent for same value).
func (m *Memory) SetCapabilityAnchorLastWriteUnix(ts uint64) {
	atomic.StoreUint64(&m.capabilityAnchorLastWriteUnix, ts)
}

// Anchoring accessors (read paths for tests / exposition)
func (m *Memory) CapabilityAnchorEmitted() uint64 {
	return atomic.LoadUint64(&m.capabilityAnchorEmitted)
}

func (m *Memory) CapabilityAnchorSkipped() uint64 {
	return atomic.LoadUint64(&m.capabilityAnchorSkipped)
}

func (m *Memory) CapabilityRegistryHashChanged() uint64 {
	return atomic.LoadUint64(&m.capabilityRegistryHashChanged)
}

func (m *Memory) CapabilityAnchorLastWriteUnix() uint64 {
	return atomic.LoadUint64(&m.capabilityAnchorLastWriteUnix)
}

// Obligations accessors (for tests / future exposition)
func (m *Memory) ObligationsExecuted() uint64 { return atomic.LoadUint64(&m.obligationsExecuted) }
func (m *Memory) ObligationsFailed() uint64   { return atomic.LoadUint64(&m.obligationsFailed) }

// MandatoryObligationFailures accessor for tests/exposition.
func (m *Memory) MandatoryObligationFailures() uint64 {
	return atomic.LoadUint64(&m.mandatoryObligationFailures)
}

// External anchor forced failure accessor (for tests / exposition)
func (m *Memory) ExternalAnchorForcedFailures() uint64 {
	return atomic.LoadUint64(&m.externalAnchorForcedFailures)
}
func (m *Memory) IncScopeViolations()       { atomic.AddUint64(&m.scopeViolations, 1) }
func (m *Memory) IncRestrictionViolations() { atomic.AddUint64(&m.restrictionViolations, 1) }
func (m *Memory) IncUnauthorized()          { atomic.AddUint64(&m.unauthorized, 1) }
func (m *Memory) IncExpired()               { atomic.AddUint64(&m.expiredDelegations, 1) }
func (m *Memory) IncRevoked()               { atomic.AddUint64(&m.revokedDelegations, 1) }

// IncDelegationsPartiallyRevoked increments counter for delegations transitioned to partially_revoked state.
func (m *Memory) IncDelegationsPartiallyRevoked() {
	atomic.AddUint64(&m.partiallyRevokedDelegations, 1)
}
func (m *Memory) IncDelegationDepthExceeded() { atomic.AddUint64(&m.delegationDepthExceeded, 1) }
func (m *Memory) SetMaxObservedDelegationDepth(depth int) {
	if depth <= 0 {
		return
	}
	for {
		cur := atomic.LoadUint64(&m.maxObservedDelegationDepth)
		if uint64(depth) <= cur {
			return
		}
		if atomic.CompareAndSwapUint64(&m.maxObservedDelegationDepth, cur, uint64(depth)) {
			return
		}
	}
}
func (m *Memory) IncCapabilityEnforceAllowed() { atomic.AddUint64(&m.capabilityEnforceAllowed, 1) }
func (m *Memory) IncCapabilityEnforceDenied()  { atomic.AddUint64(&m.capabilityEnforceDenied, 1) }
func (m *Memory) IncModelLimitExceeded()       { atomic.AddUint64(&m.modelLimitExceeded, 1) }
func (m *Memory) IncModelOutputLimitExceeded() { atomic.AddUint64(&m.modelOutputLimitExceeded, 1) }
func (m *Memory) IncModelRateLimitExceeded()   { atomic.AddUint64(&m.modelRateLimitExceeded, 1) }
func (m *Memory) IncModelUserInputLimitExceeded() {
	atomic.AddUint64(&m.modelUserInputLimitExceeded, 1)
}
func (m *Memory) IncModelUserOutputLimitExceeded() {
	atomic.AddUint64(&m.modelUserOutputLimitExceeded, 1)
}
func (m *Memory) IncModelUserRateLimitExceeded() { atomic.AddUint64(&m.modelUserRateLimitExceeded, 1) }
func (m *Memory) IncModelUnknown()               { atomic.AddUint64(&m.modelUnknown, 1) }
func (m *Memory) IncModelLimitSurge()            { atomic.AddUint64(&m.modelLimitSurges, 1) }

// IncPolicyManifestEmitted increments the RB4 policy manifest emission counter.
func (m *Memory) IncPolicyManifestEmitted() { atomic.AddUint64(&m.policyManifestEmitted, 1) }

// PolicyManifestEmitted returns total successful policy manifest emissions (RB4).
func (m *Memory) PolicyManifestEmitted() uint64 { return atomic.LoadUint64(&m.policyManifestEmitted) }

// Accessors for per-user limit exceed counters (for tests / diagnostics)
func (m *Memory) ModelUserInputLimitExceeded() uint64 {
	return atomic.LoadUint64(&m.modelUserInputLimitExceeded)
}
func (m *Memory) ModelUserOutputLimitExceeded() uint64 {
	return atomic.LoadUint64(&m.modelUserOutputLimitExceeded)
}
func (m *Memory) ModelUserRateLimitExceeded() uint64 {
	return atomic.LoadUint64(&m.modelUserRateLimitExceeded)
}

// Capability enforcement accessor helpers (for tests / diagnostics)
func (m *Memory) CapabilityEnforceAllowed() uint64 {
	return atomic.LoadUint64(&m.capabilityEnforceAllowed)
}
func (m *Memory) CapabilityEnforceDenied() uint64 {
	return atomic.LoadUint64(&m.capabilityEnforceDenied)
}
func (m *Memory) IncDelegationStatusTransitions() {
	atomic.AddUint64(&m.delegationStatusTransitions, 1)
}

func (m *Memory) IncDelegationStatusTransitionFailures() {
	atomic.AddUint64(&m.delegationStatusTransitionFailures, 1)
}
func (m *Memory) IncTokenStatusTransitions() { atomic.AddUint64(&m.tokenStatusTransitions, 1) }
func (m *Memory) IncTokenStatusTransitionFailures() {
	atomic.AddUint64(&m.tokenStatusTransitionFailures, 1)
}

func (m *Memory) RecordDecision(action, resource, outcome string) {
	if action == "" {
		action = "_"
	}
	if resource == "" {
		resource = "_"
	}
	if outcome == "" {
		outcome = unknownOutcome
	}
	key := action + "|" + resource + "|" + outcome
	m.decisionMu.Lock()
	if m.decisionCounts == nil {
		m.decisionCounts = make(map[string]uint64)
	}
	m.decisionCounts[key]++
	m.decisionMu.Unlock()
}

// IncViolation provides a generic hook for external components to increment violation-style counters.
// For now we only wire specific semantic categories (scope/restriction hygiene) into existing counters.
// If the category matches known scope/restriction hygiene categories we increment respective counters; otherwise ignored.
func (m *Memory) IncViolation(cat interface{}) {
	// Accept string or typed categories; minimal reflection avoidance.
	switch c := cat.(type) {
	case string:
		if strings.Contains(c, "scope_utf8_invalid") || strings.Contains(c, "scope_control_char") {
			m.IncScopeViolations()
		} else if strings.Contains(c, "restriction_utf8_invalid") || strings.Contains(c, "restriction_control_char") {
			m.IncRestrictionViolations()
		}
	case observability.ViolationCategory:
		// Map specific observability categories to scope/restriction hygiene counters.
		switch c {
		case observability.ScopeUTF8Invalid, observability.ScopeControlChar:
			m.IncScopeViolations()
		case observability.RestrictionUTF8Invalid, observability.RestrictionControlChar:
			m.IncRestrictionViolations()
		default:
			// Other categories currently ignored for hygiene aggregation.
		}
	default:
		// Unsupported category type: no-op (future expansion can map enum types)
	}
}

// RecordDecisionWithReason increments a labeled decision counter including a reason.
func (m *Memory) RecordDecisionWithReason(action, resource, outcome, reason string) {
	if action == "" {
		action = "_"
	}
	if resource == "" {
		resource = "_"
	}
	if outcome == "" {
		outcome = unknownOutcome
	}
	if reason == "" {
		reason = "_"
	}
	key := action + "|" + resource + "|" + outcome + "|" + reason
	m.decisionReasonMu.Lock()
	if m.decisionReasonCounts == nil {
		m.decisionReasonCounts = make(map[string]uint64)
	}
	m.decisionReasonCounts[key]++
	m.decisionReasonMu.Unlock()
}

// RecordLifecycleTransition increments a labeled lifecycle transition counter.
func (m *Memory) RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome string) {
	if entityType == "" {
		entityType = "_"
	}
	if oldStatus == "" {
		oldStatus = "_"
	}
	if newStatus == "" {
		newStatus = "_"
	}
	if outcome == "" {
		outcome = unknownOutcome
	}
	key := entityType + "|" + oldStatus + "|" + newStatus + "|" + outcome
	m.lifecycleMu.Lock()
	if m.lifecycleCounts == nil {
		m.lifecycleCounts = make(map[string]uint64)
	}
	m.lifecycleCounts[key]++
	m.lifecycleMu.Unlock()
}

// ObserveLifecycleTransitionLatency updates aggregate latency stats for lifecycle transitions.
func (m *Memory) ObserveLifecycleTransitionLatency(entityType, outcome string, d time.Duration) {
	if entityType == "" {
		entityType = "_"
	}
	if outcome == "" {
		outcome = unknownOutcome
	}
	ns := uint64(d.Nanoseconds())
	m.lifecycleLatencyMu.Lock()
	if m.lifecycleLatencyTotals == nil {
		m.lifecycleLatencyTotals = make(map[string]uint64)
	}
	if m.lifecycleLatencyCounts == nil {
		m.lifecycleLatencyCounts = make(map[string]uint64)
	}
	if m.lifecycleLatencyMax == nil {
		m.lifecycleLatencyMax = make(map[string]uint64)
	}
	key := entityType + "|" + outcome
	m.lifecycleLatencyTotals[key] += ns
	m.lifecycleLatencyCounts[key]++
	if ns > m.lifecycleLatencyMax[key] {
		m.lifecycleLatencyMax[key] = ns
	}
	// reservoir update (allocate lazily); size 64 (power of two) for masking
	if m.lifecycleLatencyRes == nil {
		m.lifecycleLatencyRes = make(map[string]*latencyReservoir)
	}
	lr := m.lifecycleLatencyRes[key]
	if lr == nil {
		lr = &latencyReservoir{samples: make([]uint64, 64)}
		m.lifecycleLatencyRes[key] = lr
	}
	//nolint:gosec // G115: reservoir size is fixed at 64, safe conversion
	pos := lr.writes & uint64(len(lr.samples)-1) // ring position
	lr.samples[pos] = ns
	lr.writes++
	m.lifecycleLatencyMu.Unlock()
}

// SetLifecycleTransitionLatencyQuantile stores latency quantile gauge value.
func (m *Memory) SetLifecycleTransitionLatencyQuantile(entityType, outcome, quantile string, value float64) {
	if entityType == "" {
		entityType = "_"
	}
	if outcome == "" {
		outcome = unknownOutcome
	}
	if quantile == "" {
		quantile = "_"
	}
	m.lifecycleLatencyQuantilesMu.Lock()
	if m.lifecycleLatencyQuantiles == nil {
		m.lifecycleLatencyQuantiles = make(map[string]uint64, 12)
	}
	key := entityType + "|" + outcome + "|" + quantile
	m.lifecycleLatencyQuantiles[key] = math.Float64bits(value)
	m.lifecycleLatencyQuantilesMu.Unlock()
}

// Snapshot returns current counters for testing/diagnostics.
func (m *Memory) Snapshot() (delegations uint64, validations uint64, totalLatencyNS uint64, minLatencyNS uint64, maxLatencyNS uint64, avgLatency time.Duration, p50 time.Duration, p90 time.Duration, p99 time.Duration, sigIssued uint64, sigIssueFail uint64, sigVerifications uint64, sigVerificationFail uint64, revIntegrityFail uint64, sigPubKeyMissing uint64, anchorAttempts uint64, anchorFailures uint64, replayStoreErrors uint64, replayStoreLatencyCount uint64, replayStoreLatencyTotalNS uint64, replayStoreLatencyMaxNS uint64, replayHits uint64, replayMisses uint64) {
	d := atomic.LoadUint64(&m.delegationsCreated)
	vc := atomic.LoadUint64(&m.validationCount)
	tot := atomic.LoadUint64(&m.validationTotalNS)
	mx := atomic.LoadUint64(&m.validationMaxNS)
	mn := atomic.LoadUint64(&m.validationMinNS)
	var avg time.Duration
	if vc > 0 {
		avg = time.Duration(tot / vc)
	}

	// Copy reservoir snapshot (lock-free; approximate)
	// Determine effective sample size (cannot exceed vc nor reservoir size)
	sampleSize := vc
	if sampleSize > uint64(len(m.reservoir)) {
		sampleSize = uint64(len(m.reservoir))
	}
	var buf []uint64
	if sampleSize > 0 {
		buf = make([]uint64, 0, sampleSize)
		// reservoirIndex points to next slot to write; we gather last sampleSize entries
		end := atomic.LoadUint64(&m.reservoirIndex)
		for i := uint64(0); i < sampleSize; i++ {
			// We walk backwards from end-1 downwards
			pos := (end - 1 - i) & 255
			buf = append(buf, m.reservoir[pos])
		}
		sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
		// helper to pick percentile
		pick := func(p float64) time.Duration {
			if len(buf) == 0 {
				return 0
			}
			rank := int(p*float64(len(buf)-1) + 0.5)
			if rank < 0 {
				rank = 0
			}
			if rank >= len(buf) {
				rank = len(buf) - 1
			}
			return time.Duration(buf[rank])
		}
		p50 = pick(0.50)
		p90 = pick(0.90)
		p99 = pick(0.99)
	}
	si := atomic.LoadUint64(&m.signaturesIssued)
	sif := atomic.LoadUint64(&m.signatureIssueFailures)
	sv := atomic.LoadUint64(&m.signatureVerifications)
	svf := atomic.LoadUint64(&m.signatureVerificationFailures)
	rif := atomic.LoadUint64(&m.revocationIntegrityFailures)
	spkm := atomic.LoadUint64(&m.signaturePublicKeyMissing)
	aa := atomic.LoadUint64(&m.anchorAttempts)
	af := atomic.LoadUint64(&m.anchorFailures)
	rse := atomic.LoadUint64(&m.replayStoreErrors)
	rslc := atomic.LoadUint64(&m.replayStoreLatencyCount)
	rslt := atomic.LoadUint64(&m.replayStoreLatencyTotalNS)
	rslm := atomic.LoadUint64(&m.replayStoreLatencyMaxNS)
	rh := atomic.LoadUint64(&m.replayHits)
	rm := atomic.LoadUint64(&m.replayMisses)
	return d, vc, tot, mn, mx, avg, p50, p90, p99, si, sif, sv, svf, rif, spkm, aa, af, rse, rslc, rslt, rslm, rh, rm
}

// SnapshotStruct mirrors Snapshot but returns a struct for easier JSON marshaling in diagnostics.
type SnapshotStruct struct {
	DelegationsCreated                     uint64        `json:"delegations_created"`
	Validations                            uint64        `json:"validations"`
	TotalLatencyNS                         uint64        `json:"total_latency_ns"`
	MinLatencyNS                           uint64        `json:"min_latency_ns"`
	MaxLatencyNS                           uint64        `json:"max_latency_ns"`
	AvgLatency                             time.Duration `json:"avg_latency"`
	P50                                    time.Duration `json:"p50"`
	P90                                    time.Duration `json:"p90"`
	P99                                    time.Duration `json:"p99"`
	SignaturesIssued                       uint64        `json:"signatures_issued"`
	SignatureIssueFailures                 uint64        `json:"signature_issue_failures"`
	SignatureVerifications                 uint64        `json:"signature_verifications"`
	SignatureVerificationFailures          uint64        `json:"signature_verification_failures"`
	EnvelopeV1Issued                       uint64        `json:"envelope_v1_issued"`
	EnvelopeV2Issued                       uint64        `json:"envelope_v2_issued"`
	MultiSignatureVerifications            uint64        `json:"multi_signature_verifications"`
	MultiSignatureVerificationFailures     uint64        `json:"multi_signature_verification_failures"`
	MultiSignatureWeightFailures           uint64        `json:"multi_signature_weight_failures"`
	MultiSignatureStructuralFailures       uint64        `json:"multi_signature_structural_failures"`
	MultiSignatureDigestFailures           uint64        `json:"multi_signature_digest_failures"`
	MultiSignaturePublicKeyMissingFailures uint64        `json:"multi_signature_public_key_missing_failures"`
	MultiSignatureInvalidSignatureFailures uint64        `json:"multi_signature_invalid_signature_failures"`
	MultiSignatureThresholdFailures        uint64        `json:"multi_signature_threshold_failures"`
	MultiSignatureBatchSizeCount           uint64        `json:"multi_signature_batch_size_count"`
	MultiSignatureBatchSizeTotal           uint64        `json:"multi_signature_batch_size_total"`
	MultiSignatureBatchSizeMax             uint64        `json:"multi_signature_batch_size_max"`
	MultiSignatureAggregateLatencyCount    uint64        `json:"multi_signature_aggregate_latency_count"`
	MultiSignatureAggregateLatencyTotalNS  uint64        `json:"multi_signature_aggregate_latency_total_ns"`
	MultiSignatureAggregateLatencyMaxNS    uint64        `json:"multi_signature_aggregate_latency_max_ns"`
	RevocationIntegrityFailures            uint64        `json:"revocation_integrity_failures"`
	SignaturePublicKeyMissing              uint64        `json:"signature_public_key_missing"`
	CryptoSignatureMissing                 uint64        `json:"crypto_signature_missing"`
	AnchorAttempts                         uint64        `json:"anchor_attempts"`
	AnchorFailures                         uint64        `json:"anchor_failures"`
	CombinedAnchorEmitted                  uint64        `json:"combined_anchor_emitted"`
	CombinedAnchorFailures                 uint64        `json:"combined_anchor_failures"`
	ReplayHits                             uint64        `json:"replay_hits"`
	ReplayMisses                           uint64        `json:"replay_misses"`
	ReplayStoreErrors                      uint64        `json:"replay_store_errors"`
	ReplayStoreLatencyCount                uint64        `json:"replay_store_latency_count"`
	ReplayStoreLatencyTotalNS              uint64        `json:"replay_store_latency_total_ns"`
	ReplayStoreLatencyMaxNS                uint64        `json:"replay_store_latency_max_ns"`
	ScopeViolations                        uint64        `json:"scope_violations"`
	RestrictionViolations                  uint64        `json:"restriction_violations"`
	UnauthorizedDecisions                  uint64        `json:"unauthorized_decisions"`
	ExpiredDelegations                     uint64        `json:"expired_delegations"`
	RevokedDelegations                     uint64        `json:"revoked_delegations"`
	PartiallyRevokedDelegations            uint64        `json:"partially_revoked_delegations"`
	DelegationStatusTransitions            uint64        `json:"delegation_status_transitions"`
	DelegationStatusTransitionFailures     uint64        `json:"delegation_status_transition_failures"`
	TokenStatusTransitions                 uint64        `json:"token_status_transitions"`
	TokenStatusTransitionFailures          uint64        `json:"token_status_transition_failures"`
	// Labeled breakdowns (only populated for *Memory implementation snapshot)
	LifecycleBreakdown      map[string]uint64 `json:"lifecycle_breakdown,omitempty"`
	DecisionBreakdown       map[string]uint64 `json:"decision_breakdown,omitempty"`
	DecisionReasonBreakdown map[string]uint64 `json:"decision_reason_breakdown,omitempty"`
	// Lifecycle latency aggregates (nanoseconds) keyed by entity|outcome (e.g. token|success)
	LifecycleLatencyTotals map[string]uint64 `json:"lifecycle_latency_totals_ns,omitempty"`
	LifecycleLatencyCounts map[string]uint64 `json:"lifecycle_latency_counts,omitempty"`
	LifecycleLatencyMax    map[string]uint64 `json:"lifecycle_latency_max_ns,omitempty"`
	// Percentile latency estimates (nanoseconds) per entity|outcome
	LifecycleLatencyP50 map[string]uint64 `json:"lifecycle_latency_p50_ns,omitempty"`
	LifecycleLatencyP90 map[string]uint64 `json:"lifecycle_latency_p90_ns,omitempty"`
	LifecycleLatencyP99 map[string]uint64 `json:"lifecycle_latency_p99_ns,omitempty"`
	// Last successful persistence timestamp (unix seconds) if persistence enabled & at least one save completed.
	LastPersistUnix             uint64 `json:"last_persist_unix,omitempty"`
	ObligationsExecuted         uint64 `json:"obligations_executed_total"`
	ObligationsFailed           uint64 `json:"obligations_failed_total"`
	ObligationLatencyCount      uint64 `json:"obligation_latency_count"`
	ObligationLatencyTotalNS    uint64 `json:"obligation_latency_total_ns"`
	ObligationLatencyMaxNS      uint64 `json:"obligation_latency_max_ns"`
	MandatoryObligationFailures uint64 `json:"mandatory_obligation_failures_total"`
	ModelLimitSurges            uint64 `json:"model_limit_surges_total"`
	// Attestation proof counters (Task 9)
	AttestationProofIssued                       uint64 `json:"attestation_proof_issued"`
	AttestationProofIssueFailures                uint64 `json:"attestation_proof_issue_failures"`
	AttestationProofVerifications                uint64 `json:"attestation_proof_verifications"`
	AttestationProofVerificationFailures         uint64 `json:"attestation_proof_verification_failures"`
	AttestationProofDigestMismatches             uint64 `json:"attestation_proof_digest_mismatches"`
	AttestationProofVerificationLatencyCount     uint64 `json:"attestation_proof_verification_latency_count"`
	AttestationProofVerificationLatencyTotalNS   uint64 `json:"attestation_proof_verification_latency_total_ns"`
	AttestationProofVerificationLatencyMaxNS     uint64 `json:"attestation_proof_verification_latency_max_ns"`
	AttestationProofTrustAnchorMissing           uint64 `json:"attestation_proof_trust_anchor_missing"`
	AttestationProofTrustAnchorAlgorithmMismatch uint64 `json:"attestation_proof_trust_anchor_algorithm_mismatch"`
	AttestationProofTrustAnchorKeyMismatch       uint64 `json:"attestation_proof_trust_anchor_key_mismatch"`
	// Per-algorithm capability anchor emission counts (only populated for Memory implementation)
	CapabilityAnchorAlgorithmCounts map[string]uint64 `json:"capability_anchor_algorithm_counts,omitempty"`
	// BLS Proof-of-Possession counters
	BLSPoPChallengesIssued     uint64 `json:"bls_pop_challenges_issued"`
	BLSPoPVerifications        uint64 `json:"bls_pop_verifications"`
	BLSPoPVerificationFailures uint64 `json:"bls_pop_verification_failures"`
	// Dual-control revocation workflow counters
	RevocationWorkflowInitiated            uint64 `json:"revocation_workflow_initiated"`
	RevocationWorkflowInitiationFailures   uint64 `json:"revocation_workflow_initiation_failures"`
	RevocationWorkflowApprovals            uint64 `json:"revocation_workflow_approvals"`
	RevocationWorkflowApprovalFailures     uint64 `json:"revocation_workflow_approval_failures"`
	RevocationWorkflowQuorumSatisfied      uint64 `json:"revocation_workflow_quorum_satisfied"`
	RevocationWorkflowCanceled             uint64 `json:"revocation_workflow_canceled"`
	RevocationWorkflowCancellationFailures uint64 `json:"revocation_workflow_cancellation_failures"`
	RevocationWorkflowUnauthorized         uint64 `json:"revocation_workflow_unauthorized"`
	// Cascade revocation metrics
	CascadeRevocationTriggered      uint64 `json:"cascade_revocation_triggered"`
	CascadeDescendantsProcessed     uint64 `json:"cascade_descendants_processed"`
	CascadeProcessingLatencyCount   uint64 `json:"cascade_processing_latency_count"`
	CascadeProcessingLatencyTotalNS uint64 `json:"cascade_processing_latency_total_ns"`
	CascadeProcessingLatencyMaxNS   uint64 `json:"cascade_processing_latency_max_ns"`
	CascadeDepthLimitReached        uint64 `json:"cascade_depth_limit_reached"`
	CascadeBatchProcessed           uint64 `json:"cascade_batch_processed"`
	CascadeMaxDepthReached          uint64 `json:"cascade_max_depth_reached"`
	CascadeProcessingErrors         uint64 `json:"cascade_processing_errors"`
}

// SnapshotEx returns the extended snapshot struct.
//
//nolint:gocyclo // Atomic snapshot of 30+ metric fields
func (m *Memory) SnapshotEx() SnapshotStruct {
	d, vc, tot, mn, mx, avg, p50, p90, p99, si, sif, sv, svf, rif, spkm, aa, af, rse, rslc, rslt, rslm, rh, rm := m.Snapshot()
	return SnapshotStruct{
		DelegationsCreated:            d,
		Validations:                   vc,
		TotalLatencyNS:                tot,
		MinLatencyNS:                  mn,
		MaxLatencyNS:                  mx,
		AvgLatency:                    avg,
		P50:                           p50,
		P90:                           p90,
		P99:                           p99,
		SignaturesIssued:              si,
		SignatureIssueFailures:        sif,
		SignatureVerifications:        sv,
		SignatureVerificationFailures: svf,
		EnvelopeV1Issued:              atomic.LoadUint64(&m.envelopeV1Issued),
		EnvelopeV2Issued:              atomic.LoadUint64(&m.envelopeV2Issued),
		// Sunset phase & cadence metrics (debug only; not yet in JSON spec)
		// Additional fields may be added when exporting via API.
		MultiSignatureVerifications:                  atomic.LoadUint64(&m.multiSignatureVerifications),
		MultiSignatureVerificationFailures:           atomic.LoadUint64(&m.multiSignatureVerificationFailures),
		MultiSignatureWeightFailures:                 atomic.LoadUint64(&m.multiSignatureWeightFailures),
		MultiSignatureStructuralFailures:             atomic.LoadUint64(&m.multiSignatureStructuralFailures),
		MultiSignatureDigestFailures:                 atomic.LoadUint64(&m.multiSignatureDigestFailures),
		MultiSignaturePublicKeyMissingFailures:       atomic.LoadUint64(&m.multiSignaturePublicKeyMissingFailures),
		MultiSignatureInvalidSignatureFailures:       atomic.LoadUint64(&m.multiSignatureInvalidSignatureFailures),
		MultiSignatureThresholdFailures:              atomic.LoadUint64(&m.multiSignatureThresholdFailures),
		MultiSignatureBatchSizeCount:                 atomic.LoadUint64(&m.multiSignatureBatchSizeCount),
		MultiSignatureBatchSizeTotal:                 atomic.LoadUint64(&m.multiSignatureBatchSizeTotal),
		MultiSignatureBatchSizeMax:                   atomic.LoadUint64(&m.multiSignatureBatchSizeMax),
		MultiSignatureAggregateLatencyCount:          atomic.LoadUint64(&m.multiSignatureAggregateLatencyCount),
		MultiSignatureAggregateLatencyTotalNS:        atomic.LoadUint64(&m.multiSignatureAggregateLatencyTotalNS),
		MultiSignatureAggregateLatencyMaxNS:          atomic.LoadUint64(&m.multiSignatureAggregateLatencyMaxNS),
		RevocationIntegrityFailures:                  rif,
		SignaturePublicKeyMissing:                    spkm,
		CryptoSignatureMissing:                       atomic.LoadUint64(&m.cryptoSignatureMissing),
		AnchorAttempts:                               aa,
		AnchorFailures:                               af,
		CombinedAnchorEmitted:                        atomic.LoadUint64(&m.combinedAnchorEmitted),
		CombinedAnchorFailures:                       atomic.LoadUint64(&m.combinedAnchorFailures),
		ReplayHits:                                   rh,
		ReplayMisses:                                 rm,
		ReplayStoreErrors:                            rse,
		ReplayStoreLatencyCount:                      rslc,
		ReplayStoreLatencyTotalNS:                    rslt,
		ReplayStoreLatencyMaxNS:                      rslm,
		ScopeViolations:                              atomic.LoadUint64(&m.scopeViolations),
		RestrictionViolations:                        atomic.LoadUint64(&m.restrictionViolations),
		UnauthorizedDecisions:                        atomic.LoadUint64(&m.unauthorized),
		ExpiredDelegations:                           atomic.LoadUint64(&m.expiredDelegations),
		RevokedDelegations:                           atomic.LoadUint64(&m.revokedDelegations),
		PartiallyRevokedDelegations:                  atomic.LoadUint64(&m.partiallyRevokedDelegations),
		DelegationStatusTransitions:                  atomic.LoadUint64(&m.delegationStatusTransitions),
		DelegationStatusTransitionFailures:           atomic.LoadUint64(&m.delegationStatusTransitionFailures),
		TokenStatusTransitions:                       atomic.LoadUint64(&m.tokenStatusTransitions),
		TokenStatusTransitionFailures:                atomic.LoadUint64(&m.tokenStatusTransitionFailures),
		ModelLimitSurges:                             atomic.LoadUint64(&m.modelLimitSurges),
		AttestationProofIssued:                       atomic.LoadUint64(&m.attestationProofIssued),
		AttestationProofIssueFailures:                atomic.LoadUint64(&m.attestationProofIssueFailures),
		AttestationProofVerifications:                atomic.LoadUint64(&m.attestationProofVerifications),
		AttestationProofVerificationFailures:         atomic.LoadUint64(&m.attestationProofVerificationFailures),
		AttestationProofDigestMismatches:             atomic.LoadUint64(&m.attestationProofDigestMismatch),
		AttestationProofVerificationLatencyCount:     atomic.LoadUint64(&m.attestationProofVerificationLatencyCount),
		AttestationProofVerificationLatencyTotalNS:   atomic.LoadUint64(&m.attestationProofVerificationLatencyTotalNS),
		AttestationProofVerificationLatencyMaxNS:     atomic.LoadUint64(&m.attestationProofVerificationLatencyMaxNS),
		AttestationProofTrustAnchorMissing:           atomic.LoadUint64(&m.attestationProofTrustAnchorMissing),
		AttestationProofTrustAnchorAlgorithmMismatch: atomic.LoadUint64(&m.attestationProofTrustAnchorAlgorithmMismatch),
		AttestationProofTrustAnchorKeyMismatch:       atomic.LoadUint64(&m.attestationProofTrustAnchorKeyMismatch),
		BLSPoPChallengesIssued:                       atomic.LoadUint64(&m.blsPoPChallengesIssued),
		BLSPoPVerifications:                          atomic.LoadUint64(&m.blsPoPVerifications),
		BLSPoPVerificationFailures:                   atomic.LoadUint64(&m.blsPoPVerificationFailures),
		RevocationWorkflowInitiated:                  atomic.LoadUint64(&m.revWorkflowInitiated),
		RevocationWorkflowInitiationFailures:         atomic.LoadUint64(&m.revWorkflowInitiationFailures),
		RevocationWorkflowApprovals:                  atomic.LoadUint64(&m.revWorkflowApprovals),
		RevocationWorkflowApprovalFailures:           atomic.LoadUint64(&m.revWorkflowApprovalFailures),
		RevocationWorkflowQuorumSatisfied:            atomic.LoadUint64(&m.revWorkflowQuorumSatisfied),
		RevocationWorkflowCanceled:                   atomic.LoadUint64(&m.revWorkflowCanceled),
		RevocationWorkflowCancellationFailures:       atomic.LoadUint64(&m.revWorkflowCancellationFailures),
		RevocationWorkflowUnauthorized:               atomic.LoadUint64(&m.revWorkflowUnauthorized),
		// Cascade revocation metrics
		CascadeRevocationTriggered:      atomic.LoadUint64(&m.cascadeRevocationTriggered),
		CascadeDescendantsProcessed:     atomic.LoadUint64(&m.cascadeDescendantsProcessed),
		CascadeProcessingLatencyCount:   atomic.LoadUint64(&m.cascadeProcessingLatencyCount),
		CascadeProcessingLatencyTotalNS: atomic.LoadUint64(&m.cascadeProcessingLatencyTotalNS),
		CascadeProcessingLatencyMaxNS:   atomic.LoadUint64(&m.cascadeProcessingLatencyMaxNS),
		CascadeDepthLimitReached:        atomic.LoadUint64(&m.cascadeDepthLimitReached),
		CascadeBatchProcessed:           atomic.LoadUint64(&m.cascadeBatchProcessed),
		CascadeMaxDepthReached:          atomic.LoadUint64(&m.cascadeMaxDepthReachedGauge),
		CascadeProcessingErrors:         atomic.LoadUint64(&m.cascadeProcessingErrors),
		CapabilityAnchorAlgorithmCounts: func() map[string]uint64 {
			m.anchorAlgoMu.Lock()
			defer m.anchorAlgoMu.Unlock()
			if m.anchorAlgoCounts == nil {
				return map[string]uint64{}
			}
			cp := make(map[string]uint64, len(m.anchorAlgoCounts))
			for k, v := range m.anchorAlgoCounts {
				cp[k] = v
			}
			return cp
		}(),
		LifecycleBreakdown: func() map[string]uint64 {
			m.lifecycleMu.Lock()
			defer m.lifecycleMu.Unlock()
			if m.lifecycleCounts == nil {
				return map[string]uint64{}
			}
			cp := make(map[string]uint64, len(m.lifecycleCounts))
			for k, v := range m.lifecycleCounts {
				cp[k] = v
			}
			return cp
		}(),
		DecisionBreakdown: func() map[string]uint64 {
			m.decisionMu.Lock()
			defer m.decisionMu.Unlock()
			if m.decisionCounts == nil {
				return map[string]uint64{}
			}
			cp := make(map[string]uint64, len(m.decisionCounts))
			for k, v := range m.decisionCounts {
				cp[k] = v
			}
			return cp
		}(),
		DecisionReasonBreakdown: func() map[string]uint64 {
			m.decisionReasonMu.Lock()
			defer m.decisionReasonMu.Unlock()
			if m.decisionReasonCounts == nil {
				return map[string]uint64{}
			}
			cp := make(map[string]uint64, len(m.decisionReasonCounts))
			for k, v := range m.decisionReasonCounts {
				cp[k] = v
			}
			return cp
		}(),
		LifecycleLatencyTotals: func() map[string]uint64 {
			m.lifecycleLatencyMu.Lock()
			defer m.lifecycleLatencyMu.Unlock()
			if m.lifecycleLatencyTotals == nil {
				return map[string]uint64{}
			}
			cp := make(map[string]uint64, len(m.lifecycleLatencyTotals))
			for k, v := range m.lifecycleLatencyTotals {
				cp[k] = v
			}
			return cp
		}(),
		LifecycleLatencyCounts: func() map[string]uint64 {
			m.lifecycleLatencyMu.Lock()
			defer m.lifecycleLatencyMu.Unlock()
			if m.lifecycleLatencyCounts == nil {
				return map[string]uint64{}
			}
			cp := make(map[string]uint64, len(m.lifecycleLatencyCounts))
			for k, v := range m.lifecycleLatencyCounts {
				cp[k] = v
			}
			return cp
		}(),
		LifecycleLatencyMax: func() map[string]uint64 {
			m.lifecycleLatencyMu.Lock()
			defer m.lifecycleLatencyMu.Unlock()
			if m.lifecycleLatencyMax == nil {
				return map[string]uint64{}
			}
			cp := make(map[string]uint64, len(m.lifecycleLatencyMax))
			for k, v := range m.lifecycleLatencyMax {
				cp[k] = v
			}
			return cp
		}(),
		LifecycleLatencyP50: func() map[string]uint64 {
			m.lifecycleLatencyMu.Lock()
			defer m.lifecycleLatencyMu.Unlock()
			if m.lifecycleLatencyRes == nil {
				return map[string]uint64{}
			}
			out := make(map[string]uint64, len(m.lifecycleLatencyRes))
			for k, lr := range m.lifecycleLatencyRes { // derive sample size
				writes := lr.writes
				if writes == 0 {
					continue
				}
				size := uint64(len(lr.samples))
				sampleSize := writes
				if sampleSize > size {
					sampleSize = size
				}
				buf := make([]uint64, 0, sampleSize)
				for i := uint64(0); i < sampleSize; i++ {
					pos := (writes - 1 - i) & (size - 1)
					buf = append(buf, lr.samples[pos])
				}
				sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
				rank := int(0.50*float64(len(buf)-1) + 0.5)
				if rank < 0 {
					rank = 0
				}
				if rank >= len(buf) {
					rank = len(buf) - 1
				}
				out[k] = buf[rank]
			}
			return out
		}(),
		LifecycleLatencyP90: func() map[string]uint64 {
			m.lifecycleLatencyMu.Lock()
			defer m.lifecycleLatencyMu.Unlock()
			if m.lifecycleLatencyRes == nil {
				return map[string]uint64{}
			}
			out := make(map[string]uint64, len(m.lifecycleLatencyRes))
			for k, lr := range m.lifecycleLatencyRes {
				writes := lr.writes
				if writes == 0 {
					continue
				}
				size := uint64(len(lr.samples))
				sampleSize := writes
				if sampleSize > size {
					sampleSize = size
				}
				buf := make([]uint64, 0, sampleSize)
				for i := uint64(0); i < sampleSize; i++ {
					pos := (writes - 1 - i) & (size - 1)
					buf = append(buf, lr.samples[pos])
				}
				sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
				rank := int(0.90*float64(len(buf)-1) + 0.5)
				if rank < 0 {
					rank = 0
				}
				if rank >= len(buf) {
					rank = len(buf) - 1
				}
				out[k] = buf[rank]
			}
			return out
		}(),
		LifecycleLatencyP99: func() map[string]uint64 {
			m.lifecycleLatencyMu.Lock()
			defer m.lifecycleLatencyMu.Unlock()
			if m.lifecycleLatencyRes == nil {
				return map[string]uint64{}
			}
			out := make(map[string]uint64, len(m.lifecycleLatencyRes))
			for k, lr := range m.lifecycleLatencyRes {
				writes := lr.writes
				if writes == 0 {
					continue
				}
				size := uint64(len(lr.samples))
				sampleSize := writes
				if sampleSize > size {
					sampleSize = size
				}
				buf := make([]uint64, 0, sampleSize)
				for i := uint64(0); i < sampleSize; i++ {
					pos := (writes - 1 - i) & (size - 1)
					buf = append(buf, lr.samples[pos])
				}
				sort.Slice(buf, func(i, j int) bool { return buf[i] < buf[j] })
				rank := int(0.99*float64(len(buf)-1) + 0.5)
				if rank < 0 {
					rank = 0
				}
				if rank >= len(buf) {
					rank = len(buf) - 1
				}
				out[k] = buf[rank]
			}
			return out
		}(),
		LastPersistUnix:             atomic.LoadUint64(&m.lastPersistUnix),
		ObligationsExecuted:         atomic.LoadUint64(&m.obligationsExecuted),
		ObligationsFailed:           atomic.LoadUint64(&m.obligationsFailed),
		ObligationLatencyCount:      atomic.LoadUint64(&m.obligationLatencyCount),
		ObligationLatencyTotalNS:    atomic.LoadUint64(&m.obligationLatencyTotalNS),
		ObligationLatencyMaxNS:      atomic.LoadUint64(&m.obligationLatencyMaxNS),
		MandatoryObligationFailures: atomic.LoadUint64(&m.mandatoryObligationFailures),
	}
}

// EnvelopeV1IssuedCount returns issued V1 envelope count.
func (m *Memory) EnvelopeV1IssuedCount() uint64 { return atomic.LoadUint64(&m.envelopeV1Issued) }

// EnvelopeV2IssuedCount returns issued V2 envelope count.
func (m *Memory) EnvelopeV2IssuedCount() uint64 { return atomic.LoadUint64(&m.envelopeV2Issued) }

// EnvelopeDigestMismatchCount returns total digest mismatches observed.
func (m *Memory) EnvelopeDigestMismatchCount() uint64 {
	return atomic.LoadUint64(&m.envelopeDigestMismatch)
}

// EnvelopeIssuanceCadenceAvgSeconds returns approximate average issuance interval seconds (if any observations recorded)
func (m *Memory) EnvelopeIssuanceCadenceAvgSeconds() float64 {
	cnt := atomic.LoadUint64(&m.envelopeIssuanceCadenceCount)
	if cnt == 0 {
		return 0
	}
	totNs := atomic.LoadUint64(&m.envelopeIssuanceCadenceTotal)
	return float64(totNs) / 1e9 / float64(cnt)
}

// EnvelopeV2AdoptionRatio returns last stored adoption ratio (0..1).
func (m *Memory) EnvelopeV2AdoptionRatio() float64 {
	bits := atomic.LoadUint64(&m.envelopeV2AdoptionRatioBits)
	return math.Float64frombits(bits)
}

// EnvelopeV1SunsetPhase returns current sunset phase enum.
func (m *Memory) EnvelopeV1SunsetPhase() uint64 { return atomic.LoadUint64(&m.envelopeV1SunsetPhase) }

// SunsetPhaseSatisfactionProgress returns last stored progress ratio (0..1)
func (m *Memory) SunsetPhaseSatisfactionProgress() float64 {
	bits := atomic.LoadUint64(&m.sunsetPhaseSatisfactionBits)
	return math.Float64frombits(bits)
}

// LastEnvelopeIssuanceUnix returns unix seconds of last envelope issuance (0 when none yet)
func (m *Memory) LastEnvelopeIssuanceUnix() uint64 {
	return atomic.LoadUint64(&m.lastEnvelopeIssuanceUnix)
}

// SetLastEnvelopeIssuanceUnix sets last issuance unix seconds.
func (m *Memory) SetLastEnvelopeIssuanceUnix(ts uint64) {
	atomic.StoreUint64(&m.lastEnvelopeIssuanceUnix, ts)
}

// BLS Proof-of-Possession counters
func (m *Memory) IncBLSPoPChallengesIssued()     { atomic.AddUint64(&m.blsPoPChallengesIssued, 1) }
func (m *Memory) IncBLSPoPVerifications()        { atomic.AddUint64(&m.blsPoPVerifications, 1) }
func (m *Memory) IncBLSPoPVerificationFailures() { atomic.AddUint64(&m.blsPoPVerificationFailures, 1) }

// Ensure Memory implements Metrics.
var _ Metrics = (*Memory)(nil)
