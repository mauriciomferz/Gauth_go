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
	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/policy"
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
	head := a.Handler.Registry.Head()
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
	head := a.Handler.Registry.Head()
	var headHash string
	if head != nil {
		headHash = head.Hash
	}
	verified := true
	var verr string
	if err := a.Handler.Registry.VerifyChain(); err != nil {
		verified = false
		verr = err.Error()
	}

	// Capture Revocation Snapshot (Roadmap Item 5)
	var revocationSnapshot interface{}
	if a.Handler.RevocationChain != nil {
		revocationSnapshot = a.Handler.RevocationChain.LatestTreeHead()
	}

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
		for _, h := range a.Handler.Registry.ChainHashes() {
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
		"chain":               a.Handler.Registry.ChainHashes(),
		"verified":            verified,
		"verification_error":  verr,
		"queried_hash":        qh,
		"length":              len(a.Handler.Registry.ChainHashes()),
		"revocation_snapshot": revocationSnapshot,
	})
}

// Chain returns paginated chain hashes with total length and verification state.
func (a *API) Chain(c *gin.Context) {
	a.Handler.EnsureInitialized() // empty chain valid
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
	hashes := a.Handler.Registry.ChainHashes()
	total := len(hashes)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	slice := hashes[offset:end]
	head := a.Handler.Registry.Head()
	var headHash string
	if head != nil {
		headHash = head.Hash
	}
	verified := true
	verr := ""
	if err := a.Handler.Registry.VerifyChain(); err != nil {
		verified = false
		verr = err.Error()
	}
	// Include versions aligned with hashes for introspection
	versions := make([]int, len(slice))
	if len(slice) > 0 {
		for _, b := range a.Handler.Registry.ChainWithVersions() {
			for i, h := range slice {
				if h == b.Hash {
					versions[i] = b.Version
				}
			}
		}
	}
	c.JSON(200, gin.H{"success": true, "head_hash": headHash, "hashes": slice, "versions": versions, "offset": offset, "limit": limit, "returned": len(slice), "total": total, "verified": verified, "verification_error": verr, "active_version": a.Handler.Metrics.ActiveVersion})
}

// Timeline returns a compact list of all bundle versions.
func (a *API) Timeline(c *gin.Context) {
	a.Handler.EnsureInitialized()
	activeVer := a.Handler.Registry.ActiveVersion()
	head := a.Handler.Registry.Head()
	rolledBack := false
	if head != nil && len(a.Handler.Registry.ChainHashes()) > 0 {
		last := a.Handler.Registry.ChainWithVersions()
		if len(last) > 0 {
			latest := last[len(last)-1].Version
			if activeVer != latest {
				rolledBack = true
			}
		}
	}
	// Build timeline
	timeline := make([]map[string]any, 0, len(a.Handler.Registry.ChainWithVersions()))
	// We need created times. In original code this accessed an unexported field via hack/helper.
	// We should just use FindByHash which is clean.
	for _, vw := range a.Handler.Registry.ChainWithVersions() {
		b := a.Handler.Registry.FindByHash(vw.Hash)
		var created time.Time
		if b != nil {
			created = b.Created
		}
		short := vw.Hash
		if len(short) > 8 {
			short = short[:8]
		}
		timeline = append(timeline, map[string]any{
			"version":    vw.Version,
			"hash":       vw.Hash,
			"short_hash": short,
			"created":    created.Format(time.RFC3339),
			"active":     vw.Version == activeVer,
		})
	}
	c.JSON(200, gin.H{"success": true, "total": len(timeline), "timeline": timeline, "active_version": activeVer, "rolled_back": rolledBack})
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
	dec, err := a.Handler.Engine.Evaluate(c.Request.Context(), policy.EvalRequest{Subject: req.Subject, Action: req.Action, Resource: req.Resource, Attrs: req.Attrs})
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
		// Define an ad-hoc struct or map matching what AuditLog expects
		// Assuming AuditLog expects *AuditEntry (internal/web type).
		// Since we can't import web here easily (cycle), we use a map or anonymous struct if allowed,
		// but the interface takes interface{}. The implementation will cast it.
		// However, to keep it type safe, we'll verify what append expects.
		// The `web` package AuditLog expects `*AuditEntry`.
		// We'll create a mirror struct or just pass a map if the logger handles it.
		// BUT `audit.Append` takes `*AuditEntry`.
		// WORKAROUND: Pass a map or create a similar struct.
		// The original code passed `&AuditEntry{...}`.
		// We can define `PolicyAuditEntry` struct here and have the implementation mapper handle it,
		// or just define `AuditEntry` if it's common.
		// BETTER: The API doesn't need to know `AuditEntry` if we inject a refined `PolicyAuditor` func.
		// Let's assume for now we pass a generic request and the adapter does the work, OR we define a struct here.
		// Let's try passing a struct compatible with json marshaling, maybe the specific Auditor implementation will handle it.
		// Actually, `web/audit.go` likely has `AuditEntry`.
		// We can't import `web`.

		// Let's pass a special PolicyAuditPayload that the main server can wrap into an AuditEntry.
		entry := map[string]interface{}{
			"type":     "policy_eval", // signal to adapter
			"at":       time.Now(),
			"actor":    "policy_evaluator",
			"action":   "evaluate",
			"resource": req.Resource,
			"outcome":  map[bool]string{true: "allow", false: "deny"}[dec.Allow],
			"meta":     map[string]string{"bundle_hash": dec.BundleHash, "chain_head": dec.ChainHead, "subject": req.Subject, "action": req.Action},
		}
		a.Auditor.Append(entry)
	}

	c.JSON(200, gin.H{"success": true, "allow": dec.Allow, "deny": dec.Deny, "reason": dec.Reason, "matched": dec.Matched, "denied_by": dec.DeniedBy, "bundle_hash": dec.BundleHash, "chain_head": dec.ChainHead, "policy_version": dec.PolicyVersion})
}

func (a *API) GetBundle(c *gin.Context) {
	if a.Handler.Registry == nil {
		c.JSON(404, gin.H{"success": false, "message": "no bundles"})
		return
	}
	hash := c.Param("hash")
	b := a.Handler.Registry.FindByHash(hash)
	if b == nil {
		c.JSON(404, gin.H{"success": false, "message": "bundle not found"})
		return
	}
	c.JSON(200, gin.H{"success": true, "bundle": b})
}

func (a *API) AddBundle(c *gin.Context) {
	adminToken := os.Getenv("GAUTH_POLICY_ADMIN_TOKEN")
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

	if err := policy.ValidateBundle(policy.Bundle{ID: req.ID, Policies: req.Policies}); err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	b, err := a.Handler.Registry.AddBundle(policy.Bundle{ID: req.ID, Policies: req.Policies})
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": err.Error()})
		return
	}

	a.Handler.Metrics.Lock()
	a.Handler.Metrics.Revisions++
	a.Handler.Metrics.ActiveVersion = b.Version
	a.Handler.Metrics.Unlock()

	vErr := a.Handler.Registry.VerifyChain()
	head := a.Handler.Registry.Head()
	var vmsg string
	verified := true
	if vErr != nil {
		verified = false
		vmsg = vErr.Error()
	}

	// Try persistence
	if a.Handler.Config.PersistPath != "" {
		_ = a.Handler.SaveState() // best effort
	}

	c.JSON(201, gin.H{"success": true, "bundle_hash": b.Hash, "head_hash": head.Hash, "policy_version": b.Version, "verified": verified, "verification_error": vmsg, "chain": a.Handler.Registry.ChainHashes()})
}

func (a *API) Rollback(c *gin.Context) {
	if a.Handler.Registry == nil {
		c.JSON(400, gin.H{"success": false, "message": "no policy chain"})
		return
	}
	adminTok := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
	if adminTok == "" {
		c.JSON(403, gin.H{"success": false, "message": "admin token required"})
		return
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
	prevActive := a.Handler.Registry.ActiveVersion()
	if err := a.Handler.Registry.Rollback(ver); err != nil {
		c.JSON(404, gin.H{"success": false, "message": err.Error()})
		return
	}

	a.Handler.Metrics.Lock()
	a.Handler.Metrics.RollbackCount++
	a.Handler.Metrics.ActiveVersion = a.Handler.Registry.ActiveVersion()
	a.Handler.Metrics.Unlock()

	head := a.Handler.Registry.Head()
	verified := true
	verr := ""
	if err := a.Handler.Registry.VerifyChain(); err != nil {
		verified = false
		verr = err.Error()
	}

	if a.Handler.Config.PersistPath != "" {
		_ = a.Handler.SaveState()
	}

	if a.Auditor != nil {
		entry := map[string]interface{}{
			"type":     "policy_rollback",
			"at":       time.Now(),
			"actor":    "policy_admin",
			"action":   "rollback",
			"resource": "policy_chain",
			"outcome":  "success",
			"meta":     map[string]string{"target_version": fmt.Sprintf("%d", ver), "previous_active_version": fmt.Sprintf("%d", prevActive), "head_hash": head.Hash},
		}
		a.Auditor.Append(entry)
	}

	c.JSON(200, gin.H{"success": true, "active_version": a.Handler.Metrics.ActiveVersion, "head_hash": head.Hash, "verified": verified, "verification_error": verr})
}

func (a *API) Diff(c *gin.Context) {
	if a.Handler.Registry == nil {
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
	diff, err := a.Handler.Registry.Diff(from, to)
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
	b.WriteString("# HELP gauth_policy_evaluations_total Total policy evaluations\n")
	b.WriteString("# TYPE gauth_policy_evaluations_total counter\n")
	b.WriteString(fmt.Sprintf("gauth_policy_evaluations_total %d\n", pm.Total))
	b.WriteString("# HELP gauth_policy_evaluations_allow_total Total allow decisions\n")
	b.WriteString("# TYPE gauth_policy_evaluations_allow_total counter\n")
	b.WriteString(fmt.Sprintf("gauth_policy_evaluations_allow_total %d\n", pm.Allow))
	b.WriteString("# HELP gauth_policy_evaluations_deny_total Total deny decisions\n")
	b.WriteString("# TYPE gauth_policy_evaluations_deny_total counter\n")
	b.WriteString(fmt.Sprintf("gauth_policy_evaluations_deny_total %d\n", pm.Deny))
	b.WriteString("# HELP gauth_policy_revisions_total Total appended policy bundle revisions\n")
	b.WriteString("# TYPE gauth_policy_revisions_total counter\n")
	b.WriteString(fmt.Sprintf("gauth_policy_revisions_total %d\n", pm.Revisions))
	b.WriteString("# HELP gauth_policy_active_version Current effective policy bundle version (rollback aware)\n")
	b.WriteString("# TYPE gauth_policy_active_version gauge\n")
	b.WriteString(fmt.Sprintf("gauth_policy_active_version %d\n", pm.ActiveVersion))
	b.WriteString("# HELP gauth_policy_rollback_total Total successful policy rollback operations\n")
	b.WriteString("# TYPE gauth_policy_rollback_total counter\n")
	b.WriteString(fmt.Sprintf("gauth_policy_rollback_total %d\n", pm.RollbackCount))
	b.WriteString("# HELP gauth_policy_diff_requests_total Total successful diff requests\n")
	b.WriteString("# TYPE gauth_policy_diff_requests_total counter\n")
	b.WriteString(fmt.Sprintf("gauth_policy_diff_requests_total %d\n", pm.DiffRequests))

	if mm, ok := a.Handler.PromMetrics.(*metrics.Memory); ok {
		b.WriteString("# HELP gauth_scope_violations_total Scope validation violations\n")
		b.WriteString("# TYPE gauth_scope_violations_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_scope_violations_total %d\n", mm.ScopeViolations()))
		b.WriteString("# HELP gauth_restriction_violations_total Restriction validation violations\n")
		b.WriteString("# TYPE gauth_restriction_violations_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_restriction_violations_total %d\n", mm.RestrictionViolations()))
		b.WriteString("# HELP gauth_unauthorized_decisions_total Unauthorized authorization decisions (policy deny)\n")
		b.WriteString("# TYPE gauth_unauthorized_decisions_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_unauthorized_decisions_total %d\n", mm.UnauthorizedDecisions()))
		b.WriteString("# HELP gauth_expired_delegations_total Expired delegations encountered\n")
		b.WriteString("# TYPE gauth_expired_delegations_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_expired_delegations_total %d\n", mm.ExpiredDelegations()))
		b.WriteString("# HELP gauth_revoked_delegations_total Revoked delegations encountered\n")
		b.WriteString("# TYPE gauth_revoked_delegations_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_revoked_delegations_total %d\n", mm.RevokedDelegations()))

		// Validation failure reason counters
		b.WriteString("# HELP gauth_validation_invalid_payload_total Validation failures due to invalid payload\n")
		b.WriteString("# TYPE gauth_validation_invalid_payload_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_validation_invalid_payload_total %d\n", mm.InvalidPayloadFailures()))
		b.WriteString("# HELP gauth_validation_unsupported_status_total Validation failures due to unsupported status values\n")
		b.WriteString("# TYPE gauth_validation_unsupported_status_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_validation_unsupported_status_total %d\n", mm.UnsupportedStatusFailures()))
		b.WriteString("# HELP gauth_validation_invalid_transition_total Validation failures due to invalid lifecycle transitions\n")
		b.WriteString("# TYPE gauth_validation_invalid_transition_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_validation_invalid_transition_total %d\n", mm.InvalidTransitionFailures()))
		b.WriteString("# HELP gauth_validation_not_found_total Validation failures due to missing entities\n")
		b.WriteString("# TYPE gauth_validation_not_found_total counter\n")
		b.WriteString(fmt.Sprintf("gauth_validation_not_found_total %d\n", mm.NotFoundFailures()))
	}

	// Histogram
	b.WriteString("# HELP gauth_policy_eval_latency_ns Evaluation latency distribution (ns)\n")
	b.WriteString("# TYPE gauth_policy_eval_latency_ns histogram\n")
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
		approxSum += uint64(segmentMid) * cnt
		b.WriteString(fmt.Sprintf("gauth_policy_eval_latency_ns_bucket{le=\"%d\"} %d\n", ub, cumulative))
		prevBound = ub
	}
	b.WriteString(fmt.Sprintf("gauth_policy_eval_latency_ns_bucket{le=\"+Inf\"} %d\n", cumulative))
	b.WriteString(fmt.Sprintf("gauth_policy_eval_latency_ns_count %d\n", cumulative))
	b.WriteString(fmt.Sprintf("gauth_policy_eval_latency_ns_sum %d\n", approxSum))
	b.WriteString(fmt.Sprintf("gauth_policy_eval_latency_ns_p99 %d\n", pm.P99LatencyNS))

	c.String(200, b.String())
}

func (a *API) AuditConsistency(c *gin.Context) {
	if a.Handler.Registry == nil {
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
	head := a.Handler.Registry.Head()
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
	if err := a.Handler.Registry.VerifyChain(); err != nil {
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
