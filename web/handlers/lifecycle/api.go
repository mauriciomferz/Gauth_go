package lifecycle

// Lifecycle handlers - extracted from server_clean.go
// Provides lifecycle metrics and timeline introspection.

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Event represents a lifecycle transition event.
type Event struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	Outcome    string    `json:"outcome"`
	Reason     string    `json:"reason"`
	LatencyNS  int64     `json:"latency_ns"`
	At         time.Time `json:"at"`
}

// MetricsSnapshot contains lifecycle metrics data.
type MetricsSnapshot struct {
	DelegationStatusTransitions        uint64            `json:"delegation_status_transitions"`
	DelegationStatusTransitionFailures uint64            `json:"delegation_status_transition_failures"`
	TokenStatusTransitions             uint64            `json:"token_status_transitions"`
	TokenStatusTransitionFailures      uint64            `json:"token_status_transition_failures"`
	MultiSignatureWeightFailures       uint64            `json:"multi_signature_weight_failures"`
	LifecycleBreakdown                 map[string]uint64 `json:"lifecycle_breakdown"`
	DecisionBreakdown                  map[string]uint64 `json:"decision_breakdown"`
	DecisionReasonBreakdown            map[string]uint64 `json:"decision_reason_breakdown"`
	LifecycleLatencyTotals             map[string]uint64 `json:"lifecycle_latency_totals_ns"`
	LifecycleLatencyCounts             map[string]uint64 `json:"lifecycle_latency_counts"`
	LifecycleLatencyMax                map[string]uint64 `json:"lifecycle_latency_max_ns"`
	LifecycleLatencyP50                map[string]uint64 `json:"lifecycle_latency_p50_ns"`
	LifecycleLatencyP90                map[string]uint64 `json:"lifecycle_latency_p90_ns"`
	LifecycleLatencyP99                map[string]uint64 `json:"lifecycle_latency_p99_ns"`
	LastPersistUnix                    uint64            `json:"last_persist_unix"`
	LegacyAliasHits                    uint64            `json:"legacy_alias_hits"`
}

// MetricsProvider provides lifecycle metrics.
type MetricsProvider interface {
	// GetLifecycleSnapshot returns the current lifecycle metrics snapshot.
	// Returns nil if metrics are not available.
	GetLifecycleSnapshot() *MetricsSnapshot
}

// EventProvider provides lifecycle events.
type EventProvider interface {
	// ListEvents returns lifecycle events matching the filter criteria.
	ListEvents(filter EventFilter) ([]*Event, string)
}

// EventFilter specifies criteria for filtering lifecycle events.
type EventFilter struct {
	EntityType string
	EntityID   string
	Since      time.Time
	Outcome    string
	Reason     string
	Cursor     string
	Limit      int
}

// API provides HTTP handlers for lifecycle metrics and timeline.
type API struct {
	metrics MetricsProvider
	events  EventProvider
}

// NewAPI creates a new lifecycle API handler.
func NewAPI(metrics MetricsProvider, events EventProvider) *API {
	return &API{metrics: metrics, events: events}
}

// RegisterRoutes registers lifecycle endpoints on the router.
func (a *API) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v1/beta/metrics/lifecycle", a.Metrics)
	r.GET("/api/v1/beta/lifecycle/timeline", a.Timeline)
}

// Metrics returns high-level lifecycle counters.
func (a *API) Metrics(c *gin.Context) {
	if a.metrics == nil {
		c.JSON(200, gin.H{"success": true, "metrics": gin.H{"available": false}})
		return
	}
	snap := a.metrics.GetLifecycleSnapshot()
	if snap == nil {
		c.JSON(200, gin.H{"success": true, "metrics": gin.H{"available": true}})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"metrics": gin.H{
			"delegation_status_transitions":         snap.DelegationStatusTransitions,
			"delegation_status_transition_failures": snap.DelegationStatusTransitionFailures,
			"token_status_transitions":              snap.TokenStatusTransitions,
			"token_status_transition_failures":      snap.TokenStatusTransitionFailures,
			"multi_signature_weight_failures":       snap.MultiSignatureWeightFailures,
			"lifecycle_breakdown":                   snap.LifecycleBreakdown,
			"decision_breakdown":                    snap.DecisionBreakdown,
			"decision_reason_breakdown":             snap.DecisionReasonBreakdown,
			"lifecycle_latency_totals_ns":           snap.LifecycleLatencyTotals,
			"lifecycle_latency_counts":              snap.LifecycleLatencyCounts,
			"lifecycle_latency_max_ns":              snap.LifecycleLatencyMax,
			"lifecycle_latency_p50_ns":              snap.LifecycleLatencyP50,
			"lifecycle_latency_p90_ns":              snap.LifecycleLatencyP90,
			"lifecycle_latency_p99_ns":              snap.LifecycleLatencyP99,
			"last_persist_unix":                     snap.LastPersistUnix,
			"legacy_alias_hits":                     snap.LegacyAliasHits,
		},
	})
}

// Timeline returns lifecycle events with filtering and pagination.
func (a *API) Timeline(c *gin.Context) {
	// Parse query parameters
	filter := EventFilter{
		EntityType: c.Query("entity_type"),
		EntityID:   c.Query("entity_id"),
		Outcome:    c.Query("outcome"),
		Reason:     c.Query("reason"),
		Cursor:     c.Query("cursor"),
		Limit:      100,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 250 {
			filter.Limit = v
		}
	}

	if sinceStr := c.Query("since"); sinceStr != "" {
		if sec, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			filter.Since = time.Unix(sec, 0)
		}
	}

	results, nextCursor := a.events.ListEvents(filter)

	// Check for CSV export
	if wantsCSV(c) {
		var b strings.Builder
		b.WriteString("entity_type,entity_id,old_status,new_status,outcome,reason,latency_ns,at\n")
		for _, ev := range results {
			b.WriteString(ev.EntityType + "," + ev.EntityID + "," + ev.OldStatus + "," +
				ev.NewStatus + "," + ev.Outcome + "," + ev.Reason + "," +
				strconv.FormatInt(ev.LatencyNS, 10) + "," + ev.At.Format(time.RFC3339) + "\n")
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		c.String(200, b.String())
		return
	}

	c.JSON(200, gin.H{
		"success":     true,
		"events":      results,
		"count":       len(results),
		"next_cursor": nextCursor,
	})
}

func wantsCSV(c *gin.Context) bool {
	if strings.EqualFold(c.Query("format"), "csv") {
		return true
	}
	accept := c.GetHeader("Accept")
	if accept == "" {
		return false
	}
	for _, part := range strings.Split(accept, ",") {
		p := strings.ToLower(strings.TrimSpace(strings.Split(part, ";")[0]))
		if p == "text/csv" || p == "application/csv" {
			return true
		}
	}
	return false
}
