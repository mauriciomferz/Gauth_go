package testutil

// This file provides lightweight parsing & validation helpers for the JSON fixture
// constants declared in fixtures.go. Keeping these here prevents test code from
// repeatedly re-declaring ephemeral structs or manual map[string]any parsing.
//
// IMPORTANT: These structs intentionally model only the fields exercised by tests.
// If the registry or policy schema evolves, these can be extended with additive
// fields without breaking existing tests. For negative fixtures (e.g. missing
// schema_version) we return structured errors.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Capability represents a single capability entry in a registry fixture.
type Capability struct {
    ID      string `json:"id"`
    Version string `json:"version"`
    Stable  bool   `json:"stable"`
}

// CapabilityRegistry models the JSON capability registry structure used in tests.
type CapabilityRegistry struct {
    SchemaVersion int                   `json:"schema_version"`
    Capabilities  []Capability          `json:"capabilities"`
    ActionMapping map[string][]string   `json:"action_mappings"`
}

// PolicyRule represents a single rule inside a policy bundle.
type PolicyRule struct {
    Actions   []string `json:"actions"`
    Resources []string `json:"resources"`
    Effect    string   `json:"effect"`
}

// Policy represents a single policy in a bundle.
type Policy struct {
    ID       string       `json:"id"`
    Subjects []string     `json:"subjects"`
    Rules    []PolicyRule `json:"rules"`
}

// PolicyBundle models the top-level bundle fixture structure.
type PolicyBundle struct {
    ID       string   `json:"id"`
    Policies []Policy `json:"policies"`
}

// ErrMissingSchemaVersion is returned when a registry JSON lacks schema_version.
var ErrMissingSchemaVersion = errors.New("capability registry missing schema_version or value is zero")

// ErrDuplicateCapabilityID is returned when the registry lists the same capability id more than once.
var ErrDuplicateCapabilityID = errors.New("capability registry contains duplicate capability id")

// RegistryErrorKind categorizes structured capability registry validation errors.
type RegistryErrorKind int

const (
    // RegistryErrorUnknownCapability indicates an action mapping references an unknown capability id.
    RegistryErrorUnknownCapability RegistryErrorKind = iota + 1
    // RegistryErrorEmptyActionMapping indicates an action maps to zero capabilities.
    RegistryErrorEmptyActionMapping
)

// CapabilityRegistryError carries structured context for advanced validation errors.
// Sentinel errors (missing schema, duplicate IDs) are still returned directly to preserve
// existing equality checks in tests.
type CapabilityRegistryError struct {
    Kind        RegistryErrorKind
    Action      string
    CapabilityID string
    Message     string
}

func (e *CapabilityRegistryError) Error() string { return e.Message }

// Is allows errors.Is matching against generic categories if needed.
func (e *CapabilityRegistryError) Is(target error) bool {
    // Support comparison with standard errors created via errors.New with same message if desired.
    if target == nil { return false }
    return e.Message == target.Error()
}

func newUnknownCapabilityError(action, id string) error {
    return &CapabilityRegistryError{
        Kind: RegistryErrorUnknownCapability,
        Action: action,
        CapabilityID: id,
        Message: fmt.Sprintf("action '%s' references unknown capability id '%s'", action, id),
    }
}

func newEmptyActionMappingError(action string) error {
    return &CapabilityRegistryError{
        Kind: RegistryErrorEmptyActionMapping,
        Action: action,
        Message: fmt.Sprintf("action '%s' maps to empty capability list", action),
    }
}

// AsCapabilityRegistryError attempts to cast err to *CapabilityRegistryError returning the typed error and true
// on success; otherwise (nil, false). This avoids callers needing a direct type assertion and centralizes future
// interface adaptation if error representation changes.
func AsCapabilityRegistryError(err error) (*CapabilityRegistryError, bool) {
    if err == nil { return nil, false }
    te, ok := err.(*CapabilityRegistryError)
    if ok { return te, true }
    return nil, false
}

// ParseCapabilityRegistry unmarshals a registry JSON string into a CapabilityRegistry.
// It performs minimal validation to surface common fixture problems early.
func ParseCapabilityRegistry(raw string) (*CapabilityRegistry, error) {
    var reg CapabilityRegistry
    if err := json.Unmarshal([]byte(raw), &reg); err != nil {
        return nil, fmt.Errorf("unmarshal capability registry: %w", err)
    }
    if reg.SchemaVersion == 0 { // missing or zero
        return nil, ErrMissingSchemaVersion
    }
    // Basic internal consistency: every action mapping refers to at least one capability id.
    if len(reg.Capabilities) == 0 {
        return &reg, nil // allow empty capability list in negative tests
    }
    ids := make(map[string]struct{}, len(reg.Capabilities))
    for _, c := range reg.Capabilities {
        if _, exists := ids[c.ID]; exists {
            return nil, ErrDuplicateCapabilityID
        }
        ids[c.ID] = struct{}{}
    }
    for action, caps := range reg.ActionMapping {
        if len(caps) == 0 {
            return nil, newEmptyActionMappingError(action)
        }
        for _, id := range caps {
            if _, ok := ids[id]; !ok {
                return nil, newUnknownCapabilityError(action, id)
            }
        }
    }
    return &reg, nil
}

// MustCapabilityRegistry parses a registry fixture and panics on error. Intended for
// use in test setup where a failure should abort immediately.
func MustCapabilityRegistry(raw string) *CapabilityRegistry {
    reg, err := ParseCapabilityRegistry(raw)
    if err != nil {
        panic(err)
    }
    return reg
}

// ParsePolicyBundle unmarshals a policy bundle fixture.
func ParsePolicyBundle(raw string) (*PolicyBundle, error) {
    var b PolicyBundle
    if err := json.Unmarshal([]byte(raw), &b); err != nil {
        return nil, fmt.Errorf("unmarshal policy bundle: %w", err)
    }
    if b.ID == "" {
        return nil, errors.New("policy bundle missing id")
    }
    return &b, nil
}

// MustPolicyBundle parses a bundle and panics on error.
func MustPolicyBundle(raw string) *PolicyBundle {
    b, err := ParsePolicyBundle(raw)
    if err != nil {
        panic(err)
    }
    return b
}

// SHA256Hex returns the lowercase hex-encoded SHA256 of the raw fixture string.
func SHA256Hex(raw string) string {
    h := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(h[:])
}

// IsValidCapabilityRegistry quickly checks if basic fields present; does not perform deep validation.
func IsValidCapabilityRegistry(raw string) bool {
    var tmp struct {
        SchemaVersion int `json:"schema_version"`
        Capabilities  []any `json:"capabilities"`
    }
    if err := json.Unmarshal([]byte(raw), &tmp); err != nil {
        return false
    }
    return tmp.SchemaVersion > 0 && len(tmp.Capabilities) > 0
}

// CanonicalRegistryHash returns the SHA256 hex of the canonicalized registry JSON.
// This allows semantically equivalent registries with different ordering to hash equally.
// For negative or unparsable fixtures it falls back to hashing the raw input.
var registryBuilderPool = sync.Pool{New: func() any { return &strings.Builder{} }}

func CanonicalRegistryHash(raw string) string {
    reg, err := ParseCapabilityRegistry(raw)
    if err != nil { return SHA256Hex(raw) }
    // Sort capabilities by ID then Version (must match previous canonical ordering).
    sort.SliceStable(reg.Capabilities, func(i, j int) bool {
        if reg.Capabilities[i].ID == reg.Capabilities[j].ID {
            return reg.Capabilities[i].Version < reg.Capabilities[j].Version
        }
        return reg.Capabilities[i].ID < reg.Capabilities[j].ID
    })
    // Sort action mapping keys.
    keys := make([]string, 0, len(reg.ActionMapping))
    for k := range reg.ActionMapping { keys = append(keys, k) }
    sort.Strings(keys)
    // Acquire pooled builder.
    b := registryBuilderPool.Get().(*strings.Builder)
    b.Reset()
    // Capacity hint.
    b.Grow(128 + len(reg.Capabilities)*64 + len(keys)*64)
    b.WriteString("{\"schema_version\":")
    b.WriteString(strconv.Itoa(reg.SchemaVersion))
    b.WriteString(",\"capabilities\":[")
    for i, c := range reg.Capabilities {
        if i > 0 { b.WriteByte(',') }
        b.WriteString("{\"id\":\"")
        b.WriteString(c.ID)
        b.WriteString("\",\"version\":\"")
        b.WriteString(c.Version)
        b.WriteString("\",\"stable\":")
        if c.Stable { b.WriteString("true") } else { b.WriteString("false") }
        b.WriteByte('}')
    }
    // IMPORTANT: Maintain exact formatting (no extra spaces) to preserve historical golden hashes.
    b.WriteString("],\"action_mappings\":{")
    for i, k := range keys {
        if i > 0 { b.WriteByte(',') }
        b.WriteString("\"")
        b.WriteString(k)
        b.WriteString("\":[")
        vals := reg.ActionMapping[k]
        for j, v := range vals {
            if j > 0 { b.WriteByte(',') }
            b.WriteString("\"")
            b.WriteString(v)
            b.WriteString("\"")
        }
        b.WriteByte(']')
    }
    b.WriteString("}}")
    // Hash and release builder.
    data := b.String()
    registryBuilderPool.Put(b)
    h := sha256.Sum256([]byte(data))
    return hex.EncodeToString(h[:])
}

// CanonicalizePolicyBundle returns a deterministic compact JSON representation of a policy bundle:
// - Policies sorted by ID
// - Rules left in original order (semantic ordering may matter)
// If parsing fails it returns the raw string.
func CanonicalizePolicyBundle(raw string) string {
    b, err := ParsePolicyBundle(raw)
    if err != nil {
        return raw
    }
    sort.SliceStable(b.Policies, func(i, j int) bool { return b.Policies[i].ID < b.Policies[j].ID })
    type canonical struct {
        ID       string   `json:"id"`
        Policies []Policy `json:"policies"`
    }
    c := canonical{ID: b.ID, Policies: b.Policies}
    out, err := json.Marshal(c)
    if err != nil {
        return raw
    }
    return string(out)
}

// CanonicalPolicyBundleHash hashes the canonical representation of a policy bundle.
// Falls back to raw hashing if parsing fails.
func CanonicalPolicyBundleHash(raw string) string {
    b, err := ParsePolicyBundle(raw)
    if err != nil {
        return SHA256Hex(raw)
    }
    sort.SliceStable(b.Policies, func(i, j int) bool { return b.Policies[i].ID < b.Policies[j].ID })
    type canonical struct {
        ID       string   `json:"id"`
        Policies []Policy `json:"policies"`
    }
    c := canonical{ID: b.ID, Policies: b.Policies}
    out, err := json.Marshal(c)
    if err != nil { return SHA256Hex(raw) }
    h := sha256.Sum256(out)
    return hex.EncodeToString(h[:])
}
