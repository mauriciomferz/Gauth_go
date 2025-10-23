package pdp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/pdp/expr"
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
type CombiningStrategy interface {
	Combine(steps []EvaluationStep) (final Effect, allowPolicies []string, denyPolicies []string, reason string)
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
}

// NewInMemoryEngine creates a new PDP engine with provided combining strategy.
func NewInMemoryEngine(strategy CombiningStrategy) *InMemoryEngine {
	if strategy == nil {
		strategy = DenyOverridesStrategy{}
	}
	return &InMemoryEngine{strategy: strategy, policies: make([]Policy, 0), policyMatches: make(map[string]uint64), latencyBuckets: []int64{50_000, 100_000, 250_000, 500_000, 1_000_000, 2_500_000, 5_000_000, 10_000_000, 25_000_000, 50_000_000, 100_000_000}, latencyBucketCounts: make([]uint64, 11)}
}

// WithMetrics attaches an external metrics implementation for decision recording.
func (e *InMemoryEngine) WithMetrics(m metrics.Metrics) *InMemoryEngine {
	e.externalMetrics = m
	return e
}

// AddPolicy appends a policy (no deduplication yet).
func (e *InMemoryEngine) AddPolicy(p Policy) { e.policies = append(e.policies, p) }

// Evaluate executes rule matching & combining.
func (e *InMemoryEngine) Evaluate(ctx context.Context, req Request) (Decision, error) {
	start := time.Now()
	steps := make([]EvaluationStep, 0, 16)
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
			step := EvaluationStep{PolicyID: p.ID, RuleID: r.ID, Matched: true, Effect: r.Effect}
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
		}
	}
	final, allowPolicies, denyPolicies, reason := e.strategy.Combine(steps)
	dec := Decision{Allow: final == EffectAllow, Reason: reason, Policies: allowPolicies, DenyPolicies: denyPolicies, Trace: steps, Metadata: map[string]string{"combining_strategy": e.strategy.Name()}}
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
		e.externalMetrics.RecordDecision(req.Action, req.Resource, out)
		if !dec.Allow {
			e.externalMetrics.IncUnauthorized()
		}
	}
	return dec, nil
}

func (e *InMemoryEngine) Metrics() MetricsSnapshot {
	pm := make(map[string]uint64, len(e.policyMatches))
	for k, v := range e.policyMatches {
		pm[k] = v
	}
	snap := MetricsSnapshot{Decisions: e.decisions, Allows: e.allows, Denies: e.denies, ExprErrors: e.exprErrors, LatencyCount: e.latencyCount, LatencyMeanNs: e.latencyMeanNs, LatencyM2: e.latencyM2, PolicyMatches: pm}
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
			safe := strings.ReplaceAll(id, "\"", "")
			fmt.Fprintf(&b, "pdp_policy_matches_total{policy=\"%s\"} %d\n", safe, c)
		}
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
