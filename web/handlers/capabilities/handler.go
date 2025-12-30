package capabilities

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/capability"
	"github.com/mauriciomferz/AgentAuth/internal/notary"
	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	// "github.com/mauriciomferz/AgentAuth/internal/anchor" // Avoid circular dependency if possible, inject interface
)

// Handler manages capability state, configuration, and enforcement logic.
type Handler struct {
	mu sync.RWMutex

	// State
	ActionMappings         map[string][]string // action -> required capability IDs
	Enforce                bool                // enforcement flag
	LifecycleStrict        bool                // strict lifecycle negotiation
	LifecycleSunsetEnforce bool                // deny usage after sunset
	Source                 string              // provenance: static | file
	LastLoaded             time.Time           // timestamp of last successful load
	SchemaVersion          int                 // schema version
	RegistryHash           string              // current registry hash
	PrevRegistryHash       string              // previous registry hash
	AnchorStale            bool                // anchor staleness flag
	AnchorLastAge          time.Duration       // last observed anchor age
	AnchorStaleThreshold   time.Duration       // staleness threshold
	RegistryChangeAt       time.Time           // timestamp of last hash change

	// Audit State
	AuditPrevHash    string
	AuditPersistPath string
	AuditClient      AnchorClient // abstract interface for anchoring

	// Key manager for EdDSA public key exposure
	KeyManager *crypto.Manager

	// Metrics interface for anchor emission counters
	Metrics CapabilityMetrics

	// OnReload callback invoked after successful reload with new hash
	OnReload func(newHash string)
}

// CapabilityMetrics defines the interface for capability anchor metrics.
type CapabilityMetrics interface {
	IncCapabilityAnchorEmitted()
	IncCapabilityAnchorSkipped()
	IncCapabilityRegistryHashChanged()
}

// AnchorClient defines the interface for anchoring operations.
type AnchorClient interface {
	Anchor(hash string) (*notary.Receipt, error)
	TotalAnchors() int64
}

// NewHandler creates a new capabilities handler with defaults.
func NewHandler() *Handler {
	st := 12 * time.Hour
	if v := os.Getenv("GAUTH_CAP_ANCHOR_STALE_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			st = d
		}
	}

	return &Handler{
		ActionMappings:         make(map[string][]string),
		Source:                 "static",
		Enforce:                os.Getenv("GAUTH_CAPABILITY_ENFORCE") == "1",
		LifecycleStrict:        os.Getenv("GAUTH_CAP_LIFECYCLE_STRICT") == "1",
		LifecycleSunsetEnforce: os.Getenv("GAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE") == "1",
		AuditPersistPath:       os.Getenv("GAUTH_CAP_AUDIT_PATH"),
		AnchorStaleThreshold:   st,
	}
}

// LoadFromFile loads capabilities and action mappings from a JSON file.
// It updates the global registry and local state transactionally.
func (h *Handler) LoadFromFile(path string) error {
	// #nosec G304
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg struct {
		Capabilities   []capability.Capability `json:"capabilities"`
		ActionMappings map[string][]string     `json:"action_mappings"`
		SchemaVersion  int                     `json:"schema_version"`
	}
	if err2 := json.Unmarshal(b, &cfg); err2 != nil {
		return err2
	}
	// Validate schema version first.
	if cfg.SchemaVersion <= 0 {
		return fmt.Errorf("invalid or missing schema_version in capability file")
	}
	// Validate capabilities (unique IDs, non-empty fields) without mutating global state.
	idSet := make(map[string]struct{}, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		if c.ID == "" || c.Version == "" {
			return fmt.Errorf("invalid capability entry id/version empty")
		}
		if _, exists := idSet[c.ID]; exists {
			return fmt.Errorf("duplicate capability id: %s", c.ID)
		}
		idSet[c.ID] = struct{}{}
	}
	// Validate action mappings reference capabilities defined in this file.
	for act, caps := range cfg.ActionMappings {
		for _, cid := range caps {
			if _, ok := idSet[cid]; !ok {
				return fmt.Errorf("action mapping references unknown capability id=%s action=%s", cid, act)
			}
		}
	}
	// All validation passed: build canonical representation for hashing then apply transactionally.
	// Sort capabilities by ID for canonical form.
	capsSorted := make([]capability.Capability, len(cfg.Capabilities))
	copy(capsSorted, cfg.Capabilities)
	sort.Slice(capsSorted, func(i, j int) bool { return capsSorted[i].ID < capsSorted[j].ID })

	// Canonical action mappings: sort actions and each capability list.
	actions := make([]string, 0, len(cfg.ActionMappings))
	for act := range cfg.ActionMappings {
		actions = append(actions, act)
	}
	sort.Strings(actions)
	canonActions := make(map[string][]string, len(actions))
	for _, act := range actions {
		lst := make([]string, len(cfg.ActionMappings[act]))
		copy(lst, cfg.ActionMappings[act])
		sort.Strings(lst)
		canonActions[act] = lst
	}

	canon := struct {
		SchemaVersion  int                     `json:"schema_version"`
		Capabilities   []capability.Capability `json:"capabilities"`
		ActionMappings map[string][]string     `json:"action_mappings"`
	}{SchemaVersion: cfg.SchemaVersion, Capabilities: capsSorted, ActionMappings: canonActions}

	enc, err := json.Marshal(canon)
	if err != nil {
		return fmt.Errorf("canonical marshal: %w", err)
	}
	hSum := sha256.Sum256(enc)
	newHash := fmt.Sprintf("sha256:%x", hSum[:])

	h.mu.Lock()
	defer h.mu.Unlock()

	// Detect semantic change or initialization
	if h.RegistryHash != newHash {
		h.PrevRegistryHash = h.RegistryHash
		h.RegistryChangeAt = time.Now()
		// TODO: Audit logging logic could go here or be exposed via event/hook
	}

	// Update state
	h.RegistryHash = newHash
	h.ActionMappings = cfg.ActionMappings
	h.SchemaVersion = cfg.SchemaVersion
	h.Source = "file"
	h.LastLoaded = time.Now()

	// Update global registry (transactional reset)
	capability.Reset(cfg.Capabilities)

	// Invoke OnReload callback if set (for anchor emission)
	if h.OnReload != nil {
		h.OnReload(newHash)
	}

	// Emit metrics for hash change
	if h.Metrics != nil && h.PrevRegistryHash != "" && h.PrevRegistryHash != newHash {
		h.Metrics.IncCapabilityRegistryHashChanged()
	}

	return nil
}

// GetRequiredCaps returns the required capabilities for a given action.
func (h *Handler) GetRequiredCaps(action string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ActionMappings[action]
}

// IsEnforced returns whether capability enforcement is enabled.
func (h *Handler) IsEnforced() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Enforce
}

// GetRegistryHash returns the current registry hash.
func (h *Handler) GetRegistryHash() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.RegistryHash
}

// GetState returns a snapshot of the current state for API consumption.
func (h *Handler) GetState() (source, hash, prevHash string, version, actions int, loaded, changed time.Time) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Source, h.RegistryHash, h.PrevRegistryHash, h.SchemaVersion, len(h.ActionMappings), h.LastLoaded, h.RegistryChangeAt
}

// IsSunsetEnforced returns whether sunset enforcement is enabled.
func (h *Handler) IsSunsetEnforced() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.LifecycleSunsetEnforce
}

// GetAuditPersistPath returns the audit persistence path.
func (h *Handler) GetAuditPersistPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.AuditPersistPath
}

// SetAuditPersistPath sets the audit persistence path.
func (h *Handler) SetAuditPersistPath(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.AuditPersistPath = path
}

// SetAnchorClient sets the anchor client for audit operations.
func (h *Handler) SetAnchorClient(client AnchorClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.AuditClient = client
}

// GetAuditPrevHash returns the previous audit hash.
func (h *Handler) GetAuditPrevHash() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.AuditPrevHash
}

// SetAuditPrevHash sets the previous audit hash.
func (h *Handler) SetAuditPrevHash(hash string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.AuditPrevHash = hash
}

// GetActionMappings returns a copy of the action mappings.
func (h *Handler) GetActionMappings() map[string][]string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string][]string, len(h.ActionMappings))
	for k, v := range h.ActionMappings {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// GetSchemaVersion returns the schema version.
func (h *Handler) GetSchemaVersion() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.SchemaVersion
}

// GetPrevRegistryHash returns the previous registry hash.
func (h *Handler) GetPrevRegistryHash() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.PrevRegistryHash
}

// GetRegistryChangeAt returns the timestamp of the last registry change.
func (h *Handler) GetRegistryChangeAt() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.RegistryChangeAt
}

// GetSource returns the source of the capabilities.
func (h *Handler) GetSource() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.Source
}

// GetLastLoaded returns the timestamp of the last successful load.
func (h *Handler) GetLastLoaded() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.LastLoaded
}

// SetAnchorState updates anchor staleness state.
func (h *Handler) SetAnchorState(stale bool, age time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.AnchorStale = stale
	h.AnchorLastAge = age
}

// GetAnchorState returns anchor staleness state.
func (h *Handler) GetAnchorState() (stale bool, age, threshold time.Duration) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.AnchorStale, h.AnchorLastAge, h.AnchorStaleThreshold
}
