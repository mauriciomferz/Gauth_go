package pdp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/internal/obligations"
	"github.com/mauriciomferz/AgentAuth/pkg/pdp/expr"
)

// Request represents an authorization evaluation request to the PDP.
type Request struct {
	Subject    string
	Action     string
	Resource   string
	Attributes map[string]string
	Time       time.Time
}

// Obligation represents an obligation that must be fulfilled by PEP after decision.
type Obligation struct {
	ID         string
	Attributes map[string]string
	// Mandatory indicates the obligation must succeed; failure may convert an allow decision to deny
	// when engine configured with denyOnMandatoryFailure.
	Mandatory bool
}

// EvaluationStep captures per-rule evaluation results for tracing.
type EvaluationStep struct {
	PolicyID string
	RuleID   string
	Matched  bool
	Effect   string // allow|deny|none
	ExprEval *ExprEval
}

// ExprEval wraps expression evaluation outcome.
type ExprEval struct {
	Expr     string
	Result   bool
	Error    string
	Duration time.Duration
}

// Decision is the final PDP output.
type Decision struct {
	Allow        bool
	Reason       string
	Policies     []string // contributing allow policies (if any)
	DenyPolicies []string // contributing deny policies (if any)
	Obligations  []Obligation
	Trace        []EvaluationStep
	Metadata     map[string]string
	// P0.2: Conflict diagnostics for policy analysis
	Conflicts    []PolicyConflict `json:"conflicts,omitempty"`
	HasConflicts bool             `json:"has_conflicts"`
}

// Engine defines the PDP interface.
type Engine interface {
	Evaluate(ctx context.Context, req Request) (Decision, error)
	Metrics() MetricsSnapshot
}

// Effect internal enum for strategy combination.
type Effect int

const (
	EffectNone Effect = iota
	EffectAllow
	EffectDeny
)

// Literal outcome/status strings reused across decision recording and combining strategies.
const (
	outcomeAllow      = "allow"
	outcomeDeny       = "deny"
	defaultDenyReason = "No matching policy - default deny"
	denyPolicyReason  = "Denied by deny policy"
	allowPolicyReason = "Allowed by allow policy"
)

// CombiningStrategy defines policy result combination behavior.
// P0.2: Enhanced to return conflict diagnostics alongside decisions.
type CombiningStrategy interface {
	Combine(steps []EvaluationStep) (final Effect, allowPolicies []string, denyPolicies []string, reason string)
	// CombineWithDiagnostics performs combination and detects conflicts
	CombineWithDiagnostics(
		steps []EvaluationStep,
		policies []Policy,
	) (
		final Effect,
		allowPolicies []string,
		denyPolicies []string,
		reason string,
		conflicts []PolicyConflict,
	)
	Name() string
}

// MetricsSnapshot basic metrics (extended later).
type MetricsSnapshot struct {
	Decisions     uint64
	Allows        uint64
	Denies        uint64
	ExprErrors    uint64
	LatencyCount  uint64
	LatencyMeanNs float64
	LatencyM2     float64
	PolicyMatches map[string]uint64
	// P2.13: Cache metrics
	CacheMetrics *PDPCacheMetrics
}

// --- Policy & Rule model (phase 1) ---

// Rule represents a single evaluable authorization rule.
type Rule struct {
	ID        string
	Actions   []string
	Resources []string
	Effect    string // "allow" | "deny"
	Expr      string // optional expression
}

// Policy groups rules under a common identifier and subject matching.
type Policy struct {
	ID       string
	Subjects []string // simple subject matching; later expand with attributes/roles
	Rules    []Rule
	Metadata map[string]string
	// Obligations enumerates mandatory post-decision actions when this policy contributes
	// to the final decision (e.g. logging, notification). These are executed after the
	// decision is finalized. Execution failures are counted via metrics but do not alter
	// the authorization outcome (Phase 1 semantics).
	Obligations []Obligation
}

// InMemoryEngine implements Engine with in-memory slice of policies.
type InMemoryEngine struct {
	policies []Policy
	strategy CombiningStrategy
	// metrics
	decisions           uint64
	allows              uint64
	denies              uint64
	exprErrors          uint64
	latencyCount        uint64
	latencyMeanNs       float64
	latencyM2           float64
	policyMatches       map[string]uint64 // policyID -> match count (any rule matched)
	latencyBuckets      []int64           // nanosecond upper bounds (sorted)
	latencyBucketCounts []uint64          // per-bucket counts
	externalMetrics     metrics.Metrics   // optional external metrics surface for decision labeling
	// obligations execution (optional)
	obligationExecutor     obligations.Executor
	obligationAuditPath    string // JSONL audit file path (append-only)
	denyOnMandatoryFailure bool   // configuration: mandatory obligation failure flips allow->deny
	// P2.1: Advice channel for non-mandatory recommendations
	adviceChannel AdviceChannel
	// P2.13: Decision caching (sec2.item5)
	cache DecisionCache
}

// NewInMemoryEngine creates a new PDP engine with provided combining strategy.
func NewInMemoryEngine(strategy CombiningStrategy) *InMemoryEngine {
	if strategy == nil {
		strategy = DenyOverridesStrategy{}
	}
	return &InMemoryEngine{
		strategy:      strategy,
		policies:      make([]Policy, 0),
		policyMatches: make(map[string]uint64),
		latencyBuckets: []int64{
			50_000,
			100_000,
			250_000,
			500_000,
			1_000_000,
			2_500_000,
			5_000_000,
			10_000_000,
			25_000_000,
			50_000_000,
			100_000_000,
		},
		latencyBucketCounts: make([]uint64, 11),
	}
}

// WithMetrics attaches an external metrics implementation for decision recording.
func (e *InMemoryEngine) WithMetrics(m metrics.Metrics) *InMemoryEngine {
	e.externalMetrics = m
	return e
}

// WithObligations configures an obligation executor and optional audit JSONL file path.
// If auditPath is non-empty, obligation execution results will be appended as JSON lines.
func (e *InMemoryEngine) WithObligations(exec obligations.Executor, auditPath string) *InMemoryEngine {
	e.obligationExecutor = exec
	e.obligationAuditPath = auditPath
	return e
}

// WithObligationFailureDenies configures whether mandatory obligation failures deny the decision.
func (e *InMemoryEngine) WithObligationFailureDenies(deny bool) *InMemoryEngine {
	e.denyOnMandatoryFailure = deny
	return e
}

// WithAdviceChannel configures an advice emission channel for non-mandatory recommendations.
// Clients should subscribe to ch.AdviceEvents() to receive advice events asynchronously.
func (e *InMemoryEngine) WithAdviceChannel(ch AdviceChannel) *InMemoryEngine {
	e.adviceChannel = ch
	return e
}

// WithCache configures decision caching with LRU+TTL eviction or Redis.
// P2.13 (sec2.item5): Enables 10-100x performance improvement for repeated requests.
//
// Parameters:
//   - cache: DecisionCache instance (nil disables caching)
//
// Configuration via environment:
//   - AGENTAUTH_PDP_CACHE_SIZE: Max entries (default 1000, 0=disabled)
//   - AGENTAUTH_PDP_CACHE_TTL: Entry lifetime (default 5m)
//
// Example:
//
//	cache := pdp.NewInMemoryCacheFromEnv()
//	engine := pdp.NewInMemoryEngine(strategy).WithCache(cache)
func (e *InMemoryEngine) WithCache(cache DecisionCache) *InMemoryEngine {
	e.cache = cache
	return e
}

// AddPolicy appends a policy (no deduplication yet).
func (e *InMemoryEngine) AddPolicy(p Policy) { e.policies = append(e.policies, p) }

// InvalidateCache clears all cached decisions.
//
// Use after:
//   - Policy updates (AddPolicy, RemovePolicy)
//   - Schema changes
//   - Administrative operations
func (e *InMemoryEngine) InvalidateCache() {
	if e.cache != nil {
		// As this is a best-effort invalidation often called outside request context,
		// we use Background context.
		// In future we might propagate context for tracing.
		_ = e.cache.InvalidateAll(context.Background())
	}
}

// Evaluate executes rule matching & combining.
//
//nolint:gocyclo // Core policy evaluation logic with multiple decision paths - refactoring would reduce readability
func (e *InMemoryEngine) Evaluate(ctx context.Context, req Request) (Decision, error) {
	start := time.Now()

	// P2.13: Check cache before policy evaluation
	if e.cache != nil {
		if cachedDec, found, _ := e.cache.Get(ctx, req); found {
			// Cache hit - return immediately
			e.decisions++
			e.recordLatency(time.Since(start))
			if cachedDec.Allow {
				e.allows++
			} else {
				e.denies++
			}
			return cachedDec, nil
		}
	}

	steps := make([]EvaluationStep, 0, 16)
	matchedObligations := make([]Obligation, 0, 8)
	// naive matching; optimize later with indexes
	for _, p := range e.policies {
		if !subjectMatches(req.Subject, p.Subjects) {
			continue
		}
		policyMatched := false
		for _, r := range p.Rules {
			if !actionMatches(req.Action, r.Actions) {
				continue
			}
			if !resourceMatches(req.Resource, r.Resources) {
				continue
			}
			step := EvaluationStep{
				PolicyID: p.ID,
				RuleID:   r.ID,
				Matched:  true,
				Effect:   r.Effect,
			}
			if r.Expr != "" {
				startExpr := time.Now()
				res, err := expr.Eval(r.Expr, req.Attributes, req.Time)
				step.ExprEval = &ExprEval{Expr: r.Expr, Result: res, Duration: time.Since(startExpr)}
				if err != nil {
					step.ExprEval.Error = err.Error()
					res = false
					e.exprErrors++
				}
				if !res { // expression failed, treat as non-match
					step.Matched = false
				}
			}
			steps = append(steps, step)
			policyMatched = true
		}
		if policyMatched {
			e.policyMatches[p.ID]++
			// Collect policy obligations (Phase 1: unconditional when any rule matched)
			if len(p.Obligations) > 0 {
				matchedObligations = append(matchedObligations, p.Obligations...)
			}
		}
	}
	final, allowPolicies, denyPolicies, reason := e.strategy.Combine(steps)
	dec := Decision{
		Allow:        final == EffectAllow,
		Reason:       reason,
		Policies:     allowPolicies,
		DenyPolicies: denyPolicies,
		Trace:        steps,
		Metadata: map[string]string{
			"combining_strategy": e.strategy.Name(),
		},
		Obligations: matchedObligations,
	}
	// Execute obligations prior to recording allow/deny metrics so mandatory failure can influence outcome.
	var mandatoryFailures []string
	if e.obligationExecutor != nil && len(dec.Obligations) > 0 {
		names := make([]string, 0, len(dec.Obligations))
		mandatoryFlags := make([]bool, 0, len(dec.Obligations))
		for _, o := range dec.Obligations {
			names = append(names, o.ID)
			mandatoryFlags = append(mandatoryFlags, o.Mandatory)
		}
		startExec := time.Now()
		results := e.obligationExecutor.Execute(ctx, names)
		perObligationStart := startExec
		for i, r := range results {
			end := time.Now()
			dur := end.Sub(perObligationStart)
			perObligationStart = end
			if e.externalMetrics != nil {
				e.externalMetrics.ObserveObligationLatency(dur)
			}
			if r.Success {
				if e.externalMetrics != nil {
					e.externalMetrics.IncObligationsExecuted()
				}
			} else {
				if e.externalMetrics != nil {
					e.externalMetrics.IncObligationsFailed()
				}
				if i < len(mandatoryFlags) && mandatoryFlags[i] {
					mandatoryFailures = append(mandatoryFailures, names[i])
					if e.externalMetrics != nil {
						e.externalMetrics.IncMandatoryObligationFailures()
					}
				}
			}

			// P2.1 (sec2.item3): Emit advice for non-mandatory obligations
			// Advice provides non-binding recommendations to clients for operational best practices.
			if i < len(mandatoryFlags) && !mandatoryFlags[i] && e.adviceChannel != nil {
				obligationType, obligationID := "", ""
				if i < len(dec.Obligations) {
					obligationType = dec.Obligations[i].Attributes["type"]
					obligationID = dec.Obligations[i].ID
				}
				adviceEvent := AdviceEvent{
					Timestamp:  time.Now(),
					Subject:    req.Subject,
					Action:     req.Action,
					Resource:   req.Resource,
					AdviceID:   obligationID,
					AdviceType: obligationType,
					Message:    fmt.Sprintf("Non-mandatory obligation '%s' executed", r.Name),
					Metadata: map[string]string{
						"success":     fmt.Sprintf("%t", r.Success),
						"duration_ms": fmt.Sprintf("%.3f", float64(dur.Microseconds())/1000.0),
					},
				}
				if r.Error != nil {
					adviceEvent.Metadata["error"] = r.Error.Error()
				}
				// Fire and forget - advice emission should not block decision flow
				_ = e.adviceChannel.Emit(ctx, adviceEvent)
			}
			if e.obligationAuditPath != "" {
				auditRec := struct {
					Timestamp  string  `json:"ts"`
					Subject    string  `json:"subject"`
					Action     string  `json:"action"`
					Resource   string  `json:"resource"`
					Allow      bool    `json:"allow"`
					Obligation string  `json:"obligation"`
					Index      int     `json:"index"`
					Success    bool    `json:"success"`
					DurationMS float64 `json:"duration_ms"`
					Mandatory  bool    `json:"mandatory"`
					Error      string  `json:"error,omitempty"`
				}{
					Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
					Subject:    req.Subject,
					Action:     req.Action,
					Resource:   req.Resource,
					Allow:      dec.Allow,
					Obligation: r.Name,
					Index:      i,
					Success:    r.Success,
					DurationMS: float64(dur.Microseconds()) / 1000.0,
					Mandatory:  i < len(mandatoryFlags) && mandatoryFlags[i],
				}
				if r.Error != nil {
					auditRec.Error = r.Error.Error()
				}
				func() {
					f, err := os.OpenFile(e.obligationAuditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
					if err != nil {
						return
					}
					defer func() { _ = f.Close() }()
					enc, err := json.Marshal(auditRec)
					if err != nil {
						return
					}
					_, _ = f.Write(append(enc, '\n'))
				}()
			}
		}
	}
	if dec.Allow && e.denyOnMandatoryFailure && len(mandatoryFailures) > 0 {
		dec.Allow = false
		dec.Reason = "Denied due to mandatory obligation failure"
		dec.Metadata["mandatory_obligation_failures"] = strings.Join(mandatoryFailures, ",")
	}
	// Now record latency & decision counters with final outcome.
	e.recordLatency(time.Since(start))
	e.decisions++
	if dec.Allow {
		e.allows++
	} else {
		e.denies++
	}
	if e.externalMetrics != nil {
		out := outcomeDeny
		if dec.Allow {
			out = outcomeAllow
		}
		e.externalMetrics.RecordDecision(req.Action, req.Resource, out, time.Duration(0))
		if !dec.Allow {
			e.externalMetrics.IncUnauthorized()
		}
	}

	// P2.13: Store decision in cache
	if e.cache != nil {
		// Mark as cache miss for this evaluation
		if dec.Metadata == nil {
			dec.Metadata = make(map[string]string)
		}
		dec.Metadata["cache_hit"] = "false"
		_ = e.cache.Set(ctx, req, dec)
	}

	return dec, nil
}

func (e *InMemoryEngine) Metrics() MetricsSnapshot {
	pm := make(map[string]uint64, len(e.policyMatches))
	for k, v := range e.policyMatches {
		pm[k] = v
	}
	snap := MetricsSnapshot{
		Decisions:     e.decisions,
		Allows:        e.allows,
		Denies:        e.denies,
		ExprErrors:    e.exprErrors,
		LatencyCount:  e.latencyCount,
		LatencyMeanNs: e.latencyMeanNs,
		LatencyM2:     e.latencyM2,
		PolicyMatches: pm,
	}

	// P2.13: Include cache metrics if enabled
	if e.cache != nil {
		cm := e.cache.GetMetrics()
		snap.CacheMetrics = &cm
	}

	return snap
}

// ExportPrometheus returns metrics lines in Prometheus exposition format (basic prototype).
// NOTE: For real usage prefer dedicated collector and proper HELP/TYPE declarations.
func (e *InMemoryEngine) ExportPrometheus() string {
	snap := e.Metrics()
	var b strings.Builder
	// HELP/TYPE lines
	fmt.Fprintf(&b, "# HELP pdp_decisions_total Total PDP evaluations\n")
	fmt.Fprintf(&b, "# TYPE pdp_decisions_total counter\n")
	fmt.Fprintf(&b, "pdp_decisions_total %d\n", snap.Decisions)
	fmt.Fprintf(&b, "# HELP pdp_decisions_allow_total Total allow decisions\n")
	fmt.Fprintf(&b, "# TYPE pdp_decisions_allow_total counter\n")
	fmt.Fprintf(&b, "pdp_decisions_allow_total %d\n", snap.Allows)
	fmt.Fprintf(&b, "# HELP pdp_decisions_deny_total Total deny decisions\n")
	fmt.Fprintf(&b, "# TYPE pdp_decisions_deny_total counter\n")
	fmt.Fprintf(&b, "pdp_decisions_deny_total %d\n", snap.Denies)
	fmt.Fprintf(&b, "# HELP pdp_expr_errors_total Expression evaluation errors\n")
	fmt.Fprintf(&b, "# TYPE pdp_expr_errors_total counter\n")
	fmt.Fprintf(&b, "pdp_expr_errors_total %d\n", snap.ExprErrors)

	// Latency histogram (convert ns to seconds)
	fmt.Fprintf(&b, "# HELP pdp_decision_latency_seconds Decision evaluation latency\n")
	fmt.Fprintf(&b, "# TYPE pdp_decision_latency_seconds histogram\n")
	var cumulative uint64
	for i, ub := range e.latencyBuckets {
		cumulative += e.latencyBucketCounts[i]
		secs := float64(ub) / 1e9
		fmt.Fprintf(&b, "pdp_decision_latency_seconds_bucket{le=\"%.6f\"} %d\n", secs, cumulative)
	}
	// +Inf bucket count equals total observations
	fmt.Fprintf(&b, "pdp_decision_latency_seconds_bucket{le=\"+Inf\"} %d\n", snap.LatencyCount)
	// Histogram count and sum (sum = mean * count)
	latencySumSeconds := (snap.LatencyMeanNs * float64(snap.LatencyCount)) / 1e9
	fmt.Fprintf(&b, "pdp_decision_latency_seconds_sum %.9f\n", latencySumSeconds)
	fmt.Fprintf(&b, "pdp_decision_latency_seconds_count %d\n", snap.LatencyCount)
	// policy matches
	if len(snap.PolicyMatches) > 0 {
		fmt.Fprintf(&b, "# HELP pdp_policy_matches_total Policy match occurrences\n")
		fmt.Fprintf(&b, "# TYPE pdp_policy_matches_total counter\n")
		for id, c := range snap.PolicyMatches {
			// sanitize policy id for label value (replace quotes)
			sid := strings.ReplaceAll(id, `"`, `\"`)
			fmt.Fprintf(&b, "pdp_policy_matches_total{policy=\"%s\"} %d\n", sid, c)
		}
	}

	// P2.13: Cache metrics
	if snap.CacheMetrics != nil {
		cm := snap.CacheMetrics
		fmt.Fprintf(&b, "# HELP pdp_cache_lookups_total Total cache lookups\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_lookups_total counter\n")
		fmt.Fprintf(&b, "pdp_cache_lookups_total %d\n", cm.Lookups)

		fmt.Fprintf(&b, "# HELP pdp_cache_hits_total Total cache hits\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_hits_total counter\n")
		fmt.Fprintf(&b, "pdp_cache_hits_total %d\n", cm.Hits)

		fmt.Fprintf(&b, "# HELP pdp_cache_misses_total Total cache misses\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_misses_total counter\n")
		fmt.Fprintf(&b, "pdp_cache_misses_total %d\n", cm.Misses)

		fmt.Fprintf(&b, "# HELP pdp_cache_hit_rate Cache hit rate (0.0-1.0)\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_hit_rate gauge\n")
		fmt.Fprintf(&b, "pdp_cache_hit_rate %.4f\n", cm.HitRate)

		fmt.Fprintf(&b, "# HELP pdp_cache_size Current cache size\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_size gauge\n")
		fmt.Fprintf(&b, "pdp_cache_size %d\n", cm.Size)

		fmt.Fprintf(&b, "# HELP pdp_cache_evictions_total Total LRU evictions\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_evictions_total counter\n")
		fmt.Fprintf(&b, "pdp_cache_evictions_total %d\n", cm.Evictions)

		fmt.Fprintf(&b, "# HELP pdp_cache_expirations_total Total TTL expirations\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_expirations_total counter\n")
		fmt.Fprintf(&b, "pdp_cache_expirations_total %d\n", cm.Expirations)

		fmt.Fprintf(&b, "# HELP pdp_cache_invalidations_total Total cache invalidations\n")
		fmt.Fprintf(&b, "# TYPE pdp_cache_invalidations_total counter\n")
		fmt.Fprintf(&b, "pdp_cache_invalidations_total %d\n", cm.Invalidations)
	}

	return b.String()
}

func (e *InMemoryEngine) recordLatency(d time.Duration) {
	val := float64(d.Nanoseconds())
	count := e.latencyCount + 1
	e.latencyCount = count
	if count == 1 {
		e.latencyMeanNs = val
		e.latencyM2 = 0
		return
	}
	delta := val - e.latencyMeanNs
	e.latencyMeanNs += delta / float64(count)
	delta2 := val - e.latencyMeanNs
	e.latencyM2 += delta * delta2
	// bucket assignment
	ns := d.Nanoseconds()
	for i, ub := range e.latencyBuckets {
		if ns <= ub {
			e.latencyBucketCounts[i]++
			break
		}
		// if larger than all explicit buckets it falls into +Inf (handled by count metric)
	}
}

// --- Combining Strategies ---

// DenyOverridesStrategy final deny if any deny rule matched; else allow if any allow; else deny (no match).
type DenyOverridesStrategy struct{}

func (DenyOverridesStrategy) Name() string { return "deny_overrides" }
func (DenyOverridesStrategy) Combine(steps []EvaluationStep) (Effect, []string, []string, string) {
	var allowIDs, denyIDs []string
	for _, s := range steps {
		switch s.Effect {
		case outcomeDeny:
			denyIDs = append(denyIDs, s.PolicyID)
		case outcomeAllow:
			allowIDs = append(allowIDs, s.PolicyID)
		}
	}
	if len(denyIDs) > 0 {
		return EffectDeny, allowIDs, denyIDs, denyPolicyReason
	}
	if len(allowIDs) > 0 {
		return EffectAllow, allowIDs, denyIDs, "Allowed by policy"
	}
	return EffectDeny, nil, nil, defaultDenyReason
}

// CombineWithDiagnostics performs combination and detects permit-deny conflicts
func (d DenyOverridesStrategy) CombineWithDiagnostics(
	steps []EvaluationStep,
	policies []Policy,
) (Effect, []string, []string, string, []PolicyConflict) {
	effect, allowIDs, denyIDs, reason := d.Combine(steps)

	var conflicts []PolicyConflict

	// Detect permit-deny conflict
	if len(allowIDs) > 0 && len(denyIDs) > 0 {
		allIDs := append(append([]string{}, allowIDs...), denyIDs...)
		conflicts = append(conflicts, PolicyConflict{
			ID:        "runtime-conflict-1",
			Type:      ConflictPermitDeny,
			Severity:  SeverityHigh,
			PolicyIDs: allIDs,
			Description: fmt.Sprintf(
				"Deny-overrides resolved conflict: %d policies allowed, %d policies denied",
				len(allowIDs),
				len(denyIDs),
			),
			Recommendation: "Deny policies took precedence. Consider making deny policies more specific " +
				"or removing redundant allow policies.",
			DetectedAt:     time.Now(),
			ResolutionHint: "DENY effect applied (deny-overrides strategy)",
		})
	}

	return effect, allowIDs, denyIDs, reason, conflicts
}

// PermitOverridesStrategy final allow if any allow rule matched; else deny if any deny; else deny (no match).
type PermitOverridesStrategy struct{}

func (PermitOverridesStrategy) Name() string { return "permit_overrides" }
func (PermitOverridesStrategy) Combine(steps []EvaluationStep) (Effect, []string, []string, string) {
	var allowIDs, denyIDs []string
	for _, s := range steps {
		switch s.Effect {
		case "deny":
			denyIDs = append(denyIDs, s.PolicyID)
		case "allow":
			allowIDs = append(allowIDs, s.PolicyID)
		}
	}
	if len(allowIDs) > 0 {
		return EffectAllow, allowIDs, denyIDs, allowPolicyReason
	}
	if len(denyIDs) > 0 {
		return EffectDeny, allowIDs, denyIDs, denyPolicyReason
	}
	return EffectDeny, nil, nil, defaultDenyReason
}

// CombineWithDiagnostics performs combination and detects permit-deny conflicts
func (p PermitOverridesStrategy) CombineWithDiagnostics(
	steps []EvaluationStep,
	policies []Policy,
) (Effect, []string, []string, string, []PolicyConflict) {
	effect, allowIDs, denyIDs, reason := p.Combine(steps)

	var conflicts []PolicyConflict

	// Detect permit-deny conflict
	if len(allowIDs) > 0 && len(denyIDs) > 0 {
		allIDs := append(append([]string{}, allowIDs...), denyIDs...)
		conflicts = append(conflicts, PolicyConflict{
			ID:        "runtime-conflict-1",
			Type:      ConflictPermitDeny,
			Severity:  SeverityCritical, // Higher severity for permit-overrides
			PolicyIDs: allIDs,
			Description: fmt.Sprintf(
				"Permit-overrides resolved conflict: %d policies allowed, %d policies denied",
				len(allowIDs),
				len(denyIDs),
			),
			Recommendation: "Allow policies took precedence. Consider adding mandatory obligations for audit logging " +
				"or making allow policies more restrictive.",
			DetectedAt:     time.Now(),
			ResolutionHint: "ALLOW effect applied (permit-overrides strategy) - ensure this is intended behavior",
		})
	}

	return effect, allowIDs, denyIDs, reason, conflicts
}

// FirstApplicableStrategy picks first step with allow/deny and uses its effect.
type FirstApplicableStrategy struct{}

func (FirstApplicableStrategy) Name() string { return "first_applicable" }
func (FirstApplicableStrategy) Combine(steps []EvaluationStep) (Effect, []string, []string, string) {
	for _, s := range steps {
		switch s.Effect {
		case outcomeDeny:
			return EffectDeny, nil, []string{s.PolicyID}, "Denied by first applicable policy"
		case outcomeAllow:
			return EffectAllow, []string{s.PolicyID}, nil, "Allowed by first applicable policy"
		}
	}
	return EffectDeny, nil, nil, defaultDenyReason
}

// CombineWithDiagnostics performs combination with conflict detection for first-applicable
func (f FirstApplicableStrategy) CombineWithDiagnostics(
	steps []EvaluationStep,
	policies []Policy,
) (Effect, []string, []string, string, []PolicyConflict) {
	effect, allowIDs, denyIDs, reason := f.Combine(steps)

	var conflicts []PolicyConflict

	// Check if there are multiple matching policies (potential ordering issue)
	var matchedPolicies []string
	for _, s := range steps {
		if s.Effect == outcomeAllow || s.Effect == outcomeDeny {
			matchedPolicies = append(matchedPolicies, s.PolicyID)
		}
	}

	if len(matchedPolicies) > 1 {
		conflicts = append(conflicts, PolicyConflict{
			ID:        "runtime-conflict-1",
			Type:      ConflictPriorityAmbiguity,
			Severity:  SeverityMedium,
			PolicyIDs: matchedPolicies,
			Description: fmt.Sprintf(
				"First-applicable strategy: %d policies matched, using first (%s)",
				len(matchedPolicies),
				matchedPolicies[0],
			),
			Recommendation: "Consider making policies mutually exclusive or using explicit priority ordering to avoid ambiguity.",
			DetectedAt:     time.Now(),
			ResolutionHint: fmt.Sprintf("Policy %s applied; remaining %d policies ignored", matchedPolicies[0], len(matchedPolicies)-1),
		})
	}

	return effect, allowIDs, denyIDs, reason, conflicts
}

// --- Match Helpers (simplistic; to be enhanced) ---
func subjectMatches(sub string, list []string) bool {
	for _, s := range list {
		if s == sub || s == "*" {
			return true
		}
	}
	return false
}

func actionMatches(action string, actions []string) bool {
	for _, a := range actions {
		if a == action || a == "*" {
			return true
		}
	}
	return false
}

func resourceMatches(res string, resources []string) bool {
	for _, r := range resources {
		if r == res || r == "*" {
			return true
		}
	}
	return false
}
