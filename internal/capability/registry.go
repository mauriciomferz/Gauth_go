// Package capability provides a lightweight in-memory capability registry used for
// governance enforcement (demo). It supports registration, listing, and basic
// validation of required capabilities. Persistence and version negotiation can
// be layered later without changing this surface.
package capability

import "sync"

// Capability describes a governed action or feature with lifecycle metadata.
type Capability struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Stable          bool     `json:"stable"`
	DeprecatedAfter string   `json:"deprecated_after,omitempty"` // RFC3339 timestamp string (optional)
	SunsetAfter     string   `json:"sunset_after,omitempty"`     // planned full removal (RFC3339) optional
	Versions        []string `json:"versions,omitempty"`         // optional multi-version list (superset includes Version)
}

// Registry holds capabilities in-memory; lightweight for now (can extend to persistent backend later).
type Registry struct {
	mu           sync.RWMutex
	capabilities map[string]Capability
}

var defaultRegistry = NewRegistry()

// NewRegistry constructs a new empty registry.
func NewRegistry() *Registry { return &Registry{capabilities: make(map[string]Capability)} }

// Register adds/overwrites a capability.
func (r *Registry) Register(c Capability) {
	r.mu.Lock()
	r.capabilities[c.ID] = c
	r.mu.Unlock()
}

// Register global convenience.
func Register(c Capability) { defaultRegistry.Register(c) }

// List returns a snapshot of all capabilities.
func (r *Registry) List() []Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Capability, 0, len(r.capabilities))
	for _, c := range r.capabilities {
		out = append(out, c)
	}
	return out
}

// DefaultRegistry returns global registry instance.
func DefaultRegistry() *Registry { return defaultRegistry }

// Reset atomically replaces the contents of the default registry with the provided slice.
// Intended for transactional reload operations where validation occurs prior to mutation.
func Reset(list []Capability) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	// Replace map to drop any stale capabilities.
	newMap := make(map[string]Capability, len(list))
	for _, c := range list {
		newMap[c.ID] = c
	}
	defaultRegistry.capabilities = newMap
}

// BuildProvided converts a slice into a set for validation.
func BuildProvided(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, v := range list {
		m[v] = true
	}
	return m
}

// ValidateCapabilities returns missing required IDs.
func ValidateCapabilities(required []string, provided map[string]bool) (missing []string) {
	for _, req := range required {
		if !provided[req] {
			missing = append(missing, req)
		}
	}
	return missing
}
