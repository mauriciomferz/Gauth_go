package web

// RB13 Capability Diff Endpoint (skeleton implementation).
// Path: GET /api/v1/capabilities/diff?since=<hash>
// Returns a diff between the capability registry state identified by the 'since' hash
// and the current registry. If 'since' is missing or equals current hash the diff arrays
// will be empty and base/current hashes identical.
//
// Future enhancements:
// - Historical snapshot retention (in-memory ring + optional persistence)
// - Modified detection with field-level changes
// - Pagination for large diffs (>1000 entries)
// - Signed diff artifact
// - Provider supplied ETag + If-None-Match handling
// - Labeled metrics by diff size bucket

import (
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/capability"
	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/tracing"
	"github.com/gin-gonic/gin"
)

// capabilityDiffResponse defines the JSON structure returned by the diff endpoint.
type capabilityDiffResponse struct {
	BaseHash      string                 `json:"base_hash"`
	CurrentHash   string                 `json:"current_hash"`
	Added         []simpleCapabilityMeta `json:"added"`
	Removed       []simpleCapabilityMeta `json:"removed"`
	Modified      []capabilityModified   `json:"modified"`
	GeneratedAt   string                 `json:"generated_at"`
	SchemaVersion int                    `json:"schema_version"`
}

type simpleCapabilityMeta struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// capabilityModified contains a before/after pair for a changed capability.
type capabilityModified struct {
	ID     string                `json:"id"`
	Before *simpleCapabilityMeta `json:"before,omitempty"`
	After  *simpleCapabilityMeta `json:"after,omitempty"`
}

// diffCapabilities computes added, removed, modified sets between base and current.
func diffCapabilities(base, current []capability.Capability) (added, removed []simpleCapabilityMeta, modified []capabilityModified) {
	baseMap := make(map[string]capability.Capability, len(base))
	for _, c := range base {
		baseMap[c.ID] = c
	}
	currMap := make(map[string]capability.Capability, len(current))
	for _, c := range current {
		currMap[c.ID] = c
	}
	// Added & modified
	for id, curr := range currMap {
		b, exists := baseMap[id]
		if !exists {
			added = append(added, simpleCapabilityMeta{ID: curr.ID, Version: curr.Version, Stable: curr.Stable})
			continue
		}
		// Detect modification (semantic field difference)
		if curr.Version != b.Version || curr.Stable != b.Stable || curr.DeprecatedAfter != b.DeprecatedAfter || curr.SunsetAfter != b.SunsetAfter || len(curr.Versions) != len(b.Versions) {
			modified = append(modified, capabilityModified{ID: id, Before: &simpleCapabilityMeta{ID: b.ID, Version: b.Version, Stable: b.Stable}, After: &simpleCapabilityMeta{ID: curr.ID, Version: curr.Version, Stable: curr.Stable}})
			continue
		}
		// Versions slice content comparison when lengths equal
		for i := range curr.Versions {
			if curr.Versions[i] != b.Versions[i] {
				modified = append(modified, capabilityModified{ID: id, Before: &simpleCapabilityMeta{ID: b.ID, Version: b.Version, Stable: b.Stable}, After: &simpleCapabilityMeta{ID: curr.ID, Version: curr.Version, Stable: curr.Stable}})
				break
			}
		}
	}
	// Removed
	for id, b := range baseMap {
		if _, exists := currMap[id]; !exists {
			removed = append(removed, simpleCapabilityMeta{ID: b.ID, Version: b.Version, Stable: b.Stable})
		}
	}
	// Deterministic ordering
	sort.Slice(added, func(i, j int) bool { return added[i].ID < added[j].ID })
	sort.Slice(removed, func(i, j int) bool { return removed[i].ID < removed[j].ID })
	sort.Slice(modified, func(i, j int) bool { return modified[i].ID < modified[j].ID })
	return
}

// registerCapabilityDiff mounts the capability diff endpoint.
func (s *BetaServer) registerCapabilityDiff() {
	s.router.GET("/api/v1/capabilities/diff", func(c *gin.Context) {
		start := time.Now()
		// RB9 tracing span (capability.diff)
		var span *tracing.Span
		if s.tracerProvider != nil && (s.tracerSampleRatio <= 0 || rand.Float64() < s.tracerSampleRatio) {
			_, span = s.tracerProvider.StartSpan(c.Request.Context(), "capability.diff")
			if span != nil {
				sinceParam := c.Query("since")
				span.SetTag("since_param", sinceParam)
				span.SetTag("has_since", sinceParam != "")
			}
		}
		since := c.Query("since")
		caps := capability.DefaultRegistry().List()
		currentHash := capability.RegistryHash(caps)
		// Record snapshot for current state (retention capacity governs history depth)
		if s.capDiffSnapshots != nil {
			s.capDiffSnapshots.Add(caps, currentHash)
		}
		baseHash := since
		var baseCaps []capability.Capability
		if baseHash == "" || baseHash == currentHash {
			baseHash = currentHash
			baseCaps = caps
		} else {
			// Lookup snapshot by hash
			if s.capDiffSnapshots == nil {
				respondError(c, http.StatusNotFound, "capability_version_not_found", "version_not_found", "snapshot retention disabled", "rfc111:capability_diff", nil)
				if span != nil {
					span.SetTag("outcome", "error")
					span.SetTag("error_code", "snapshot_disabled")
					span.End()
				}
				return
			}
			snap, ok := s.capDiffSnapshots.Get(baseHash)
			if !ok {
				respondError(c, http.StatusNotFound, "capability_version_not_found", "version_not_found", "base capability hash unknown", "rfc111:capability_diff", nil)
				if span != nil {
					span.SetTag("outcome", "error")
					span.SetTag("error_code", "base_hash_unknown")
					span.End()
				}
				return
			}
			baseCaps = snap.Capabilities
		}
		added, removed, modified := diffCapabilities(baseCaps, caps)
		resp := capabilityDiffResponse{
			BaseHash:      baseHash,
			CurrentHash:   currentHash,
			Added:         added,
			Removed:       removed,
			Modified:      modified,
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
			SchemaVersion: 1,
		}
		c.JSON(200, resp)
		if span != nil {
			span.SetTag("outcome", "success")
			span.SetTag("base_hash", baseHash)
			span.SetTag("current_hash", currentHash)
			span.SetTag("equal", baseHash == currentHash)
			span.SetTag("added_count", len(added))
			span.SetTag("removed_count", len(removed))
			span.SetTag("modified_count", len(modified))
			span.End()
		}
		if m, ok := s.metrics.(interface {
			IncCapabilityDiffRequests()
			ObserveCapabilityDiffLatency(d time.Duration)
		}); ok {
			m.IncCapabilityDiffRequests()
			m.ObserveCapabilityDiffLatency(time.Since(start))
		}
		_ = imetrics.Noop // reference to ensure import retained until full implementation
	})
}
