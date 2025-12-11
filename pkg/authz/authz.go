package authz

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Common metadata and operator string constants to reduce duplication.
const (
	metadataCacheHitTrue  = "true"
	metadataCacheHitFalse = "false"
	operatorRegex         = "regex"
)

// Request represents an authorization request
type Request struct {
	Subject  string            `json:"subject"`
	Resource string            `json:"resource"`
	Action   string            `json:"action"`
	Context  map[string]string `json:"context,omitempty"`
}

// Decision represents an authorization decision
type Decision struct {
	Allow    bool              `json:"allow"`
	Allowed  bool              `json:"allowed"` // Compatibility field
	Policies []string          `json:"policies,omitempty"`
	Reason   string            `json:"reason,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Permission represents a permission
type Permission struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
	Granted  bool     `json:"granted"`
}

// Subject, Action, Resource simple legacy compatibility wrappers
type Subject struct {
	ID string
}

type Action struct {
	Name string
}

type Resource struct {
	ID string
}

// AddPolicy adds a new policy
func (e *BasicEnforcer) AddPolicy(ctx context.Context, policy *Policy) error {
	e.policies[policy.ID] = policy
	return nil
}

// RemovePolicy removes a policy
func (e *BasicEnforcer) RemovePolicy(ctx context.Context, policyID string) error {
	delete(e.policies, policyID)
	return nil
}

// Authorize authorizes a request (compatibility method)
func (e *BasicEnforcer) Authorize(ctx context.Context, reqOrSubject interface{}, actionOrNil ...interface{}) (*Decision, error) {
	if req, ok := reqOrSubject.(*Request); ok {
		// New interface: single Request parameter
		decision, err := e.Evaluate(ctx, req)
		if decision != nil {
			decision.Allowed = decision.Allow // Set compatibility field for new interface too
		}
		return decision, err
	}

	// Legacy interface: separate Subject, Action, Resource parameters
	if len(actionOrNil) >= 2 {
		subject := reqOrSubject.(Subject)
		action := actionOrNil[0].(Action)
		resource := actionOrNil[1].(Resource)

		req := &Request{
			Subject:  subject.ID,
			Resource: resource.ID,
			Action:   action.Name,
			Context:  make(map[string]string),
		}

		decision, err := e.Evaluate(ctx, req)
		if decision != nil {
			decision.Allowed = decision.Allow // Set compatibility field
		}
		return decision, err
	}

	return nil, fmt.Errorf("invalid parameters")
}

// AuthorizeWithParams is an alias for Authorize for compatibility
func (e *BasicEnforcer) AuthorizeWithParams(ctx context.Context, subject Subject, action Action, resource Resource) (*Decision, error) {
	return e.Authorize(ctx, subject, action, resource)
}

// matchesPattern checks if a pattern matches a value, supporting wildcards
func (e *BasicEnforcer) matchesPattern(pattern, value string) bool {
	// Handle wildcard patterns
	if pattern == "*" {
		return true
	}

	// Handle prefix wildcard patterns like "/docs/*"
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(value) >= len(prefix) && value[:len(prefix)] == prefix
	}

	// Exact match
	return pattern == value
}

// Policy represents an authorization policy
type Policy struct {
	ID         string      `json:"id"`
	Subject    string      `json:"subject"`
	Resource   string      `json:"resource"`
	Actions    []string    `json:"actions"`
	Effect     Effect      `json:"effect"`
	Conditions []Condition `json:"conditions,omitempty"`
	// Expression is an optional advanced boolean expression evaluated against request context.
	// If present it MUST evaluate to true for the policy to match. Grammar supports identifiers,
	// boolean logic (&&, ||, !), comparison (==, !=, >, >=, <, <=), membership (in [..]). Identifiers:
	//   subject, resource, action, and any request context key directly (e.g. env) or via ctx.<key>.
	// Errors in parsing or evaluation fail CLOSED (policy does not match).
	Expression     string            `json:"expression,omitempty"`
	Validators     []string          `json:"validators,omitempty"` // validator IDs enforced (all must pass)
	Metadata       map[string]string `json:"metadata,omitempty"`
	Roles          []string          `json:"roles,omitempty"`           // optional role-based matching (RBAC)
	RequiredScopes []string          `json:"required_scopes,omitempty"` // all scopes must be present in request context
	Version        int64             `json:"version,omitempty"`         // policy set version (assigned by authorizer persistence)
	Obligations    []Obligation      `json:"obligations,omitempty"`     // mandatory or optional post-decision actions
	Advice         []Advice          `json:"advice,omitempty"`          // non-mandatory advisory actions (failures do not affect decision)
}

// Effect represents the effect of a policy
type Effect string

const (
	// Allow grants access
	Allow Effect = "allow"
	// Deny denies access
	Deny Effect = "deny"
)

// Condition represents a policy condition
type Condition struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

// MemoryAuthorizer implements Authorizer using in-memory policies
type MemoryAuthorizer struct {
	policiesMu sync.RWMutex // protects policies slice during reload
	policies   []Policy
	version    int64      // monotonically increasing policy set version
	versions   []struct { // snapshot history for rollback (shallow copy of slice)
		version  int64
		policies []Policy
	}
	roles map[string][]string // subject -> roles
	// caching fields
	cacheEnabled bool
	cacheTTL     time.Duration
	cache        map[string]cachedDecision
	cacheMu      sync.RWMutex // protects decision cache
	combining    CombiningStrategy
	// metrics counters (uint64 atomics)
	metricDecisions   uint64 // total decisions made
	metricCacheHits   uint64 // cache hit count
	metricCacheMisses uint64 // cache miss count
	metricReloads     uint64 // policy reloads (updated externally by persistence wrapper)
	// latency tracking (Welford): store count, mean (ns), M2
	latencyMu    sync.Mutex // protects latencyMean and latencyM2 (floats can't be atomic)
	latencyCount uint64
	latencyMean  float64
	latencyM2    float64
	// latency histogram buckets (nanoseconds upper bounds)
	latencyBuckets      []int64  // immutable after init
	latencyBucketCounts []uint64 // atomic counters per bucket
	// conflict counter
	metricConflicts uint64
	// regex compilation caching
	regexCache               map[string]*regexp.Regexp
	regexMu                  sync.RWMutex
	metricRegexCompiles      uint64 // number of successful regex compilations
	metricRegexCompileErrors uint64 // number of failed regex compilations
	// LRU/TTL tracking for regex cache
	regexLastAccess      map[string]time.Time // last access timestamps
	regexAddedAt         map[string]time.Time // insertion time (for TTL)
	regexCapacity        int                  // max entries (<=0 unlimited)
	regexTTL             time.Duration        // ttl duration (<=0 disabled)
	metricRegexEvictions uint64               // count of evicted patterns
	// regex match frequency tracking
	regexMatchCounts   map[string]uint64 // guarded by regexMu for updates
	metricRegexMatches uint64            // total successful regex matches (atomic)
	// Authorization LRU cache (Task 4)
	decisionCache *AuthorizationCache
	jurisdiction  string // current jurisdiction scope (empty => global)
	// Advice / obligation execution
	obligationExecutor ObligationExecutor
	metricsProvider    interface {
		IncObligationsExecuted()
		IncObligationsFailed()
		IncMandatoryObligationFailures()
		ObserveObligationLatency(d time.Duration)
		RecordDecision(action, resource, decision string, duration time.Duration)
	} // minimal metrics subset
	validatorRegistry *ValidatorRegistry
}

// NewMemoryAuthorizer creates a new in-memory authorizer
func NewMemoryAuthorizer() *MemoryAuthorizer {
	return &MemoryAuthorizer{
		policies: make([]Policy, 0),
		version:  1,
		versions: make([]struct {
			version  int64
			policies []Policy
		}, 0, 8),
		roles:               make(map[string][]string),
		cache:               make(map[string]cachedDecision),
		combining:           DenyOverrides, // sensible secure default
		regexCache:          make(map[string]*regexp.Regexp),
		regexLastAccess:     make(map[string]time.Time),
		regexAddedAt:        make(map[string]time.Time),
		regexMatchCounts:    make(map[string]uint64),
		regexCapacity:       256,
		latencyBuckets:      []int64{50_000, 100_000, 250_000, 500_000, 1_000_000, 2_500_000, 5_000_000, 10_000_000, 25_000_000, 50_000_000, 100_000_000},
		latencyBucketCounts: make([]uint64, 11),
		obligationExecutor:  &DefaultObligationExecutor{},
	}
}

// SetDecisionCache attaches an external authorization decision cache.
func (ma *MemoryAuthorizer) SetDecisionCache(c *AuthorizationCache) { ma.decisionCache = c }

// SetJurisdiction sets the active jurisdiction and invalidates cache (simplistic full flush for now).
func (ma *MemoryAuthorizer) SetJurisdiction(j string) {
	if j == ma.jurisdiction {
		return
	}
	ma.jurisdiction = j
	if ma.decisionCache != nil {
		ma.decisionCache.InvalidateAll()
	}
}

// SetObligationExecutor overrides default executor.
func (ma *MemoryAuthorizer) SetObligationExecutor(exec ObligationExecutor) {
	if exec != nil {
		ma.obligationExecutor = exec
	}
}

// SetMetricsProvider sets metrics subset implementation (Noop if nil).
func (ma *MemoryAuthorizer) SetMetricsProvider(mp interface {
	IncObligationsExecuted()
	IncObligationsFailed()
	IncMandatoryObligationFailures()
	ObserveObligationLatency(d time.Duration)
	RecordDecision(action, resource, decision string, duration time.Duration)
}) {
	ma.metricsProvider = mp
}

// SetValidatorRegistry attaches a validator registry.
func (ma *MemoryAuthorizer) SetValidatorRegistry(vr *ValidatorRegistry) { ma.validatorRegistry = vr }

// InvalidateOnCryptoRotation flushes decision cache on cryptographic key rotation events.
// External rotation managers SHOULD invoke this after completing a rotation to avoid serving decisions
// tied to previous key material (e.g., scope validations, signature-based attribute derivations).
func (ma *MemoryAuthorizer) InvalidateOnCryptoRotation() {
	if ma.decisionCache != nil {
		ma.decisionCache.InvalidateAll()
	}
}

// AuthorizationCacheMetrics returns snapshot metrics from attached decision cache (nil if none).
func (ma *MemoryAuthorizer) AuthorizationCacheMetrics() *AuthorizationCacheMetrics {
	if ma.decisionCache == nil {
		return nil
	}
	snap := ma.decisionCache.Snapshot()
	return &snap
}

// executePostDecision runs obligations (mandatory may flip allow->deny) and advice (never flips outcome).
func (ma *MemoryAuthorizer) executePostDecision(dec *Decision, policy Policy, req Request) {
	if ma.obligationExecutor == nil {
		return
	}
	// Obligations
	for _, ob := range policy.Obligations {
		start := time.Now()
		err := ma.obligationExecutor.Execute(ob, map[string]interface{}{"request_subject": req.Subject, "request_action": req.Action, "request_resource": req.Resource})
		if ma.metricsProvider != nil {
			ma.metricsProvider.ObserveObligationLatency(time.Since(start))
		}
		if err != nil {
			if ma.metricsProvider != nil {
				ma.metricsProvider.IncObligationsFailed()
			}
			if ob.Mandatory && dec.Allow {
				dec.Allow = false
				dec.Reason = fmt.Sprintf("mandatory obligation %s failed: %v", ob.ID, err)
				if ma.metricsProvider != nil {
					ma.metricsProvider.IncMandatoryObligationFailures()
				}
			}
			if dec.Metadata == nil {
				dec.Metadata = make(map[string]string)
			}
			dec.Metadata["obligation_failure"] = ob.ID
			continue
		}
		if ma.metricsProvider != nil {
			ma.metricsProvider.IncObligationsExecuted()
		}
	}
	// Advice (non-mandatory): failures recorded but no decision change
	for _, adv := range policy.Advice {
		start := time.Now()
		err := ma.obligationExecutor.Execute(Obligation{ID: adv.ID, Type: adv.Type, Params: adv.Params, Mandatory: false}, map[string]interface{}{"request_subject": req.Subject, "request_action": req.Action, "request_resource": req.Resource, "advice": true})
		if ma.metricsProvider != nil {
			ma.metricsProvider.ObserveObligationLatency(time.Since(start))
		}
		if err != nil {
			if ma.metricsProvider != nil {
				ma.metricsProvider.IncObligationsFailed()
			}
			if dec.Metadata == nil {
				dec.Metadata = make(map[string]string)
			}
			dec.Metadata["advice_failure"] = adv.ID
			continue
		}
		if ma.metricsProvider != nil {
			ma.metricsProvider.IncObligationsExecuted()
		}
	}
}

// currentPolicyVersion returns stable snapshot version used for caching (last snapshot boundary).
func (ma *MemoryAuthorizer) currentPolicyVersion() int64 {
	v := ma.version
	if v <= 1 {
		return 1
	}
	return v - 1
}

// SetRegexCacheCapacity sets maximum compiled regex entries retained (<=0 -> unlimited/no eviction).
func (ma *MemoryAuthorizer) SetRegexCacheCapacity(cap int) {
	ma.regexMu.Lock()
	ma.regexCapacity = cap
	ma.regexMu.Unlock()
}

// SetRegexCacheTTL sets a time-to-live for compiled regex entries (<=0 disables TTL expiry).
func (ma *MemoryAuthorizer) SetRegexCacheTTL(ttl time.Duration) {
	ma.regexMu.Lock()
	ma.regexTTL = ttl
	ma.regexMu.Unlock()
}

// pruneRegexCache evicts expired (TTL) and over-capacity entries using LRU.
func (ma *MemoryAuthorizer) pruneRegexCache() {
	ma.regexMu.Lock()
	now := time.Now()
	// TTL eviction first
	if ma.regexTTL > 0 {
		for pattern, added := range ma.regexAddedAt {
			if now.Sub(added) > ma.regexTTL {
				delete(ma.regexCache, pattern)
				delete(ma.regexLastAccess, pattern)
				delete(ma.regexAddedAt, pattern)
				atomic.AddUint64(&ma.metricRegexEvictions, 1)
			}
		}
	}
	// Capacity eviction (LRU scan) if needed
	if ma.regexCapacity > 0 && len(ma.regexCache) > ma.regexCapacity {
		excess := len(ma.regexCache) - ma.regexCapacity
		// build slice of patterns with lastAccess
		type entry struct {
			p string
			t time.Time
		}
		list := make([]entry, 0, len(ma.regexCache))
		for p := range ma.regexCache {
			list = append(list, entry{p, ma.regexLastAccess[p]})
		}
		// simple selection of oldest 'excess' entries
		for i := 0; i < excess; i++ {
			// find oldest
			oldIdx := -1
			var oldest time.Time
			for idx, e := range list {
				if e.p == "" {
					continue
				}
				if oldIdx == -1 || e.t.Before(oldest) {
					oldIdx = idx
					oldest = e.t
				}
			}
			if oldIdx == -1 {
				break
			}
			pat := list[oldIdx].p
			delete(ma.regexCache, pat)
			delete(ma.regexLastAccess, pat)
			delete(ma.regexAddedAt, pat)
			list[oldIdx].p = "" // mark removed
			atomic.AddUint64(&ma.metricRegexEvictions, 1)
		}
	}
	ma.regexMu.Unlock()
}

// CombiningStrategy defines how multiple matching policies are resolved.
type CombiningStrategy string

const (
	// DenyOverrides forces a final deny if any matching deny policy exists.
	DenyOverrides CombiningStrategy = "deny_overrides"
	// PermitOverrides forces a final allow if any matching allow policy exists (unless no allow match).
	PermitOverrides CombiningStrategy = "permit_overrides"
	// FirstApplicable selects the first matching policy as the final decision.
	FirstApplicable CombiningStrategy = "first_applicable"
)

// SetCombiningStrategy changes the strategy (invalid values ignored).
func (ma *MemoryAuthorizer) SetCombiningStrategy(s CombiningStrategy) {
	switch s {
	case DenyOverrides, PermitOverrides, FirstApplicable:
		ma.combining = s
	}
}

// cachedDecision wraps a Decision with expiry timestamp
type cachedDecision struct {
	decision Decision
	expires  time.Time
}

// EnableCaching configures caching with a given TTL.
func (ma *MemoryAuthorizer) EnableCaching(ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	ma.cacheEnabled = true
	ma.cacheTTL = ttl
}

// DisableCaching turns off decision caching.
func (ma *MemoryAuthorizer) DisableCaching() { ma.cacheEnabled = false }

// InvalidateAll clears the entire decision cache (used after policy reload).
func (ma *MemoryAuthorizer) InvalidateAll() {
	ma.cacheMu.Lock()
	ma.cache = make(map[string]cachedDecision)
	ma.cacheMu.Unlock()
}

// InvalidateSubject removes cached decisions for a given subject.
func (ma *MemoryAuthorizer) InvalidateSubject(subject string) {
	if !ma.cacheEnabled {
		return
	}
	ma.cacheMu.Lock()
	for k := range ma.cache {
		// key format: subject|resource|action|roles|scopes
		if strings.HasPrefix(k, subject+"|") {
			delete(ma.cache, k)
		}
	}
	ma.cacheMu.Unlock()
}

// InvalidateResource removes cached decisions for a given resource pattern (exact match or prefix if ends with *).
func (ma *MemoryAuthorizer) InvalidateResource(resource string) {
	if !ma.cacheEnabled {
		return
	}
	prefixMode := strings.HasSuffix(resource, "*")
	base := strings.TrimSuffix(resource, "*")
	ma.cacheMu.Lock()
	for k := range ma.cache {
		// subject|resource|...
		parts := strings.SplitN(k, "|", 3)
		if len(parts) < 2 {
			continue
		}
		res := parts[1]
		if (!prefixMode && res == resource) || (prefixMode && strings.HasPrefix(res, base)) {
			delete(ma.cache, k)
		}
	}
	ma.cacheMu.Unlock()
}

// cacheKey creates a key from request fields + context relevant values.
func (ma *MemoryAuthorizer) cacheKey(r Request) string {
	// Limit context elements to stable keys; iterate deterministically
	var b strings.Builder
	b.WriteString(r.Subject)
	b.WriteByte('|')
	b.WriteString(r.Resource)
	b.WriteByte('|')
	b.WriteString(r.Action)
	b.WriteByte('|')
	// include roles & scopes if present (impact policy matching)
	b.WriteString(r.Context["roles"])
	b.WriteByte('|')
	b.WriteString(r.Context["scopes"]) // final
	return b.String()
}

// AddPolicy adds a policy to the authorizer
func (ma *MemoryAuthorizer) AddPolicy(policy Policy) {
	// assign current version to policy and append
	policy.Version = ma.version
	ma.policies = append(ma.policies, policy)
}

// Snapshot creates a new version snapshot of current policies and increments version counter.
// Should be called after a batch of policy mutations to delineate a stable version boundary.
func (ma *MemoryAuthorizer) Snapshot() int64 {
	snap := make([]Policy, len(ma.policies))
	copy(snap, ma.policies)
	ma.versions = append(ma.versions, struct {
		version  int64
		policies []Policy
	}{version: ma.version, policies: snap})
	ma.version++
	return ma.version - 1
}

// ListVersions returns known snapshot version numbers in ascending order (excluding current working set if not snapshotted yet).
func (ma *MemoryAuthorizer) ListVersions() []int64 {
	out := make([]int64, 0, len(ma.versions))
	for _, v := range ma.versions {
		out = append(out, v.version)
	}
	return out
}

// Rollback replaces current policies with the snapshot of the specified version.
func (ma *MemoryAuthorizer) Rollback(version int64) error {
	for i := len(ma.versions) - 1; i >= 0; i-- { // reverse search (likely latest requested)
		if ma.versions[i].version == version {
			snap := make([]Policy, len(ma.versions[i].policies))
			copy(snap, ma.versions[i].policies)
			ma.policies = snap
			// Set working version to one greater than rolled back version to maintain monotonicity for future snapshots.
			ma.version = version + 1
			return nil
		}
	}
	return fmt.Errorf("rollback: version %d not found", version)
}

// AssignRoles attaches roles to a subject (replaces existing set).
func (ma *MemoryAuthorizer) AssignRoles(subject string, roles ...string) {
	ma.roles[subject] = roles
}

// Authorize makes an authorization decision

func (ma *MemoryAuthorizer) Authorize(ctx context.Context, request Request) (Decision, error) {
	start := time.Now()
	var lruKey string
	if ma.decisionCache != nil {
		lruKey = makeKey(request.Subject, request.Action, request.Resource, ma.currentPolicyVersion(), ma.jurisdiction)
		entry, ok := ma.decisionCache.Get(lruKey)
		if ok {
			if entry.PolicyVersion == ma.currentPolicyVersion() && entry.Jurisdiction == ma.jurisdiction {
				dec := entry.Decision
				if dec.Metadata == nil {
					dec.Metadata = make(map[string]string)
				}
				dec.Metadata["cache_hit"] = metadataCacheHitTrue
				atomic.AddUint64(&ma.metricDecisions, 1)
				atomic.AddUint64(&ma.metricCacheHits, 1)
				dur := time.Since(start)
				ma.recordLatency(dur)
				if ma.metricsProvider != nil {
					outcome := "deny"
					if dec.Allow {
						outcome = "allow"
					}
					ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, dur)
				}
				return dec, nil
			}
			ma.decisionCache.MarkStale(lruKey)
		}
	}
	if ma.cacheEnabled {
		key := ma.cacheKey(request)
		ma.cacheMu.RLock()
		cd, ok := ma.cache[key]
		ma.cacheMu.RUnlock()
		if ok {
			if time.Now().Before(cd.expires) {
				// create shallow copy to avoid mutating shared cached metadata
				orig := cd.decision
				var metaCopy map[string]string
				if orig.Metadata != nil {
					metaCopy = make(map[string]string, len(orig.Metadata)+1)
					for k, v := range orig.Metadata {
						metaCopy[k] = v
					}
				} else {
					metaCopy = make(map[string]string, 1)
				}
				metaCopy["cache_hit"] = metadataCacheHitTrue
				d := orig
				d.Metadata = metaCopy
				atomic.AddUint64(&ma.metricDecisions, 1)
				atomic.AddUint64(&ma.metricCacheHits, 1)
				dur := time.Since(start)
				ma.recordLatency(dur)
				if ma.metricsProvider != nil {
					outcome := "deny"
					if d.Allow {
						outcome = "allow"
					}
					ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, dur)
				}
				return d, nil
			}
			// expired
			ma.cacheMu.Lock()
			delete(ma.cache, key)
			ma.cacheMu.Unlock()
		}
		// initial miss path will set cache_hit=false on resulting decision
	}
	matched := make([]Policy, 0)
	var denyList, allowList []Policy
	ma.policiesMu.RLock()
	policies := ma.policies // make a local copy of the slice reference for iteration
	ma.policiesMu.RUnlock()
	for _, policy := range policies {
		if ma.matchesPolicy(request, policy) {
			matched = append(matched, policy)
			if policy.Effect == Deny {
				denyList = append(denyList, policy)
			} else {
				allowList = append(allowList, policy)
			}
			if ma.combining == FirstApplicable { // legacy shortcut
				dec := ma.buildDecisionFromPolicy(request, policy, start)
				ma.annotateConflict(&dec, denyList, allowList)
				if ma.metricsProvider != nil {
					outcome := "deny"
					if dec.Allow {
						outcome = "allow"
					}
					ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, time.Since(start))
				}
				return dec, nil
			}
		}
	}
	if len(matched) > 0 {
		switch ma.combining {
		case DenyOverrides:
			if len(denyList) > 0 {
				dec := ma.buildDecisionFromPolicy(request, denyList[0], start)
				ma.executePostDecision(&dec, denyList[0], request)
				ma.annotateConflict(&dec, denyList, allowList)
				if ma.metricsProvider != nil {
					outcome := "deny"
					if dec.Allow {
						outcome = "allow"
					}
					ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, time.Since(start))
				}
				return dec, nil
			}
			dec := ma.buildDecisionFromPolicy(request, allowList[0], start)
			ma.executePostDecision(&dec, allowList[0], request)
			ma.annotateConflict(&dec, denyList, allowList)
			if ma.metricsProvider != nil {
				outcome := "deny"
				if dec.Allow {
					outcome = "allow"
				}
				ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, time.Since(start))
			}
			return dec, nil
		case PermitOverrides:
			if len(allowList) > 0 {
				dec := ma.buildDecisionFromPolicy(request, allowList[0], start)
				ma.executePostDecision(&dec, allowList[0], request)
				ma.annotateConflict(&dec, denyList, allowList)
				if ma.metricsProvider != nil {
					outcome := "deny"
					if dec.Allow {
						outcome = "allow"
					}
					ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, time.Since(start))
				}
				return dec, nil
			}
			dec := ma.buildDecisionFromPolicy(request, denyList[0], start)
			ma.executePostDecision(&dec, denyList[0], request)
			ma.annotateConflict(&dec, denyList, allowList)
			if ma.metricsProvider != nil {
				outcome := "deny"
				if dec.Allow {
					outcome = "allow"
				}
				ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, time.Since(start))
			}
			return dec, nil
		default: // fallback first matched
			dec := ma.buildDecisionFromPolicy(request, matched[0], start)
			ma.executePostDecision(&dec, matched[0], request)
			ma.annotateConflict(&dec, denyList, allowList)
			if ma.metricsProvider != nil {
				outcome := "deny"
				if dec.Allow {
					outcome = "allow"
				}
				ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, time.Since(start))
			}
			return dec, nil
		}
	}

	// Default deny
	dec := Decision{Allow: false, Reason: "No matching policy found - default deny"}
	if ma.cacheEnabled {
		ma.storeInCache(request, &dec)
	} else {
		// Ensure consistent metadata key for tests even when cache disabled
		dec.Metadata = map[string]string{"cache_hit": metadataCacheHitFalse}
	}
	if ma.decisionCache != nil {
		ma.decisionCache.Set(lruKey, AuthorizationCacheEntry{Decision: dec, PolicyVersion: ma.currentPolicyVersion(), Jurisdiction: ma.jurisdiction, Inserted: time.Now()})
	}
	atomic.AddUint64(&ma.metricDecisions, 1)
	atomic.AddUint64(&ma.metricCacheMisses, 1)
	dur := time.Since(start)
	ma.recordLatency(dur)
	if ma.metricsProvider != nil {
		outcome := "deny"
		if dec.Allow {
			outcome = "allow"
		}
		ma.metricsProvider.RecordDecision(request.Action, "resource", outcome, dur)
	}
	return dec, nil
}

// storeInCache persists a decision with TTL.
func (ma *MemoryAuthorizer) storeInCache(r Request, d *Decision) {
	if !ma.cacheEnabled {
		return
	}
	if d.Metadata == nil {
		d.Metadata = make(map[string]string)
	}
	d.Metadata["cache_hit"] = metadataCacheHitFalse
	ma.cacheMu.Lock()
	ma.cache[ma.cacheKey(r)] = cachedDecision{decision: *d, expires: time.Now().Add(ma.cacheTTL)}
	ma.cacheMu.Unlock()
}

// finalizeDecision constructs and caches decision for a policy.
// buildDecisionFromPolicy constructs Decision and records metrics + latency.
func (ma *MemoryAuthorizer) buildDecisionFromPolicy(request Request, policy Policy, start time.Time) Decision {
	effStr := string(policy.Effect)
	if len(effStr) > 0 {
		effStr = strings.ToUpper(effStr[:1]) + effStr[1:]
	}
	dec := Decision{Allow: policy.Effect == Allow, Reason: fmt.Sprintf("%s by policy %s", effStr, policy.ID), Metadata: policy.Metadata}
	if ma.cacheEnabled {
		ma.storeInCache(request, &dec)
	} else {
		if dec.Metadata == nil {
			dec.Metadata = make(map[string]string)
		}
		dec.Metadata["cache_hit"] = metadataCacheHitFalse
	}
	if ma.decisionCache != nil {
		key := makeKey(request.Subject, request.Action, request.Resource, ma.currentPolicyVersion(), ma.jurisdiction)
		ma.decisionCache.Set(key, AuthorizationCacheEntry{Decision: dec, PolicyVersion: ma.currentPolicyVersion(), Jurisdiction: ma.jurisdiction, Inserted: time.Now()})
	}
	atomic.AddUint64(&ma.metricDecisions, 1)
	atomic.AddUint64(&ma.metricCacheMisses, 1)
	ma.recordLatency(time.Since(start))
	return dec
}

// annotateConflict adds policy_conflict metadata if both allow and deny lists non-empty.
func (ma *MemoryAuthorizer) annotateConflict(dec *Decision, denyList, allowList []Policy) {
	if len(denyList) > 0 && len(allowList) > 0 {
		if dec.Metadata == nil {
			dec.Metadata = make(map[string]string)
		}
		ids := make([]string, 0, len(denyList)+len(allowList))
		for _, p := range denyList {
			ids = append(ids, p.ID)
		}
		for _, p := range allowList {
			ids = append(ids, p.ID)
		}
		dec.Metadata["policy_conflict"] = strings.Join(ids, ",")
		atomic.AddUint64(&ma.metricConflicts, 1)
	}
}

// recordLatency updates Welford statistics for latency.
func (ma *MemoryAuthorizer) recordLatency(d time.Duration) {
	// Non-atomic float updates; acceptable minor race for observability.
	// Capture duration in nanoseconds as float64.
	val := float64(d.Nanoseconds())
	// Histogram bucket increment
	durNs := d.Nanoseconds()
	for i, upper := range ma.latencyBuckets {
		if durNs <= upper {
			atomic.AddUint64(&ma.latencyBucketCounts[i], 1)
			break
		}
	}
	count := atomic.AddUint64(&ma.latencyCount, 1)
	ma.latencyMu.Lock()
	defer ma.latencyMu.Unlock()
	if count == 1 {
		ma.latencyMean = val
		ma.latencyM2 = 0
		return
	}
	delta := val - ma.latencyMean
	ma.latencyMean += delta / float64(count)
	delta2 := val - ma.latencyMean
	ma.latencyM2 += delta * delta2
}

// MetricsSnapshot represents current metrics counters and latency stats.
type MetricsSnapshot struct {
	Decisions          uint64           `json:"decisions"`
	CacheHits          uint64           `json:"cache_hits"`
	CacheMisses        uint64           `json:"cache_misses"`
	Reloads            uint64           `json:"reloads"`
	Conflicts          uint64           `json:"conflicts"`
	AvgLatencyNs       float64          `json:"avg_latency_ns"`
	P99LatencyNs       float64          `json:"p99_latency_ns"` // approximated assuming normal dist
	RegexCompiles      uint64           `json:"regex_compiles"`
	RegexCompileErrors uint64           `json:"regex_compile_errors"`
	RegexCacheSize     int              `json:"regex_cache_size"`
	RegexEvictions     uint64           `json:"regex_evictions"`
	RegexMatches       uint64           `json:"regex_matches"`
	LatencyHistogram   map[int64]uint64 `json:"latency_histogram"` // bucket upper bound ns -> count
}

// GetMetricsSnapshot returns a snapshot of metrics.
func (ma *MemoryAuthorizer) GetMetricsSnapshot() MetricsSnapshot {
	dec := atomic.LoadUint64(&ma.metricDecisions)
	hits := atomic.LoadUint64(&ma.metricCacheHits)
	miss := atomic.LoadUint64(&ma.metricCacheMisses)
	rel := atomic.LoadUint64(&ma.metricReloads)
	conf := atomic.LoadUint64(&ma.metricConflicts)
	regexCompiles := atomic.LoadUint64(&ma.metricRegexCompiles)
	regexErrors := atomic.LoadUint64(&ma.metricRegexCompileErrors)
	ma.regexMu.RLock()
	rcSize := len(ma.regexCache)
	ma.regexMu.RUnlock()
	evict := atomic.LoadUint64(&ma.metricRegexEvictions)
	count := atomic.LoadUint64(&ma.latencyCount)
	ma.latencyMu.Lock()
	mean := ma.latencyMean
	m2 := ma.latencyM2
	ma.latencyMu.Unlock()
	var p99 float64
	if count > 1 {
		variance := m2 / float64(count-1)
		// approximate p99 ~ mean + 2.326 * stddev (normal assumption)
		if variance > 0 {
			p99 = mean + 2.326*sqrt(variance)
		}
	} else {
		p99 = mean
	}
	regexMatches := atomic.LoadUint64(&ma.metricRegexMatches)
	// snapshot histogram buckets
	histo := make(map[int64]uint64, len(ma.latencyBuckets))
	for i, upper := range ma.latencyBuckets {
		cnt := atomic.LoadUint64(&ma.latencyBucketCounts[i])
		if cnt > 0 {
			histo[upper] = cnt
		}
	}
	return MetricsSnapshot{Decisions: dec, CacheHits: hits, CacheMisses: miss, Reloads: rel, Conflicts: conf, AvgLatencyNs: mean, P99LatencyNs: p99, RegexCompiles: regexCompiles, RegexCompileErrors: regexErrors, RegexCacheSize: rcSize, RegexEvictions: evict, RegexMatches: regexMatches, LatencyHistogram: histo}
}

// sqrt helper to avoid importing math for a single use if desired
func sqrt(x float64) float64 {
	// Simple Newton iteration (5 rounds sufficient for metrics precision)
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 5; i++ {
		z = 0.5 * (z + x/z)
	}
	return z
}

// GetPermissions retrieves permissions for a subject
func (ma *MemoryAuthorizer) GetPermissions(ctx context.Context, subject string) ([]Permission, error) {
	permissionMap := make(map[string][]string)

	for _, policy := range ma.policies {
		if ma.matchesSubject(subject, policy.Subject) && policy.Effect == Allow {
			if actions, exists := permissionMap[policy.Resource]; exists {
				permissionMap[policy.Resource] = append(actions, policy.Actions...)
			} else {
				permissionMap[policy.Resource] = policy.Actions
			}
		}
	}

	var permissions []Permission
	for resource, actions := range permissionMap {
		permissions = append(permissions, Permission{
			Resource: resource,
			Actions:  actions,
			Granted:  true,
		})
	}

	return permissions, nil
}

// matchesPolicy checks if a request matches a policy
func (ma *MemoryAuthorizer) matchesPolicy(request Request, policy Policy) bool {
	// Subject direct match or role-based match if roles defined on policy.
	if policy.Subject != "" && !ma.matchesSubject(request.Subject, policy.Subject) {
		// allow fallback to role matching
		if len(policy.Roles) == 0 {
			return false
		}
	}
	if len(policy.Roles) > 0 {
		// extract roles from context (comma separated)
		ctxRoles := strings.Split(request.Context["roles"], ",")
		roleMatch := false
		for _, pr := range policy.Roles {
			for _, cr := range ctxRoles {
				if pr != "" && strings.TrimSpace(cr) == pr {
					roleMatch = true
					break
				}
			}
		}
		if !roleMatch {
			return false
		}
	}
	if !ma.matchesResource(request.Resource, policy.Resource) {
		return false
	}
	if !ma.matchesAction(request.Action, policy.Actions) {
		return false
	}
	// Check required scopes (if defined) against context scopes (space or comma separated list)
	if len(policy.RequiredScopes) > 0 {
		ctxScopesRaw := request.Context["scopes"]
		ctxScopes := make(map[string]struct{})
		for _, s := range strings.Fields(strings.ReplaceAll(ctxScopesRaw, ",", " ")) {
			ctxScopes[s] = struct{}{}
		}
		for _, rs := range policy.RequiredScopes {
			if _, ok := ctxScopes[rs]; !ok {
				return false
			}
		}
	}
	if !ma.matchesConditions(request, policy.Conditions) {
		return false
	}
	// Advanced expression evaluation (Task 6). Fail closed on error.
	if policy.Expression != "" {
		ok, err := EvaluateExpression(policy.Expression, request, nil)
		if err != nil || !ok {
			return false
		}
	}
	// Validator enforcement: all listed validators must pass; missing registry or validator ID => fail closed.
	if len(policy.Validators) > 0 {
		if ma.validatorRegistry == nil {
			return false
		}
		for _, vid := range policy.Validators {
			if err := ma.validatorRegistry.Invoke(vid, request, policy); err != nil {
				return false
			}
		}
	}
	return true
}

// matchesSubject checks if a subject matches a policy subject pattern
func (ma *MemoryAuthorizer) matchesSubject(subject, pattern string) bool {
	if pattern == "*" {
		return true
	}
	return subject == pattern
}

// matchesResource checks if a resource matches a policy resource pattern
func (ma *MemoryAuthorizer) matchesResource(resource, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(resource, prefix)
	}
	return resource == pattern
}

// matchesAction checks if an action matches any of the policy actions
func (ma *MemoryAuthorizer) matchesAction(action string, actions []string) bool {
	for _, policyAction := range actions {
		if policyAction == "*" || policyAction == action {
			return true
		}
	}
	return false
}

// matchesConditions checks if request context matches policy conditions
func (ma *MemoryAuthorizer) matchesConditions(request Request, conditions []Condition) bool {
	for _, condition := range conditions {
		if !ma.evaluateCondition(request, condition) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (ma *MemoryAuthorizer) evaluateCondition(request Request, condition Condition) bool {
	contextValue, exists := request.Context[condition.Key]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "equals":
		return ma.evalEquals(contextValue, condition.Values)
	case "not_equals":
		return ma.evalNotEquals(contextValue, condition.Values)
	case "in":
		return ma.evalIn(contextValue, condition.Values)
	case "contains":
		return ma.evalContains(contextValue, condition.Values)
	case "prefix":
		return ma.evalPrefix(contextValue, condition.Values)
	case "suffix":
		return ma.evalSuffix(contextValue, condition.Values)
	case operatorRegex:
		return ma.evalRegex(contextValue, condition.Values)
	case "numeric_gt":
		return ma.evalNumericGt(contextValue, condition.Values)
	case "numeric_lt":
		return ma.evalNumericLt(contextValue, condition.Values)
	case "time_before":
		return ma.evalTimeBefore(contextValue, condition.Values)
	case "time_after":
		return ma.evalTimeAfter(contextValue, condition.Values)
	default:
		return false
	}
}

func (ma *MemoryAuthorizer) evalEquals(contextValue string, values []string) bool {
	for _, value := range values {
		if contextValue == value {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalNotEquals(contextValue string, values []string) bool {
	for _, value := range values {
		if contextValue == value {
			return false
		}
	}
	return true
}

func (ma *MemoryAuthorizer) evalIn(contextValue string, values []string) bool {
	for _, value := range values {
		if contextValue == value {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalContains(contextValue string, values []string) bool {
	for _, value := range values {
		if strings.Contains(contextValue, value) {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalPrefix(contextValue string, values []string) bool {
	for _, value := range values {
		if strings.HasPrefix(contextValue, value) {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalSuffix(contextValue string, values []string) bool {
	for _, value := range values {
		if strings.HasSuffix(contextValue, value) {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalRegex(contextValue string, patterns []string) bool {
	for _, pattern := range patterns {
		// Fast path: check if compiled regex exists
		ma.regexMu.RLock()
		rx, ok := ma.regexCache[pattern]
		ma.regexMu.RUnlock()
		if !ok {
			// Compile new pattern
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				atomic.AddUint64(&ma.metricRegexCompileErrors, 1)
				continue // try next pattern value
			}
			// Insert into cache (double-check another goroutine didn't add meanwhile)
			ma.regexMu.Lock()
			if existing, exists := ma.regexCache[pattern]; exists {
				// Use existing compiled version
				rx = existing
			} else {
				ma.regexCache[pattern] = compiled
				ma.regexAddedAt[pattern] = time.Now()
				ma.regexLastAccess[pattern] = time.Now()
				rx = compiled
				atomic.AddUint64(&ma.metricRegexCompiles, 1)
			}
			ma.regexMu.Unlock()
			// Perform pruning post-insert (outside lock inside prune)
			ma.pruneRegexCache()
		} else {
			// Update last access (write requires full Lock)
			ma.regexMu.Lock()
			ma.regexLastAccess[pattern] = time.Now()
			ma.regexMu.Unlock()
		}
		// Safeguard: rx can still be nil theoretically (should not), guard anyway
		if rx != nil && rx.MatchString(contextValue) {
			// increment per-pattern and total counters
			ma.regexMu.Lock()
			ma.regexMatchCounts[pattern]++
			ma.regexMu.Unlock()
			atomic.AddUint64(&ma.metricRegexMatches, 1)
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalNumericGt(contextValue string, thresholds []string) bool {
	cv, err := strconv.ParseFloat(contextValue, 64)
	if err != nil {
		return false
	}
	for _, thr := range thresholds {
		v, err := strconv.ParseFloat(thr, 64)
		if err != nil {
			continue
		}
		if cv > v {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalNumericLt(contextValue string, thresholds []string) bool {
	cv, err := strconv.ParseFloat(contextValue, 64)
	if err != nil {
		return false
	}
	for _, thr := range thresholds {
		v, err := strconv.ParseFloat(thr, 64)
		if err != nil {
			continue
		}
		if cv < v {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalTimeBefore(contextValue string, values []string) bool {
	ct, err := time.Parse(time.RFC3339, contextValue)
	if err != nil {
		return false
	}
	for _, tv := range values {
		pt, err := time.Parse(time.RFC3339, tv)
		if err != nil {
			continue
		}
		if ct.Before(pt) {
			return true
		}
	}
	return false
}

func (ma *MemoryAuthorizer) evalTimeAfter(contextValue string, values []string) bool {
	ct, err := time.Parse(time.RFC3339, contextValue)
	if err != nil {
		return false
	}
	for _, tv := range values {
		pt, err := time.Parse(time.RFC3339, tv)
		if err != nil {
			continue
		}
		if ct.After(pt) {
			return true
		}
	}
	return false
}

// BasicEnforcer minimal structure for backward compatibility
type BasicEnforcer struct {
	policies map[string]*Policy
}

// NewBasicEnforcer creates a new BasicEnforcer
func NewBasicEnforcer() *BasicEnforcer { return &BasicEnforcer{policies: make(map[string]*Policy)} }

// Evaluate evaluates a request (simple matching)
func (e *BasicEnforcer) Evaluate(ctx context.Context, req *Request) (*Decision, error) {
	for _, p := range e.policies {
		if e.matchesPattern(p.Subject, req.Subject) && e.matchesPattern(p.Resource, req.Resource) && actionsContain(p.Actions, req.Action) {
			if p.Effect == Allow {
				return &Decision{Allow: true, Reason: fmt.Sprintf("allowed by policy %s", p.ID)}, nil
			}
			return &Decision{Allow: false, Reason: fmt.Sprintf("denied by policy %s", p.ID)}, nil
		}
	}
	return &Decision{Allow: false, Reason: "no matching policy"}, nil
}

// Helper retained for earlier patch references (no-op wrappers)

// actionsContain returns true if action matches any element or wildcard
func actionsContain(actions []string, action string) bool {
	for _, a := range actions {
		if a == action || a == "*" {
			return true
		}
	}
	return false
}
