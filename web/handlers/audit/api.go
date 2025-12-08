package audit

// Audit API handlers - extracted from server_clean.go
// Provides endpoints for audit entry listing, creation, filtering, and streaming.

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Entry represents an audit log entry interface for type-safe access.
type Entry interface {
	GetID() string
	GetAt() time.Time
	GetActor() string
	GetAction() string
	GetResource() string
	GetOutcome() string
	GetMeta() any
}

// Provider abstracts the audit log operations.
type Provider interface {
	ListEntries(limit int) []Entry
	ListEntriesAfter(cursor string, limit int) ([]Entry, string)
	AppendEntry(id string, at time.Time, actor, action, resource, outcome string, meta any)
	SubscribeEntries() chan Entry
	UnsubscribeEntries(ch chan Entry)
}

// RandomNonceFunc generates random nonce strings for entry IDs.
type RandomNonceFunc func(length int) string

// API provides HTTP handlers for audit endpoints.
type API struct {
	Provider    Provider
	RandomNonce RandomNonceFunc
}

// NewAPI creates a new audit API instance.
func NewAPI(provider Provider, nonceFunc RandomNonceFunc) *API {
	return &API{Provider: provider, RandomNonce: nonceFunc}
}

// RegisterRoutes registers audit endpoints on the router.
func (a *API) RegisterRoutes(r *gin.Engine) {
	// Beta endpoint
	r.GET("/api/v1/beta/audit", a.Entries)
	// Standard endpoints
	r.GET("/api/v1/audit/logs", a.List)
	r.GET("/api/v1/audit/capabilities", a.Capabilities)
	r.POST("/api/v1/audit/record", a.Record)
	r.GET("/api/v1/audit/stream", a.Stream)
}

// entryToMap converts an Entry to a gin.H map for JSON response.
func entryToMap(e Entry) gin.H {
	return gin.H{
		"id":       e.GetID(),
		"at":       e.GetAt().Format(time.RFC3339),
		"actor":    e.GetActor(),
		"action":   e.GetAction(),
		"resource": e.GetResource(),
		"outcome":  e.GetOutcome(),
		"meta":     e.GetMeta(),
	}
}

// Entries returns all in-memory audit entries (GET /api/v1/beta/audit).
func (a *API) Entries(c *gin.Context) {
	if a.Provider == nil {
		c.JSON(200, gin.H{"success": true, "entries": []interface{}{}, "total": 0})
		return
	}
	raw := a.Provider.ListEntries(0)
	out := make([]gin.H, 0, len(raw))
	for _, e := range raw {
		out = append(out, entryToMap(e))
	}
	c.JSON(200, gin.H{"success": true, "entries": out, "total": len(out)})
}

// Capabilities returns capability enforcement related audit entries with pagination.
// Query params: limit (int, default 50, max 500), cursor (string), outcome (string), action (string)
func (a *API) Capabilities(c *gin.Context) {
	if a.Provider == nil {
		c.JSON(200, gin.H{"success": true, "entries": []any{}, "count": 0, "has_more": false})
		return
	}
	// Parse limit
	limit := 50
	if lStr := c.Query("limit"); lStr != "" {
		if v, err := strconv.Atoi(lStr); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 500 {
		limit = 500
	}
	// Parse cursor
	cursor := 0
	if cStr := c.Query("cursor"); cStr != "" {
		if v, err := strconv.Atoi(cStr); err == nil && v >= 0 {
			cursor = v
		}
	}
	// Filter params
	outcomeFilter := c.Query("outcome")
	actionFilter := c.Query("action")

	// Get all entries and filter
	all := a.Provider.ListEntries(0)
	capActions := []string{"capability_create", "capability_revoke", "capability_denied", "capability_enforce", "delegation_create", "delegation_revoke", "capability:enforce", "delegation:create", "delegation:revoke"}

	var filtered []Entry
	for _, e := range all {
		// Check if it's a capability-related action
		isCapAction := false
		for _, ca := range capActions {
			if e.GetAction() == ca {
				isCapAction = true
				break
			}
		}
		if !isCapAction {
			continue
		}
		// Apply outcome filter
		if outcomeFilter != "" && e.GetOutcome() != outcomeFilter {
			continue
		}
		// Apply action filter
		if actionFilter != "" && e.GetAction() != actionFilter {
			continue
		}
		filtered = append(filtered, e)
	}

	totalFiltered := len(filtered)
	// Apply cursor/pagination
	if cursor >= len(filtered) {
		c.JSON(200, gin.H{"success": true, "entries": []any{}, "count": 0, "has_more": false, "total_filtered": totalFiltered})
		return
	}
	end := cursor + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[cursor:end]
	hasMore := end < len(filtered)
	nextCursor := ""
	if hasMore {
		nextCursor = strconv.Itoa(end)
	}

	// Convert to output format
	out := make([]gin.H, 0, len(page))
	for _, e := range page {
		out = append(out, entryToMap(e))
	}

	c.JSON(200, gin.H{
		"success":        true,
		"entries":        out,
		"count":          len(out),
		"has_more":       hasMore,
		"next_cursor":    nextCursor,
		"total_filtered": totalFiltered,
	})
}

// Record creates a new audit entry (POST /api/v1/audit/record).
func (a *API) Record(c *gin.Context) {
	var req struct {
		Actor    string `json:"actor"`
		Action   string `json:"action"`
		Resource string `json:"resource"`
		Outcome  string `json:"outcome"`
		Meta     any    `json:"meta"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Action == "" {
		c.JSON(400, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	id := a.RandomNonce(6)
	at := time.Now()
	a.Provider.AppendEntry(id, at, req.Actor, req.Action, req.Resource, req.Outcome, req.Meta)
	c.JSON(201, gin.H{"success": true, "entry": gin.H{
		"id":       id,
		"at":       at.Format(time.RFC3339),
		"actor":    req.Actor,
		"action":   req.Action,
		"resource": req.Resource,
		"outcome":  req.Outcome,
		"meta":     req.Meta,
	}})
}

// List returns paginated audit entries with optional cursor and CSV export.
// Query params: limit (int), cursor (string), format (csv)
func (a *API) List(c *gin.Context) {
	limit := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	cursorParam := c.Query("cursor")
	var entries []Entry
	var nextCursor string
	if cursorParam != "" {
		entries, nextCursor = a.Provider.ListEntriesAfter(cursorParam, limit)
	} else {
		entries = a.Provider.ListEntries(limit)
		if len(entries) > 0 {
			nextCursor = entries[len(entries)-1].GetID()
		}
	}

	wantsCSV := func() bool {
		if strings.EqualFold(c.Query("format"), "csv") {
			return true
		}
		accept := c.GetHeader("Accept")
		if accept == "" {
			return false
		}
		for _, mt := range strings.Split(accept, ",") {
			mt = strings.TrimSpace(strings.Split(mt, ";")[0])
			if mt == "text/csv" || mt == "application/csv" {
				return true
			}
		}
		return false
	}

	if wantsCSV() {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=audit.csv")
		w := csv.NewWriter(c.Writer)
		_ = w.Write([]string{"id", "at", "actor", "action", "resource", "outcome", "reason"})
		for _, e := range entries {
			// Extract reason from meta if present
			reason := ""
			if meta := e.GetMeta(); meta != nil {
				if m, ok := meta.(map[string]any); ok {
					if rv, ok2 := m["reason"].(string); ok2 {
						reason = rv
					}
				}
			}
			_ = w.Write([]string{e.GetID(), e.GetAt().Format(time.RFC3339), e.GetActor(), e.GetAction(), e.GetResource(), e.GetOutcome(), reason})
		}
		w.Flush()
		return
	}

	out := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		out = append(out, entryToMap(e))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"entries":     out,
		"count":       len(out),
		"next_cursor": nextCursor,
	})
}

// Stream provides SSE streaming of audit events (GET /api/v1/audit/stream).
func (a *API) Stream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := a.Provider.SubscribeEntries()
	defer a.Provider.UnsubscribeEntries(ch)

	// Send recent history snapshot (last 20)
	history := a.Provider.ListEntries(20)
	for _, e := range history {
		if b, err := json.Marshal(entryToMap(e)); err == nil {
			fmt.Fprintf(c.Writer, "event: audit\ndata: %s\n\n", b)
		}
	}
	fmt.Fprint(c.Writer, "event: open\ndata: {\"ok\":true}\n\n")
	c.Writer.Flush()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case e := <-ch:
			if b, err := json.Marshal(entryToMap(e)); err == nil {
				fmt.Fprintf(c.Writer, "event: audit\ndata: %s\n\n", b)
				c.Writer.Flush()
			}
		case <-ticker.C:
			fmt.Fprint(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		}
	}
}
