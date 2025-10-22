package metrics

// Prometheus adapter for the lightweight in‑memory metrics interface.
//
// Rationale:
// The project uses a deliberately tiny Metrics interface (counters + latency
// observation) to keep core packages decoupled from any specific telemetry
// vendor libraries. For production / integration environments we still want to
// expose standard Prometheus metrics. This adapter implements the Metrics
// interface while delegating to Prometheus counters + histogram so it can be
// passed anywhere a Metrics is expected (e.g. WithMetrics(...)). Tests and
// benchmarks can continue to rely on the pure in‑memory implementation.
//
// Metric naming conventions follow: <namespace>_<subsystem>_<metric>_total for
// monotonically increasing counters. Latency is exported as a histogram with
// explicit buckets sized for sub‑millisecond to ~100ms ranges which should
// cover typical delegation validation latencies. (Adjust / make configurable
// later if real workloads exceed.)
//
// Gauge style metrics are not needed yet; if later added we can extend the
// interface or provide a richer wrapper while keeping backward compatibility.

import (
	"errors"
	"strings"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
)

// Receipt chain integrity status constants (avoid goconst duplication):
const (
	receiptsIntegrityOK          = "ok"
	receiptsIntegrityMismatch    = "mismatch"
	receiptsIntegrityUnconfigured = "unconfigured"
	receiptsIntegrityLegacy      = "legacy" // treated same as unconfigured
)

// PrometheusMetrics implements Metrics backed by Prometheus collectors.
type PrometheusMetrics struct {
	reg prom.Registerer

	delegationsCreated                     prom.Counter
	signaturesIssued                       prom.Counter
	signatureIssueFailures                 prom.Counter
	signatureVerifications                 prom.Counter
	signatureVerificationFailures          prom.Counter
	envelopeV1Issued                       prom.Counter
	envelopeV2Issued                       prom.Counter
	envelopeV2AdoptionRatio                prom.Gauge       // ratio 0-1 of V2 issuance vs total (best-effort)
	envelopeDigestMismatch                 prom.Counter     // canonical digest mismatch at verification events
	envelopeDigestMismatchReason           *prom.CounterVec // labeled digest mismatch reasons
	envelopeIssuanceCadenceHist            prom.Histogram   // histogram of seconds between consecutive envelope issuances
	envelopeV1SunsetPhaseGauge             prom.Gauge       // gauge enumerating sunset phase (0..5)
	sunsetPhaseSatisfactionGauge           prom.Gauge       // gauge 0..1 satisfaction progress toward next phase promotion window
	multiSignatureVerifications            prom.Counter
	multiSignatureVerificationFailures     prom.Counter
	multiSignatureWeightFailures           prom.Counter
	multiSignatureStructuralFailures       prom.Counter
	multiSignatureDigestFailures           prom.Counter
	multiSignaturePublicKeyMissingFailures prom.Counter
	multiSignatureInvalidSignatureFailures prom.Counter
	multiSignatureThresholdFailures        prom.Counter
	multiSignatureVerificationLatency      prom.Histogram
	revocationIntegrityFailures            prom.Counter
	validationLatency                      prom.Histogram
	signaturePublicKeyMissing              prom.Counter
	anchorAttempts                         prom.Counter
	anchorFailures                         prom.Counter
	replayHits                             prom.Counter
	replayMisses                           prom.Counter
	replayStoreErrors                      prom.Counter
	replayStoreLatency                     prom.Histogram
	scopeViolations                        prom.Counter
	restrictionViolations                  prom.Counter
	unauthorizedDecisions                  prom.Counter
	expiredDelegations                     prom.Counter
	revokedDelegations                     prom.Counter
	// Capability enforcement decision counters
	capabilityEnforceAllowed               prom.Counter
	capabilityEnforceDenied                prom.Counter
	modelLimitExceeded                     prom.Counter
	modelOutputLimitExceeded               prom.Counter
	modelRateLimitExceeded                 prom.Counter
	modelUserInputLimitExceeded            prom.Counter
	modelUserOutputLimitExceeded           prom.Counter
	modelUserRateLimitExceeded             prom.Counter
	modelUnknown                           prom.Counter
	modelLimitSurges                       prom.Counter
	delegationStatusTransitions            prom.Counter
	delegationStatusTransitionFailures     prom.Counter
	tokenStatusTransitions                 prom.Counter
	tokenStatusTransitionFailures          prom.Counter
	capabilityAnchorEmitted                prom.Counter
	capabilityAnchorSkipped                prom.Counter
	capabilityRegistryHashChanged          prom.Counter
	obligationsExecuted                    prom.Counter
	obligationsFailed                      prom.Counter
	// RawPOA embedding counters
	envelopeRawPOAEmbedded               prom.Counter
	envelopeRawPOATooLarge               prom.Counter
	capabilityAnchorLastWriteUnix        uint64         // stored locally for status endpoint exposure via type assertion (not a Prom metric yet)
	capabilityAnchorLastWriteGauge       prom.Gauge     // gauge exposing last write unix seconds (optional)
	capabilityAnchorAgeGauge             prom.Gauge     // gauge exposing current age seconds (set externally)
	capabilityAnchorStaleGauge           prom.Gauge     // gauge 0/1 stale state
	capabilityAnchorEmissionIntervalHist prom.Histogram // histogram of successful emission intervals (seconds)
	capabilityAnchorEmissionJitterGauge  prom.Gauge     // rolling stddev of last N intervals
	// Notarization metrics
	capabilityAnchorNotarizationLatencyHist prom.Histogram // latency histogram for external notarization operations
	capabilityAnchorNotarizedAgeGauge       prom.Gauge     // age in seconds since last successful notarization receipt
	capabilityAnchorNotarizationFailures    prom.Counter   // failures submitting hash to external notary
	// Provider-labeled variants (added for enhanced attribution). Backward compatibility: existing methods write to unlabeled collectors.
	capabilityAnchorNotarizationLatencyHistVec *prom.HistogramVec // labels: provider
	capabilityAnchorNotarizationFailuresVec    *prom.CounterVec   // labels: provider
	// Receipt chain integrity gauge (ok=1 mismatch=0 unconfigured=-1)
	capabilityAnchorNotarizationReceiptsIntegrityGauge prom.Gauge
	// Receipt chain last verification age gauge (seconds since last verification; 0 if never)
	capabilityAnchorNotarizationReceiptsLastVerifyAgeGauge prom.Gauge
	// External capability anchoring provider metrics (distinct from notarization)
	externalAnchorLatencyHist       prom.Histogram
	externalAnchorLatencyHistVec    *prom.HistogramVec // provider labeled
	externalAnchorFailures          prom.Counter
	externalAnchorFailuresVec       *prom.CounterVec // provider labeled
	externalAnchorAttempts          prom.Counter
	externalAnchorAttemptsVec       *prom.CounterVec // provider labeled
	externalAnchorForcedFailures    prom.Counter
	externalAnchorForcedFailuresVec *prom.CounterVec // provider labeled forced failures
	externalAnchorAgeGauge          prom.Gauge       // seconds since last external anchor success
	externalAnchorLastHashGauge     prom.Gauge       // expose last external anchor hash length as proxy (0 none)
	// External anchor receipt chain metrics
	externalAnchorReceiptsIntegrityGauge     prom.Gauge
	externalAnchorReceiptsLastVerifyAgeGauge prom.Gauge
	externalAnchorReceiptsTotalCounter       prom.Counter
	// decision labeled counter (action, resource, outcome)
	decisionCounter       *prom.CounterVec
	decisionReasonCounter *prom.CounterVec // labels: action,resource,outcome,reason
	// lifecycle labeled counters
	tokenLifecycleCounter      *prom.CounterVec   // labels: old_status,new_status,outcome
	delegationLifecycleCounter *prom.CounterVec   // labels: old_status,new_status,outcome
	lifecycleTransitionLatency *prom.HistogramVec // labels: entity (token|delegation), outcome
}

// PrometheusAdapterOptions allows optional customization when constructing.
type PrometheusAdapterOptions struct {
	Namespace string          // e.g. "gauth"
	Subsystem string          // e.g. "delegation" (used as prefix component)
	Registry  prom.Registerer // optional; defaults to global if nil
	Buckets   []float64       // optional custom latency buckets
}

// DefaultBuckets target fast in‑memory validation path (<1ms typical) while
// still tracking tail latencies up to 100ms.
var DefaultBuckets = []float64{
	0.0001, 0.00025, 0.0005,
	0.001, 0.0025, 0.005,
	0.010, 0.025, 0.050,
	0.075, 0.100,
}

// NewPrometheusMetrics constructs a Prometheus-backed Metrics implementation.
// Safe for repeated invocation; if collectors are already registered and the
// same registry is reused the function returns the existing metrics (the
// Prometheus client returns an AlreadyRegisteredError which we treat as reuse).
func NewPrometheusMetrics(opts PrometheusAdapterOptions) *PrometheusMetrics {
	if opts.Namespace == "" {
		opts.Namespace = "gauth"
	}
	if opts.Subsystem == "" {
		opts.Subsystem = "rfc0111"
	}
	if len(opts.Buckets) == 0 {
		opts.Buckets = DefaultBuckets
	}
	reg := opts.Registry
	if reg == nil {
		reg = prom.DefaultRegisterer
	}

	labels := prom.Labels{}
	msLatency := prom.NewHistogram(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "multi_signature_verification_latency_seconds", Help: "Latency of multi-signature verification operations", Buckets: opts.Buckets, ConstLabels: labels})
	if err := reg.Register(msLatency); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if h, ok2 := are.ExistingCollector.(prom.Histogram); ok2 {
				msLatency = h
			}
		}
	}
	// Helper to build fully qualified counter.
	fqCounter := func(name, help string) prom.Counter {
		c := prom.NewCounter(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: name, Help: help, ConstLabels: labels})
		if err := reg.Register(c); err != nil {
			if are, ok := err.(prom.AlreadyRegisteredError); ok {
				return are.ExistingCollector.(prom.Counter)
			}
		}
		return c
	}
	hist := prom.NewHistogram(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "validation_latency_seconds", Help: "Delegation validation latency", Buckets: opts.Buckets, ConstLabels: labels})
	// Replay store latency histogram uses same bucket set (fast path expected <1ms typical)
	rsHist := prom.NewHistogram(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "replay_store_latency_seconds", Help: "Latency of replay store (Seen/Record) operations", Buckets: opts.Buckets, ConstLabels: labels})
	if err := reg.Register(hist); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			// ExistingCollector is a Histogram (interface); attempt type assert.
			if h, ok2 := are.ExistingCollector.(prom.Histogram); ok2 {
				hist = h
			}
		}
	}
	if err := reg.Register(rsHist); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if h, ok2 := are.ExistingCollector.(prom.Histogram); ok2 {
				rsHist = h
			}
		}
	}
	pm := &PrometheusMetrics{
		reg:                                    reg,
		delegationsCreated:                     fqCounter("delegations_created_total", "Delegations (POAs) successfully created"),
		signaturesIssued:                       fqCounter("signatures_issued_total", "POA signatures successfully issued"),
		signatureIssueFailures:                 fqCounter("signature_issue_failures_total", "Failed attempts to issue a POA signature"),
		signatureVerifications:                 fqCounter("signature_verifications_total", "Successful signature verifications"),
		signatureVerificationFailures:          fqCounter("signature_verification_failures_total", "Failed signature verifications"),
		envelopeV1Issued:                       fqCounter("envelope_v1_issued_total", "Delegation tokens issued using envelope version 1"),
		envelopeV2Issued:                       fqCounter("envelope_v2_issued_total", "Delegation tokens issued using envelope version 2"),
		envelopeDigestMismatch:                 fqCounter("envelope_digest_mismatch_total", "Canonical digest mismatch detected during envelope verification"),
		envelopeRawPOAEmbedded:                 fqCounter("envelope_raw_poa_embedded_total", "Delegation tokens issued with embedded RawPOA canonical serialization"),
		envelopeRawPOATooLarge:                 fqCounter("envelope_raw_poa_too_large_total", "Delegation tokens where RawPOA embedding omitted due to size limit"),
		multiSignatureVerifications:            fqCounter("multi_signature_verifications_total", "Successful multi-signature threshold verifications"),
		multiSignatureVerificationFailures:     fqCounter("multi_signature_verification_failures_total", "Failed multi-signature threshold verifications (generic)"),
		multiSignatureWeightFailures:           fqCounter("multi_signature_weight_failures_total", "Multi-signature weight threshold failures"),
		multiSignatureStructuralFailures:       fqCounter("multi_signature_structural_failures_total", "Multi-signature structural precondition failures"),
		multiSignatureDigestFailures:           fqCounter("multi_signature_digest_failures_total", "Multi-signature canonical digest failures"),
		multiSignaturePublicKeyMissingFailures: fqCounter("multi_signature_public_key_missing_failures_total", "Multi-signature public key missing events"),
		multiSignatureInvalidSignatureFailures: fqCounter("multi_signature_invalid_signature_failures_total", "Multi-signature invalid signature cryptographic failures"),
		multiSignatureThresholdFailures:        fqCounter("multi_signature_threshold_failures_total", "Multi-signature count-based threshold failures"),
		multiSignatureVerificationLatency:      msLatency,
		revocationIntegrityFailures:            fqCounter("revocation_integrity_failures_total", "Revocation chain integrity verification failures"),
		signaturePublicKeyMissing:              fqCounter("signature_public_key_missing_total", "Signature present but public key not found (soft skip)"),
		validationLatency:                      hist,
		anchorAttempts:                         fqCounter("anchor_attempt_total", "Attempts to externally anchor chain tip"),
		anchorFailures:                         fqCounter("anchor_failure_total", "Failures anchoring chain tip"),
		replayHits:                             fqCounter("replay_hits_total", "Replay token detections (rejected)"),
		replayMisses:                           fqCounter("replay_misses_total", "Unique tokens accepted (first-seen)"),
		replayStoreErrors:                      fqCounter("replay_store_errors_total", "Errors communicating with replay store (fail-open)"),
		replayStoreLatency:                     rsHist,
		scopeViolations:                        fqCounter("scope_violations_total", "Scope validation violations"),
		restrictionViolations:                  fqCounter("restriction_violations_total", "Restriction validation violations"),
		unauthorizedDecisions:                  fqCounter("unauthorized_total", "Unauthorized decisions (authz denied due to policy)"),
		expiredDelegations:                     fqCounter("expired_delegations_total", "Expired delegations encountered in validation"),
		revokedDelegations:                     fqCounter("revoked_delegations_total", "Revoked delegations encountered in validation"),
		capabilityEnforceAllowed:               fqCounter("capability_enforce_allowed_total", "Capability enforcement allow decisions"),
		capabilityEnforceDenied:                fqCounter("capability_enforce_denied_total", "Capability enforcement denied decisions"),
			modelLimitExceeded:                     fqCounter("model_limit_exceeded_total", "Model input token limit exceeded decisions"),
			modelOutputLimitExceeded:               fqCounter("model_output_limit_exceeded_total", "Model output token limit exceeded decisions"),
			modelRateLimitExceeded:                 fqCounter("model_rate_limit_exceeded_total", "Model per-minute request rate limit exceeded decisions"),
			modelUserInputLimitExceeded:            fqCounter("model_user_input_limit_exceeded_total", "Per-user scoped model input token limit exceeded decisions"),
			modelUserOutputLimitExceeded:           fqCounter("model_user_output_limit_exceeded_total", "Per-user scoped model output token limit exceeded decisions"),
			modelUserRateLimitExceeded:             fqCounter("model_user_rate_limit_exceeded_total", "Per-user scoped model per-minute request rate limit exceeded decisions"),
			modelUnknown:                           fqCounter("model_unknown_total", "Unknown model validation requests denied due to strict mode"),
			modelLimitSurges:                       fqCounter("model_limit_exceed_surge_total", "Model limit exceed surge detection triggers"),
		delegationStatusTransitions:            fqCounter("delegation_status_transitions_total", "Successful delegation status transitions"),
		delegationStatusTransitionFailures:     fqCounter("delegation_status_transition_failures_total", "Failed delegation status transitions"),
		tokenStatusTransitions:                 fqCounter("token_status_transitions_total", "Successful token status transitions"),
		tokenStatusTransitionFailures:          fqCounter("token_status_transition_failures_total", "Failed token status transitions"),
		capabilityAnchorEmitted:                fqCounter("capability_anchor_emitted_total", "Capability registry anchor artifacts emitted"),
		capabilityAnchorSkipped:                fqCounter("capability_anchor_skipped_total", "Capability anchor emission attempts skipped due to interval throttle"),
		capabilityRegistryHashChanged:          fqCounter("capability_registry_hash_changed_total", "Capability registry hash change events (semantic changes)"),
		obligationsExecuted:                    fqCounter("obligations_executed_total", "Successfully executed obligations/advice actions"),
		obligationsFailed:                      fqCounter("obligations_failed_total", "Failed obligation/advice executions"),
	}
	// Capability anchoring gauges (best-effort registration; ignore AlreadyRegistered errors)
	createGauge := func(name, help string) prom.Gauge {
		g := prom.NewGauge(prom.GaugeOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: name, Help: help, ConstLabels: labels})
		if err := reg.Register(g); err != nil {
			if are, ok := err.(prom.AlreadyRegisteredError); ok {
				if eg, ok2 := are.ExistingCollector.(prom.Gauge); ok2 {
					g = eg
				}
			}
		}
		return g
	}
	pm.capabilityAnchorLastWriteGauge = createGauge("capability_anchor_last_write_seconds", "Unix epoch seconds of last capability anchor artifact emission")
	pm.envelopeV2AdoptionRatio = createGauge("envelope_v2_adoption_ratio", "Ratio (0-1) of envelope V2 issuance vs total issuance")
	pm.envelopeIssuanceCadenceHist = prom.NewHistogram(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "envelope_issuance_cadence_seconds", Help: "Interval in seconds between consecutive envelope issuances (any version)", Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120, 300}, ConstLabels: labels})
	if err := reg.Register(pm.envelopeIssuanceCadenceHist); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if h, ok2 := are.ExistingCollector.(prom.Histogram); ok2 {
				pm.envelopeIssuanceCadenceHist = h
			}
		}
	}
	pm.envelopeV1SunsetPhaseGauge = createGauge("envelope_v1_sunset_phase", "Envelope V1 sunset lifecycle phase (0 Pilot,1 Broad,2 Stabilization,3 SoftDep,4 Sunset,5 PostVerify)")
	pm.sunsetPhaseSatisfactionGauge = createGauge("envelope_v1_sunset_phase_satisfaction_progress", "Fraction (0-1) of current phase promotion window satisfied under threshold criteria")
	pm.capabilityAnchorAgeGauge = createGauge("capability_anchor_age_seconds", "Seconds since last capability anchor artifact emission (0 when never emitted)")
	pm.capabilityAnchorStaleGauge = createGauge("capability_anchor_stale", "Capability anchor stale state (1 if age exceeds SLA threshold, else 0)")
	// Emission interval histogram & jitter gauge
	intervalBuckets := []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800}
	cih := prom.NewHistogram(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_anchor_emission_interval_seconds", Help: "Histogram of intervals between successful capability anchor emissions", Buckets: intervalBuckets, ConstLabels: labels})
	if err := reg.Register(cih); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if h, ok2 := are.ExistingCollector.(prom.Histogram); ok2 {
				cih = h
			}
		}
	}
	pm.capabilityAnchorEmissionIntervalHist = cih
	pm.capabilityAnchorEmissionJitterGauge = createGauge("capability_anchor_emission_jitter_seconds", "Rolling standard deviation of recent capability anchor emission intervals")
	// Notarization specific collectors
	notarizationBuckets := []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30}
	nl := prom.NewHistogram(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_anchor_notarization_latency_seconds", Help: "Latency of external capability anchor notarization operations", Buckets: notarizationBuckets, ConstLabels: labels})
	if err := reg.Register(nl); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if h, ok2 := are.ExistingCollector.(prom.Histogram); ok2 {
				nl = h
			}
		}
	}
	pm.capabilityAnchorNotarizationLatencyHist = nl
	// Provider-labeled histogram vector
	nlVec := prom.NewHistogramVec(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_anchor_notarization_latency_provider_seconds", Help: "Latency of capability anchor notarization operations labeled by provider", Buckets: notarizationBuckets, ConstLabels: labels}, []string{"provider"})
	if err := reg.Register(nlVec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if hv, ok2 := are.ExistingCollector.(*prom.HistogramVec); ok2 {
				nlVec = hv
			}
		}
	}
	pm.capabilityAnchorNotarizationLatencyHistVec = nlVec
	pm.capabilityAnchorNotarizedAgeGauge = createGauge("capability_anchor_notarized_age_seconds", "Seconds since last successful capability anchor notarization receipt (0 when never)")
	pm.capabilityAnchorNotarizationFailures = fqCounter("capability_anchor_notarization_failures_total", "Failures submitting capability anchor hash to external notary service")
	// Provider-labeled failure counter vector
	nfVec := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_anchor_notarization_failures_provider_total", Help: "Capability anchor notarization failures labeled by provider", ConstLabels: labels}, []string{"provider"})
	if err := reg.Register(nfVec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				nfVec = cv
			}
		}
	}
	pm.capabilityAnchorNotarizationFailuresVec = nfVec
	// Receipt chain integrity gauge
	rInt := prom.NewGauge(prom.GaugeOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_anchor_notarization_receipts_integrity", Help: "Integrity status of notarization receipt persistence chain (ok=1 mismatch=0 unconfigured=-1)", ConstLabels: labels})
	if err := reg.Register(rInt); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if eg, ok2 := are.ExistingCollector.(prom.Gauge); ok2 {
				rInt = eg
			}
		}
	}
	pm.capabilityAnchorNotarizationReceiptsIntegrityGauge = rInt
	// Last verification age gauge
	lv := prom.NewGauge(prom.GaugeOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_anchor_notarization_receipts_last_verify_age_seconds", Help: "Seconds since last successful receipt chain integrity verification (0 when never)", ConstLabels: labels})
	if err := reg.Register(lv); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if eg, ok2 := are.ExistingCollector.(prom.Gauge); ok2 {
				lv = eg
			}
		}
	}
	pm.capabilityAnchorNotarizationReceiptsLastVerifyAgeGauge = lv
	// External anchor metrics (latency + attempts/failures) using similar bucket sizing to notarization
	exAnchorBuckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5}
	eal := prom.NewHistogram(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "external_anchor_latency_seconds", Help: "Latency of external capability anchoring provider operations", Buckets: exAnchorBuckets, ConstLabels: labels})
	if err := reg.Register(eal); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if h, ok2 := are.ExistingCollector.(prom.Histogram); ok2 {
				eal = h
			}
		}
	}
	pm.externalAnchorLatencyHist = eal
	ealv := prom.NewHistogramVec(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "external_anchor_latency_provider_seconds", Help: "Latency of external capability anchoring operations labeled by provider", Buckets: exAnchorBuckets, ConstLabels: labels}, []string{"provider"})
	if err := reg.Register(ealv); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if hv, ok2 := are.ExistingCollector.(*prom.HistogramVec); ok2 {
				ealv = hv
			}
		}
	}
	pm.externalAnchorLatencyHistVec = ealv
	eaf := fqCounter("external_anchor_failures_total", "Failures performing external capability anchoring")
	pm.externalAnchorFailures = eaf
	// Forced failures (deterministic override path) counters
	eff := fqCounter("external_anchor_forced_failures_total", "Forced external capability anchoring failures (deterministic override before probabilistic model)")
	pm.externalAnchorForcedFailures = eff
	effVec := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "external_anchor_forced_failures_provider_total", Help: "Forced external capability anchoring failures labeled by provider", ConstLabels: labels}, []string{"provider"})
	if err := reg.Register(effVec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				effVec = cv
			}
		}
	}
	pm.externalAnchorForcedFailuresVec = effVec
	eafVec := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "external_anchor_failures_provider_total", Help: "External capability anchoring failures labeled by provider", ConstLabels: labels}, []string{"provider"})
	if err := reg.Register(eafVec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				eafVec = cv
			}
		}
	}
	pm.externalAnchorFailuresVec = eafVec
	eaAtt := fqCounter("external_anchor_attempts_total", "Attempts to perform external capability anchoring")
	pm.externalAnchorAttempts = eaAtt
	eaAttVec := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "external_anchor_attempts_provider_total", Help: "External capability anchoring attempts labeled by provider", ConstLabels: labels}, []string{"provider"})
	if err := reg.Register(eaAttVec); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				eaAttVec = cv
			}
		}
	}
	pm.externalAnchorAttemptsVec = eaAttVec
	pm.externalAnchorAgeGauge = createGauge("external_anchor_age_seconds", "Seconds since last successful external capability anchoring (0 when never)")
	pm.externalAnchorLastHashGauge = createGauge("external_anchor_last_hash_len", "Length of last external anchor hash (0 when none)")
	// External anchor receipt chain collectors
	exrInt := prom.NewGauge(prom.GaugeOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_external_anchor_receipts_integrity", Help: "Integrity status of external anchor receipt chain (ok=1 mismatch=0 unconfigured=-1)", ConstLabels: labels})
	if err := reg.Register(exrInt); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if eg, ok2 := are.ExistingCollector.(prom.Gauge); ok2 {
				exrInt = eg
			}
		}
	}
	pm.externalAnchorReceiptsIntegrityGauge = exrInt
	exrAge := prom.NewGauge(prom.GaugeOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "capability_external_anchor_receipts_last_verify_age_seconds", Help: "Seconds since last external anchor receipt chain integrity verification (0 when never)", ConstLabels: labels})
	if err := reg.Register(exrAge); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if eg, ok2 := are.ExistingCollector.(prom.Gauge); ok2 {
				exrAge = eg
			}
		}
	}
	pm.externalAnchorReceiptsLastVerifyAgeGauge = exrAge
	exrTot := fqCounter("capability_external_anchor_receipts_total", "Total successful external anchor receipts persisted")
	pm.externalAnchorReceiptsTotalCounter = exrTot
	// digest mismatch reasons
	dmrc := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "envelope_digest_mismatch_reason_total", Help: "Envelope digest mismatches labeled by reason", ConstLabels: labels}, []string{"reason"})
	if err := reg.Register(dmrc); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				dmrc = cv
			}
		}
	}
	pm.envelopeDigestMismatchReason = dmrc
	// decisionCounter vector
	dc := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "decision_total", Help: "Authorization decisions by action, resource, outcome", ConstLabels: labels}, []string{"action", "resource", "outcome"})
	if err := reg.Register(dc); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				dc = cv
			}
		}
	}
	pm.decisionCounter = dc
	// decisionReasonCounter vector
	drc := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "decision_reason_total", Help: "Authorization decisions by action, resource, outcome, reason", ConstLabels: labels}, []string{"action", "resource", "outcome", "reason"})
	if err := reg.Register(drc); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				drc = cv
			}
		}
	}
	pm.decisionReasonCounter = drc
	// lifecycle labeled counters
	tlc := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "token_lifecycle_transition_total", Help: "Token lifecycle transitions labeled by old_status,new_status,outcome", ConstLabels: labels}, []string{"old_status", "new_status", "outcome"})
	if err := reg.Register(tlc); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				tlc = cv
			}
		}
	}
	dlc := prom.NewCounterVec(prom.CounterOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "delegation_lifecycle_transition_total", Help: "Delegation lifecycle transitions labeled by old_status,new_status,outcome", ConstLabels: labels}, []string{"old_status", "new_status", "outcome"})
	if err := reg.Register(dlc); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if cv, ok2 := are.ExistingCollector.(*prom.CounterVec); ok2 {
				dlc = cv
			}
		}
	}
	pm.tokenLifecycleCounter = tlc
	pm.delegationLifecycleCounter = dlc
	// lifecycle transition latency histogram
	lth := prom.NewHistogramVec(prom.HistogramOpts{Namespace: opts.Namespace, Subsystem: opts.Subsystem, Name: "lifecycle_transition_seconds", Help: "Latency of lifecycle status transitions", Buckets: opts.Buckets, ConstLabels: labels}, []string{"entity", "outcome"})
	if err := reg.Register(lth); err != nil {
		if are, ok := err.(prom.AlreadyRegisteredError); ok {
			if hv, ok2 := are.ExistingCollector.(*prom.HistogramVec); ok2 {
				lth = hv
			}
		}
	}
	pm.lifecycleTransitionLatency = lth
	return pm
}

// Ensure interface compliance.
var _ Metrics = (*PrometheusMetrics)(nil)

// Counter increments
func (p *PrometheusMetrics) IncDelegationsCreated()            { p.delegationsCreated.Inc() }
func (p *PrometheusMetrics) IncSignaturesIssued()              { p.signaturesIssued.Inc() }
func (p *PrometheusMetrics) IncSignatureIssueFailures()        { p.signatureIssueFailures.Inc() }
func (p *PrometheusMetrics) IncSignatureVerifications()        { p.signatureVerifications.Inc() }
func (p *PrometheusMetrics) IncSignatureVerificationFailures() { p.signatureVerificationFailures.Inc() }
func (p *PrometheusMetrics) IncEnvelopeV1Issued()              { p.envelopeV1Issued.Inc() }
func (p *PrometheusMetrics) IncEnvelopeV2Issued()              { p.envelopeV2Issued.Inc() }
func (p *PrometheusMetrics) SetEnvelopeV2AdoptionRatio(r float64) {
	if p.envelopeV2AdoptionRatio != nil {
		p.envelopeV2AdoptionRatio.Set(r)
	}
}
func (p *PrometheusMetrics) IncEnvelopeDigestMismatch() {
	if p.envelopeDigestMismatch != nil {
		p.envelopeDigestMismatch.Inc()
	}
}
func (p *PrometheusMetrics) IncEnvelopeDigestMismatchReason(reason string) {
	if p.envelopeDigestMismatchReason != nil {
		if reason == "" {
			reason = "other"
		}
		p.envelopeDigestMismatchReason.WithLabelValues(reason).Inc()
	}
}
func (p *PrometheusMetrics) ObserveEnvelopeIssuanceCadence(seconds float64) {
	if p.envelopeIssuanceCadenceHist != nil {
		p.envelopeIssuanceCadenceHist.Observe(seconds)
	}
}
func (p *PrometheusMetrics) SetEnvelopeV1SunsetPhase(phase int) {
	if p.envelopeV1SunsetPhaseGauge != nil {
		if phase < 0 {
			phase = 0
		}
		p.envelopeV1SunsetPhaseGauge.Set(float64(phase))
	}
}

// SetSunsetPhaseSatisfactionProgress sets 0..1 gauge indicating fraction of window satisfied.
func (p *PrometheusMetrics) SetSunsetPhaseSatisfactionProgress(progress float64) {
	if p.sunsetPhaseSatisfactionGauge == nil {
		return
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	p.sunsetPhaseSatisfactionGauge.Set(progress)
}

// RawPOA embedding counters
// IncEnvelopeRawPOAEmbedded increments count of tokens issued with embedded canonical RawPOA JSON.
func (p *PrometheusMetrics) IncEnvelopeRawPOAEmbedded() {
	if p.envelopeRawPOAEmbedded != nil {
		p.envelopeRawPOAEmbedded.Inc()
	}
}

// IncEnvelopeRawPOATooLarge increments count of tokens where embedding skipped due to size cap.
func (p *PrometheusMetrics) IncEnvelopeRawPOATooLarge() {
	if p.envelopeRawPOATooLarge != nil {
		p.envelopeRawPOATooLarge.Inc()
	}
}
func (p *PrometheusMetrics) IncMultiSignatureVerifications() { p.multiSignatureVerifications.Inc() }
func (p *PrometheusMetrics) IncMultiSignatureVerificationFailures() {
	p.multiSignatureVerificationFailures.Inc()
}
func (p *PrometheusMetrics) IncMultiSignatureWeightFailures() { p.multiSignatureWeightFailures.Inc() }
func (p *PrometheusMetrics) IncMultiSignatureStructuralFailures() {
	p.multiSignatureStructuralFailures.Inc()
}
func (p *PrometheusMetrics) IncMultiSignatureDigestFailures() { p.multiSignatureDigestFailures.Inc() }
func (p *PrometheusMetrics) IncMultiSignaturePublicKeyMissing() {
	p.multiSignaturePublicKeyMissingFailures.Inc()
}
func (p *PrometheusMetrics) IncMultiSignatureInvalidSignatureFailures() {
	p.multiSignatureInvalidSignatureFailures.Inc()
}
func (p *PrometheusMetrics) IncMultiSignatureThresholdFailures() {
	p.multiSignatureThresholdFailures.Inc()
}
func (p *PrometheusMetrics) ObserveMultiSignatureVerificationLatency(d time.Duration) {
	p.multiSignatureVerificationLatency.Observe(d.Seconds())
}
func (p *PrometheusMetrics) IncRevocationIntegrityFailures() { p.revocationIntegrityFailures.Inc() }
func (p *PrometheusMetrics) IncSignaturePublicKeyMissing()   { p.signaturePublicKeyMissing.Inc() }
func (p *PrometheusMetrics) IncAnchorAttempts()              { p.anchorAttempts.Inc() }
func (p *PrometheusMetrics) IncAnchorFailures()              { p.anchorFailures.Inc() }
func (p *PrometheusMetrics) IncReplayHits()                  { p.replayHits.Inc() }
func (p *PrometheusMetrics) IncReplayMisses()                { p.replayMisses.Inc() }
func (p *PrometheusMetrics) ObserveValidationLatency(d time.Duration) {
	p.validationLatency.Observe(d.Seconds())
}
func (p *PrometheusMetrics) IncReplayStoreErrors() { p.replayStoreErrors.Inc() }
func (p *PrometheusMetrics) ObserveReplayStoreLatency(d time.Duration) {
	p.replayStoreLatency.Observe(d.Seconds())
}
func (p *PrometheusMetrics) IncScopeViolations()             { p.scopeViolations.Inc() }
func (p *PrometheusMetrics) IncRestrictionViolations()       { p.restrictionViolations.Inc() }
func (p *PrometheusMetrics) IncUnauthorized()                { p.unauthorizedDecisions.Inc() }
func (p *PrometheusMetrics) IncExpired()                     { p.expiredDelegations.Inc() }
func (p *PrometheusMetrics) IncRevoked()                     { p.revokedDelegations.Inc() }
func (p *PrometheusMetrics) IncDelegationStatusTransitions() { p.delegationStatusTransitions.Inc() }
func (p *PrometheusMetrics) IncDelegationStatusTransitionFailures() {
	p.delegationStatusTransitionFailures.Inc()
}
func (p *PrometheusMetrics) IncCapabilityEnforceAllowed() { if p.capabilityEnforceAllowed != nil { p.capabilityEnforceAllowed.Inc() } }
func (p *PrometheusMetrics) IncCapabilityEnforceDenied()  { if p.capabilityEnforceDenied != nil { p.capabilityEnforceDenied.Inc() } }
func (p *PrometheusMetrics) IncModelLimitExceeded() { if p.modelLimitExceeded != nil { p.modelLimitExceeded.Inc() } }
func (p *PrometheusMetrics) IncModelOutputLimitExceeded() { if p.modelOutputLimitExceeded != nil { p.modelOutputLimitExceeded.Inc() } }
func (p *PrometheusMetrics) IncModelRateLimitExceeded() { if p.modelRateLimitExceeded != nil { p.modelRateLimitExceeded.Inc() } }
func (p *PrometheusMetrics) IncModelUserInputLimitExceeded() { if p.modelUserInputLimitExceeded != nil { p.modelUserInputLimitExceeded.Inc() } }
func (p *PrometheusMetrics) IncModelUserOutputLimitExceeded() { if p.modelUserOutputLimitExceeded != nil { p.modelUserOutputLimitExceeded.Inc() } }
func (p *PrometheusMetrics) IncModelUserRateLimitExceeded() { if p.modelUserRateLimitExceeded != nil { p.modelUserRateLimitExceeded.Inc() } }
func (p *PrometheusMetrics) IncModelUnknown() { if p.modelUnknown != nil { p.modelUnknown.Inc() } }
func (p *PrometheusMetrics) IncModelLimitSurge() { if p.modelLimitSurges != nil { p.modelLimitSurges.Inc() } }
func (p *PrometheusMetrics) IncTokenStatusTransitions()        { p.tokenStatusTransitions.Inc() }
func (p *PrometheusMetrics) IncTokenStatusTransitionFailures() { p.tokenStatusTransitionFailures.Inc() }
func (p *PrometheusMetrics) IncCapabilityAnchorEmitted()       { p.capabilityAnchorEmitted.Inc() }
func (p *PrometheusMetrics) IncCapabilityAnchorSkipped()       { p.capabilityAnchorSkipped.Inc() }
func (p *PrometheusMetrics) IncCapabilityRegistryHashChanged() { p.capabilityRegistryHashChanged.Inc() }
func (p *PrometheusMetrics) IncObligationsExecuted() {
	if p.obligationsExecuted != nil {
		p.obligationsExecuted.Inc()
	}
}
func (p *PrometheusMetrics) IncObligationsFailed() {
	if p.obligationsFailed != nil {
		p.obligationsFailed.Inc()
	}
}
func (p *PrometheusMetrics) SetCapabilityAnchorLastWriteUnix(ts uint64) {
	p.capabilityAnchorLastWriteUnix = ts
	if p.capabilityAnchorLastWriteGauge != nil {
		p.capabilityAnchorLastWriteGauge.Set(float64(ts))
	}
}

// SetCapabilityAnchorAgeSeconds updates age gauge (call from server SLA monitor loop).
func (p *PrometheusMetrics) SetCapabilityAnchorAgeSeconds(age uint64) {
	if p.capabilityAnchorAgeGauge != nil {
		p.capabilityAnchorAgeGauge.Set(float64(age))
	}
}

// SetCapabilityAnchorStale sets stale gauge (1=stale,0=fresh).
func (p *PrometheusMetrics) SetCapabilityAnchorStale(stale bool) {
	if p.capabilityAnchorStaleGauge != nil {
		if stale {
			p.capabilityAnchorStaleGauge.Set(1)
		} else {
			p.capabilityAnchorStaleGauge.Set(0)
		}
	}
}

// ObserveCapabilityAnchorEmissionInterval records a successful emission interval (seconds).
func (p *PrometheusMetrics) ObserveCapabilityAnchorEmissionInterval(d time.Duration) {
	if p.capabilityAnchorEmissionIntervalHist != nil {
		p.capabilityAnchorEmissionIntervalHist.Observe(d.Seconds())
	}
}

// SetCapabilityAnchorEmissionJitter sets rolling stddev gauge.
func (p *PrometheusMetrics) SetCapabilityAnchorEmissionJitter(jitterSeconds float64) {
	if p.capabilityAnchorEmissionJitterGauge != nil {
		p.capabilityAnchorEmissionJitterGauge.Set(jitterSeconds)
	}
}

// ObserveCapabilityAnchorNotarizationLatency records notarization latency duration.
func (p *PrometheusMetrics) ObserveCapabilityAnchorNotarizationLatency(d time.Duration) {
	if p.capabilityAnchorNotarizationLatencyHist != nil {
		p.capabilityAnchorNotarizationLatencyHist.Observe(d.Seconds())
	}
}

// ObserveCapabilityAnchorNotarizationLatencyProvider records latency with provider label.
func (p *PrometheusMetrics) ObserveCapabilityAnchorNotarizationLatencyProvider(provider string, d time.Duration) {
	if provider == "" {
		provider = "_"
	}
	if p.capabilityAnchorNotarizationLatencyHistVec != nil {
		p.capabilityAnchorNotarizationLatencyHistVec.WithLabelValues(provider).Observe(d.Seconds())
	}
	// Also record in unlabeled histogram for continuity.
	p.ObserveCapabilityAnchorNotarizationLatency(d)
}

// SetCapabilityAnchorNotarizedAgeSeconds sets age since last notarization.
func (p *PrometheusMetrics) SetCapabilityAnchorNotarizedAgeSeconds(age uint64) {
	if p.capabilityAnchorNotarizedAgeGauge != nil {
		p.capabilityAnchorNotarizedAgeGauge.Set(float64(age))
	}
}

// IncCapabilityAnchorNotarizationFailures increments notarization failure counter.
func (p *PrometheusMetrics) IncCapabilityAnchorNotarizationFailures() {
	if p.capabilityAnchorNotarizationFailures != nil {
		p.capabilityAnchorNotarizationFailures.Inc()
	}
}

// IncCapabilityAnchorNotarizationFailuresProvider increments failure counter labeled with provider.
func (p *PrometheusMetrics) IncCapabilityAnchorNotarizationFailuresProvider(provider string) {
	if provider == "" {
		provider = "_"
	}
	if p.capabilityAnchorNotarizationFailuresVec != nil {
		p.capabilityAnchorNotarizationFailuresVec.WithLabelValues(provider).Inc()
	}
	p.IncCapabilityAnchorNotarizationFailures()
}

// External anchor metrics helpers
func (p *PrometheusMetrics) ObserveExternalAnchorLatency(provider string, d time.Duration) {
	if provider == "" {
		provider = "_"
	}
	if p.externalAnchorLatencyHist != nil {
		p.externalAnchorLatencyHist.Observe(d.Seconds())
	}
	if p.externalAnchorLatencyHistVec != nil {
		p.externalAnchorLatencyHistVec.WithLabelValues(provider).Observe(d.Seconds())
	}
}

func (p *PrometheusMetrics) IncExternalAnchorFailures(provider string) {
	if p.externalAnchorFailures != nil {
		p.externalAnchorFailures.Inc()
	}
	if provider == "" {
		provider = "_"
	}
	if p.externalAnchorFailuresVec != nil {
		p.externalAnchorFailuresVec.WithLabelValues(provider).Inc()
	}
}

// IncExternalAnchorForcedFailures increments forced failure counters (unlabeled + provider labeled)
// IncExternalAnchorForcedFailures implements the Metrics interface (unlabeled forced failure increment only).
func (p *PrometheusMetrics) IncExternalAnchorForcedFailures() {
	if p.externalAnchorForcedFailures != nil {
		p.externalAnchorForcedFailures.Inc()
	}
	// Forced failures are also counted as general failures for aggregate visibility.
	if p.externalAnchorFailures != nil {
		p.externalAnchorFailures.Inc()
	}
}

// IncExternalAnchorForcedFailuresProvider increments provider-labeled forced failure counters (internal use by server code).
func (p *PrometheusMetrics) IncExternalAnchorForcedFailuresProvider(provider string) {
	if provider == "" {
		provider = "_"
	}
	if p.externalAnchorForcedFailures != nil {
		p.externalAnchorForcedFailures.Inc()
	}
	if p.externalAnchorForcedFailuresVec != nil {
		p.externalAnchorForcedFailuresVec.WithLabelValues(provider).Inc()
	}
	p.IncExternalAnchorFailures(provider)
}

func (p *PrometheusMetrics) IncExternalAnchorAttempts(provider string) {
	if p.externalAnchorAttempts != nil {
		p.externalAnchorAttempts.Inc()
	}
	if provider == "" {
		provider = "_"
	}
	if p.externalAnchorAttemptsVec != nil {
		p.externalAnchorAttemptsVec.WithLabelValues(provider).Inc()
	}
}

func (p *PrometheusMetrics) SetExternalAnchorAgeSeconds(age uint64) {
	if p.externalAnchorAgeGauge != nil {
		p.externalAnchorAgeGauge.Set(float64(age))
	}
}

func (p *PrometheusMetrics) SetExternalAnchorLastHashLen(n int) {
	if p.externalAnchorLastHashGauge != nil {
		p.externalAnchorLastHashGauge.Set(float64(n))
	}
}

// RecordExternalAnchorResult atomically records an external anchoring attempt result.
// It increments attempts, and conditionally failures or latency observation + hash len.
// Use this helper when performing a single provider anchoring operation to keep
// unlabeled and provider-labeled counters/histograms synchronized.
func (p *PrometheusMetrics) RecordExternalAnchorResult(provider string, success bool, latency time.Duration, hashLen int) {
	if provider == "" {
		provider = "_"
	}
	// Attempts counter (unlabeled + provider)
	if p.externalAnchorAttempts != nil {
		p.externalAnchorAttempts.Inc()
	}
	if p.externalAnchorAttemptsVec != nil {
		p.externalAnchorAttemptsVec.WithLabelValues(provider).Inc()
	}
	if !success {
		if p.externalAnchorFailures != nil {
			p.externalAnchorFailures.Inc()
		}
		if p.externalAnchorFailuresVec != nil {
			p.externalAnchorFailuresVec.WithLabelValues(provider).Inc()
		}
		return
	}
	// Success path latency observation
	if p.externalAnchorLatencyHist != nil {
		p.externalAnchorLatencyHist.Observe(latency.Seconds())
	}
	if p.externalAnchorLatencyHistVec != nil {
		p.externalAnchorLatencyHistVec.WithLabelValues(provider).Observe(latency.Seconds())
	}
	// Hash length gauge (only set on success)
	if hashLen > 0 && p.externalAnchorLastHashGauge != nil {
		p.externalAnchorLastHashGauge.Set(float64(hashLen))
	}
}

// SetCapabilityAnchorNotarizationReceiptsIntegrity maps status string to numeric gauge value.
// Status mapping: ok=1 mismatch=0 unconfigured=-1 (legacy treated as -1).
func (p *PrometheusMetrics) SetCapabilityAnchorNotarizationReceiptsIntegrity(status string) {
	if p.capabilityAnchorNotarizationReceiptsIntegrityGauge == nil {
		return
	}
	val := -1.0
	switch status {
	case receiptsIntegrityOK:
		val = 1
	case receiptsIntegrityMismatch:
		val = 0
	case receiptsIntegrityUnconfigured, receiptsIntegrityLegacy:
		val = -1
	}
	p.capabilityAnchorNotarizationReceiptsIntegrityGauge.Set(val)
}

// SetCapabilityAnchorNotarizationReceiptsLastVerifyAge sets seconds since last integrity verification.
func (p *PrometheusMetrics) SetCapabilityAnchorNotarizationReceiptsLastVerifyAge(age uint64) {
	if p.capabilityAnchorNotarizationReceiptsLastVerifyAgeGauge != nil {
		p.capabilityAnchorNotarizationReceiptsLastVerifyAgeGauge.Set(float64(age))
	}
}

// External anchor receipt chain helpers
func (p *PrometheusMetrics) SetExternalAnchorReceiptsIntegrity(status string) {
	if p.externalAnchorReceiptsIntegrityGauge == nil {
		return
	}
	val := -1.0
	switch status {
	case receiptsIntegrityOK:
		val = 1
	case receiptsIntegrityMismatch:
		val = 0
	case receiptsIntegrityUnconfigured:
		val = -1
	}
	p.externalAnchorReceiptsIntegrityGauge.Set(val)
}

func (p *PrometheusMetrics) SetExternalAnchorReceiptsLastVerifyAge(age uint64) {
	if p.externalAnchorReceiptsLastVerifyAgeGauge != nil {
		p.externalAnchorReceiptsLastVerifyAgeGauge.Set(float64(age))
	}
}
func (p *PrometheusMetrics) IncExternalAnchorReceiptsTotal() {
	if p.externalAnchorReceiptsTotalCounter != nil {
		p.externalAnchorReceiptsTotalCounter.Inc()
	}
}

// IncViolation implements the generic violation increment hook. For now we map
// specific hygiene categories onto existing scope/restriction counters. Unknown
// categories are ignored (future expansion may introduce dedicated counters).
func (p *PrometheusMetrics) IncViolation(cat interface{}) {
	switch c := cat.(type) {
	case string:
		if strings.Contains(c, "scope_utf8_invalid") || strings.Contains(c, "scope_control_char") {
			p.IncScopeViolations()
		} else if strings.Contains(c, "restriction_utf8_invalid") || strings.Contains(c, "restriction_control_char") {
			p.IncRestrictionViolations()
		}
	default:
		// no-op
	}
}

func (p *PrometheusMetrics) RecordDecision(action, resource, outcome string) {
	if action == "" {
		action = "_"
	}
	if resource == "" {
		resource = "_"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	p.decisionCounter.WithLabelValues(action, resource, outcome).Inc()
}

// RecordDecisionWithReason records decision with reason label.
func (p *PrometheusMetrics) RecordDecisionWithReason(action, resource, outcome, reason string) {
	if action == "" {
		action = "_"
	}
	if resource == "" {
		resource = "_"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	if reason == "" {
		reason = "_"
	}
	if p.decisionReasonCounter != nil {
		p.decisionReasonCounter.WithLabelValues(action, resource, outcome, reason).Inc()
	}
}

// RecordLifecycleTransition records a labeled lifecycle transition.
func (p *PrometheusMetrics) RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome string) {
	if oldStatus == "" {
		oldStatus = "_"
	}
	if newStatus == "" {
		newStatus = "_"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	switch entityType {
	case "token":
		if p.tokenLifecycleCounter != nil {
			p.tokenLifecycleCounter.WithLabelValues(oldStatus, newStatus, outcome).Inc()
		}
	case "delegation":
		if p.delegationLifecycleCounter != nil {
			p.delegationLifecycleCounter.WithLabelValues(oldStatus, newStatus, outcome).Inc()
		}
	default:
		// ignore unknown entity types silently
	}
}

// ObserveLifecycleTransitionLatency records latency of a lifecycle transition.
func (p *PrometheusMetrics) ObserveLifecycleTransitionLatency(entityType, outcome string, d time.Duration) {
	if entityType == "" {
		entityType = "_"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	if p.lifecycleTransitionLatency != nil {
		p.lifecycleTransitionLatency.WithLabelValues(entityType, outcome).Observe(d.Seconds())
	}
}

// Registry returns the underlying registerer (for advanced integration / custom handler tests).
func (p *PrometheusMetrics) Registry() prom.Registerer { return p.reg }

// Describe implements prom.Collector only if explicitly cast; no-op so we do not
// accidentally re-register (adapter itself does not implement Collector; the
// individual counters / histogram are registered already). Provided for future
// extension – returns error to signal misuse currently.
func (p *PrometheusMetrics) Describe(ch chan<- *prom.Desc) { /* intentionally empty */ }
func (p *PrometheusMetrics) Collect(ch chan<- prom.Metric) { /* intentionally empty */ }

// ErrNotCollector signals that PrometheusMetrics should not be registered directly.
var ErrNotCollector = errors.New("PrometheusMetrics adapter is not itself a collector; counters are already registered")
