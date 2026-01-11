package delegation

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/internal/capability"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/internal/tracing"
)

// Status constants
const (
	StatusActive           = "active"
	StatusSuspended        = "suspended"
	StatusTerminated       = "terminated"
	StatusDeprecated       = "deprecated"
	StatusSunset           = "sunset"
	StatusPartiallyRevoked = "partially_revoked"
)

// EnforceFunc is the signature for capability enforcement.
type EnforceFunc func(action string, claims map[string]any) (bool, []string)

// GetRequiredCapsFunc is the signature for looking up required capabilities.
type GetRequiredCapsFunc func(action string) []string

// MetricsProvider defines dependencies for recording metrics.
type MetricsProvider interface {
	IncViolation(cat interface{})
	IncCapabilityEnforceDenied()
	IncCapabilityEnforceAllowed()
	IncDelegationStatusTransitions()
	IncDelegationStatusTransitionFailures()
	RecordLifecycleTransition(entityType, oldStatus, newStatus, outcome string)
	ObserveLifecycleTransitionLatency(entityType, outcome string, d time.Duration)
}

// AuditProvider defines dependencies for audit logging.
type AuditProvider interface {
	AppendEntry(id string, at time.Time, actor, action, resource, outcome string, meta any)
}

// LifecycleRecorder defines dependencies for recording lifecycle events.
type LifecycleRecorder interface {
	RecordEvent(entityType, entityID, oldStatus, newStatus, outcome, reason string, latencyNS int64)
}

// Handler manages delegation related endpoints.
type Handler struct {
	mu         sync.RWMutex
	status     map[string]string
	metrics    MetricsProvider
	audit      AuditProvider
	lifecycle  LifecycleRecorder
	enforce    EnforceFunc
	tracer     *tracing.TracerProvider
	getReqCaps GetRequiredCapsFunc
}

// NewHandler creates a new Delegation handler.
func NewHandler(
	metrics MetricsProvider,
	audit AuditProvider,
	lifecycle LifecycleRecorder,
	enforce EnforceFunc,
	tracer *tracing.TracerProvider,
	getReqCaps GetRequiredCapsFunc,
) *Handler {
	return &Handler{
		status:     make(map[string]string),
		metrics:    metrics,
		audit:      audit,
		lifecycle:  lifecycle,
		enforce:    enforce,
		tracer:     tracer,
		getReqCaps: getReqCaps,
	}
}

// RegisterRoutes registers the delegation endpoints.
func (h *Handler) RegisterRoutes(r *gin.Engine, betaGroup *gin.RouterGroup) {
	// Public/API routes
	r.POST("/api/v1/delegation/create", h.Create)
	r.POST("/api/v1/delegation/revoke", h.Revoke)
	r.POST("/api/v1/delegation/status/update", h.StatusUpdate)

	// Beta alias/legacy routes
	if betaGroup != nil {
		betaGroup.POST("/delegation/create", h.Create)
		betaGroup.POST("/delegation/revoke", h.Revoke)
		betaGroup.POST("/delegation/status/update", h.StatusUpdate)
	}
}

func randomNonce(n int) string {
	// Simplified placeholders for internal utils - in real extracting we should export or copy util
	// For now we'll just use a time-based pseudo ID if we can't access utils,
	// but to match server_clean behavior exactly we might need the original randomNonce.
	// Assuming randomNonce is not exported from server_clean.
	// We'll implement a simple one here for the handler's internal use.
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	t := time.Now().UnixNano()
	for i := range b {
		b[i] = letterBytes[t%int64(len(letterBytes))]
		t /= 62
	}
	return string(b)
}

// Create creates a new delegation status entry.
func (h *Handler) Create(c *gin.Context) {
	var in struct {
		DelegationID string         `json:"delegation_id"`
		Subject      string         `json:"subject"`
		Delegate     string         `json:"delegate"`
		Claims       map[string]any `json:"claims"`
	}
	if err := c.BindJSON(&in); err != nil || in.DelegationID == "" {
		c.JSON(400, gin.H{"success": false, "error": "invalid_payload"})
		return
	}

	allowed, missing := h.enforce("delegation:create", in.Claims)
	if !allowed {
		h.recordDenial("delegation:create", in.DelegationID, in.Subject, missing)
		c.JSON(403, gin.H{"success": false, "error": "capability_denied", "missing": missing})
		return
	}

	if h.metrics != nil {
		h.metrics.IncCapabilityEnforceAllowed()
	}

	h.mu.Lock()
	h.status[in.DelegationID] = StatusActive
	h.mu.Unlock()

	// Append audit entry
	meta := map[string]any{"delegation_id": in.DelegationID, "delegate": in.Delegate}
	if in.Claims != nil {
		if caps, ok := in.Claims["cap"].([]string); ok {
			meta["caps"] = caps
		}
	}
	h.appendAudit(in.Subject, "delegation:create", "active", meta)

	c.JSON(200, gin.H{"success": true, "delegation_id": in.DelegationID, "status": "active"})
}

// Revoke updates status to terminated.
func (h *Handler) Revoke(c *gin.Context) {
	var in struct {
		DelegationID string         `json:"delegation_id"`
		Claims       map[string]any `json:"claims"`
		Reason       string         `json:"reason"`
	}
	if err := c.BindJSON(&in); err != nil || in.DelegationID == "" {
		c.JSON(400, gin.H{"success": false, "error": "invalid_payload"})
		return
	}

	allowed, missing := h.enforce("delegation:revoke", in.Claims)
	if !allowed {
		h.recordDenial("delegation:revoke", in.DelegationID, "revoker", missing)
		c.JSON(403, gin.H{"success": false, "error": "capability_denied", "missing": missing})
		return
	}

	if h.metrics != nil {
		h.metrics.IncCapabilityEnforceAllowed()
	}

	h.mu.Lock()
	h.status[in.DelegationID] = StatusTerminated
	h.mu.Unlock()

	meta := map[string]any{"delegation_id": in.DelegationID, "reason": in.Reason}
	h.appendAudit("revoker", "delegation:revoke", "terminated", meta)

	c.JSON(200, gin.H{"success": true, "delegation_id": in.DelegationID, "status": "terminated"})
}

// StatusUpdate handles manual status updates (prototype).
func (h *Handler) StatusUpdate(c *gin.Context) {
	start := time.Now()
	var span *tracing.Span
	if h.tracer != nil {
		_, span = h.tracer.StartSpan(context.Background(), "delegation_status_update")
	}
	var req struct {
		DelegationID string `json:"delegation_id"`
		NewStatus    string `json:"new_status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DelegationID == "" || req.NewStatus == "" {
		if h.metrics != nil {
			h.metrics.IncDelegationStatusTransitionFailures()
			h.metrics.RecordLifecycleTransition("delegation", "_", "_", "failure")
			if mm, ok := h.metrics.(*metrics.Memory); ok {
				mm.IncInvalidPayloadFailure()
			}
		}
		c.JSON(400, gin.H{"success": false, "error": "invalid_payload"})
		return
	}
	// Validate new status
	switch req.NewStatus {
	case StatusActive, StatusSuspended, StatusTerminated, StatusPartiallyRevoked:
		// ok
	default:
		if h.metrics != nil {
			if mm, ok := h.metrics.(*metrics.Memory); ok {
				mm.IncUnsupportedStatusFailure()
			}
		}
		c.JSON(400, gin.H{"success": false, "error": "invalid_status"})
		return
	}
	h.mu.Lock()
	old, exists := h.status[req.DelegationID]
	// Check transition validity
	if exists && old == StatusTerminated {
		if h.metrics != nil {
			h.metrics.IncDelegationStatusTransitionFailures()
			h.metrics.RecordLifecycleTransition("delegation", old, req.NewStatus, "failure")
			if mm, ok := h.metrics.(*metrics.Memory); ok {
				mm.IncInvalidTransitionFailure()
			}
		}
		h.mu.Unlock()
		c.JSON(409, gin.H{"success": false, "error": "cannot_transition", "current": "terminated"})
		return
	}
	if exists && old == StatusPartiallyRevoked && req.NewStatus == StatusActive {
		if h.metrics != nil {
			h.metrics.IncDelegationStatusTransitionFailures()
			h.metrics.RecordLifecycleTransition("delegation", old, req.NewStatus, "failure")
			if mm, ok := h.metrics.(*metrics.Memory); ok {
				mm.IncInvalidTransitionFailure()
			}
		}
		h.mu.Unlock()
		c.JSON(409, gin.H{"success": false, "error": "cannot_transition", "current": "partially_revoked"})
		return
	}
	initialized := !exists
	noChange := exists && old == req.NewStatus

	h.status[req.DelegationID] = req.NewStatus
	h.mu.Unlock()

	elapsed := time.Since(start)
	if h.lifecycle != nil && !noChange {
		h.lifecycle.RecordEvent("delegation", req.DelegationID, old, req.NewStatus, "success", "manual_update", elapsed.Nanoseconds())
	}
	if h.metrics != nil && !noChange {
		h.metrics.IncDelegationStatusTransitions()
		h.metrics.RecordLifecycleTransition("delegation", old, req.NewStatus, "success")
		h.metrics.ObserveLifecycleTransitionLatency("delegation", "success", elapsed)
	}

	if span != nil {
		span.End()
	}

	// Audit is optional here in original code, but we can add lightly if needed or just skip to match exact logic.
	// Original code only logged via logger (fmt.Printf equivalent) or just updated map.
	// Looking at original code: it simply updated the map and returned.

	c.JSON(200, gin.H{
		"success":       true,
		"delegation_id": req.DelegationID,
		"old_status":    old,
		"new_status":    req.NewStatus,
		"took_ms":       elapsed.Milliseconds(),
		"initialized":   initialized,
		"no_change":     noChange,
	})
}

// Snapshot returns a copy of the current delegation status map.
func (h *Handler) Snapshot() map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]string, len(h.status))
	for k, v := range h.status {
		out[k] = v
	}
	return out
}

func (h *Handler) recordDenial(action, id, actor string, missing []string) {
	if h.metrics != nil {
		h.metrics.IncViolation("capability_denied")
		h.metrics.IncCapabilityEnforceDenied()
	}
	if h.audit != nil {
		meta := map[string]any{"delegation_id": id, "missing": missing, "action": action}

		// Reconstruct lifecycle metadata logic for audit
		caps := capability.DefaultRegistry().List()
		reg := make(map[string]capability.Capability, len(caps))
		for _, cobj := range caps {
			reg[cobj.ID] = cobj
		}
		var lifecycle []map[string]any
		reqs := []string{}
		if h.getReqCaps != nil {
			reqs = h.getReqCaps(action)
		}
		for _, need := range reqs {
			co, ok := reg[need]
			if !ok {
				continue
			}
			phase := StatusActive
			if co.DeprecatedAfter != "" {
				if t, err := time.Parse(time.RFC3339, co.DeprecatedAfter); err == nil && time.Now().After(t) {
					phase = StatusDeprecated
				}
			}
			if co.SunsetAfter != "" {
				if t, err := time.Parse(time.RFC3339, co.SunsetAfter); err == nil && time.Now().After(t) {
					phase = StatusSunset
				}
			}
			lifecycle = append(lifecycle, map[string]any{
				"id":               need,
				"deprecated_after": co.DeprecatedAfter,
				"sunset_after":     co.SunsetAfter,
				"phase":            phase,
			})
		}
		meta["lifecycle"] = lifecycle
		h.appendAudit(actor, "capability:enforce", "denied", meta)
	}
}

func (h *Handler) appendAudit(actor, action, outcome string, meta any) {
	if h.audit != nil {
		h.audit.AppendEntry(randomNonce(6), time.Now(), actor, action, "delegation", outcome, meta)
	}
}
