package policy

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// Auditor abstracts the audit logging capability
type Auditor interface {
	Append(entry interface{})
	List(limit int) []interface{}
}

type API struct {
	Handler *Handler
	Auditor Auditor
}

func (a *API) RegisterRoutes(r *gin.Engine) {
	// Public verification / status
	r.GET("/api/v1/policy/provenance", a.Provenance)
	r.GET("/api/v1/policy/chain", a.Chain)
	r.GET("/api/v1/policy/timeline", a.Timeline)

	// Core operations
	r.POST("/api/v1/policy/evaluate", a.Evaluate)
	r.GET("/api/v1/policy/bundles/:hash", a.GetBundle)

	// Admin / Governance
	r.POST("/api/v1/policy/bundles", a.AddBundle)
	r.POST("/api/v1/policy/rollback", a.Rollback)
	r.GET("/api/v1/policy/diff", a.Diff)

	// Monitoring / Consistency
	r.GET("/api/v1/policy/metrics", a.Metrics)
	r.GET("/api/v1/policy/metrics/prometheus", a.PrometheusMetrics)
	r.GET("/policy/metrics/prometheus", a.PrometheusMetrics) // Alias
	r.GET("/api/v1/policy/audit-consistency", a.AuditConsistency)
	// Additional convenience
	r.GET("/api/v1/policy/head/policies", a.HeadPolicies)
}

func (a *API) HeadPolicies(c *gin.Context) {
	a.Handler.EnsureInitialized()
	head, err := a.Handler.Store.Head(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}
	if head == nil {
		c.JSON(200, gin.H{"success": true, "head_hash": "", "policies": []policy.Policy{}, "count": 0})
		return
	}
	// Defensive copy
	pols := make([]policy.Policy, len(head.Policies))
	copy(pols, head.Policies)
	c.JSON(200, gin.H{"success": true, "head_hash": head.Hash, "policies": pols, "count": len(pols)})
}

// Provenance returns current policy bundle chain head and verification status.
func (a *API) Provenance(c *gin.Context) {
	a.Handler.EnsureInitialized()
	ctx := c.Request.Context()
	head, err := a.Handler.Store.Head(ctx)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}
	var headHash string
	if head != nil {
		headHash = head.Hash
	}
	verified := true
	var verr string
	if err := a.Handler.Store.VerifyChain(ctx); err != nil {
		verified = false
		verr = err.Error()
	}

	// Capture Revocation Snapshot (Roadmap Item 5)
	var revocationSnapshot interface{}
	if a.Handler.RevocationChain != nil {
		revocationSnapshot = a.Handler.RevocationChain.LatestTreeHead()
	}

	chainHashes, _ := a.Handler.Store.ChainHashes(ctx) // Ignore error?

	// Optional hash query parameter
	qh := strings.TrimSpace(c.Query("hash"))
	if qh != "" {
		// Validate hash format (SHA-256 hex)
		if len(qh) != 64 {
			c.JSON(400, gin.H{"success": false, "message": "invalid hash format: length must be 64"})
			return
		}
		// Check hex chars
		for _, r := range qh {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				c.JSON(400, gin.H{"success": false, "message": "invalid hash format: must be hex"})
				return
			}
		}

		found := false
		for _, h := range chainHashes {
			if h == qh {
				found = true
				break
			}
		}
		if !found {
			verified = false
			if verr == "" {
				verr = "hash_not_found"
			}
		}
	}
	c.JSON(200, gin.H{
		"success":             true,
		"head_hash":           headHash,
		"chain":               chainHashes,
		"verified":            verified,
		"verification_error":  verr,
		"queried_hash":        qh,
		"length":              len(chainHashes),
		"revocation_snapshot": revocationSnapshot,
	})
}

// Chain returns paginated chain hashes with total length and verification state.
func (a *API) Chain(c *gin.Context) {
	a.Handler.EnsureInitialized()
	ctx := c.Request.Context()
	// Parse offset & limit; default limit 50
	offStr := c.Query("offset")
	limStr := c.Query("limit")
	offset, limit := 0, 50
	if offStr != "" {
		if n, err := strconv.Atoi(offStr); err == nil && n >= 0 {
			offset = n
		}
	}
	if limStr != "" {
		if n, err := strconv.Atoi(limStr); err == nil && n > 0 {
			limit = n
		}
	}

	// Use List to get paginated chunks directly?
	// Store.List uses offset/limit for Bundles.
	// But we only want hashes...
	// Postgres Store.List returns bundles (heavier). Store.ChainHashes returns all hashes.
	// For now, use ChainHashes and slice in memory to match previous behavior
	// (Postgres ChainHashes loads all IDs, which is lightweight enough for now).
	allHashes, err := a.Handler.Store.ChainHashes(ctx)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}

	total := len(allHashes)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	slice := allHashes[offset:end]

	head, _ := a.Handler.Store.Head(ctx)
	var headHash string
	if head != nil {
		headHash = head.Hash
	}
	verified := true
	verr := ""
	if err := a.Handler.Store.VerifyChain(ctx); err != nil {
		verified = false
		verr = err.Error()
	}

	// Include versions aligned with hashes for introspection
	versions := make([]int, len(slice))
	// Optimize: Only fetch versions for sliced hashes?
	// Or use GetByHash.
	for i, h := range slice {
		b, _ := a.Handler.Store.GetByHash(ctx, h)
		if b != nil {
			versions[i] = b.Version
		}
	}

	activeVer, _ := a.Handler.Store.ActiveVersion(ctx)

	c.JSON(200, gin.H{
		"success":            true,
		"head_hash":          headHash,
		"hashes":             slice,
		"versions":           versions,
		"offset":             offset,
		"limit":              limit,
		"returned":           len(slice),
		"total":              total,
		"verified":           verified,
		"verification_error": verr,
		"active_version":     activeVer,
	})
}

// Timeline returns a compact list of all bundle versions.
func (a *API) Timeline(c *gin.Context) {
	a.Handler.EnsureInitialized()
	ctx := c.Request.Context()
	activeVer, _ := a.Handler.Store.ActiveVersion(ctx)
	head, _ := a.Handler.Store.Head(ctx)
	rolledBack := false
	if head != nil && activeVer != head.Version {
		rolledBack = true
	}

	// Fetch all bundles logic
	// Ideally we paginate or limit. For UI/Timeline we often want all or recent N.
	// Using a reasonable limit 1000.
	bundles, total, err := a.Handler.Store.List(ctx, 0, 1000)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Build timeline
	timeline := make([]map[string]any, 0, len(bundles))
	maxVer := 0
	for _, b := range bundles {
		if b.Version > maxVer {
			maxVer = b.Version
		}
		short := b.Hash
		if len(short) > 8 {
			short = short[:8]
		}
		timeline = append(timeline, map[string]any{
			"version":    b.Version,
			"hash":       b.Hash,
			"short_hash": short,
			"created":    b.Created.Format(time.RFC3339),
			"active":     b.Version == activeVer,
		})
	}

	if maxVer > activeVer {
		rolledBack = true
	}

	c.JSON(200, gin.H{"success": true, "total": total, "timeline": timeline, "active_version": activeVer, "rolled_back": rolledBack})
}

// Evaluate evaluates a request against current head.
func (a *API) Evaluate(c *gin.Context) {
	a.Handler.EnsureInitialized()
	var req struct {
		Subject  string            `json:"subject"`
		Action   string            `json:"action"`
		Resource string            `json:"resource"`
		Attrs    map[string]string `json:"attrs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	start := time.Now()
	// Pass context to Evaluate
	dec, err := a.Handler.Engine.Evaluate(c.Request.Context(), policy.EvalRequest{
		Subject:  req.Subject,
		Action:   req.Action,
		Resource: req.Resource,
		Attrs:    req.Attrs,
	})
	elapsedNS := time.Since(start).Nanoseconds()
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Metrics
	a.Handler.Metrics.Lock()
	a.Handler.Metrics.Total++
	if dec.Allow {
		a.Handler.Metrics.Allow++
	} else {
		a.Handler.Metrics.Deny++
	}
	a.Handler.Metrics.LastReason = dec.Reason
	a.Handler.Metrics.LastAt = time.Now().UTC()
	a.Handler.Metrics.LastMatched = len(dec.Matched)
	a.Handler.Metrics.LastDeniedBy = len(dec.DeniedBy)
	a.Handler.Metrics.Unlock()
	a.Handler.RecordLatency(time.Duration(elapsedNS)) // updates buckets + P99

	// Audit event - requires interface adaptation for AuditEntry
	if a.Auditor != nil {
		entry := map[string]interface{}{
			"type":     "policy_eval", // signal to adapter
			"at":       time.Now(),
			"actor":    "policy_evaluator",
			"action":   "evaluate",
			"resource": req.Resource,
			"outcome":  map[bool]string{true: "allow", false: "deny"}[dec.Allow],
			"meta": map[string]string{
				"bundle_hash": dec.BundleHash,
				"chain_head":  dec.ChainHead,
				"subject":     req.Subject,
				"action":      req.Action,
			},
		}
		a.Auditor.Append(entry)
	}

	c.JSON(200, gin.H{
		"success":        true,
		"allow":          dec.Allow,
		"deny":           dec.Deny,
		"reason":         dec.Reason,
		"matched":        dec.Matched,
		"denied_by":      dec.DeniedBy,
		"bundle_hash":    dec.BundleHash,
		"chain_head":     dec.ChainHead,
		"policy_version": dec.PolicyVersion,
	})
}

func (a *API) GetBundle(c *gin.Context) {
	if a.Handler.Store == nil {
		c.JSON(404, gin.H{"success": false, "message": "store not initialized"})
		return
	}
	hash := c.Param("hash")
	b, err := a.Handler.Store.GetByHash(c.Request.Context(), hash)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}
	if b == nil {
		c.JSON(404, gin.H{"success": false, "message": "bundle not found"})
		return
	}
	c.JSON(200, gin.H{"success": true, "bundle": b})
}

func (a *API) AddBundle(c *gin.Context) {
	adminToken := os.Getenv("AGENTAUTH_POLICY_ADMIN_TOKEN")
	if adminToken != "" && c.GetHeader("X-Admin-Token") != adminToken {
		c.JSON(401, gin.H{"success": false, "message": "unauthorized"})
		return
	}
	if a.Handler.RateLimiter != nil && !a.Handler.RateLimiter.Allow(c.ClientIP()) {
		c.JSON(429, gin.H{"success": false, "message": "rate limit exceeded"})
		return
	}
	var req struct {
		ID       string          `json:"id"`
		Policies []policy.Policy `json:"policies"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	a.Handler.EnsureInitialized()
	ctx := c.Request.Context()

	if err := policy.ValidateBundle(policy.Bundle{ID: req.ID, Policies: req.Policies}); err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	b, err := a.Handler.Store.AppendBundle(ctx, policy.Bundle{ID: req.ID, Policies: req.Policies})
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}

	a.Handler.Metrics.Lock()
	a.Handler.Metrics.Revisions++
	a.Handler.Metrics.ActiveVersion = b.Version
	a.Handler.Metrics.Unlock()

	vErr := a.Handler.Store.VerifyChain(ctx)
	head, _ := a.Handler.Store.Head(ctx)
	var vmsg string
	verified := true
	if vErr != nil {
		verified = false
		vmsg = vErr.Error()
	}

	// Try persistence (Not needed if Store handles it, but maybe legacy hook?)
	if a.Handler.Config.PersistPath != "" {
		_ = a.Handler.SaveState() // best effort
	}

	if a.Handler.OnPolicyChange != nil {
		a.Handler.OnPolicyChange()
	}

	chain, _ := a.Handler.Store.ChainHashes(ctx)
	headHash := ""
	if head != nil {
		headHash = head.Hash
	}

	c.JSON(201, gin.H{
		"success":            true,
		"bundle_hash":        b.Hash,
		"head_hash":          headHash,
		"policy_version":     b.Version,
		"verified":           verified,
		"verification_error": vmsg,
		"chain":              chain,
	})
}

func (a *API) Rollback(c *gin.Context) {
	if a.Handler.Store == nil {
		c.JSON(400, gin.H{"success": false, "message": "no policy chain"})
		return
	}
	adminToken := os.Getenv("AGENTAUTH_POLICY_ADMIN_TOKEN")
	if adminToken != "" {
		if c.GetHeader("X-Admin-Token") != adminToken {
			c.JSON(403, gin.H{"success": false, "message": "unauthorized"})
			return
		}
	}
	verStr := c.Query("version")
	if verStr == "" {
		c.JSON(400, gin.H{"success": false, "message": "version required"})
		return
	}
	ver, err := strconv.Atoi(verStr)
	if err != nil || ver <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "invalid version"})
		return
	}

	ctx := c.Request.Context()
	prevActive, _ := a.Handler.Store.ActiveVersion(ctx)

	if err := a.Handler.Store.Rollback(ctx, ver); err != nil {
		c.JSON(404, gin.H{"success": false, "message": err.Error()})
		return
	}

	active, _ := a.Handler.Store.ActiveVersion(ctx)
	a.Handler.Metrics.Lock()
	a.Handler.Metrics.RollbackCount++
	a.Handler.Metrics.ActiveVersion = active
	a.Handler.Metrics.Unlock()

	head, _ := a.Handler.Store.Head(ctx)
	headHash := ""
	if head != nil {
		headHash = head.Hash
	}
	verified := true
	verr := ""
	if err := a.Handler.Store.VerifyChain(ctx); err != nil {
		verified = false
		verr = err.Error()
	}

	if a.Handler.Config.PersistPath != "" {
		_ = a.Handler.SaveState()
	}

	if a.Handler.OnPolicyChange != nil {
		a.Handler.OnPolicyChange()
	}

	if a.Auditor != nil {
		entry := map[string]interface{}{
			"type":     "policy_rollback",
			"at":       time.Now(),
			"actor":    "policy_admin",
			"action":   "rollback",
			"resource": "policy_chain",
			"outcome":  "success",
			"meta": map[string]string{
				"target_version":          fmt.Sprintf("%d", ver),
				"previous_active_version": fmt.Sprintf("%d", prevActive),
				"head_hash":               headHash,
			},
		}
		a.Auditor.Append(entry)
	}

	c.JSON(200, gin.H{
		"success":            true,
		"active_version":     active,
		"head_hash":          headHash,
		"verified":           verified,
		"verification_error": verr,
	})
}

func (a *API) Diff(c *gin.Context) {
	if a.Handler.Store == nil {
		c.JSON(400, gin.H{"success": false, "message": "no policy chain"})
		return
	}
	fromStr := c.Query("from")
	toStr := c.Query("to")
	from, to := 0, 0
	if fromStr != "" {
		if v, err := strconv.Atoi(fromStr); err == nil {
			from = v
		}
	}
	if toStr != "" {
		if v, err := strconv.Atoi(toStr); err == nil {
			to = v
		}
	}
	// Use standalone Diff function
	diff, err := policy.Diff(c.Request.Context(), a.Handler.Store, from, to)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	atomic.AddUint64(&a.Handler.Metrics.DiffRequests, 1)
	c.JSON(200, gin.H{"success": true, "diff": diff})
}

func (a *API) Metrics(c *gin.Context) {
	pm := a.Handler.Metrics
	pm.RLock()
	defer pm.RUnlock()

	latHist := make(map[string]uint64, len(pm.LatencyBuckets))
	for ub, ptr := range pm.LatencyBuckets {
		latHist[fmt.Sprintf("%d", ub)] = atomic.LoadUint64(ptr)
	}
	c.JSON(200, gin.H{
		"success":           true,
		"total":             pm.Total,
		"allow":             pm.Allow,
		"deny":              pm.Deny,
		"last_reason":       pm.LastReason,
		"last_at":           pm.LastAt.Format(time.RFC3339Nano),
		"last_matched":      pm.LastMatched,
		"last_denied_by":    pm.LastDeniedBy,
		"latency_histogram": latHist,
		"p99_latency_ns":    pm.P99LatencyNS,
		"revisions":         pm.Revisions,
		"active_version":    pm.ActiveVersion,
		"rollback_count":    pm.RollbackCount,
		"diff_requests":     pm.DiffRequests,
	})
}

func (a *API) PrometheusMetrics(c *gin.Context) {
	pm := a.Handler.Metrics
	pm.RLock()
	defer pm.RUnlock()

	c.Header("Content-Type", "text/plain; version=0.0.4")
	var b strings.Builder
	b.WriteString("# HELP agentauth_policy_evaluations_total Total policy evaluations\n")
	b.WriteString("# TYPE agentauth_policy_evaluations_total counter\n")
	b.WriteString(fmt.Sprintf("agentauth_policy_evaluations_total %d\n", pm.Total))
	b.WriteString("# HELP agentauth_policy_evaluations_allow_total Total allow decisions\n")
	b.WriteString("# TYPE agentauth_policy_evaluations_allow_total counter\n")
	b.WriteString(fmt.Sprintf("agentauth_policy_evaluations_allow_total %d\n", pm.Allow))
	b.WriteString("# HELP agentauth_policy_evaluations_deny_total Total deny decisions\n")
	b.WriteString("# TYPE agentauth_policy_evaluations_deny_total counter\n")
	b.WriteString(fmt.Sprintf("agentauth_policy_evaluations_deny_total %d\n", pm.Deny))
	b.WriteString("# HELP agentauth_policy_revisions_total Total appended policy bundle revisions\n")
	b.WriteString("# TYPE agentauth_policy_revisions_total counter\n")
	b.WriteString(fmt.Sprintf("agentauth_policy_revisions_total %d\n", pm.Revisions))
	b.WriteString("# HELP agentauth_policy_active_version Current effective policy bundle version (rollback aware)\n")
	b.WriteString("# TYPE agentauth_policy_active_version gauge\n")
	b.WriteString(fmt.Sprintf("agentauth_policy_active_version %d\n", pm.ActiveVersion))
	b.WriteString("# HELP agentauth_policy_rollback_total Total successful policy rollback operations\n")
	b.WriteString("# TYPE agentauth_policy_rollback_total counter\n")
	b.WriteString(fmt.Sprintf("agentauth_policy_rollback_total %d\n", pm.RollbackCount))
	b.WriteString("# HELP agentauth_policy_diff_requests_total Total successful diff requests\n")
	b.WriteString("# TYPE agentauth_policy_diff_requests_total counter\n")
	b.WriteString(fmt.Sprintf("agentauth_policy_diff_requests_total %d\n", pm.DiffRequests))

	if mm, ok := a.Handler.PromMetrics.(*metrics.Memory); ok {
		b.WriteString("# HELP agentauth_scope_violations_total Scope validation violations\n")
		b.WriteString("# TYPE agentauth_scope_violations_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_scope_violations_total %d\n", mm.ScopeViolations()))
		b.WriteString("# HELP agentauth_restriction_violations_total Restriction validation violations\n")
		b.WriteString("# TYPE agentauth_restriction_violations_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_restriction_violations_total %d\n", mm.RestrictionViolations()))
		b.WriteString("# HELP agentauth_unauthorized_decisions_total Unauthorized authorization decisions (policy deny)\n")
		b.WriteString("# TYPE agentauth_unauthorized_decisions_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_unauthorized_decisions_total %d\n", mm.UnauthorizedDecisions()))
		b.WriteString("# HELP agentauth_expired_delegations_total Expired delegations encountered\n")
		b.WriteString("# TYPE agentauth_expired_delegations_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_expired_delegations_total %d\n", mm.ExpiredDelegations()))
		b.WriteString("# HELP agentauth_revoked_delegations_total Revoked delegations encountered\n")
		b.WriteString("# TYPE agentauth_revoked_delegations_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_revoked_delegations_total %d\n", mm.RevokedDelegations()))

		// Validation failure reason counters
		b.WriteString("# HELP agentauth_validation_invalid_payload_total Validation failures due to invalid payload\n")
		b.WriteString("# TYPE agentauth_validation_invalid_payload_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_validation_invalid_payload_total %d\n", mm.InvalidPayloadFailures()))
		b.WriteString("# HELP agentauth_validation_unsupported_status_total Validation failures due to unsupported status values\n")
		b.WriteString("# TYPE agentauth_validation_unsupported_status_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_validation_unsupported_status_total %d\n", mm.UnsupportedStatusFailures()))
		b.WriteString("# HELP agentauth_validation_invalid_transition_total Validation failures due to invalid lifecycle transitions\n")
		b.WriteString("# TYPE agentauth_validation_invalid_transition_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_validation_invalid_transition_total %d\n", mm.InvalidTransitionFailures()))
		b.WriteString("# HELP agentauth_validation_not_found_total Validation failures due to missing entities\n")
		b.WriteString("# TYPE agentauth_validation_not_found_total counter\n")
		b.WriteString(fmt.Sprintf("agentauth_validation_not_found_total %d\n", mm.NotFoundFailures()))
	}

	// Histogram
	b.WriteString("# HELP agentauth_policy_eval_latency_ns Evaluation latency distribution (ns)\n")
	b.WriteString("# TYPE agentauth_policy_eval_latency_ns histogram\n")
	var bounds []int64
	for ub := range pm.LatencyBuckets {
		bounds = append(bounds, ub)
	}
	sort.Slice(bounds, func(i, j int) bool { return bounds[i] < bounds[j] })
	var cumulative uint64
	var approxSum uint64
	var prevBound int64
	for i, ub := range bounds {
		cnt := atomic.LoadUint64(pm.LatencyBuckets[ub])
		cumulative += cnt
		low := prevBound
		if i == 0 {
			low = 0
		}
		segmentMid := (low + ub) / 2
		// #nosec G115
		approxSum += uint64(segmentMid) * cnt
		b.WriteString(fmt.Sprintf("agentauth_policy_eval_latency_ns_bucket{le=\"%d\"} %d\n", ub, cumulative))
		prevBound = ub
	}
	b.WriteString(fmt.Sprintf("agentauth_policy_eval_latency_ns_bucket{le=\"+Inf\"} %d\n", cumulative))
	b.WriteString(fmt.Sprintf("agentauth_policy_eval_latency_ns_count %d\n", cumulative))
	b.WriteString(fmt.Sprintf("agentauth_policy_eval_latency_ns_sum %d\n", approxSum))
	b.WriteString(fmt.Sprintf("agentauth_policy_eval_latency_ns_p99 %d\n", pm.P99LatencyNS))

	c.String(200, b.String())
}

func (a *API) AuditConsistency(c *gin.Context) {
	if a.Handler.Store == nil {
		c.JSON(200, gin.H{"success": true, "evaluations": 0, "consistent": true, "message": "no policy chain"})
		return
	}

	if a.Auditor == nil {
		c.JSON(200, gin.H{"success": true, "message": "auditor not configured"})
		return
	}

	entries := a.Auditor.List(20)
	evalCount := 0
	consistent := true
	ctx := c.Request.Context()
	head, _ := a.Handler.Store.Head(ctx)
	var lastEvalChainHead string

	for _, e := range entries {
		// Reflection to check generic entries for Action=="evaluate" and Meta.chain_head
		val := reflect.ValueOf(e)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			continue
		}

		actField := val.FieldByName("Action")
		if !actField.IsValid() || actField.String() != "evaluate" {
			continue
		}

		evalCount++
		metaField := val.FieldByName("Meta")
		if metaField.IsValid() {
			metaVal := metaField.Interface()
			if m, ok := metaVal.(map[string]string); ok {
				if ch, exists := m["chain_head"]; exists {
					lastEvalChainHead = ch
					if head != nil && ch != head.Hash {
						consistent = false
					}
				}
			} else if m, ok := metaVal.(map[string]interface{}); ok {
				if ch, exists := m["chain_head"]; exists {
					if chStr, okStr := ch.(string); okStr {
						lastEvalChainHead = chStr
						if head != nil && chStr != head.Hash {
							consistent = false
						}
					}
				}
			}
		}
	}

	// If no evaluations, consistent is trivially true
	// Verify chain integrity
	verified := true
	if err := a.Handler.Store.VerifyChain(ctx); err != nil {
		verified = false
	}

	c.JSON(200, gin.H{
		"success":              true,
		"chain_verified":       verified,
		"consistent":           consistent,
		"evaluations":          evalCount,
		"last_eval_chain_head": lastEvalChainHead,
		"current_head": func() string {
			if head != nil {
				return head.Hash
			}
			return ""
		}(),
	})
}
